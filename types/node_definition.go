package types

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	importEnvsKey = "__IMPORT_ENVS"
)

// NodeCredentials holds login material for SSH/NETCONF/GNMI/etc. (topology
// defaults/kinds/groups/nodes).
type NodeCredentials struct {
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"-" yaml:"password,omitempty"`
}

// NodeDefinition represents a configuration a given node can have in the lab definition file.
type NodeDefinition struct {
	Kind                  string                `yaml:"kind,omitempty"`
	Group                 string                `yaml:"group,omitempty"`
	Type                  string                `yaml:"type,omitempty"`
	StartupConfig         string                `yaml:"startup-config,omitempty"`
	StartupDelay          uint                  `yaml:"startup-delay,omitempty"`
	EnforceStartupConfig  *bool                 `yaml:"enforce-startup-config,omitempty"`
	SuppressStartupConfig *bool                 `yaml:"suppress-startup-config,omitempty"`
	AutoRemove            *bool                 `yaml:"auto-remove,omitempty"`
	RestartPolicy         string                `yaml:"restart-policy,omitempty"`
	Config                *ConfigDispatcher     `yaml:"config,omitempty"`
	Image                 string                `yaml:"image,omitempty"`
	ImageBuild            *ImageBuildDefinition `yaml:"-"`
	ImagePullPolicy       string                `yaml:"image-pull-policy,omitempty"`
	License               string                `yaml:"license,omitempty"`
	Position              string                `yaml:"position,omitempty"`
	Entrypoint            string                `yaml:"entrypoint,omitempty"`
	Cmd                   string                `yaml:"cmd,omitempty"`
	// list of commands to run in container
	Exec []string `yaml:"exec,omitempty"`
	// list of bind mount compatible strings
	Binds []string `yaml:"binds,omitempty"`
	// list of devices to map in the container
	Devices []string `yaml:"devices,omitempty"`
	// List of capabilities to add for the container
	CapAdd []string `yaml:"cap-add,omitempty"`
	// Set the shared memory size allocated to the container
	ShmSize string `yaml:"shm-size,omitempty"`
	// list of port bindings
	Ports []string `yaml:"ports,omitempty"`
	// user-defined IPv4 address in the management network
	MgmtIPv4 string `yaml:"mgmt-ipv4,omitempty"`
	// user-defined IPv6 address in the management network
	MgmtIPv6 string `yaml:"mgmt-ipv6,omitempty"`
	// environment variables
	Env map[string]string `yaml:"env,omitempty"`
	// external file containing environment variables
	EnvFiles []string `yaml:"env-files,omitempty"`
	// linux user used in a container
	User string `yaml:"user,omitempty"`
	// container labels
	Labels map[string]string `yaml:"labels,omitempty"`
	// container networking mode. if set to `host` the host networking will be used for this node,
	//  else bridged network
	NetworkMode string `yaml:"network-mode,omitempty"`
	// container cgroup namespace mode. Valid values are runtime dependent, typically host or private.
	CgroupnsMode string `yaml:"cgroupns-mode,omitempty"`
	// Override container runtime
	Runtime string `yaml:"runtime,omitempty"`
	// Set node CPU (cgroup or hypervisor)
	CPU float64 `yaml:"cpu,omitempty"`
	// Set node CPUs to use
	CPUSet string `yaml:"cpu-set,omitempty"`
	// Set node Memory (cgroup or hypervisor)
	Memory string `yaml:"memory,omitempty"`
	// Set the nodes Sysctl
	Sysctls map[string]string `yaml:"sysctls,omitempty"`
	// Extra options, may be kind specific
	Extras *Extras `yaml:"extras,omitempty"`
	// Deployment stages
	Stages *Stages `yaml:"stages,omitempty"`
	// DNS configuration
	DNS *DNSConfig `yaml:"dns,omitempty"`
	// Certificate configuration
	Certificate *CertificateConfig `yaml:"certificate,omitempty"`
	// Healthcheck configuration
	HealthCheck *HealthcheckConfig `yaml:"healthcheck,omitempty"`
	// Credentials for SSH/NETCONF/GNMI/etc. (overrides kind default when set).
	Credentials NodeCredentials `yaml:"credentials,omitempty"`
	// Network aliases
	Aliases    []string     `yaml:"aliases,omitempty"`
	Components []*Component `yaml:"components,omitempty"`
}

// Interface compliance.
var _ yaml.Unmarshaler = &NodeDefinition{}

// UnmarshalYAML is a custom unmarshaler for NodeDefinition type that allows to map old attributes
// to new ones.
func (n *NodeDefinition) UnmarshalYAML(unmarshal func(any) error) error {
	// define an alias type to avoid recursion during unmarshaling
	type NodeDefinitionAlias NodeDefinition

	// NodeDefinitionWithDeprecatedFields can contain fields that are deprecated
	// but still supported for backward compatibility.
	type NodeDefinitionWithDeprecatedFields struct {
		NodeDefinitionAlias `yaml:",inline"`
		LegacyUsername      string `yaml:"username,omitempty"`
		LegacyPassword      string `yaml:"password,omitempty"`
	}

	raw := map[interface{}]interface{}{}
	if err := unmarshal(raw); err != nil {
		return err
	}

	imageBuild, err := normalizeImageDefinition(raw)
	if err != nil {
		return err
	}
	nodeBuild, err := normalizeNodeBuildDefinition(raw)
	if err != nil {
		return err
	}
	if imageBuild != nil && nodeBuild != nil {
		return fmt.Errorf("image.build and build cannot both be set on the same node")
	}

	encoded, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}

	nd := &NodeDefinitionWithDeprecatedFields{}
	nd.NodeDefinitionAlias = NodeDefinitionAlias(*n)
	if err := yaml.Unmarshal(encoded, nd); err != nil {
		return err
	}

	*n = NodeDefinition(nd.NodeDefinitionAlias)
	n.ImageBuild = imageBuild
	if nodeBuild != nil {
		n.ImageBuild = nodeBuild
	}

	if nd.LegacyUsername != "" && n.Credentials.Username == "" {
		n.Credentials.Username = nd.LegacyUsername
	}
	if nd.LegacyPassword != "" && n.Credentials.Password == "" {
		n.Credentials.Password = nd.LegacyPassword
	}

	return nil
}

func normalizeNodeBuildDefinition(raw map[interface{}]interface{}) (*ImageBuildDefinition, error) {
	buildRaw, ok := raw["build"]
	if !ok {
		return nil, nil
	}
	encoded, err := yaml.Marshal(buildRaw)
	if err != nil {
		return nil, err
	}
	build := &ImageBuildDefinition{}
	if err := yaml.Unmarshal(encoded, build); err != nil {
		return nil, err
	}
	return build, nil
}

func normalizeImageDefinition(raw map[interface{}]interface{}) (*ImageBuildDefinition, error) {
	imageRaw, ok := raw["image"]
	if !ok {
		return nil, nil
	}

	switch image := imageRaw.(type) {
	case string:
		return nil, nil
	case map[interface{}]interface{}:
		nameRaw, ok := image["name"]
		if !ok {
			return nil, fmt.Errorf("image.name is required when image is an object")
		}
		name, ok := nameRaw.(string)
		if !ok || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("image.name must be a non-empty string")
		}

		var build *ImageBuildDefinition
		if buildRaw, ok := image["build"]; ok {
			encoded, err := yaml.Marshal(buildRaw)
			if err != nil {
				return nil, err
			}
			build = &ImageBuildDefinition{}
			if err := yaml.Unmarshal(encoded, build); err != nil {
				return nil, err
			}
			if err := build.Validate(name); err != nil {
				return nil, err
			}
		}

		raw["image"] = name
		return build, nil
	default:
		return nil, fmt.Errorf("image must be a string or an object")
	}
}

// ImportEnvs imports all environment variables defined in the shell
// if __IMPORT_ENVS is set to true.
func (n *NodeDefinition) ImportEnvs() {
	if n == nil || n.Env == nil {
		return
	}

	var importEnvs bool

	for k, v := range n.Env {
		if k == importEnvsKey && v == "true" {
			importEnvs = true
			break
		}
	}

	if !importEnvs {
		return
	}

	for _, e := range os.Environ() {
		kv := strings.SplitN(e, "=", 2)
		if len(kv) < 2 {
			continue
		}
		if _, exists := n.Env[kv[0]]; exists {
			continue
		}

		n.Env[kv[0]] = kv[1]
	}
}
