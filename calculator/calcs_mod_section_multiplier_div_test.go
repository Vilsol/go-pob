package calculator

import "testing"

// TestModTagTextPrintsAnExplicitDivOfOne: CalcBreakdownControl.lua:452 prints a Multiplier tag's div whenever the tag carries one, 1 included.
func TestModTagTextPrintsAnExplicitDivOfOne(t *testing.T) {
	requireGameData(t, "3.29")
	entry := ParseMod("Gain 1% of Cold Damage as Extra Fire Damage per 1% Chill Effect on Enemy", false)
	if entry == nil || len(entry.ModList) != 1 || entry.Extra != "" {
		t.Fatalf("parse: %+v", entry)
	}
	if got, want := modTagText(entry.ModList[0]), "+1 per 1 Chill Effect"; got != want {
		t.Errorf("tags %q, PoB %q", got, want)
	}
	entry = ParseMod("10% increased Damage per Frenzy Charge", false)
	if got, want := modTagText(entry.ModList[0]), "10% per Frenzy Charge"; got != want {
		t.Errorf("control: tags %q, PoB %q", got, want)
	}
}
