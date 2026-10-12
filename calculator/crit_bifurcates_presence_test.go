package calculator

import (
	"math"
	"testing"
)

// TestCritBifurcatesIsWrittenWhenCritChanceIsZero: CalcOffence.lua:3137 guards the crit-bifurcate block with a Lua PRESENCE test.
//
//	if env.mode_effective and skillModList:Flag(cfg, "BifurcateCrit") and output.PreBifurcateCritChance
//	   and not (critOverride == 100) then
func TestCritBifurcatesIsWrittenWhenCritChanceIsZero(t *testing.T) {
	build := loadCorpusBuild(t, "1573-assassin-raise-spectre.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)

	if got, want := env.Player.Output["PreBifurcateCritChance"], 0.0; math.Abs(got-want) > 1e-9 {
		t.Fatalf("REFUSING: PreBifurcateCritChance = %v, want %v — this fixture only exercises the "+
			"presence-vs-zero distinction while the value is 0", got, want)
	}
	if got, want := env.Player.Output["CritBifurcates"], 1.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("CritBifurcates = %v, want %v (PoB enters the block on a PRESENT key, "+
			"CalcOffence.lua:3137, and 0 is truthy in Lua)", got, want)
	}
}
