package calculator

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestTreeDependentsCoverClusterSubgraphs: what the tree draws red on hovering an allocated node.
func TestTreeDependentsCoverClusterSubgraphs(t *testing.T) {
	t.Parallel()
	for _, file := range []string{"0719-gladiator-herald-of-agony-2.xml", "0511-elementalist-tornado-of-elemental-turbulence.xml", "0549-guardian-tempest-shield.xml"} {
		t.Run(file, func(t *testing.T) {
			t.Parallel()
			build := parseCorpusBuild(t, file)
			b := GetMiscCalculator(build).NewPowerBuilder(PowerStats()[0], 1)
			spec := build.Tree.Specs[build.ActiveSpecIndex()]
			version := data.TreeVersions[data.LatestTreeVersion]
			var roots []int64
			for _, n := range version.Tree().Nodes {
				if n.ClassStartIndex != nil && *n.ClassStartIndex == int64(spec.ClassID) {
					for _, key := range append(slices.Clone(n.Out), n.In...) {
						if o, ok := version.Tree().Nodes[key]; ok && o.Skill != nil && o.AscendancyName == nil {
							id, _ := strconv.ParseInt(key, 10, 64)
							roots = append(roots, id)
						}
					}
				}
			}
			var subgraph []data.SubgraphNode
			for _, n := range TreeClusterNodes(build) {
				subgraph = append(subgraph, data.SubgraphNode{ID: n.ID, Type: n.Type, Linked: n.Linked})
			}
			active := slices.Clone(build.Build.PassiveNodes)
			reaches := 0
			for _, id := range active {
				want := b.depends(id)
				got := append([]int64{id}, version.CalculateDependentNodesWith(active, id, roots, subgraph)...)
				slices.Sort(want)
				slices.Sort(got)
				if !slices.Equal(got, want) {
					t.Errorf("node %d: depends %v, want %v", id, got, want)
				}
				if slices.ContainsFunc(got, func(o int64) bool { return o != id && o >= 1<<16 }) {
					reaches++
				}
			}
			t.Logf("%d allocated, %d subgraph nodes, %d whose dependents reach into a subgraph", len(active), len(subgraph), reaches)
			if reaches == 0 {
				t.Fatal("REFUSING: no allocated node's dependents reach a subgraph node")
			}
		})
	}
}
