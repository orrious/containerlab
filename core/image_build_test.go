package core

import (
	"strings"
	"testing"

	clabmocksmocknodes "github.com/srl-labs/containerlab/mocks/mocknodes"
	clabnodes "github.com/srl-labs/containerlab/nodes"
	clabtypes "github.com/srl-labs/containerlab/types"
	"go.uber.org/mock/gomock"
	"gopkg.in/yaml.v2"
)

func TestConfigParsesRootImages(t *testing.T) {
	t.Parallel()
	var cfg Config
	err := yaml.UnmarshalStrict([]byte(`
name: img-lab
images:
  cm-base:
    image: localhost/cm-base:latest
    build:
      mode: pre-deploy
      context: ./cm-base
topology:
  nodes:
    card-01:
      kind: linux
      image: localhost/cm-base:latest
`), &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Images["cm-base"].Image != "localhost/cm-base:latest" {
		t.Fatalf("unexpected root image %#v", cfg.Images["cm-base"])
	}
	if cfg.Images["cm-base"].Build == nil || cfg.Images["cm-base"].Build.Context != "./cm-base" {
		t.Fatalf("unexpected build %#v", cfg.Images["cm-base"].Build)
	}
}

func TestResolveImageBuildTargetsDedupesIdenticalDefinitions(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	nodeBuild := topologyBuild()
	c := &CLab{
		Config: &Config{
			Images: map[string]*clabtypes.ImageDefinition{
				"card": {
					Image: "localhost/card:stage3",
					Build: topologyBuild(),
				},
			},
		},
		Nodes: map[string]clabnodes.Node{
			"card-01": imageBuildTestNode(ctrl, "card-01", "localhost/card:stage3", nodeBuild),
		},
	}

	targets, err := c.resolveImageBuildTargets()
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected one deduped target, got %d", len(targets))
	}
	if targets[0].NodeName != "card-01" {
		t.Fatalf("expected target bound to card-01, got %q", targets[0].NodeName)
	}
}

func TestResolveImageBuildTargetsRejectsConflictingDuplicateImages(t *testing.T) {
	t.Parallel()
	c := &CLab{
		Config: &Config{
			Images: map[string]*clabtypes.ImageDefinition{
				"one": {
					Image: "localhost/shared:latest",
					Build: &clabtypes.ImageBuildDefinition{Context: "./one"},
				},
				"two": {
					Image: "localhost/shared:latest",
					Build: &clabtypes.ImageBuildDefinition{Context: "./two"},
				},
			},
		},
	}

	_, err := c.resolveImageBuildTargets()
	if err == nil || !strings.Contains(err.Error(), "duplicate image") {
		t.Fatalf("expected duplicate image error, got %v", err)
	}
}

func TestResolveImageBuildTargetsRejectsAmbiguousTopologyBinding(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	c := &CLab{
		Config: &Config{
			Images: map[string]*clabtypes.ImageDefinition{
				"card": {
					Image: "localhost/card:stage3",
					Build: topologyBuild(),
				},
			},
		},
		Nodes: map[string]clabnodes.Node{
			"card-01": imageBuildTestNode(ctrl, "card-01", "localhost/card:stage3", nil),
			"card-02": imageBuildTestNode(ctrl, "card-02", "localhost/card:stage3", nil),
		},
	}

	_, err := c.resolveImageBuildTargets()
	if err == nil || !strings.Contains(err.Error(), "consumed by multiple nodes") {
		t.Fatalf("expected ambiguous topology binding error, got %v", err)
	}
}

func TestResolveImageBuildTargetsRejectsUnknownNodeBinding(t *testing.T) {
	t.Parallel()
	c := &CLab{
		Config: &Config{
			Images: map[string]*clabtypes.ImageDefinition{
				"card": {
					Image: "localhost/card:stage3",
					Build: func() *clabtypes.ImageBuildDefinition {
						b := topologyBuild()
						b.Node = "missing"
						return b
					}(),
				},
			},
		},
		Nodes: map[string]clabnodes.Node{},
	}

	_, err := c.resolveImageBuildTargets()
	if err == nil || !strings.Contains(err.Error(), "references an unknown node") {
		t.Fatalf("expected unknown node error, got %v", err)
	}
}

func TestResolveImageBuildTargetsRejectsNodeBuildWithoutImage(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	c := &CLab{
		Config: &Config{},
		Nodes: map[string]clabnodes.Node{
			"card-01": imageBuildTestNode(
				ctrl,
				"card-01",
				"",
				&clabtypes.ImageBuildDefinition{Mode: clabtypes.ImageBuildModePreDeploy},
			),
		},
	}

	_, err := c.resolveImageBuildTargets()
	if err == nil || !strings.Contains(err.Error(), "image is required") {
		t.Fatalf("expected missing image error, got %v", err)
	}
}

func imageBuildTestNode(
	ctrl *gomock.Controller,
	name string,
	image string,
	build *clabtypes.ImageBuildDefinition,
) clabnodes.Node {
	node := clabmocksmocknodes.NewMockNode(ctrl)
	node.EXPECT().Config().Return(&clabtypes.NodeConfig{
		ShortName:  name,
		LongName:   "clab-test-" + name,
		Image:      image,
		ImageBuild: build,
		Stages:     clabtypes.NewStages(),
	}).AnyTimes()
	return node
}

func topologyBuild() *clabtypes.ImageBuildDefinition {
	return &clabtypes.ImageBuildDefinition{
		Mode: clabtypes.ImageBuildModeTopology,
		Builder: &clabtypes.ImageBuildBuilder{
			Image: "localhost/base:latest",
			Cmd:   "/usr/local/bin/build",
		},
	}
}
