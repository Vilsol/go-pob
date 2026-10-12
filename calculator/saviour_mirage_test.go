package calculator

import (
	"math"
	"os"
	"testing"
)

// TestSaviourMirageCopiesTheSkill pins CalcMirages.lua:118-162 through calculateMirage.
func TestSaviourMirageCopiesTheSkill(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	build := loadCorpusBuild(t, "1344-slayer-reave-3.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)

	const want = 56987034.02939816
	if got := env.Player.Output["FullDPS"]; math.Abs(got-want) > want*1e-6 {
		t.Errorf("FullDPS = %v, want %v (PoB's value, to 1e-6)", got, want)
	}
}
