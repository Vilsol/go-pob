package calculator

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestMinionItemSetTooltip: The minion dropdown of an item-set minion.
func TestMinionItemSetTooltip(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), "0030-saboteur-cyclone-2.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	build.Build.MainSocketGroup = 2
	build.Skills.SkillSets[0].Skills[1].MainActiveSkill = 2
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	controls := env.MainSkillControls()
	if controls == nil || !controls.MinionShown || len(controls.Minions) != 2 {
		t.Fatalf("want the two item sets in the minion dropdown, got %+v", controls)
	}
	ag := controls.Minions[1]
	if ag.Label != "AG" || ag.ItemSetID != 2 {
		t.Fatalf("second option: got %+v", ag)
	}
	want := []string{
		"^7Weapon 1: ^xAF6025Dying Breath, Iron Staff",
		"^7Helmet: ^xAF6025Ylfeban's Trickery, Hubris Circlet",
		"^7Body Armour: ^xAF6025Zahndethus' Cassock, Sage's Robe",
		"^7Gloves: ^xAF6025Southbound, Soldier Gloves",
		"^7Boots: ^xAF6025Wake of Destruction, Mesh Boots",
		"",
		"^x80A080Tip: You can drag items from the Items tab onto this dropdown to equip them onto the minion.",
	}
	if !sameStrings(ag.Tooltip, want) {
		t.Errorf("AG tooltip:\n got %q\nwant %q", ag.Tooltip, want)
	}
	first := controls.Minions[0].Tooltip
	if len(first) == 0 || first[0] != "^7Weapon 1: ^xFFFF77New Item, Void Sceptre" {
		t.Errorf("Default Item Set tooltip starts %q", first)
	}
	if want := []string{"Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots"}; !slices.Equal(controls.MinionUses, want) {
		t.Errorf("MinionUses %q, want %q", controls.MinionUses, want)
	}
	if got := DropItemPrimarySlot(build, 0, ItemDBUnique, "Southbound, Soldier Gloves"); got != "Gloves" {
		t.Errorf("Southbound primary slot %q", got)
	}
	if got := DropItemPrimarySlot(build, 0, ItemDBUnique, "Kaom's Heart, Glorious Plate"); got != "Body Armour" {
		t.Errorf("Kaom's Heart primary slot %q", got)
	}
	if got := DropItemPrimarySlot(build, 0, ItemDBUnique, "Astramentis, Onyx Amulet"); got != "Amulet" {
		t.Errorf("Astramentis primary slot %q", got)
	}
}

// TestActiveSetSlotItemName: The "Socketed in" dropdown names the active item set's item in a slot (SkillsTab.lua:191-196).
func TestActiveSetSlotItemName(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), "0030-saboteur-cyclone-2.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := ActiveSetSlotItemName(build, "Weapon 1"); got != "New Item, Void Sceptre" {
		t.Errorf("Weapon 1: %q", got)
	}
	if got := ActiveSetSlotItemName(build, "Ring 3"); got != "" {
		t.Errorf("Ring 3: %q", got)
	}
	build.Items.ActiveItemSet = 2
	if got := ActiveSetSlotItemName(build, "Weapon 1"); got != "Dying Breath, Iron Staff" {
		t.Errorf("Weapon 1 in the AG set: %q", got)
	}
}
