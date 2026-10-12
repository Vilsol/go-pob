package calculator

import (
	"math"
	"os"
	"testing"
)

// TestArcanistBrandPublishesTriggerRate pins CalcTriggers.lua:1321-1341's ["arcanist brand"] entry and the two reads it feeds.
func TestArcanistBrandPublishesTriggerRate(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range []string{
		"0027-saboteur-spark-of-the-nova.xml",
		"0032-saboteur-spark-of-the-nova-2.xml",
		"1102-elementalist-frost-bomb.xml",
		"3493-ladder-elatycs_7290.ALL_YOUR_BASE.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}

			env := corpusEnvForLoadedData(t, build)
			if sd := env.Player.MainSkill.SkillData; sd == nil || !sd.TriggeredByBrand {
				t.Fatalf("REFUSING: %s's main skill is not supported by Arcanist Brand, so it no "+
					"longer exercises this rung", build)
			}
			for _, key := range []string{
				"TriggerRateCap", "EffectiveSourceRate", "SkillTriggerRate",
			} {
				want, ok := rec.Output[key]
				if !ok || want == 0 {
					t.Fatalf("REFUSING: PoB publishes no nonzero %s for %s; a build that does "+
						"not reach the dispatcher cannot pin this entry", key, build)
				}
				got, present := env.Player.Output[key]
				if !present {
					t.Errorf("%s: we publish no %s at all; PoB reports %v", build, key, want)
					continue
				}
				if math.Abs(got-want) > 1e-9 {
					t.Errorf("%s: %s = %v, PoB reports %v", build, key, got, want)
				}
			}
		})
	}
}
