//go:build linux && podman
// +build linux,podman

package podman

import (
	"testing"

	"github.com/containers/buildah/define"
	podmantypes "github.com/containers/podman/v5/pkg/domain/entities/types"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func TestSplitImageName(t *testing.T) {
	tests := []struct {
		name      string
		imageName string
		wantRepo  string
		wantTag   string
	}{
		{
			name:      "localhost with port and tag",
			imageName: "localhost:5000/lab/wic:stage2",
			wantRepo:  "localhost:5000/lab/wic",
			wantTag:   "stage2",
		},
		{
			name:      "default tag",
			imageName: "localhost/lab/wic",
			wantRepo:  "localhost/lab/wic",
			wantTag:   "latest",
		},
		{
			name:      "short image",
			imageName: "wic:stage2",
			wantRepo:  "docker.io/library/wic",
			wantTag:   "stage2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, tag, err := splitImageName(tt.imageName)
			if err != nil {
				t.Fatalf("splitImageName returned error: %v", err)
			}
			if repo != tt.wantRepo || tag != tt.wantTag {
				t.Fatalf("splitImageName = (%q, %q), want (%q, %q)", repo, tag, tt.wantRepo, tt.wantTag)
			}
		})
	}
}

func TestApplyBuildNetwork(t *testing.T) {
	tests := []struct {
		name       string
		network    string
		wantPolicy define.NetworkConfigurationPolicy
		wantHost   bool
		wantPath   string
		wantOption bool
	}{
		{name: "default", wantPolicy: define.NetworkDefault},
		{name: "topology", network: "topology", wantPolicy: define.NetworkDefault},
		{name: "host", network: "host", wantPolicy: define.NetworkEnabled, wantHost: true, wantOption: true},
		{name: "none", network: "none", wantPolicy: define.NetworkDisabled, wantOption: true},
		{name: "named", network: "build-net", wantPolicy: define.NetworkEnabled, wantPath: "build-net", wantOption: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &podmantypes.BuildOptions{}
			applyBuildNetwork(opts, tt.network)
			if opts.ConfigureNetwork != tt.wantPolicy {
				t.Fatalf("ConfigureNetwork = %v, want %v", opts.ConfigureNetwork, tt.wantPolicy)
			}
			var network *define.NamespaceOption
			for i := range opts.NamespaceOptions {
				if opts.NamespaceOptions[i].Name == string(specs.NetworkNamespace) {
					network = &opts.NamespaceOptions[i]
				}
			}
			if !tt.wantOption {
				if network != nil {
					t.Fatalf("network namespace option = %#v, want nil", network)
				}
				return
			}
			if network == nil {
				t.Fatal("network namespace option is nil")
			}
			if network.Host != tt.wantHost || network.Path != tt.wantPath {
				t.Fatalf("network namespace option = %#v, want host=%v path=%q", network, tt.wantHost, tt.wantPath)
			}
		})
	}
}
