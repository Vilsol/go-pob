package calculator

import (
	"math"
	"os"
	"testing"
)

// TestTriggerRotationEntriesCarryICDR pins CalcTriggers.lua:100's `(skill.icdr or 1)`.
//
//pob:block:src/Modules/CalcTriggers.lua:94-100
func TestTriggerRotationEntriesCarryICDR(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	builds := []string{
		"0393-elementalist-reap.xml",
		"1192-elementalist-reap-8.xml",
		"0184-elementalist-exsanguinate-of-transmission.xml",
	}

	graded := 0
	for _, build := range builds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		wantTime, ok := rec.Output["Time"]
		if !ok || wantTime == 0 {
			t.Fatalf("REFUSING: PoB reports no nonzero Time for %s", build)
		}
		wantRate := rec.Output["SkillTriggerRate"]
		if wantRate == 0 {
			t.Fatalf("REFUSING: PoB reports no nonzero SkillTriggerRate for %s — this build no "+
				"longer exercises the rotation simulation", build)
		}
		graded++

		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			out := env.Player.Output

			esr := out["EffectiveSourceRate"]
			if esr == 0 {
				t.Fatalf("REFUSING: our EffectiveSourceRate is 0, so the simTime identity below " +
					"cannot be evaluated and this fixture proves nothing")
			}
			if simTime := 1000 / esr; math.Abs(out["Time"]-simTime) <= math.Abs(simTime)*1e-9 {
				t.Errorf("%s: Time %v IS the simulation window (1000/EffectiveSourceRate = %v) — "+
					"the rotation fired once, so an entry reached the simulation with icdr unset",
					build, out["Time"], simTime)
			}
			for _, c := range []struct {
				key  string
				want float64
			}{{"Time", wantTime}, {"SkillTriggerRate", wantRate}} {
				if got := out[c.key]; math.Abs(got-c.want) > math.Abs(c.want)*oracleDumpTolerance {
					t.Errorf("%s: %s = %v, PoB reports %v (ratio %.4f)", build, c.key, got, c.want, got/c.want)
				}
			}
		})
	}

	if graded != len(builds) {
		t.Fatalf("REFUSING: graded %d of %d fixtures", graded, len(builds))
	}
}
