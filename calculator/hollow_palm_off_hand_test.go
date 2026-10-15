package calculator

import (
	"math"
	"testing"
)

// TestHollowPalmGivesUnarmedOffHand pins CalcSetup.lua:1468-1476: with no weapon and no gloves.
func TestHollowPalmGivesUnarmedOffHand(t *testing.T) {
	build := loadCorpusBuild(t, "15647-slayer-cyclone.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
	off := env.Player.OutputTable[OutTableOffHand]
	for stat, want := range map[string]float64{"Accuracy": 8503, "CastRate": 16.812, "AverageHit": 8703798.5727} {
		if got := off[stat]; math.Abs(got-want) > 1e-6*want {
			t.Errorf("OffHand.%s = %v, want %v", stat, got, want)
		}
	}
	if got, want := env.Player.Output["Speed"], 16.098991071429; math.Abs(got-want) > 1e-9*want {
		t.Errorf("Speed = %v, want %v", got, want)
	}
}
