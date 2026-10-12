package data

import (
	"context"
	"slices"
	"strconv"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

func init() {
	config.InitLogging(false)

	if err := poe.InitializeAll(raw.DataContext(context.Background()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		panic(err)
	}
}

func TestLoadTreeGraph(t *testing.T) {
	TreeVersions[TreeVersion3_18].getGraph()
}

func TestCalculateAllocationPaths(t *testing.T) {
	activeNonRootNodes := []int64{33296, 1957, 739, 18866, 37569, 36542, 4397}
	activeRootNodes := []int64{57264}
	inactiveRootNodes := []int64{57226}

	activeNodes := slices.Concat(activeNonRootNodes, activeRootNodes)
	rootNodes := slices.Concat(activeRootNodes, inactiveRootNodes)

	actual := TreeVersions[TreeVersion3_18].CalculateAllocationPaths(activeNodes, rootNodes)

	for _, node := range activeNonRootNodes {
		testza.AssertEqual(t, actual[node], int64(-1), "Active non-root nodes should be mapped to -1")
	}
	for _, node := range inactiveRootNodes {
		testza.AssertNotEqual(t, actual[node], int64(-1), "Inactive root nodes hop to their class start")
	}
	for _, node := range activeRootNodes {
		testza.AssertEqual(t, actual[node], int64(-1), "Active root nodes should be mapped to -1")
	}

	testza.AssertEqual(t, actual[31875], int64(4397), "Neighbor of active node should point to the neighboring active")

	testza.AssertEqual(t, actual[59650], int64(57226), "Neighbor of inactive root node should point to the neighboring root")

	testza.AssertEqual(t, actual[11420], int64(4397), "Neighbor of multiple active nodes should point to the one with the lowest node ID")

	testza.AssertEqual(t, actual[27929], int64(32024), "Node equidistant from active node and inactive root should point towards active node")

	testza.AssertEqual(t, actual[61198], int64(20551), "Faraway node with uniquely best path should connect to that path")

	testza.AssertEqual(t, actual[26294], int64(17833), "Faraway node with multiple shortest-paths should connect to the one with the lowest node ID")

	_, ok := actual[44470]
	testza.AssertFalse(t, ok, "PassiveSpec.lua:44 holds no position proxy, so none is on a path")

	_, ok = actual[62596]
	testza.AssertFalse(t, ok, "Disconnected nodes should not be included in the output")

	_, ok = actual[62504]
	testza.AssertFalse(t, ok, "Disconnected nodes should not be included in the output")

	testza.AssertLen(t, actual, 1992, "All connected spec nodes should have a path")
}

func TestCalculateAllocationPathsStability(t *testing.T) {
	lastResult := map[int64]int64{}
	for i := 0; i < 20; i++ {
		result := TreeVersions[TreeVersion3_18].CalculateAllocationPaths([]int64{48828, 55373, 2151, 47062, 15144, 62103}, []int64{48828})
		if i > 0 {
			testza.AssertEqual(t, result, lastResult, "Results should be stable between runs")
		}
		lastResult = result
	}
}

func BenchmarkCalculateAllocationPaths(b *testing.B) {
	TreeVersions[TreeVersion3_18].getGraph()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TreeVersions[TreeVersion3_18].CalculateAllocationPaths([]int64{48828, 55373, 2151, 47062, 15144, 62103}, []int64{48828})
	}
}

func TestCalculatePrunableNodesSinglePath(t *testing.T) {
	activeNonRootNodes := []int64{35179, 9009, 19506}
	activeRootNodes := []int64{15144}
	activeNodes := slices.Concat(activeNonRootNodes, activeRootNodes)

	actual := TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	testza.AssertLen(t, actual, 0, "There should be no prunable nodes")

	activeNonRootNodes = []int64{35179, 19506}
	activeNodes = slices.Concat(activeNonRootNodes, activeRootNodes)

	actual = TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	expected := []int64{19506}
	testza.AssertEqual(t, expected, actual, "Node should be pruned")
}

func TestCalculatePrunableNodesMultiPath(t *testing.T) {
	activeNonRootNodes := []int64{35179, 60532, 9009, 19506}
	activeRootNodes := []int64{15144, 62103}
	activeNodes := slices.Concat(activeNonRootNodes, activeRootNodes)

	actual := TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	testza.AssertLen(t, actual, 0, "There should be no prunable nodes (1)")

	activeNonRootNodes = []int64{60532, 9009, 19506}
	activeNodes = slices.Concat(activeNonRootNodes, activeRootNodes)

	actual = TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	testza.AssertLen(t, actual, 0, "There should be no prunable nodes (2)")

	activeRootNodes = []int64{15144}
	activeNodes = slices.Concat(activeNonRootNodes, activeRootNodes)

	actual = TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	expected := []int64{60532, 9009, 19506}
	testza.AssertEqual(t, expected, actual, "All non-root nodes should be pruned")
}

func TestCalculatePrunableNodesCircular(t *testing.T) {
	activeNonRootNodes := []int64{35179, 9009, 19506, 8348, 6534, 20812, 44103, 32091, 17814, 60204}
	activeRootNodes := []int64{15144}
	activeNodes := slices.Concat(activeNonRootNodes, activeRootNodes)

	actual := TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	testza.AssertLen(t, actual, 0, "There should be no prunable nodes (1)")

	activeNonRootNodes = []int64{35179, 9009, 19506, 8348, 6534, 20812, 44103, 17814, 60204}
	activeNodes = slices.Concat(activeNonRootNodes, activeRootNodes)

	actual = TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	testza.AssertLen(t, actual, 0, "There should be no prunable nodes (2)")

	activeNonRootNodes = []int64{35179, 9009, 8348, 6534, 20812, 44103, 17814, 60204}
	activeNodes = slices.Concat(activeNonRootNodes, activeRootNodes)

	actual = TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	expected := []int64{8348, 6534, 20812, 44103, 17814, 60204}
	testza.AssertEqual(t, expected, actual, "All nodes in the circle should be pruned")
}

// TestCalculateAllocationPathsInactiveRootHopsToClassStart: PoB allocates the class start.
func TestCalculateAllocationPathsInactiveRootHopsToClassStart(t *testing.T) {
	root := int64(57226)
	actual := TreeVersions[TreeVersion3_18].CalculateAllocationPaths(nil, []int64{root})
	start := actual[root]
	node, ok := TreeVersions[TreeVersion3_18].Tree().Nodes[strconv.FormatInt(start, 10)]
	if !ok || node.ClassStartIndex == nil {
		t.Fatalf("root %d hops to %d, not its class start", root, start)
	}
	testza.AssertEqual(t, actual[start], int64(-1), "the class start is the anchor")
	testza.AssertEqual(t, actual[59650], root, "a neighbour of the root hops through it")
}
