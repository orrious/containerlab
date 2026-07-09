package core

import (
	"context"
	"slices"
	"testing"
	"time"

	clabmocksmocknodes "github.com/srl-labs/containerlab/mocks/mocknodes"
	clabnodes "github.com/srl-labs/containerlab/nodes"
	clabtypes "github.com/srl-labs/containerlab/types"
	"go.uber.org/mock/gomock"
)

func TestLifecycleStartNodesIncludesDependenciesInOrder(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	c := CLab{Nodes: getNodeMap(mockCtrl)}

	nodes, deps, err := c.lifecycleStartNodes([]string{"node5"})
	if err != nil {
		t.Fatalf("lifecycleStartNodes() error = %v", err)
	}
	got := make([]string, 0, len(nodes))
	for _, node := range nodes {
		got = append(got, node.Config().ShortName)
	}
	want := []string{"node1", "node2", "node3", "node4", "node5"}
	if !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if len(deps["node5"]) != 2 {
		t.Fatalf("node5 dependencies = %d, want 2", len(deps["node5"]))
	}
}

func TestLifecycleStartNodesIncludesContainerNetworkDependency(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	c := CLab{Nodes: getNodeMap(mockCtrl)}
	_, deps, err := c.lifecycleStartNodes([]string{"node3"})
	if err != nil {
		t.Fatalf("lifecycleStartNodes() error = %v", err)
	}
	for _, dep := range deps["node3"] {
		if dep.Node == "node2" && dep.Stage == clabtypes.WaitForCreate {
			return
		}
	}
	t.Fatalf("node3 dependencies = %#v, want node2 create", deps["node3"])
}

func TestLifecycleStartNodesRejectsCycles(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	nodes := getNodeMap(mockCtrl)
	nodes["node1"].Config().Stages.Create.WaitFor = clabtypes.WaitForList{
		&clabtypes.WaitFor{Node: "node2", Stage: clabtypes.WaitForCreate},
	}
	c := CLab{Nodes: nodes}
	if _, _, err := c.lifecycleStartNodes([]string{"node2"}); err == nil {
		t.Fatal("lifecycleStartNodes() error = nil, want cycle error")
	}
}

func TestLifecycleStartNodesRejectsMissingDependency(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	nodes := getNodeMap(mockCtrl)
	nodes["node1"].Config().Stages.Create.WaitFor = clabtypes.WaitForList{
		&clabtypes.WaitFor{Node: "missing", Stage: clabtypes.WaitForCreate},
	}
	c := CLab{Nodes: nodes}
	if _, _, err := c.lifecycleStartNodes([]string{"node1"}); err == nil {
		t.Fatal("lifecycleStartNodes() error = nil, want missing dependency error")
	}
}

func TestWaitForLifecycleNodeHealthy(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	node := clabmocksmocknodes.NewMockNode(mockCtrl)
	node.EXPECT().IsHealthy(gomock.Any()).Return(true, nil)
	c := CLab{Nodes: map[string]clabnodes.Node{"dependency": node}, timeout: time.Second}
	if err := c.waitForLifecycleNodeHealthy(context.Background(), "dependent", "dependency"); err != nil {
		t.Fatalf("waitForLifecycleNodeHealthy() error = %v", err)
	}
}
