package exposition

import (
	"slices"
	"testing"
)

// TestTreeTattooNodesReadAsTheTattoo: 0020's Hinekora tattoos read as the tattoo (Data/TattooPassives.lua's dn / sd / reminderText), not the tree's node.
func TestTreeTattooNodesReadAsTheTattoo(t *testing.T) {
	initItemData(t)
	build := corpusBuild(t, "0020-assassin-storm-burst.xml")
	tips := TreeTattooNodes(build)
	if len(tips) != len(build.Tree.Specs[build.ActiveSpecIndex()].Overrides) {
		t.Fatalf("%d tips for %d overrides", len(tips), len(build.Tree.Specs[build.ActiveSpecIndex()].Overrides))
	}
	i := slices.IndexFunc(tips, func(x TattooNodeTip) bool { return x.NodeID == 15549 })
	if i < 0 {
		t.Fatal("no tip for node 15549")
	}
	if got := tips[i]; got.Name != "Tattoo of the Hinekora Storyteller" || !slices.Equal(got.Stats, []string{"+3% to Chaos Resistance"}) || len(got.Reminder) != 0 {
		t.Errorf("node 15549: %+v", got)
	}
	build.Tree.Specs[build.ActiveSpecIndex()].Overrides[0].DN = "Honoured Tattoo of the Makanga"
	got := TreeTattooNodes(build)[0]
	if got.Name != "Honoured Tattoo of the Makanga" || !slices.Equal(got.Stats, []string{"+2% to all maximum Resistances"}) ||
		!slices.Equal(got.Reminder, []string{"Requires at least 8 adjacent Passive Skills to be allocated"}) {
		t.Errorf("Honoured Tattoo of the Makanga: %+v", got)
	}
	build.Tree.Specs[build.ActiveSpecIndex()].Overrides[0].DN = "No such tattoo"
	if len(TreeTattooNodes(build)) != len(tips)-1 {
		t.Error("an unknown tattoo still has a tip")
	}
}
