package podman

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/containers/buildah/define"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	podmantypes "github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/distribution/reference"
	dockercontainer "github.com/docker/docker/api/types/container"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	clabtypes "github.com/srl-labs/containerlab/types"
	clabutils "github.com/srl-labs/containerlab/utils"
)

func (r *PodmanRuntime) ImageExists(ctx context.Context, imageName string) (bool, error) {
	ctx, err := r.connect(ctx)
	if err != nil {
		return false, err
	}
	return images.Exists(ctx, clabutils.GetCanonicalImageName(imageName), &images.ExistsOptions{})
}

func (r *PodmanRuntime) BuildImage(ctx context.Context, opts *clabtypes.ImageBuildOptions) error {
	if opts == nil {
		return fmt.Errorf("image build options are required")
	}
	ctx, err := r.connect(ctx)
	if err != nil {
		return err
	}

	buildOpts := podmantypes.BuildOptions{}
	buildOpts.ContextDirectory = opts.Context
	buildOpts.Output = opts.Name
	buildOpts.Out = os.Stdout
	buildOpts.Err = os.Stderr
	applyBuildNetwork(&buildOpts, opts.Network)

	containerfile := opts.Dockerfile
	if !filepath.IsAbs(containerfile) {
		containerfile = filepath.Join(opts.Context, containerfile)
	}
	if _, err := images.Build(ctx, []string{containerfile}, buildOpts); err != nil {
		return err
	}
	log.Info("Done building image", "image", opts.Name)
	return nil
}

func applyBuildNetwork(opts *podmantypes.BuildOptions, networkName string) {
	if networkName == "" || networkName == "topology" {
		return
	}

	network := define.NamespaceOption{Name: string(specs.NetworkNamespace)}
	switch networkName {
	case "host":
		network.Host = true
		opts.ConfigureNetwork = define.NetworkEnabled
	case "none":
		opts.ConfigureNetwork = define.NetworkDisabled
	default:
		network.Path = networkName
		opts.ConfigureNetwork = define.NetworkEnabled
	}
	opts.NamespaceOptions.AddOrReplace(network)
}

func (r *PodmanRuntime) CommitContainer(
	ctx context.Context,
	containerName string,
	imageName string,
	commit *clabtypes.ImageBuildCommit,
) error {
	ctx, err := r.connect(ctx)
	if err != nil {
		return err
	}

	repo, tag, err := splitImageName(imageName)
	if err != nil {
		return err
	}
	options := new(containers.CommitOptions).WithRepo(repo).WithTag(tag)
	if commit != nil {
		config, err := json.Marshal(&dockercontainer.Config{
			Entrypoint: commit.Entrypoint,
			Cmd:        commit.Cmd,
		})
		if err != nil {
			return fmt.Errorf("marshal commit config: %w", err)
		}
		options.WithConfig(bytes.NewReader(config))
	}

	if _, err := containers.Commit(ctx, containerName, options); err != nil {
		return fmt.Errorf("commit container %q as image %q: %w", containerName, imageName, err)
	}
	return nil
}

func splitImageName(imageName string) (string, string, error) {
	named, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return "", "", fmt.Errorf("parse image name %q: %w", imageName, err)
	}
	named = reference.TagNameOnly(named)
	tagged, ok := named.(reference.Tagged)
	if !ok {
		return "", "", fmt.Errorf("image name %q has no tag", imageName)
	}
	return named.Name(), tagged.Tag(), nil
}
