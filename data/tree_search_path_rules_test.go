package data

import (
	"sort"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestAllocationPathsObeyPoBPathRules pins the three conjuncts of PassiveSpec.lua:926 that CalculateAllocationPaths did not carry.
//
//	node.type ~= "Mastery" and other.type ~= "ClassStart" and other.type ~= "AscendClassStart"
func TestAllocationPathsObeyPoBPathRules(t *testing.T) {
	versions := make([]string, 0, len(TreeVersions))
	for v := range TreeVersions {
		versions = append(versions, string(v))
	}
	sort.Strings(versions)
	if len(versions) == 0 {
		t.Fatal("REFUSING: no tree versions are registered, so this test would assert over nothing")
	}

	for _, name := range versions {
		t.Run(name, func(t *testing.T) {
			v := TreeVersions[pob.TreeVersion(name)]
			nodes := v.Tree().Nodes

			starts := make([]int64, 0, 7)
			for id, node := range nodes {
				if node.ClassStartIndex != nil {
					if n, err := strconv.ParseInt(id, 10, 64); err == nil {
						starts = append(starts, n)
					}
				}
			}
			sort.Slice(starts, func(a, b int) bool { return starts[a] < starts[b] })
			if len(starts) == 0 {
				t.Fatalf("REFUSING: %s has no classStartIndex node, so the fixture has no root", name)
			}
			root := starts[0]

			nextHops := v.CalculateAllocationPaths(nil, []int64{root})
			if len(nextHops) < 500 {
				t.Fatalf("REFUSING: the BFS reached %d nodes from class start %d; the fixture is "+
					"not exercising the tree", len(nextHops), root)
			}

			nodeFor := func(id int64) (Node, bool) {
				n, ok := nodes[strconv.FormatInt(id, 10)]
				return n, ok
			}
			isMastery := func(n Node) bool { return n.IsMastery != nil && *n.IsMastery }
			isClassStart := func(n Node) bool { return n.ClassStartIndex != nil }
			isAscendStart := func(n Node) bool { return n.IsAscendancyStart != nil && *n.IsAscendancyStart }
			ascend := func(n Node) string {
				if n.AscendancyName == nil {
					return ""
				}
				return *n.AscendancyName
			}

			ids := make([]int64, 0, len(nextHops))
			for id := range nextHops {
				ids = append(ids, id)
			}
			sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })

			var viaMastery, intoStart, crossAscend int
			for _, id := range ids {
				parent := nextHops[id]
				if parent == -1 {
					continue
				}
				child, okC := nodeFor(id)
				par, okP := nodeFor(parent)
				if !okC || !okP {
					continue
				}

				if isMastery(par) {
					viaMastery++
				}
				if isClassStart(child) || isAscendStart(child) {
					intoStart++
				}
				if ascend(child) != ascend(par) {
					crossAscend++
				}
			}

			if viaMastery > 0 {
				t.Errorf("%s: %d nodes are reached THROUGH a mastery node — PassiveSpec.lua:926's "+
					"`node.type ~= \"Mastery\"` conjunct is missing", name, viaMastery)
			}
			if intoStart > 0 {
				t.Errorf("%s: %d nodes reached are themselves a ClassStart or AscendClassStart — "+
					"PassiveSpec.lua:926 never enqueues one", name, intoStart)
			}
			if crossAscend > 0 {
				t.Errorf("%s: %d edges cross an ascendancy boundary at distance >= 1 — "+
					"PassiveSpec.lua:926 allows that only from a distance-0 node into the main tree",
					name, crossAscend)
			}
		})
	}
}

// TestAscendancyEdgesAreDroppedAtGraphBuild pins the reason rule 3 of PassiveSpec.lua:926 is NOT implemented in CalculateAllocationPaths.
func TestAscendancyEdgesAreDroppedAtGraphBuild(t *testing.T) {
	versions := make([]string, 0, len(TreeVersions))
	for v := range TreeVersions {
		versions = append(versions, string(v))
	}
	sort.Strings(versions)
	if len(versions) == 0 {
		t.Fatal("REFUSING: no tree versions are registered")
	}

	for _, name := range versions {
		t.Run(name, func(t *testing.T) {
			v := TreeVersions[pob.TreeVersion(name)]
			nodes := v.Tree().Nodes
			_, adjacency := v.getGraph()

			ascendancy := 0
			crossing := 0
			ids := make([]string, 0, len(nodes))
			for id := range nodes {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			for _, id := range ids {
				node := nodes[id]
				if node.AscendancyName == nil {
					continue
				}
				ascendancy++
				n, err := strconv.ParseInt(id, 10, 64)
				if err != nil {
					continue
				}
				for adj := range adjacency[n] {
					other, ok := nodes[strconv.FormatInt(adj, 10)]
					if !ok {
						continue
					}
					if other.AscendancyName == nil || *other.AscendancyName != *node.AscendancyName {
						crossing++
					}
				}
			}

			if ascendancy == 0 {
				t.Fatalf("REFUSING: %s has no ascendancy nodes, so the count below means nothing", name)
			}
			if crossing != 0 {
				t.Errorf("%s: %d ascendancy-crossing edges exist among %d ascendancy nodes. "+
					"tree_versions.go:323-327 no longer drops them, so PassiveSpec.lua:926's rule-3 "+
					"conjunct — including the `curDist == 0` exit into the main tree — must now be "+
					"ported into CalculateAllocationPaths, which is the only place that knows curDist.",
					name, crossing, ascendancy)
			}
		})
	}
}

// TestAllocationPathsReachOnlySpecNodes pins PassiveSpec.lua:44: the spec holds no node without a group.
func TestAllocationPathsReachOnlySpecNodes(t *testing.T) {
	versions := make([]string, 0, len(TreeVersions))
	for v := range TreeVersions {
		versions = append(versions, string(v))
	}
	sort.Strings(versions)
	if len(versions) == 0 {
		t.Fatal("REFUSING: no tree versions are registered")
	}

	for _, name := range versions {
		t.Run(name, func(t *testing.T) {
			v := TreeVersions[pob.TreeVersion(name)]
			tree := v.Tree()
			starts := make([]int64, 0, 7)
			nested := 0
			for id, node := range tree.Nodes {
				if node.ExpansionJewel != nil && node.ExpansionJewel.Parent != nil {
					nested++
				}
				if node.ClassStartIndex != nil {
					if n, err := strconv.ParseInt(id, 10, 64); err == nil {
						starts = append(starts, n)
					}
				}
			}
			sort.Slice(starts, func(a, b int) bool { return starts[a] < starts[b] })
			if len(starts) == 0 || nested == 0 {
				t.Skipf("%s: %d class starts, %d nested cluster sockets — nothing to check", name, len(starts), nested)
			}

			nextHops := v.CalculateAllocationPaths(nil, starts)
			if len(nextHops) < 500 {
				t.Fatalf("REFUSING: the BFS reached %d nodes", len(nextHops))
			}
			var outside []int64
			for id := range nextHops {
				if !tree.inSpec(strconv.FormatInt(id, 10)) {
					outside = append(outside, id)
				}
			}
			sort.Slice(outside, func(a, b int) bool { return outside[a] < outside[b] })
			if len(outside) > 0 {
				t.Errorf("%s: %d nodes PassiveSpec.lua:44 leaves out of the spec are on a path, e.g. %v", name, len(outside), outside[:min(5, len(outside))])
			}
		})
	}
}
