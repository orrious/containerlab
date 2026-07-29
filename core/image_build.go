package core

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	clabexec "github.com/srl-labs/containerlab/exec"
	clablinks "github.com/srl-labs/containerlab/links"
	clabnodes "github.com/srl-labs/containerlab/nodes"
	clabruntime "github.com/srl-labs/containerlab/runtime"
	clabtypes "github.com/srl-labs/containerlab/types"
	clabutils "github.com/srl-labs/containerlab/utils"
	"github.com/vishvananda/netns"
)

type imageBuildTarget struct {
	Key      string
	Image    string
	Build    *clabtypes.ImageBuildDefinition
	Node     clabnodes.Node
	NodeName string
	Source   string
}

func (c *CLab) validateImageBuildDefinitions() error {
	targets, err := c.resolveImageBuildTargets()
	if err != nil {
		return err
	}
	c.setImageBuildTargets(targets)
	return nil
}

func (c *CLab) ensureImageBuildTargets() error {
	if c.imageBuildTargets != nil {
		return nil
	}
	targets, err := c.resolveImageBuildTargets()
	if err != nil {
		return err
	}
	c.setImageBuildTargets(targets)
	return nil
}

func (c *CLab) setImageBuildTargets(targets []*imageBuildTarget) {
	c.imageBuildTargets = targets
	c.managedImageNames = make(map[string]struct{}, len(targets))
	for _, target := range targets {
		c.managedImageNames[target.Image] = struct{}{}
	}
}

func (c *CLab) resolveImageBuildTargets() ([]*imageBuildTarget, error) {
	targets := make([]*imageBuildTarget, 0)
	byImage := map[string]*imageBuildTarget{}

	addTarget := func(target *imageBuildTarget) error {
		target.Image = strings.TrimSpace(target.Image)
		if target.Image == "" {
			return fmt.Errorf("%s.image is required when build is set", target.Source)
		}
		target.Build.Normalize()
		if target.Build.Mode == clabtypes.ImageBuildModeTopology {
			if err := c.bindTopologyImageBuildTarget(target); err != nil {
				return err
			}
		}
		if err := target.Build.Validate(target.Image); err != nil {
			return fmt.Errorf("%s.build: %w", target.Source, err)
		}

		if existing, ok := byImage[target.Image]; ok {
			if !reflect.DeepEqual(existing.Build, target.Build) {
				return fmt.Errorf(
					"image build target %q conflicts with %s; duplicate image %q must use an identical build definition",
					target.Source,
					existing.Source,
					target.Image,
				)
			}
			return nil
		}
		byImage[target.Image] = target
		targets = append(targets, target)
		return nil
	}

	rootImageKeys := make([]string, 0, len(c.Config.Images))
	for key := range c.Config.Images {
		rootImageKeys = append(rootImageKeys, key)
	}
	sort.Strings(rootImageKeys)
	for _, key := range rootImageKeys {
		imageDef := c.Config.Images[key]
		source := fmt.Sprintf("images.%s", key)
		if imageDef == nil {
			return nil, fmt.Errorf("%s must not be empty", source)
		}
		if strings.TrimSpace(imageDef.Image) == "" {
			return nil, fmt.Errorf("%s.image is required", source)
		}
		if imageDef.Build == nil {
			continue
		}
		if err := addTarget(&imageBuildTarget{
			Key:    key,
			Image:  imageDef.Image,
			Build:  imageDef.Build,
			Source: source,
		}); err != nil {
			return nil, err
		}
	}

	nodeNames := make([]string, 0, len(c.Nodes))
	for nodeName := range c.Nodes {
		nodeNames = append(nodeNames, nodeName)
	}
	sort.Strings(nodeNames)
	for _, nodeName := range nodeNames {
		node := c.Nodes[nodeName]
		cfg := node.Config()
		build := cfg.ImageBuild
		if build == nil {
			continue
		}
		if build.Node != "" && build.Node != nodeName {
			return nil, fmt.Errorf("node %q build.node must either be omitted or match the node name", nodeName)
		}
		build.Node = nodeName
		if err := addTarget(&imageBuildTarget{
			Key:      nodeName,
			Image:    cfg.Image,
			Build:    build,
			Node:     node,
			NodeName: nodeName,
			Source:   fmt.Sprintf("node %q", nodeName),
		}); err != nil {
			return nil, err
		}
	}

	return targets, nil
}

