package calculator

import (
	"slices"
	"testing"
)

// TestCalcSpecIsTheBuildOnThatTree: The tree select's what-if for another tree is the build switched to it.
func TestCalcSpecIsTheBuildOnThatTree(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	if len(build.Tree.Specs) < 2 {
		t.Fatal("fixture has one tree")
	}
	active, nodes := build.Tree.ActiveSpec, slices.Clone(build.Build.PassiveNodes)
	other := 1
	if active == 1 {
		other = 2
	}

	misc := GetMiscCalculator(build)
	got := misc.CalcSpec(other).Player.Output
	if build.Tree.ActiveSpec != active || !slices.Equal(build.Build.PassiveNodes, nodes) {
		t.Fatalf("the build's tree changed: %d, was %d", build.Tree.ActiveSpec, active)
	}
	base := misc.Base().Player.Output

	build.SetActiveSpec(other)
	want := GetMiscCalculator(build).Base().Player.Output
	changed := 0
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %v, the build on tree %d %v", k, got[k], other, v)
		}
		if base[k] != v {
			changed++
		}
	}
	if changed == 0 {
		t.Fatalf("tree %d changes nothing: no control", other)
	}
}

// TestTreeSpecsPointsCountSockets: The tooltip's jewel sockets: a tree with sockets allocated counts them, an empty one none.
func TestTreeSpecsPointsCountSockets(t *testing.T) {
	build := parseCorpusBuild(t, "0005-occultist-vaal-double-strike.xml")
	points := TreeSpecsPoints(build)
	if points[build.ActiveSpecIndex()].Sockets == 0 {
		t.Fatalf("active tree has no sockets counted: %+v", points)
	}
	NewTreeSpec(build, "empty")
	if p := TreeSpecsPoints(build); p[len(p)-1].Sockets != 0 || p[len(p)-1].Used != 0 {
		t.Fatalf("an empty tree counts %+v", p[len(p)-1])
	}
}
