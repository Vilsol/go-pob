package calculator

import (
	"math"
	"slices"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

func sameKindMasteries(t *testing.T, build *pob.PathOfBuilding, selected int64) int64 {
	t.Helper()
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	sel := tree.Nodes[strconv.FormatInt(selected, 10)]
	for id, n := range tree.Nodes {
		if !isMasteryNode(n) || id == strconv.FormatInt(selected, 10) || n.Name == nil || *n.Name != *sel.Name {
			continue
		}
		nid, _ := strconv.ParseInt(id, 10, 64)
		if !slices.Contains(build.Build.PassiveNodes, nid) {
			return nid
		}
	}
	t.Fatalf("REFUSING: no other unallocated %s in the tree", *sel.Name)
	return 0
}

func TestMasteryOptionsLeaveOutEffectsHeldElsewhere(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	opts := MasteryOptions(build, 63268)
	if len(opts) < 2 {
		t.Fatalf("63268 offers %d effects", len(opts))
	}
	i := slices.IndexFunc(opts, func(o MasteryOption) bool { return o.Selected })
	if i < 0 || opts[i].Effect != 6216 {
		t.Fatalf("selected option = %d, want effect 6216 (%+v)", i, opts)
	}
	other := sameKindMasteries(t, build, 63268)
	for _, o := range MasteryOptions(build, other) {
		if o.Effect == 6216 {
			t.Fatalf("mastery %d offers effect 6216, which 63268 holds", other)
		}
	}
	if len(MasteryOptions(build, other)) != len(opts)-1 {
		t.Errorf("mastery %d offers %d effects, want %d", other, len(MasteryOptions(build, other)), len(opts)-1)
	}
	if MasteryOptions(build, 54447) != nil {
		t.Error("a class start offers mastery effects")
	}
}

func TestSelectMasteryEffectAllocatesAndMovesTheChoice(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	opts := MasteryOptions(build, 63268)
	var next int64
	for _, o := range opts {
		if !o.Selected {
			next = o.Effect
			break
		}
	}
	if err := SelectMasteryEffect(build, 63268, next, nil); err != nil {
		t.Fatal(err)
	}
	sel := pob.SpecMasterySelections(build.Tree.Specs[1])
	if i := slices.IndexFunc(sel, func(s pob.MasterySelection) bool { return s.Node == 63268 }); i < 0 || sel[i].Effect != next || len(sel) != 5 {
		t.Fatalf("selections = %v, want 63268 -> %d among 5", sel, next)
	}
	other := sameKindMasteries(t, build, 63268)
	if err := SelectMasteryEffect(build, other, next, nil); err == nil {
		t.Fatal("an effect another mastery holds was accepted")
	}
	free := MasteryOptions(build, other)[0].Effect
	before := len(build.Build.PassiveNodes)
	if err := SelectMasteryEffect(build, other, free, []int64{other}); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(build.Build.PassiveNodes, other) || len(build.Build.PassiveNodes) != before+1 {
		t.Fatalf("mastery %d not allocated (%d -> %d nodes)", other, before, len(build.Build.PassiveNodes))
	}
	build.DeallocateNodes([]int64{other})
	if slices.ContainsFunc(pob.SpecMasterySelections(build.Tree.Specs[1]), func(s pob.MasterySelection) bool { return s.Node == other }) {
		t.Fatal("a deallocated mastery kept its effect")
	}
}

// TestMasteryWithoutAnEffectGrantsNothing: PoB leaves a mastery with no selected effect unallocated, so it adds no stats.
func TestMasteryWithoutAnEffectGrantsNothing(t *testing.T) {
	life := func(extra []int64, effect int64) float64 {
		build := parseCorpusBuild(t, "0056-saboteur-ice-trap.xml")
		build.AllocateNodes(append([]int64{9877, 25067, 31315}, extra...))
		if effect != 0 {
			build.SetMasteryEffect(44316, effect)
		}
		return (&Calculator{PoB: build}).BuildOutput(OutputModeMain).Player.Output["Life"]
	}
	// PoB at the pin: 3544 without the mastery, 3544 with it allocated and no effect, 3619 with "+30 to maximum Life".
	for _, c := range []struct {
		name   string
		extra  []int64
		effect int64
		want   float64
	}{
		{"no mastery", nil, 0, 3544},
		{"mastery without an effect", []int64{44316}, 0, 3544},
		{"mastery with +30 to maximum Life", []int64{44316}, 47642, 3619},
	} {
		if got := life(c.extra, c.effect); math.Round(got) != c.want {
			t.Errorf("%s: Life = %v, PoB %v", c.name, got, c.want)
		}
	}
}
