package calculator

import (
	"slices"
	"testing"
)

// TestGrandSpectrumMultiplierSourcedByItemName: Item.lua:2415-2417 source the Grand Spectrum multiplier with the bare item name.
func TestGrandSpectrumMultiplierSourcedByItemName(t *testing.T) {
	env, cases := oracleModSectionTables(t, "0004-trickster-lightning-tendrils-of-eccentricity.xml")
	checked := 0
	for _, c := range cases {
		if !slices.Contains(c.ModName, "ElementalResist") {
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
		t.Fatal("no ElementalResist table")
	}
}
