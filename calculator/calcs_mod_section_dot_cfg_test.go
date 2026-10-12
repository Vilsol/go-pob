package calculator

import (
	"slices"
	"testing"
)

// TestCalcsModSectionDotTypeCfg: CalcOffence.lua:5679 and :5688 keep dotCfg and dot<Type>Cfg on the skill.
func TestCalcsModSectionDotTypeCfg(t *testing.T) {
	env, cases := oracleModSectionTables(t, "0044-assassin-charged-dash.xml")
	checked := 0
	for _, c := range cases {
		if c.Cfg != "dotLightning" || c.ModType != "INC" {
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
		t.Fatal("no dotLightning INC table")
	}
}
