package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestCalcMasteryEffectIsTheBuildWithThatEffect: The mastery popup's what-if for another effect of an allocated mastery.
func TestCalcMasteryEffectIsTheBuildWithThatEffect(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	spec := build.Tree.Specs[build.ActiveSpecIndex()]
	saved := spec.MasteryEffects
	tree := specTree()
	var node, other int64
	for _, s := range pob.SpecMasterySelections(spec) {
		for _, e := range tree.Nodes[nodeKey(s.Node)].MasteryEffects {
			if e.Effect != s.Effect && statsModKey(e.Stats) != "" {
				if _, held := heldBy(masterySelectionMap(spec), e.Effect); !held {
					node, other = s.Node, e.Effect
				}
			}
		}
		if node != 0 {
			break
		}
	}
	if node == 0 {
		t.Fatal("fixture has no allocated mastery with a free other effect")
	}

	misc := GetMiscCalculator(build)
	got := misc.CalcMasteryEffect(node, other, []int64{node}).Player.Output
	if build.Tree.Specs[build.ActiveSpecIndex()].MasteryEffects != saved {
		t.Fatalf("the build's selection changed: %q, was %q", build.Tree.Specs[build.ActiveSpecIndex()].MasteryEffects, saved)
	}
	base := misc.Base().Player.Output

	build.SetMasteryEffect(node, other)
	want := GetMiscCalculator(build).Base().Player.Output
	changed := 0
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %v, the build with the effect %v", k, got[k], v)
		}
		if base[k] != v {
			changed++
		}
	}
	if changed == 0 {
		t.Fatalf("effect %d of mastery %d changes nothing: no control", other, node)
	}
}

func masterySelectionMap(spec pob.Spec) map[int64]int64 {
	out := map[int64]int64{}
	for _, s := range pob.SpecMasterySelections(spec) {
		out[s.Node] = s.Effect
	}
	return out
}
