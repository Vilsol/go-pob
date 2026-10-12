package calculator

import (
	"slices"
	"testing"
)

// TestCalcsModSectionFlagSourceTotals: The per-source totals Combine every mod type present.
func TestCalcsModSectionFlagSourceTotals(t *testing.T) {
	env, cases := oracleModSectionTables(t, "0001-necromancer-unknown.xml")
	checked := 0
	for _, c := range cases {
		if !slices.Contains(c.ModName, "ShockImmune") && !slices.Contains(c.ModName, "BleedImmune") {
			continue
		}
		var want, got []string
		for _, r := range c.Rows {
			want = append(want, modSectionRowText(r.Value, r.Name, r.Flags, r.Tags, r.Source, r.SourceName, r.Totals))
		}
		for _, r := range env.CalcsModSection(c.data(), false) {
			got = append(got, modSectionRowText(r.Value, r.Name, r.Flags, r.Tags, r.Source, r.SourceName, r.Totals))
		}
		slices.Sort(want)
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("%v:\n got %q\nPoB %q", c.ModName, got, want)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no ShockImmune / BleedImmune table")
	}
}
