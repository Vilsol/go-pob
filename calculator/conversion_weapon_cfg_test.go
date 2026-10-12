package calculator

import (
	"math"
	"os"
	"testing"
)

// TestEnemyPhysicalDamageReductionCap pins the ENEMY cap on physical damage reduction.
func TestEnemyPhysicalDamageReductionCap(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}

	builds := []string{
		"0077-pathfinder-toxic-rain-2",
		"0165-occultist-unknown-2",
		"0166-occultist-unknown-3",
	}

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("cannot load the oracle blob: %v", err)
	}

	keys := []string{
		"PhysicalHitAverage", "PhysicalCritAverage",
		"FireHitAverage", "ColdHitAverage", "LightningHitAverage", "ChaosHitAverage",
		"TotalMin", "TotalMax",
	}

	for _, name := range builds {
		t.Run(name, func(t *testing.T) {
			rec, ok := oracle[name+".xml"]
			if !ok {
				t.Fatalf("%s has no record in the oracle blob", name)
			}
			if rec.TreeSources < oracleMinTreeSources {
				t.Fatalf("%s is an EXCLUDED build (treeSources %d) — the oracle never grades it, so "+
					"there is no divergence here to pin", name, rec.TreeSources)
			}
			if len(rec.MainHand) == 0 {
				t.Fatalf("%s carries no MainHand table in the oracle — it is not an attack build, "+
					"so it cannot exercise the per-weapon conversion table at all", name)
			}

			env := corpusEnvForLoadedData(t, name)
			got := env.Player.OutputTable[OutTableMainHand]

			for _, k := range keys {
				want, hasWant := rec.MainHand[k]
				if !hasWant {
					continue
				}
				if math.Abs(got[k]-want) > math.Abs(want)*oracleDumpTolerance {
					t.Errorf("MainHand.%s: PoB %g, ours %g (ratio %.4f)", k, want, got[k],
						got[k]/want)
				}
			}
		})
	}
}
