package calculator

import (
	"math"
	"os"
	"testing"
)

// TestUniqueItemTriggerRotationCarriesICDR pins the contract stated on triggerRotationSkill.
//
//pob:src/Modules/CalcTriggers.lua:91-134
func TestUniqueItemTriggerRotationCarriesICDR(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	builds := []string{
		"0184-elementalist-exsanguinate-of-transmission.xml",
		"0393-elementalist-reap.xml",
		"1192-elementalist-reap-8.xml",
	}
	keys := []string{"SkillTriggerRate", "Time", "TriggerTime", "Speed"}

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
					"ok and cannot see the simulation being starved", key, build)
			}
		}
		for _, key := range []string{"EffectiveSourceRate", "TriggerRateCap"} {
			if want, ok := rec.Output[key]; !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero %s for %s", key, build)
			}
		}
		graded++

		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			for _, key := range append(keys, "EffectiveSourceRate", "TriggerRateCap") {
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
