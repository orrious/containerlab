//go:build linux && podman
// +build linux,podman

package podman

import "testing"

func TestSelectPodmanSocket(t *testing.T) {
	tests := []struct {
		name       string
		runtimeDir string
		exists     bool
		want       string
	}{
		{name: "rootful", want: "/run/podman/podman.sock"},
		{name: "rootless socket present", runtimeDir: "/run/user/1000", exists: true, want: "/run/user/1000/podman/podman.sock"},
		{name: "rootless socket absent", runtimeDir: "/run/user/1000", want: "/run/podman/podman.sock"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectPodmanSocket(tt.runtimeDir, func(path string) bool {
				if tt.runtimeDir != "" && path != tt.runtimeDir+"/podman/podman.sock" {
					t.Fatalf("exists path = %q", path)
				}
				return tt.exists
			})
			if got != tt.want {
				t.Fatalf("selectPodmanSocket() = %q, want %q", got, tt.want)
			}
		})
	}
}
