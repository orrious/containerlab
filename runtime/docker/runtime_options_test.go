package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	clabtypes "github.com/srl-labs/containerlab/types"
)

func TestProcessCgroupnsMode(t *testing.T) {
	hostConfig := &container.HostConfig{}
	err := (&DockerRuntime{}).processCgroupnsMode(&clabtypes.NodeConfig{CgroupnsMode: "host"}, hostConfig)
	if err != nil {
		t.Fatalf("processCgroupnsMode() error = %v", err)
	}
	if hostConfig.CgroupnsMode != container.CgroupnsModeHost {
		t.Fatalf("CgroupnsMode = %q, want host", hostConfig.CgroupnsMode)
	}

	err = (&DockerRuntime{}).processCgroupnsMode(&clabtypes.NodeConfig{CgroupnsMode: "invalid"}, hostConfig)
	if err == nil {
		t.Fatal("processCgroupnsMode() error = nil, want invalid mode error")
	}
}
