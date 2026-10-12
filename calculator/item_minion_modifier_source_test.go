package calculator

import (
	"slices"
	"testing"
)

// TestItemMinionModifierInnerModCarriesTheItemSource: modLib.setSource sources a MinionModifier's inner mod too.
func TestItemMinionModifierInnerModCarriesTheItemSource(t *testing.T) {
	env, cases := oracleModSectionTables(t, "0031-guardian-raise-spectre.xml")
	checked := 0
	for _, c := range cases {
		if c.Cfg != "skill" || c.ModType != "INC" || !slices.Equal(c.ModName, []string{"Damage"}) {
			continue
		}
		var got []string
		for _, r := range env.CalcsModSection(c.data(), false) {
			got = append(got, r.Value+" | "+r.Source+" | "+r.SourceName)
		}
		for _, r := range c.Rows {
			if r.Value != "101.5" {
				continue
			}
			checked++
			if want := r.Value + " | " + r.Source + " | " + r.SourceName; !slices.Contains(got, want) {
				t.Errorf("no row %q in %q", want, got)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no converted Damage row")
	}
}
