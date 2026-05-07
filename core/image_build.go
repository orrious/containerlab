package core

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	clabexec "github.com/srl-labs/containerlab/exec"
	clabnodes "github.com/srl-labs/containerlab/nodes"
	clabruntime "github.com/srl-labs/containerlab/runtime"
	clabtypes "github.com/srl-labs/containerlab/types"
)

func (c *CLab) validateImageBuildDefinitions() error {
	for _, node := range c.Nodes {
		cfg := node.Config()
		if err := cfg.ImageBuild.Validate(cfg.Image); err != nil {
			return fmt.Errorf("node %q image.build: %w", cfg.ShortName, err)
		}
	}
	return nil
}

func (c *CLab) buildPreDeployImages(ctx context.Context) error {
	for _, node := range c.Nodes {
		build := node.Config().ImageBuild
		if build == nil || build.Mode != clabtypes.ImageBuildModePreDeploy {
			continue
		}
		if err := c.buildNodeImage(ctx, node); err != nil {
			return err
		}
	}
	return nil
}

func (c *CLab) buildNodeImage(ctx context.Context, node clabnodes.Node) error {
	cfg := node.Config()
	build := cfg.ImageBuild
	if build == nil {
		return nil
	}
	if err := build.Validate(cfg.Image); err != nil {
		return fmt.Errorf("node %q image.build: %w", cfg.ShortName, err)
	}

	exists, err := node.GetRuntime().ImageExists(ctx, cfg.Image)
	if err != nil {
		return fmt.Errorf("node %q inspect image %q: %w", cfg.ShortName, cfg.Image, err)
	}

	switch build.Rebuild {
	case clabtypes.ImageBuildRebuildNever:
		if !exists {
			return fmt.Errorf("node %q image %q is missing and image.build.rebuild=never prevents building it", cfg.ShortName, cfg.Image)
		}
		return nil
	case clabtypes.ImageBuildRebuildIfMissing:
		if exists {
			log.Debugf("image %s present, skip build", cfg.Image)
			return nil
		}
	case clabtypes.ImageBuildRebuildAlways:
	}

	contextPath := build.Context
	if !filepath.IsAbs(contextPath) {
		contextPath = filepath.Join(c.TopoPaths.TopologyFileDir(), contextPath)
	}

	log.Info("Building image", "image", cfg.Image, "node", cfg.ShortName)
	return node.GetRuntime().BuildImage(ctx, &clabtypes.ImageBuildOptions{
		Name:       cfg.Image,
		Context:    contextPath,
		Dockerfile: build.Dockerfile,
		Network:    build.Network,
	})
}

func shouldSkipPullForBuiltNodeImage(node clabnodes.Node, imageKey string) bool {
	build := node.Config().ImageBuild
	return imageKey == clabnodes.ImageKey && build != nil && build.Mode == clabtypes.ImageBuildModePreDeploy
}

func topologyBuilderImage(node clabnodes.Node, imageKey string) (string, bool) {
	build := node.Config().ImageBuild
	if imageKey != clabnodes.ImageKey || build == nil || build.Mode != clabtypes.ImageBuildModeTopology {
		return "", false
	}
	if build.Builder == nil {
		return "", true
	}
	return build.Builder.Image, true
}

func (c *CLab) buildTopologyImage(ctx context.Context, node clabnodes.Node, _ bool, _ *clabexec.ExecCollection) error {
	cfg := node.Config()
	build := cfg.ImageBuild
	if build == nil || build.Mode != clabtypes.ImageBuildModeTopology {
		return nil
	}
	if err := build.Validate(cfg.Image); err != nil {
		return fmt.Errorf("node %q image.build: %w", cfg.ShortName, err)
	}

	exists, err := node.GetRuntime().ImageExists(ctx, cfg.Image)
	if err != nil {
		return fmt.Errorf("node %q inspect image %q: %w", cfg.ShortName, cfg.Image, err)
	}
	if build.Rebuild == clabtypes.ImageBuildRebuildNever {
		if !exists {
			return fmt.Errorf("node %q image %q is missing and image.build.rebuild=never prevents building it", cfg.ShortName, cfg.Image)
		}
		return nil
	}
	if build.Rebuild == clabtypes.ImageBuildRebuildIfMissing && exists {
		log.Debugf("image %s present, skip topology build", cfg.Image)
		return nil
	}

	origImage, origCmd, origEntrypoint := cfg.Image, cfg.Cmd, cfg.Entrypoint
	cfg.Image = build.Builder.Image
	cfg.Cmd = "sleep infinity"
	if build.Builder.Entrypoint != "" {
		cfg.Entrypoint = build.Builder.Entrypoint
	}
	defer func() {
		cfg.Image = origImage
		cfg.Cmd = origCmd
		cfg.Entrypoint = origEntrypoint
	}()

	log.Info("Starting topology image builder", "node", cfg.ShortName, "image", origImage)
	if err := node.Deploy(ctx, &clabnodes.DeployParams{Nodes: c.Nodes}); err != nil {
		return fmt.Errorf("builder deploy: %w", err)
	}
	if err := node.UpdateConfigWithRuntimeInfo(ctx); err != nil {
		log.Warnf("failed to update builder runtime information for node %s: %v", cfg.ShortName, err)
	}
	if err := node.DeployEndpoints(ctx); err != nil {
		return fmt.Errorf("builder deploy links: %w", err)
	}
	execCmd, err := clabexec.NewExecCmdFromString(build.Builder.Cmd)
	if err != nil {
		return fmt.Errorf("parse topology builder command: %w", err)
	}
	execResult, err := node.RunExec(ctx, execCmd)
	if err != nil {
		return fmt.Errorf("run topology builder command: %w", err)
	}
	if execResult.GetReturnCode() != 0 {
		return fmt.Errorf("topology builder command failed: %s", execResult)
	}
	if err := node.GetRuntime().StopContainer(ctx, cfg.LongName, clabtypes.SIGKILL); err != nil {
		return fmt.Errorf("stop topology builder: %w", err)
	}
	if err := waitForContainerStopped(ctx, node); err != nil {
		return err
	}

	containerName := node.Config().LongName
	if err := node.GetRuntime().CommitContainer(ctx, containerName, origImage, build.Commit); err != nil {
		return fmt.Errorf("commit topology builder: %w", err)
	}
	if err := removeNodeLinks(ctx, node); err != nil {
		return fmt.Errorf("remove topology builder links: %w", err)
	}
	if err := node.GetRuntime().DeleteContainer(ctx, containerName); err != nil {
		return fmt.Errorf("delete topology builder container: %w", err)
	}
	return nil
}

func waitForContainerStopped(ctx context.Context, node clabnodes.Node) error {
	for {
		status := node.GetContainerStatus(ctx)
		if status == clabruntime.Stopped {
			return nil
		}
		if status == clabruntime.NotFound {
			return fmt.Errorf("builder container %q disappeared before commit", node.Config().LongName)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func removeNodeLinks(ctx context.Context, node clabnodes.Node) error {
	seen := map[string]struct{}{}
	for _, ep := range node.GetEndpoints() {
		link := ep.GetLink()
		if link == nil {
			continue
		}
		key := fmt.Sprintf("%p", link)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if err := link.Remove(ctx); err != nil {
			return err
		}
	}
	return nil
}