func (c *CLab) bindTopologyImageBuildTarget(target *imageBuildTarget) error {
	if target.Build.Node != "" {
		node, ok := c.Nodes[target.Build.Node]
		if !ok {
			return fmt.Errorf("%s.build.node %q references an unknown node", target.Source, target.Build.Node)
		}
		target.Node = node
		target.NodeName = target.Build.Node
		return nil
	}
	if target.NodeName != "" {
		target.Build.Node = target.NodeName
		return nil
	}

	consumers := make([]string, 0)
	for nodeName, node := range c.Nodes {
		if node.Config().Image == target.Image {
			consumers = append(consumers, nodeName)
		}
	}
	sort.Strings(consumers)
	switch len(consumers) {
	case 0:
		return fmt.Errorf("%s.build.node is required because no node consumes image %q", target.Source, target.Image)
	case 1:
		target.NodeName = consumers[0]
		target.Node = c.Nodes[consumers[0]]
		target.Build.Node = consumers[0]
		return nil
	default:
		return fmt.Errorf(
			"%s.build.node is required because image %q is consumed by multiple nodes: %s",
			target.Source,
			target.Image,
			strings.Join(consumers, ", "),
		)
	}
}

func (c *CLab) buildPreDeployImages(ctx context.Context) error {
	if err := c.ensureImageBuildTargets(); err != nil {
		return err
	}
	for _, target := range c.imageBuildTargets {
		if target.Build.Mode != clabtypes.ImageBuildModePreDeploy {
			continue
		}
		if err := c.buildImageTarget(ctx, target); err != nil {
			return err
		}
	}
	return nil
}

func (c *CLab) buildImageTarget(ctx context.Context, target *imageBuildTarget) error {
	build := target.Build
	if err := build.Validate(target.Image); err != nil {
		return fmt.Errorf("%s.build: %w", target.Source, err)
	}

	rt := c.globalRuntime()
	if target.Node != nil {
		rt = target.Node.GetRuntime()
	}

	exists, err := rt.ImageExists(ctx, target.Image)
	if err != nil {
		return fmt.Errorf("%s inspect image %q: %w", target.Source, target.Image, err)
	}

	switch build.Rebuild {
	case clabtypes.ImageBuildRebuildNever:
		if !exists {
			return fmt.Errorf("%s image %q is missing and build.rebuild=never prevents building it", target.Source, target.Image)
		}
		return nil
	case clabtypes.ImageBuildRebuildIfMissing:
		if exists {
			log.Debugf("image %s present, skip build", target.Image)
			return nil
		}
	case clabtypes.ImageBuildRebuildAlways:
	}

	contextPath := build.Context
	if !filepath.IsAbs(contextPath) {
		contextPath = filepath.Join(c.TopoPaths.TopologyFileDir(), contextPath)
	}

	log.Info("Building image", "image", target.Image, "source", target.Source)
	return rt.BuildImage(ctx, &clabtypes.ImageBuildOptions{
		Name:       target.Image,
		Context:    contextPath,
		Dockerfile: build.Dockerfile,
		Network:    build.Network,
	})
}

func (c *CLab) shouldSkipPullForManagedImage(imageKey, imageName string) bool {
	if imageKey != clabnodes.ImageKey {
		return false
	}
	if err := c.ensureImageBuildTargets(); err != nil {
		return false
	}
	_, ok := c.managedImageNames[imageName]
	return ok
}

