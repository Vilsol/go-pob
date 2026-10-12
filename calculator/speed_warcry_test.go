package calculator

import (
	"math"
	"os"
	"testing"
)

// TestWarcrySpeedComesFromWarcryCastTime pins CalcOffence.lua:2334-2335, the warcry arm of the Speed assignment.
//
//	if skillFlags.warcry then
//	else
//	end
func TestWarcrySpeedComesFromWarcryCastTime(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0473-luminary-battlemages-cry.xml")

	const (
		wantWarcryCastTime = 0.40201005025126
		wantCastRate       = 2.4875
		wantSpeed          = 0.12470382840753
	)

	if got := env.Player.Output["WarcryCastTime"]; math.Abs(got-wantWarcryCastTime) > 1e-9 {
		t.Fatalf("WarcryCastTime = %v, want %v (PoB) — the producer moved, so this fixture no "+
			"longer isolates the missing consumer at CalcOffence.lua:2335", got, wantWarcryCastTime)
	}

	if got := env.Player.Output["CastRate"]; math.Abs(got-wantCastRate) > 1e-6 {
		t.Errorf("CastRate = %v, want %v (PoB) — CalcOffence.lua:2334-2335 sets a warcry's Speed to "+
			"1 / WarcryCastTime (%v) rather than the generic 1/baseTime * round((1+inc/100)*more, 2)",
			got, wantCastRate, 1/wantWarcryCastTime)
	}

	if got := env.Player.Output["Speed"]; math.Abs(got-wantSpeed) > 1e-6 {
		t.Errorf("Speed = %v, want %v (PoB) — 1/WarcryCastTime clamped by :2359 to "+
			"Repeats/Cooldown", got, wantSpeed)
	}
}
