package calculator

import "testing"

// TestCraftBaseSearchFilter: Data.lua:1256-1281, values from PoB's data.itemBaseLists.
func TestCraftBaseSearchFilter(t *testing.T) {
	initItemsTabData(t)
	want := map[string]string{
		"Two-Stone Ring (Cold/Lightning)": "Two-Stone Ring (Cold/Lightning) +(12-16)% to Cold and Lightning Resistances",
		"Vaal Regalia":                    "Vaal Regalia",
		"Chieftain Talisman":              "Chieftain Talisman 16% increased Area of Effect",
		"Prismatic Tincture":              "Prismatic Tincture (70-100)% increased Elemental Damage with Melee Weapons",
		"Iron Flask":                      "Iron Flask Restores Ward on use +200 to Ward",
	}
	seen := 0
	for _, g := range CraftBaseGroups() {
		for k, name := range g.Bases {
			w, ok := want[name]
			if !ok {
				continue
			}
			seen++
			got := ""
			if k < len(g.SearchFilters) {
				got = g.SearchFilters[k]
			}
			if got != w {
				t.Errorf("%s: %q, PoB %q", name, got, w)
			}
		}
	}
	if seen != len(want) {
		t.Fatalf("REFUSING: %d of %d bases listed", seen, len(want))
	}
}
