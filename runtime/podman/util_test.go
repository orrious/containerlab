//go:build linux && podman
// +build linux,podman

package podman

import (
	"context"
	"testing"

	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/srl-labs/containerlab/types"
)

func TestCreateContainerSpecAppliesRuntimeOptions(t *testing.T) {
	r := &PodmanRuntime{mgmt: &types.MgmtNet{Network: "clab"}}
	cfg := &types.NodeConfig{
		LongName:     "clab-test-node1",
		ShortName:    "node1",
		Image:        "localhost/test:latest",
		Labels:       map[string]string{},
		NetworkMode:  "host",
		Privileged:   false,
		CapAdd:       []string{"NET_ADMIN"},
		CgroupnsMode: "host",
		ShmSize:      "64m",
		Tmpfs:        map[string]string{"/run": "rw,nosuid,nodev", "/run/lock": "rw"},
		SecurityOpts: []string{"no-new-privileges"},
		ExtraHosts:   []string{"example:127.0.0.1"},
	}

	sg, err := r.createContainerSpec(context.Background(), cfg)
	if err != nil {
		t.Fatalf("createContainerSpec() error = %v", err)
	}
	if sg.Privileged == nil || *sg.Privileged {
		t.Fatalf("Privileged = %v, want false", sg.Privileged)
	}
	if sg.CgroupNS.NSMode != specgen.Host {
		t.Fatalf("CgroupNS mode = %q, want host", sg.CgroupNS.NSMode)
	}
	if sg.ShmSize == nil || *sg.ShmSize != 64*1000*1000 {
		t.Fatalf("ShmSize = %v, want 64000000", sg.ShmSize)
	}
	if sg.NoNewPrivileges == nil || !*sg.NoNewPrivileges {
		t.Fatalf("NoNewPrivileges = %v, want true", sg.NoNewPrivileges)
	}
	if len(sg.CapAdd) != 1 || sg.CapAdd[0] != "NET_ADMIN" {
		t.Fatalf("CapAdd = %v, want NET_ADMIN", sg.CapAdd)
	}

	tmpfs := map[string]bool{}
	for _, mount := range sg.Mounts {
		if mount.Type == "tmpfs" {
			tmpfs[mount.Destination] = true
		}
	}
	if !tmpfs["/run"] || !tmpfs["/run/lock"] {
		t.Fatalf("tmpfs mounts = %#v, want /run and /run/lock", tmpfs)
	}
}

func TestApplySecurityOptsRejectsUnsupportedOption(t *testing.T) {
	if err := applySecurityOpts([]string{"unsupported=value"}, &specgen.ContainerSecurityConfig{}); err == nil {
		t.Fatal("applySecurityOpts() error = nil, want unsupported option error")
	}
}
