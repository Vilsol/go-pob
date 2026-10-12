package calculator

import (
	"math"
	"os"
	"testing"
)

// TestHolyRelicTriggerCapAndSourceRateMatchPoB is RECOVERED FROM A DELETED BRANCH; the HitSpeed gap it declared has closed.
func TestHolyRelicTriggerCapAndSourceRateMatchPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	build := loadCorpusBuild(t, "0140-necromancer-summon-holy-relic-of-conviction.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
	if env.Minion == nil || env.Minion.Actor == nil {
		t.Fatal("REFUSING: build has no minion actor, so this fixture proves nothing")
	}
	out := env.Minion.Output

	const wantCap = 4.3290043290043
	if got := out["TriggerRateCap"]; math.Abs(got-wantCap) > wantCap*1e-6 {
		t.Errorf("Minion.TriggerRateCap = %v, want %v (PoB's value, to 1e-6)", got, wantCap)
	}

	const wantSourceRate = 26.4576
	got := out["EffectiveSourceRate"]
	if math.Abs(got-wantSourceRate) > wantSourceRate*1e-6 {
		t.Errorf("Minion.EffectiveSourceRate = %v, want %v (PoB's value, to 1e-6)", got, wantSourceRate)
	}
}
