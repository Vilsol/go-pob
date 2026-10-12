package calculator

import (
	"math"
	"os"
	"testing"
)

// local damageCategoryConfig = env.configInput.enemyDamageType or "Average"
// if damageCategoryConfig ~= "DamageOverTime" then
//
//	output.ConfiguredEvadeChance = output[damageCategoryConfig.."EvadeChance"] or 0
//	if worstOf > 1 then ... square it, and again when worstOf == 4 ... end
//
// end
var configuredEvadeChanceBuilds = []string{
	"0011-saboteur-cyclone.xml",
	"0017-deadeye-spectral-throw-of-materialising.xml",
	"0026-ascendant-fire-trap-of-blasting.xml",
}

func TestConfiguredEvadeChanceIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range configuredEvadeChanceBuilds {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			want, ok := rec.Output["ConfiguredEvadeChance"]
			if !ok {
				t.Fatalf("REFUSING: PoB reports no ConfiguredEvadeChance for %s", build)
			}
			if want == 0 {
				t.Fatalf("REFUSING: PoB reports ConfiguredEvadeChance 0 for %s, so the case agrees "+
					"with an absent producer and cannot discriminate", build)
			}

			env := corpusEnvForLoadedData(t, build)
			got := env.Player.Output["ConfiguredEvadeChance"]
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: ConfiguredEvadeChance = %v, PoB reports %v (got 0 means nothing "+
					"publishes the key, or it is published after its consumer reads it)", build, got, want)
			}

			if wantCrit, ok := rec.Output["EnemyCritChance"]; ok && wantCrit > 0 {
				gotCrit := env.Player.Output["EnemyCritChance"]
				if math.Abs(gotCrit-wantCrit) > math.Abs(wantCrit)*oracleDumpTolerance {
					t.Errorf("%s: EnemyCritChance = %v, PoB reports %v -- ConfiguredEvadeChance is "+
						"correct above, so its consumer at CalcDefence.lua:1694 is reading it too early",
						build, gotCrit, wantCrit)
				}
			}
		})
	}
}
