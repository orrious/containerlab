package core

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	claberrors "github.com/srl-labs/containerlab/errors"
	clabnodes "github.com/srl-labs/containerlab/nodes"
	clabruntime "github.com/srl-labs/containerlab/runtime"
	clabtypes "github.com/srl-labs/containerlab/types"
)

// lifecycleNodes resolves the requested node names without applying lifecycle policy.
func (c *CLab) lifecycleNodes(nodeNames []string) ([]clabnodes.Node, error) {
	var nodes []clabnodes.Node

	if len(nodeNames) > 0 {
		for _, name := range nodeNames {
			n, ok := c.Nodes[name]
			if !ok {
				// node filter case, where referenced node in filter is not in topo file
				return nil, fmt.Errorf(
					"%w: node %q is not present in the topology",
					claberrors.ErrIncorrectInput,
					name,
				)
			}

			nodes = append(nodes, n)
		}
	} else {
		for _, n := range c.Nodes {
			nodes = append(nodes, n)
		}
	}

	return nodes, nil
}

func (c *CLab) lifecycleStartNodes(
	nodeNames []string,
) ([]clabnodes.Node, map[string]clabtypes.WaitForList, error) {
	selected := map[string]struct{}{}
	deps := map[string]clabtypes.WaitForList{}

	var addWithDependencies func(string) error
	addWithDependencies = func(name string) error {
		node, ok := c.Nodes[name]
		if !ok {
			return fmt.Errorf("%w: node %q is not present in the topology", claberrors.ErrIncorrectInput, name)
		}
		if _, exists := selected[name]; exists {
			return nil
		}
		selected[name] = struct{}{}

		for _, dep := range c.lifecycleDependencies(node) {
			if _, ok := c.Nodes[dep.Node]; !ok {
				return fmt.Errorf("dependee node %q not found", dep.Node)
			}
			deps[name] = append(deps[name], dep)
			if err := addWithDependencies(dep.Node); err != nil {
				return err
			}
		}
		return nil
	}

	if len(nodeNames) == 0 {
		nodeNames = make([]string, 0, len(c.Nodes))
		for name := range c.Nodes {
			nodeNames = append(nodeNames, name)
		}
		slices.Sort(nodeNames)
	}
	for _, name := range nodeNames {
		if err := addWithDependencies(name); err != nil {
			return nil, nil, err
		}
	}

	orderedNames, err := lifecycleToposort(selected, deps)
	if err != nil {
		return nil, nil, err
	}
	nodes := make([]clabnodes.Node, 0, len(orderedNames))
	for _, name := range orderedNames {
		nodes = append(nodes, c.Nodes[name])
	}
	return nodes, deps, nil
}

func (c *CLab) lifecycleDependencies(node clabnodes.Node) clabtypes.WaitForList {
	var deps clabtypes.WaitForList
	if stages := node.Config().Stages; stages != nil {
		for _, waitForNodes := range stages.GetWaitFor() {
			deps = append(deps, waitForNodes...)
		}
	}

	netMode := strings.SplitN(node.Config().NetworkMode, ":", 2) //nolint:mnd
	if len(netMode) == 2 && netMode[0] == "container" {
		if _, exists := c.Nodes[netMode[1]]; exists {
			deps = append(deps, &clabtypes.WaitFor{Node: netMode[1], Stage: clabtypes.WaitForCreate})
		}
	}
	return deps
}

