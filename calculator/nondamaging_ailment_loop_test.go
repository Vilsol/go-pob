package calculator

import (
	"math"
	"os"
	"testing"
)

// TestNonDamagingAilmentLoopPortsBothDroppedPieces covers CalcOffence.lua:5292-5314, the loop over the non-damaging ailments.
func TestNonDamagingAilmentLoopPortsBothDroppedPieces(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	builds := []string{
		"0421-hierophant-vaal-arc.xml",
		"1066-slayer-wild-strike.xml",
		"0118-reliquarian-unknown-2.xml",
	}

	graded := 0
	for _, build := range builds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		want, ok := rec.Output["ShockEffectMod"]
		if !ok || want == 0 {
			t.Fatalf("REFUSING: PoB reports no nonzero ShockEffectMod for %s", build)
		}
		if want == 1 {
			t.Fatalf("REFUSING: PoB's ShockEffectMod for %s is exactly 1, so the enemy-side "+
				"factor this test is about cannot be seen on it", build)
		}
		graded++

		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			got := env.Player.Output["ShockEffectMod"]
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: ShockEffectMod = %v, PoB reports %v (ratio %.4f)",
					build, got, want, got/want)
			}
		})
	}

	if graded == 0 {
		t.Fatal("REFUSING: no fixture graded ShockEffectMod — the population emptied rather " +
			"than this test passing")
	}
}
