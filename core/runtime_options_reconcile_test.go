package core

import (
	"testing"

	clabtypes "github.com/srl-labs/containerlab/types"
)

func TestResolveNodeConfigIncludesRuntimeOptions(t *testing.T) {
	privileged := false
	topo := &clabtypes.Topology{
		Kinds: map[string]*clabtypes.NodeDefinition{"linux": {}},
		Nodes: map[string]*clabtypes.NodeDefinition{"node1": {
			Kind:         "linux",
			Privileged:   &privileged,
			CgroupnsMode: "host",
			PidMode:      "host",
			Tmpfs:        map[string]string{"/run": "rw"},
			SecurityOpts: []string{"seccomp=unconfined"},
		}},
	}

	cfg := resolveNodeConfigFromTopology(topo, "node1")
	if cfg.Privileged {
		t.Fatal("Privileged = true, want false")
	}
	if cfg.CgroupnsMode != "host" || cfg.PidMode != "host" {
		t.Fatalf("namespace modes = %q/%q, want host/host", cfg.CgroupnsMode, cfg.PidMode)
	}
	if cfg.Tmpfs["/run"] != "rw" {
		t.Fatalf("Tmpfs = %#v, want /run=rw", cfg.Tmpfs)
	}
	if len(cfg.SecurityOpts) != 1 || cfg.SecurityOpts[0] != "seccomp=unconfined" {
		t.Fatalf("SecurityOpts = %#v", cfg.SecurityOpts)
	}
	for _, field := range []string{"Privileged", "CgroupnsMode", "PidMode", "Tmpfs", "SecurityOpts"} {
		if !clabtypes.RecreateFields[field] {
			t.Fatalf("RecreateFields[%q] = false, want true", field)
		}
	}
}
