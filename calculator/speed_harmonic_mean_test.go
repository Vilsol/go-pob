package calculator

import (
	"math"
	"os"
	"testing"
)

// TestDualWieldSpeedUsesHarmonicMean pins CalcOffence.lua:2444, `combineStat("Speed", "HARMONICMEAN")`.
func TestDualWieldSpeedUsesHarmonicMean(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0101-champion-lancing-steel.xml")

	mh := env.Player.OutputTable[OutTableMainHand]["Speed"]
	oh := env.Player.OutputTable[OutTableOffHand]["Speed"]
	if mh == oh {
		t.Fatalf("fixture is useless: both hands are at %v, where the arithmetic and harmonic "+
			"means coincide — pick a build whose weapons differ", mh)
	}

	const want = 1.7284374
	if got := env.Player.Output["Speed"]; math.Abs(got-want) > 1e-6 {
		t.Errorf("Speed = %v, want %v (PoB) — MainHand %v / OffHand %v; the arithmetic mean is %v, "+
			"and CalcOffence.lua:2444 combines Speed as HARMONICMEAN",
			got, want, mh, oh, (mh+oh)/2)
	}
}
