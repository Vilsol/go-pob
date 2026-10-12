package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestResetTreeAndRemoveAllTattoos: Reset Tree keeps only the start nodes and forgets the mastery effects; Remove All Tattoos leaves the tree's nodes alone.
func TestResetTreeAndRemoveAllTattoos(t *testing.T) {
	build := parseCorpusBuild(t, "0020-assassin-storm-burst.xml")
	i := build.ActiveSpecIndex()
	if len(build.Tree.Specs[i].Overrides) == 0 || build.Tree.Specs[i].MasteryEffects == "" {
		t.Fatal("fixture has no tattoo or no mastery effect")
	}
	if lines := TattooCountLines(build); len(lines) == 0 {
		t.Fatal("no tattoo count lines")
	}
	nodes := len(build.Build.PassiveNodes)
	RemoveAllTattoos(build)
	if len(build.Tree.Specs[i].Overrides) != 0 || len(build.Build.PassiveNodes) != nodes || len(TattooCountLines(build)) != 0 {
		t.Fatalf("%d overrides, %d nodes (was %d)", len(build.Tree.Specs[i].Overrides), len(build.Build.PassiveNodes), nodes)
	}
	ResetTree(build)
	tree := specTree()
	for _, id := range build.Build.PassiveNodes {
		if n := tree.Nodes[nodeKey(id)]; !isClassStartNode(n) && !isAscendClassStartNode(n) {
			t.Fatalf("node %d kept", id)
		}
	}
	if len(build.Build.PassiveNodes) == 0 || build.Tree.Specs[i].MasteryEffects != "" || len(pob.SpecMasterySelections(build.Tree.Specs[i])) != 0 {
		t.Fatalf("%v, effects %q", build.Build.PassiveNodes, build.Tree.Specs[i].MasteryEffects)
	}
}
