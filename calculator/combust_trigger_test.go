package calculator

import (
	"math"
	"os"
	"testing"
)

// TestCombustTriggersFromInfernalCryUptime pins CalcTriggers.lua:1347-1354 and :502-519.
func TestCombustTriggersFromInfernalCryUptime(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	build := loadCorpusBuild(t, "0995-berserker-earthshatter-17.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)

	const want = 43115977.47914907
	if got := env.Player.Output["FullDPS"]; math.Abs(got-want) > want*1e-6 {
		t.Errorf("FullDPS = %v, want %v (PoB's value, to 1e-6)", got, want)
	}
}