func lifecycleToposort(
	selected map[string]struct{},
	deps map[string]clabtypes.WaitForList,
) ([]string, error) {
	ordered := make([]string, 0, len(selected))
	visiting := map[string]bool{}
	visited := map[string]bool{}

	var visit func(string) error
	visit = func(name string) error {
		if visited[name] {
			return nil
		}
		if visiting[name] {
			return fmt.Errorf("cyclic lifecycle dependencies found at node %q", name)
		}
		visiting[name] = true
		for _, dep := range deps[name] {
			if _, ok := selected[dep.Node]; ok {
				if err := visit(dep.Node); err != nil {
					return err
				}
			}
		}
		visiting[name] = false
		visited[name] = true
		ordered = append(ordered, name)
		return nil
	}

	names := make([]string, 0, len(selected))
	for name := range selected {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func (c *CLab) waitForLifecycleNodeHealthy(ctx context.Context, nodeName, depName string) error {
	dep, ok := c.Nodes[depName]
	if !ok {
		return fmt.Errorf("dependee node %q not found", depName)
	}

	var deadline time.Time
	if c.timeout > 0 {
		deadline = time.Now().Add(c.timeout)
	}
	for {
		healthy, err := dep.IsHealthy(ctx)
		if err != nil {
			return fmt.Errorf("node %q waiting for node %q healthy stage: %w", nodeName, depName, err)
		}
		if healthy {
			return nil
		}
		if !deadline.IsZero() && time.Now().After(deadline) {
			return fmt.Errorf("node %q timed out waiting for node %q healthy stage", nodeName, depName)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func (c *CLab) parkRecreatedNodes(ctx context.Context, plan *applyPlan) error {
	for _, nodeName := range sortedStringSet(plan.parkedNodeSet) {
		node, exists := c.Nodes[nodeName]
		if !exists {
			continue
		}
		parker, ok := node.(interface{ ParkEndpoints(context.Context) error })
		if !ok {
			return fmt.Errorf("node %q does not support endpoint parking", nodeName)
		}
		log.Info("Parking links for recreate", "node", nodeName)
		if err := parker.ParkEndpoints(ctx); err != nil {
			return fmt.Errorf("failed parking endpoints for node %q: %w", nodeName, err)
		}
	}

	return nil
}

func (c *CLab) restoreRecreatedNodes(ctx context.Context, plan *applyPlan) error {
	for _, nodeName := range sortedStringSet(plan.parkedNodeSet) {
		node, exists := c.Nodes[nodeName]
		if !exists {
			continue
		}
		restorer, ok := node.(interface{ RestoreEndpoints(context.Context) error })
		if !ok {
			return fmt.Errorf("node %q does not support endpoint restore", nodeName)
		}
		log.Info("Restoring links after recreate", "node", nodeName)
		if err := restorer.RestoreEndpoints(ctx); err != nil {
			return fmt.Errorf("failed restoring endpoints for node %q: %w", nodeName, err)
		}
	}

	return nil
}

func (c *CLab) startStoppedNodes(ctx context.Context, plan *applyPlan) error {
	for _, nodeName := range sortedStringSet(plan.startNodeSet) {
		node, exists := c.Nodes[nodeName]
		if !exists {
			continue
		}
		log.Info("Starting stopped node", "node", nodeName)
		if err := node.Start(ctx); err != nil {
			return fmt.Errorf("failed starting node %q: %w", nodeName, err)
		}
	}

	return nil
}

func (c *CLab) restartApplyNodes(
	ctx context.Context,
	nodeSet map[string]struct{},
) error {
	nodeNames := sortedStringSet(nodeSet)

	for _, nodeName := range nodeNames {
		node := c.Nodes[nodeName]
		if node == nil {
			continue
		}

		log.Info("Restarting node after link apply", "node", nodeName)
		if err := node.Stop(ctx); err != nil {
			return err
		}
		if err := node.Start(ctx); err != nil {
			return err
		}
		if err := c.waitNodeRunning(ctx, node); err != nil {
			return err
		}
		if err := c.waitNodeHealthyIfAvailable(ctx, node); err != nil {
			return err
		}
	}

	return nil
}

func (c *CLab) waitNodeRunning(ctx context.Context, node clabnodes.Node) error {
	deadline := time.Now().Add(c.timeout)

	for {
		status := node.GetContainerStatus(ctx)
		if clabruntime.ContainerHasJoinableNetns(status) {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf(
				"node %q did not reach running state, current state %s",
				node.GetShortName(),
				status,
			)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func (c *CLab) waitNodeHealthyIfAvailable(ctx context.Context, node clabnodes.Node) error {
	if node.Config().Healthcheck == nil {
		return nil
	}

	deadline := time.Now().Add(c.timeout)

	for {
		healthy, err := node.IsHealthy(ctx)
		if healthy {
			return nil
		}
		if err != nil && strings.Contains(err.Error(), "no health information") {
			return nil
		}
		if err != nil {
			log.Debugf("error checking node %q health: %v", node.GetShortName(), err)
		}

		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("node %q did not become healthy: %w", node.GetShortName(), err)
			}
			return fmt.Errorf("node %q did not become healthy", node.GetShortName())
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
