package calculator

import (
	"math"
	"os"
	"testing"
)

// TestSealBlockHonoursNoRepeatBonuses pins CalcOffence.lua:910's `not NoRepeatBonuses` conjunct.
func TestSealBlockHonoursNoRepeatBonuses(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0503-reliquarian-vaal-blade-vortex.xml")
	if !env.Player.MainSkill.SkillModList.Flag(nil, "NoRepeatBonuses") {
		t.Fatal("premise broken: main skill does not carry NoRepeatBonuses")
	}
	out := env.Player.Output
	if _, ok := out["SealMax"]; ok {
		t.Errorf("SealMax = %v, want absent (PoB dump has no SealMax)", out["SealMax"])
	}
	if got, want := out["TotalDPS"], 8.88913423224e+06; math.Abs(got-want) > 1e-6*want {
		t.Errorf("TotalDPS = %v, want %v (PoB)", got, want)
	}
}
