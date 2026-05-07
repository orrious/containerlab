package docker

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	buildapi "github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	dockerC "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
	clabtypes "github.com/srl-labs/containerlab/types"
	clabutils "github.com/srl-labs/containerlab/utils"
)

func (d *DockerRuntime) ImageExists(ctx context.Context, imageName string) (bool, error) {
	canonicalImageName := clabutils.GetCanonicalImageName(imageName)
	_, _, err := d.Client.ImageInspectWithRaw(ctx, canonicalImageName)
	if err == nil {
		return true, nil
	}
	if dockerC.IsErrNotFound(err) {
		return false, nil
	}
	return false, err
}

func (d *DockerRuntime) BuildImage(ctx context.Context, opts *clabtypes.ImageBuildOptions) error {
	if opts == nil {
		return fmt.Errorf("image build options are required")
	}
	buildCtx, err := tarBuildContext(opts.Context)
	if err != nil {
		return err
	}
	defer buildCtx.Close()

	buildOpts := buildapi.ImageBuildOptions{
		Tags:       []string{opts.Name},
		Dockerfile: opts.Dockerfile,
		Remove:     true,
	}
	if opts.Network != "" && opts.Network != "topology" {
		buildOpts.NetworkMode = opts.Network
	}

	resp, err := d.Client.ImageBuild(ctx, buildCtx, buildOpts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	terminalFd, isTerminal := term.GetFdInfo(os.Stdout)
	if err := jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stdout, terminalFd, isTerminal, nil); err != nil {
		return err
	}
	log.Info("Done building image", "image", opts.Name)
	return nil
}

func tarBuildContext(root string) (io.ReadCloser, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("image build context %q is not a directory", root)
	}

	pr, pw := io.Pipe()
	go func() {
		tw := tar.NewWriter(pw)
		walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if rel == "." {
				return nil
			}
			header.Name = filepath.ToSlash(rel)
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			if entry.Type().IsRegular() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				_, copyErr := io.Copy(tw, file)
				closeErr := file.Close()
				if copyErr != nil {
					return copyErr
				}
				if closeErr != nil {
					return closeErr
				}
			}
			return nil
		})
		closeErr := tw.Close()
		if walkErr != nil {
			_ = pw.CloseWithError(walkErr)
			return
		}
		_ = pw.CloseWithError(closeErr)
	}()
	return pr, nil
}

func (d *DockerRuntime) CommitContainer(ctx context.Context, containerName string, imageName string, commit *clabtypes.ImageBuildCommit) error {
	var config *container.Config
	if commit != nil {
		config = &container.Config{
			Entrypoint: commit.Entrypoint,
			Cmd:        commit.Cmd,
		}
	}

	_, err := d.Client.ContainerCommit(ctx, containerName, container.CommitOptions{
		Reference: imageName,
		Config:    config,
	})
	return err
}
