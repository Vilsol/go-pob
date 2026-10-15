package calculator

import (
	"math"
	"os"
	"testing"
)

// TestMirageCopySkipsTriggers pins CalcTriggers.lua:1593: the Mirage Archer copy of a Manaforged Arrows Frenzy is in env.limitedSkills.
func TestMirageCopySkipsTriggers(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	build := loadCorpusBuild(t, "0597-warden-split-arrow.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)

	const want = 4788782.1090658
	if got := env.Player.Output["FullDPS"]; math.Abs(got-want) > want*1e-6 {
		t.Errorf("FullDPS = %v, want %v (PoB's value, to 1e-6)", got, want)
	}
}