func (c *CLab) topologyBuilderImagesForNode(node clabnodes.Node) ([]string, error) {
	if err := c.ensureImageBuildTargets(); err != nil {
		return nil, err
	}
	nodeName := node.Config().ShortName
	images := make([]string, 0)
	seen := map[string]struct{}{}
	for _, target := range c.imageBuildTargets {
		if target.Build.Mode != clabtypes.ImageBuildModeTopology || target.NodeName != nodeName {
			continue
		}
		if target.Build.Builder == nil || target.Build.Builder.Image == "" {
			continue
		}
		image := target.Build.Builder.Image
		if _, managed := c.managedImageNames[image]; managed {
			continue
		}
		if _, ok := seen[image]; ok {
			continue
		}
		seen[image] = struct{}{}
		images = append(images, image)
	}
	sort.Strings(images)
	return images, nil
}

func (c *CLab) topologyTargetsForNode(node clabnodes.Node) ([]*imageBuildTarget, error) {
	if err := c.ensureImageBuildTargets(); err != nil {
		return nil, err
	}
	nodeName := node.Config().ShortName
	targets := make([]*imageBuildTarget, 0)
	for _, target := range c.imageBuildTargets {
		if target.Build.Mode == clabtypes.ImageBuildModeTopology && target.NodeName == nodeName {
			targets = append(targets, target)
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Source < targets[j].Source
	})
	return targets, nil
}

func (c *CLab) buildTopologyImage(ctx context.Context, node clabnodes.Node, _ bool, _ *clabexec.ExecCollection) error {
	targets, err := c.topologyTargetsForNode(node)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if err := c.buildTopologyImageTarget(ctx, target); err != nil {
			return err
		}
	}
	return nil
}

