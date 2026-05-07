package types

import (
	"fmt"
	"strings"
)

const (
	ImageBuildModePreDeploy ImageBuildMode = "pre-deploy"
	ImageBuildModeTopology  ImageBuildMode = "topology"

	ImageBuildRebuildIfMissing ImageBuildRebuild = "if-missing"
	ImageBuildRebuildAlways    ImageBuildRebuild = "always"
	ImageBuildRebuildNever     ImageBuildRebuild = "never"
)

type ImageBuildMode string

type ImageBuildRebuild string

type ImageBuildDefinition struct {
	Mode       ImageBuildMode     `json:"mode,omitempty" yaml:"mode,omitempty"`
	Rebuild    ImageBuildRebuild  `json:"rebuild,omitempty" yaml:"rebuild,omitempty"`
	Context    string             `json:"context,omitempty" yaml:"context,omitempty"`
	Dockerfile string             `json:"dockerfile,omitempty" yaml:"dockerfile,omitempty"`
	Network    string             `json:"network,omitempty" yaml:"network,omitempty"`
	Builder    *ImageBuildBuilder `json:"builder,omitempty" yaml:"builder,omitempty"`
	Commit     *ImageBuildCommit  `json:"commit,omitempty" yaml:"commit,omitempty"`
}

type ImageBuildBuilder struct {
	Image      string `json:"image,omitempty" yaml:"image,omitempty"`
	Entrypoint string `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	Cmd        string `json:"cmd,omitempty" yaml:"cmd,omitempty"`
}

type ImageBuildCommit struct {
	Entrypoint []string `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	Cmd        []string `json:"cmd,omitempty" yaml:"cmd,omitempty"`
}

type ImageBuildOptions struct {
	Name       string
	Context    string
	Dockerfile string
	Network    string
}

func (b *ImageBuildDefinition) Normalize() {
	if b == nil {
		return
	}
	if b.Mode == "" {
		b.Mode = ImageBuildModePreDeploy
	}
	if b.Rebuild == "" {
		b.Rebuild = ImageBuildRebuildIfMissing
	}
	if b.Context == "" {
		b.Context = "."
	}
	if b.Dockerfile == "" {
		b.Dockerfile = "Dockerfile"
	}
}

func (b *ImageBuildDefinition) Validate(imageName string) error {
	if b == nil {
		return nil
	}
	b.Normalize()
	if strings.TrimSpace(imageName) == "" {
		return fmt.Errorf("image.name is required when image.build is set")
	}
	switch b.Mode {
	case ImageBuildModePreDeploy, ImageBuildModeTopology:
	default:
		return fmt.Errorf("unsupported image.build.mode %q", b.Mode)
	}
	switch b.Rebuild {
	case ImageBuildRebuildIfMissing, ImageBuildRebuildAlways, ImageBuildRebuildNever:
	default:
		return fmt.Errorf("unsupported image.build.rebuild %q", b.Rebuild)
	}
	if b.Mode == ImageBuildModeTopology {
		if b.Builder == nil || strings.TrimSpace(b.Builder.Image) == "" {
			return fmt.Errorf("image.build.builder.image is required when image.build.mode is topology")
		}
		if strings.TrimSpace(b.Builder.Cmd) == "" {
			return fmt.Errorf("image.build.builder.cmd is required when image.build.mode is topology")
		}
	}
	return nil
}
