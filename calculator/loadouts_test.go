package calculator

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestNewLoadoutIsListedAndSelects: New Loadout makes a tree and an item.
func TestNewLoadoutIsListedAndSelects(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	if slices.Contains(Loadouts(build).Loadouts, "Mapping") {
		t.Fatal("fixture already has a Mapping loadout")
	}
	NewLoadout(build, "Mapping")
	list := Loadouts(build)
	k := slices.Index(list.Loadouts, "Mapping")
	if k < 0 || list.Selected == k {
		t.Fatalf("%+v", list)
	}
	if !SelectLoadout(build, "Mapping") {
		t.Fatal("not selected")
	}
	if pob.SpecTitle(build.Tree.Specs[build.ActiveSpecIndex()]) != "Mapping" ||
		orDefault(build.Skills.SkillSets[build.ActiveSkillSetIndex()].Title) != "Mapping" ||
		activeTitle(loadoutItemSets(build), build.Items.ActiveItemSet) != "Mapping" ||
		activeTitle(loadoutConfigSets(build), build.Config.ActiveConfigSet) != "Mapping" {
		t.Fatalf("active: tree %d, skills %d, items %d, config %d", build.Tree.ActiveSpec, build.Skills.ActiveSkillSet, build.Items.ActiveItemSet, build.Config.ActiveConfigSet)
	}
	if got := Loadouts(build); got.Selected != slices.Index(got.Loadouts, "Mapping") {
		t.Fatalf("selected %d of %v", got.Selected, got.Loadouts)
	}
	if SelectLoadout(build, "No such loadout") {
		t.Fatal("an unknown loadout selected")
	}
}

// TestLinkedLoadout: A "{id}" in the titles links sets of different names into one loadout, named by the tree's.
func TestLinkedLoadout(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	NewLoadout(build, "x")
	title := func(s string) *string { return &s }
	build.Tree.Specs[0].Title = title("Leveling {7}")
	build.Items.ItemSets[0].Title = title("{7} gear")
	build.Skills.SkillSets[0].Title = title("lvl {7}")
	build.Config.ConfigSets[0].Title = title("{7}")
	list := Loadouts(build)
	if !slices.Contains(list.Loadouts, "Leveling {7}") {
		t.Fatalf("%v", list.Loadouts)
	}
	build.SetActiveSpec(len(build.Tree.Specs))
	if !SelectLoadout(build, "Leveling {7}") || build.Tree.ActiveSpec != 1 {
		t.Fatalf("active tree %d", build.Tree.ActiveSpec)
	}
	id, _ := strconv.Atoi(build.Items.ItemSets[0].ID)
	if build.Items.ActiveItemSet != id || build.Skills.ActiveSkillSet != build.Skills.SkillSets[0].ID || build.Config.ActiveConfigSet != build.Config.ConfigSets[0].ID {
		t.Fatalf("sets %d %d %d", build.Items.ActiveItemSet, build.Skills.ActiveSkillSet, build.Config.ActiveConfigSet)
	}
}
