package calculator

import (
	"math"
	"os"
	"testing"
)

// TestSoulGainPreventionDurationNameListIsNotInverted pins CalcOffence.lua:1577.
func TestSoulGainPreventionDurationNameListIsNotInverted(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	builds := []string{
		"1612-elementalist-vaal-arc-3.xml",
		"5485-ladder-MiskoSnow_1580.misko__cws.xml",
		"4309-ladder-Caneja_1041.Caneja.xml",
		"3495-ladder-eoenoco_7144.댝댝이.xml",
	}

	graded := 0
	for _, build := range builds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		want, ok := rec.Output["SoulGainPreventionDuration"]
		if !ok || want == 0 {
			t.Fatalf("REFUSING: PoB reports no nonzero SoulGainPreventionDuration for %s", build)
		}
		graded++

		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			got := env.Player.Output["SoulGainPreventionDuration"]
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: SoulGainPreventionDuration = %v, PoB reports %v (ratio %.4f)",
					build, got, want, got/want)
			}
		})
	}

	if graded == 0 {
		t.Fatal("REFUSING: no fixture graded SoulGainPreventionDuration — the population " +
			"emptied rather than this test passing")
	}
}