func (c *CLab) buildTopologyImageTarget(ctx context.Context, target *imageBuildTarget) error {
	node := target.Node
	cfg := node.Config()
	build := target.Build
	if err := build.Validate(target.Image); err != nil {
		return fmt.Errorf("%s.build: %w", target.Source, err)
	}

	exists, err := node.GetRuntime().ImageExists(ctx, target.Image)
	if err != nil {
		return fmt.Errorf("%s inspect image %q: %w", target.Source, target.Image, err)
	}
	if build.Rebuild == clabtypes.ImageBuildRebuildNever {
		if !exists {
			return fmt.Errorf("%s image %q is missing and build.rebuild=never prevents building it", target.Source, target.Image)
		}
		return nil
	}
	if build.Rebuild == clabtypes.ImageBuildRebuildIfMissing && exists {
		log.Debugf("image %s present, skip topology build", target.Image)
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

	log.Info("Starting topology image builder", "node", cfg.ShortName, "image", target.Image)
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
	if err := parkTopologyBuilderLinks(ctx, node); err != nil {
		return fmt.Errorf("park topology builder links: %w", err)
	}
	if err := node.GetRuntime().StopContainer(ctx, cfg.LongName, clabtypes.SIGKILL); err != nil {
		return fmt.Errorf("stop topology builder: %w", err)
	}
	if err := waitForContainerStopped(ctx, node); err != nil {
		return err
	}

	containerName := node.Config().LongName
	if err := node.GetRuntime().CommitContainer(ctx, containerName, target.Image, build.Commit); err != nil {
		return fmt.Errorf("commit topology builder: %w", err)
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

func topologyImageParkingNetNSName(node clabnodes.Node) string {
	return clabutils.ParkingNetnsName(node.Config().LongName)
}

func topologyImageParkingNode(node clabnodes.Node, create bool) (*clablinks.ParkingNode, error) {
	var (
		path string
		err  error
	)
	if create {
		path, err = clabutils.CreateOrGetNamedNetNS(topologyImageParkingNetNSName(node))
	} else {
		path, err = clabutils.GetNamedNetNS(topologyImageParkingNetNSName(node))
	}
	if err != nil {
		return nil, err
	}

	return clablinks.NewParkingNode(node.Config().LongName, path), nil
}

func parkTopologyBuilderLinks(ctx context.Context, node clabnodes.Node) error {
	parkingNode, err := topologyImageParkingNode(node, true)
	if err != nil {
		return err
	}

	moved := make([]clablinks.Endpoint, 0, len(node.GetEndpoints()))
	for _, ep := range node.GetEndpoints() {
		link := ep.GetLink()
		if link == nil {
			continue
		}
		// A veth whose endpoints are both owned by the builder cannot be
		// moved into another namespace one endpoint at a time: moving the
		// second endpoint back beside its peer fails with EEXIST. It has no
		// external peer to preserve, so remove it after parking external
		// links and let the final node's normal DeployEndpoints pass recreate
		// it from the topology.
		if topologyImageLinkIsInternal(link, node) {
			continue
		}
		if link.GetType() != clablinks.LinkTypeVEth {
			for i := len(moved) - 1; i >= 0; i-- {
				_ = clablinks.RestoreParkedEndpointInterface(ctx, parkingNode, moved[i])
			}
			_ = netns.DeleteNamed(topologyImageParkingNetNSName(node))
			return fmt.Errorf(
				"node %q endpoint %q is linked via %q, but topology image parking supports only veth links",
				node.Config().ShortName,
				ep.GetIfaceName(),
				link.GetType(),
			)
		}

		if err := clablinks.ParkEndpointInterface(ctx, ep, parkingNode); err != nil {
			for i := len(moved) - 1; i >= 0; i-- {
				_ = clablinks.RestoreParkedEndpointInterface(ctx, parkingNode, moved[i])
			}
			_ = netns.DeleteNamed(topologyImageParkingNetNSName(node))
			return fmt.Errorf("park endpoint %q: %w", ep.GetIfaceName(), err)
		}
		moved = append(moved, ep)
	}

	removed := make(map[clablinks.Link]clablinks.Endpoint)
	for _, ep := range node.GetEndpoints() {
		link := ep.GetLink()
		if link == nil || !topologyImageLinkIsInternal(link, node) {
			continue
		}
		if _, ok := removed[link]; ok {
			continue
		}
		if err := link.Remove(ctx); err != nil {
			for removedLink, removedEP := range removed {
				_ = removedLink.Deploy(ctx, removedEP)
			}
			for i := len(moved) - 1; i >= 0; i-- {
				_ = clablinks.RestoreParkedEndpointInterface(ctx, parkingNode, moved[i])
			}
			_ = netns.DeleteNamed(topologyImageParkingNetNSName(node))
			return fmt.Errorf("remove builder-internal link for endpoint %q: %w", ep.GetIfaceName(), err)
		}
		removed[link] = ep
	}

	return nil
}

func restoreTopologyBuilderLinks(ctx context.Context, node clabnodes.Node) error {
	parkingNode, err := topologyImageParkingNode(node, false)
	if err != nil {
		return nil
	}

	restored := make([]clablinks.Endpoint, 0, len(node.GetEndpoints()))
	for _, ep := range node.GetEndpoints() {
		if ep.GetLink() == nil ||
			ep.GetLink().GetType() != clablinks.LinkTypeVEth ||
			topologyImageLinkIsInternal(ep.GetLink(), node) {
			continue
		}
		if err := clablinks.RestoreParkedEndpointInterface(ctx, parkingNode, ep); err != nil {
			for i := len(restored) - 1; i >= 0; i-- {
				_ = clablinks.ParkEndpointInterface(ctx, restored[i], parkingNode)
			}
			return err
		}
		restored = append(restored, ep)
	}

	if err := netns.DeleteNamed(topologyImageParkingNetNSName(node)); err != nil {
		return fmt.Errorf("cleanup topology image parking netns: %w", err)
	}

	return nil
}

func topologyImageLinkIsInternal(link clablinks.Link, node clabnodes.Node) bool {
	endpoints := link.GetRuntimeEndpoints()
	if len(endpoints) < 2 {
		return false
	}
	for _, ep := range endpoints {
		if ep.GetNode() == nil || ep.GetNode().GetShortName() != node.GetShortName() {
			return false
		}
	}
	return true
}
