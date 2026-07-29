package types

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestNodeDefinitionImageScalar(t *testing.T) {
	t.Parallel()
	var n NodeDefinition
	if err := yaml.Unmarshal([]byte("image: alpine:3\n"), &n); err != nil {
		t.Fatal(err)
	}
	if n.Image != "alpine:3" {
		t.Fatalf("got image %q", n.Image)
	}
	if n.ImageBuild != nil {
		t.Fatalf("expected nil image build, got %#v", n.ImageBuild)
	}
}

func TestNodeDefinitionImageObjectBuild(t *testing.T) {
	t.Parallel()
	var n NodeDefinition
	if err := yaml.Unmarshal([]byte(`
image:
  name: localhost/example:latest
  build:
    mode: pre-deploy
    rebuild: always
    context: ./build
    dockerfile: Dockerfile.node
`), &n); err != nil {
		t.Fatal(err)
	}
	if n.Image != "localhost/example:latest" {
		t.Fatalf("got image %q", n.Image)
	}
	if n.ImageBuild == nil {
		t.Fatal("expected image build")
	}
	if n.ImageBuild.Mode != ImageBuildModePreDeploy || n.ImageBuild.Rebuild != ImageBuildRebuildAlways {
		t.Fatalf("unexpected build config %#v", n.ImageBuild)
	}
}

func TestNodeDefinitionBuildShorthand(t *testing.T) {
	t.Parallel()
	var n NodeDefinition
	if err := yaml.Unmarshal([]byte(`
image: localhost/example:latest
build:
  mode: topology
  rebuild: if-missing
  builder:
    image: localhost/base:latest
    cmd: /usr/local/bin/build
`), &n); err != nil {
		t.Fatal(err)
	}
	if n.Image != "localhost/example:latest" {
		t.Fatalf("got image %q", n.Image)
	}
	if n.ImageBuild == nil {
		t.Fatal("expected image build")
	}
	if n.ImageBuild.Mode != ImageBuildModeTopology || n.ImageBuild.Rebuild != ImageBuildRebuildIfMissing {
		t.Fatalf("unexpected build config %#v", n.ImageBuild)
	}
}

func TestNodeDefinitionImageObjectBuildAndShorthandConflict(t *testing.T) {
	t.Parallel()
	var n NodeDefinition
	if err := yaml.Unmarshal([]byte(`
image:
  name: localhost/example:latest
  build:
    mode: pre-deploy
build:
  mode: pre-deploy
`), &n); err == nil {
		t.Fatal("expected error")
	}
}

func TestNodeDefinitionImageObjectRequiresName(t *testing.T) {
	t.Parallel()
	var n NodeDefinition
	if err := yaml.Unmarshal([]byte(`image: {build: {mode: pre-deploy}}`), &n); err == nil {
		t.Fatal("expected error")
	}
}

func TestImageBuildValidateTopologyRequiresBuilderImage(t *testing.T) {
	t.Parallel()
	b := &ImageBuildDefinition{Mode: ImageBuildModeTopology, Builder: &ImageBuildBuilder{Image: "builder:latest"}}
	if err := b.Validate("localhost/example:latest"); err == nil {
		t.Fatal("expected error")
	}
}

func TestImageBuildValidateDefaults(t *testing.T) {
	t.Parallel()
	b := &ImageBuildDefinition{}
	if err := b.Validate("localhost/example:latest"); err != nil {
		t.Fatal(err)
	}
	if b.Mode != ImageBuildModePreDeploy || b.Rebuild != ImageBuildRebuildIfMissing {
		t.Fatalf("defaults not applied: %#v", b)
	}
}

func TestImageBuildValidateTopologyRequiresBuilderCmd(t *testing.T) {
	t.Parallel()
	b := &ImageBuildDefinition{Mode: ImageBuildModeTopology, Builder: &ImageBuildBuilder{Image: "builder:latest"}}
	if err := b.Validate("localhost/example:latest"); err == nil {
		t.Fatal("expected error")
	}
}
