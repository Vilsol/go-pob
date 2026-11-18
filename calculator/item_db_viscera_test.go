package calculator

import (
	"testing"
)

// TestVisceraDBAgainstPoB: Modules/Main.lua:204-212 keys data.viscera's 441 items (Data/Viscera.lua at PoB 2.68.1) by name,
// so the four whose title repeats their base collapse to 437; the values are PoB's file.
func TestVisceraDBAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	items := ItemDB(ItemDBViscera)
	if len(items) != 437 {
		t.Fatalf("%d Viscera items, PoB 437", len(items))
	}
	byTitle := map[string]*Item{}
	for _, item := range items {
		byTitle[item.Title] = item
	}
	for title, want := range map[string]struct{ base, implicit string }{
		"Lesser Scorched Arm":                   {"Scorched Arm", "(10-15)% chance to Ignite"},
		"Incarnation of Fear's Head":            {"Incarnation of Fear's Head", "Trigger Level 20 Summon Void Spawn every 4 seconds"},
		"Greater Incarnation of Neglect's Cube": {"Incarnation of Neglect's Cube", "Damaging Retaliation Skills become Usable every 4 seconds"},
	} {
		item := byTitle[title]
		if item == nil {
			t.Errorf("%s: absent", title)
			continue
		}
		if item.BaseName != want.base || item.Type != "Viscera" || item.Rarity != RarityRare || len(item.ImplicitModLines) != 1 || item.ImplicitModLines[0].Line != want.implicit {
			t.Errorf("%s: base %q type %q rarity %q implicits %v, want %q / %q", title, item.BaseName, item.Type, item.Rarity, item.ImplicitModLines, want.base, want.implicit)
		}
	}
}
