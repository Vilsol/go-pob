package data

import (
	"context"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

func init() {
	config.InitLogging(false)

	if err := poe.InitializeAll(context.Background(), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		panic(err)
	}
}

func TestLoadTreeGraph(t *testing.T) {
	TreeVersions[TreeVersion3_18].getGraph()
}

func TestCalculateAllocationPaths(t *testing.T) {
	// Starting from the witch spell damage root, path up through both
	// the small int nodes and the cast speed small nodes, in both paths
	// stopping just short of Arcanist's Dominion.
	activeNonRootNodes := []int64{33296, 1957, 739, 18866, 37569, 36542, 4397}
	activeRootNodes := []int64{57264}   // witch spell damage root
	inactiveRootNodes := []int64{57226} // witch ES root

	activeNodes := append(activeNonRootNodes, activeRootNodes...)
	rootNodes := append(activeRootNodes, inactiveRootNodes...)

	actual := TreeVersions[TreeVersion3_18].CalculateAllocationPaths(activeNodes, rootNodes)

	for _, node := range activeNonRootNodes {
		testza.AssertEqual(t, actual[node], int64(-1), "Active non-root nodes should be mapped to -1")
	}
	for _, node := range inactiveRootNodes {
		testza.AssertEqual(t, actual[node], int64(-1), "Inactive root nodes should be mapped to -1")
	}
	for _, node := range activeRootNodes {
		testza.AssertEqual(t, actual[node], int64(-1), "Active root nodes should be mapped to -1")
	}

	// Small str node below Enduring Bond wheel should point to the small int node to its right
	testza.AssertEqual(t, actual[31875], int64(4397), "Neighbor of active node should point to the neighboring active")

	// ES+mana node adjacent to the inactive ES+mana root should point to that root
	testza.AssertEqual(t, actual[59650], int64(57226), "Neighbor of inactive root node should point to the neighboring root")

	// Arcanist's Dominion is adjacent to both small int node 4397 and cast speed node 18866
	testza.AssertEqual(t, actual[11420], int64(4397), "Neighbor of multiple active nodes should point to the one with the lowest node ID")

	// 27929 Deep Wisdom is distance 4 to all of the following:
	// - small int node 4397 (via 11420 Arcanist's Dominion, 60554 Minion Damage, 32024 Minion Life)
	// - cast speed node 18866 (also via 11420 Arcanist's Dominion, 60554 Minion Damage, 32024 Minion Life)
	// - inactive ES+mana root 57226 (via small int nodes 21678, 32210, 8948)
	//
	// Even though the smallest-node-ID next hop is small int node 8948, it should recognize that because
	// the ES+mana root is inactive, it needs to be treated as though that path requires 1 additional distance
	// than the path to the already-active frontier would.
	testza.AssertEqual(t, actual[27929], int64(32024), "Node equidistant from active node and inactive root should point towards active node")

	// Heart of the Warrior should connect to the small node above it
	testza.AssertEqual(t, actual[61198], int64(20551), "Faraway node with uniquely best path should connect to that path")

	// Bloodletting should connect to the lower-ID'd of the two small nodes it touches
	testza.AssertEqual(t, actual[26294], int64(17833), "Faraway node with multiple shortest-paths should connect to the one with the lowest node ID")

	// Bottom-right small jewel node should point to the corresponding jewel socket
	testza.AssertEqual(t, actual[44470], int64(12161), "Position proxies should still appear in the graph")

	// Mystic Talents can only be reached via anointment
	_, ok := actual[62596]
	testza.AssertFalse(t, ok, "Disconnected nodes should not be included in the output")

	// Forbidden Power is an ascendancy node, it isn't connected to the main graph
	_, ok = actual[62504]
	testza.AssertFalse(t, ok, "Disconnected nodes should not be included in the output")

	testza.AssertLen(t, actual, 2075, "All connected nodes should have a path")
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
	// Starting from ranger attack speed root node, pathing to 'Path of the Hunter'
	activeNonRootNodes := []int64{35179, 9009, 19506}
	activeRootNodes := []int64{15144}
	activeNodes := append(activeNonRootNodes, activeRootNodes...)

	actual := TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	testza.AssertLen(t, actual, 0, "There should be no prunable nodes")

	// Delete the node before 'Path of the Hunter' which should cause it to be pruned
	activeNonRootNodes = []int64{35179, 19506}
	activeNodes = append(activeNonRootNodes, activeRootNodes...)

	actual = TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	expected := []int64{19506}
	testza.AssertEqual(t, expected, actual, "Node should be pruned")
}

func TestCalculatePrunableNodesMultiPath(t *testing.T) {
	// Starting from both ranger root nodes, path to 'Path of the Hunter'
	activeNonRootNodes := []int64{35179, 60532, 9009, 19506}
	activeRootNodes := []int64{15144, 62103}
	activeNodes := append(activeNonRootNodes, activeRootNodes...)

	actual := TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	testza.AssertLen(t, actual, 0, "There should be no prunable nodes (1)")

	// Delete the 2nd node after the top root node which wouldn't require anything
	// to be pruned
	activeNonRootNodes = []int64{60532, 9009, 19506}
	activeNodes = append(activeNonRootNodes, activeRootNodes...)

	actual = TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	testza.AssertLen(t, actual, 0, "There should be no prunable nodes (2)")

	// Delete the bottom root node which should cause all non-root nodes to be pruned
	activeRootNodes = []int64{15144}
	activeNodes = append(activeNonRootNodes, activeRootNodes...)

	actual = TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	expected := []int64{60532, 9009, 19506}
	testza.AssertEqual(t, expected, actual, "All non-root nodes should be pruned")
}

func TestCalculatePrunableNodesCircular(t *testing.T) {
	// Starting from ranger attack speed root node, path the entire circle between
	// 'Path of the Hunter' and 'Reflexes'
	activeNonRootNodes := []int64{35179, 9009, 19506, 8348, 6534, 20812, 44103, 32091, 17814, 60204}
	activeRootNodes := []int64{15144}
	activeNodes := append(activeNonRootNodes, activeRootNodes...)

	actual := TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	testza.AssertLen(t, actual, 0, "There should be no prunable nodes (1)")

	// Delete the node below 'Reflexes' which wouldn't require anything to be pruned
	activeNonRootNodes = []int64{35179, 9009, 19506, 8348, 6534, 20812, 44103, 17814, 60204}
	activeNodes = append(activeNonRootNodes, activeRootNodes...)

	actual = TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	testza.AssertLen(t, actual, 0, "There should be no prunable nodes (2)")

	// Delete the 'Path of the Hunter' node which should cause all nodes in the circle
	// to be pruned
	activeNonRootNodes = []int64{35179, 9009, 8348, 6534, 20812, 44103, 17814, 60204}
	activeNodes = append(activeNonRootNodes, activeRootNodes...)

	actual = TreeVersions[TreeVersion3_18].CalculatePrunableNodes(activeNodes, activeRootNodes)
	expected := []int64{8348, 6534, 20812, 44103, 17814, 60204}
	testza.AssertEqual(t, expected, actual, "All nodes in the circle should be pruned")
}
