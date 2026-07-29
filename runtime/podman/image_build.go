package podman

import (
	"context"
	"fmt"

	"github.com/containers/podman/v5/pkg/bindings/images"
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

func (*PodmanRuntime) BuildImage(context.Context, *clabtypes.ImageBuildOptions) error {
	return fmt.Errorf("image.build is not supported by the podman runtime yet")
}

func (*PodmanRuntime) CommitContainer(context.Context, string, string, *clabtypes.ImageBuildCommit) error {
	return fmt.Errorf("image.build mode topology is not supported by the podman runtime yet")
}
