package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestCalcsPassReadsTheCalcsTabInputs: CalcSetup.lua:462-485: the CALCS pass takes its buff mode from the Calcs tab's misc_buffMode.
func TestCalcsPassReadsTheCalcsTabInputs(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	modes := func(mode OutputMode) [3]bool {
		env := (&Calculator{PoB: build}).BuildOutput(mode)
		return [3]bool{env.ModeBuffs, env.ModeCombat, env.ModeEffective}
	}
	for _, c := range []struct {
		buffMode string
		want     [3]bool
	}{
		{"EFFECTIVE", [3]bool{true, true, true}},
		{"COMBAT", [3]bool{true, true, false}},
		{"BUFFED", [3]bool{true, false, false}},
		{"UNBUFFED", [3]bool{false, false, false}},
	} {
		build.SetCalcsInputString("misc_buffMode", c.buffMode)
		if got := modes(OutputModeCalcs); got != c.want {
			t.Errorf("CALCS with %s: buffs/combat/effective %v, want %v", c.buffMode, got, c.want)
		}
		aligned := (&Calculator{PoB: build, AlignCalcsSkillToMain: true}).BuildOutput(OutputModeCalcs)
		if got := [3]bool{aligned.ModeBuffs, aligned.ModeCombat, aligned.ModeEffective}; got != c.want {
			t.Errorf("CALCS aligned to MAIN with %s: %v", c.buffMode, got)
		}
		if got := modes(OutputModeMain); got != [3]bool{true, true, true} {
			t.Errorf("MAIN with the Calcs tab on %s: %v", c.buffMode, got)
		}
	}

	groups := len(build.Skills.SkillSets[build.ActiveSkillSetIndex()].Skills)
	if groups < 2 {
		t.Fatalf("REFUSING: the fixture needs two socket groups, has %d", groups)
	}
	build.SetCalcsInputNumber("skill_number", 2)
	if env := (&Calculator{PoB: build}).BuildOutput(OutputModeCalcs); env.MainSocketGroup != 1 {
		t.Errorf("CALCS on skill_number 2: socket group %d", env.MainSocketGroup+1)
	}
	build.SetCalcsInputNumber("skill_number", 99)
	if env := (&Calculator{PoB: build}).BuildOutput(OutputModeCalcs); env.MainSocketGroup != groups-1 {
		t.Errorf("CALCS on skill_number 99 clamps to %d: got %d", groups, env.MainSocketGroup+1)
	}
}
