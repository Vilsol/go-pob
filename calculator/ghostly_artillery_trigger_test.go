package calculator

import (
	"math"
	"os"
	"testing"
)

// TestGhostlyArtilleryPublishesATriggerRate pins CalcTriggers.lua:1530-1538.
//
//pob:src/Modules/CalcTriggers.lua:1535-1543
func TestGhostlyArtilleryPublishesATriggerRate(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	builds := []string{
		"0751-deadeye-ghostly-artillery.xml",
		"1058-warden-ghostly-artillery.xml",
		"1631-luminary-ghostly-artillery.xml",
	}
	keys := []string{"EffectiveSourceRate", "TriggerRateCap", "SkillTriggerRate"}

	graded := 0
	for _, build := range builds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		for _, key := range keys {
			want, ok := rec.Output[key]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero %s for %s — a 0-vs-0 fixture scores "+
					"ok and cannot see the entry being deleted", key, build)
			}
		}
		graded++

		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			for _, key := range keys {
				want := rec.Output[key]
				got := env.Player.Output[key]
				if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
					t.Errorf("%s: %s = %v, PoB reports %v (ratio %.5f)", build, key, got, want, got/want)
				}
			}
		})
	}

	if graded != len(builds) {
		t.Fatalf("REFUSING: graded %d of %d fixtures", graded, len(builds))
	}
}
