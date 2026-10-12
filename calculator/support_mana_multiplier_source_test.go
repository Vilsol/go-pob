package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestSupportManaMultiplierCarriesTheSupportSource: CalcActiveSkill.lua:514 sources each support's SupportManaMultiplier with the support's modSource.
func TestSupportManaMultiplierCarriesTheSupportSource(t *testing.T) {
	requireGameData(t, "3.29")
	raw, err := os.ReadFile("../testdata/corpus-3.29/0001-necromancer-unknown.xml")
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeCalcs, NewEnvironmentCache())
	rows := env.CalcsModSection(ModSectionData{ModName: []string{"ManaReserved", "Reserved", "SupportManaMultiplier"}, Multi: true, Cfg: "skill"}, false)
	seen := 0
	for _, r := range rows {
		if r.Name != "Support Mana Multiplier" {
			continue
		}
		seen++
		if r.Source != "Skill" || r.SourceName == "" {
			t.Errorf("%s %s: source %q / %q, PoB Skill / the support's name", r.Value, r.Name, r.Source, r.SourceName)
		}
	}
	if seen == 0 {
		t.Fatal("no Support Mana Multiplier row")
	}
}
