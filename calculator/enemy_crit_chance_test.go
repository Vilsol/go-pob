package calculator

import (
	"math"
	"os"
	"testing"
)

// local enemyCritChance = enemyDB:Flag(nil, "NeverCrit") and 0 or enemyDB:Flag(nil, "AlwaysCrit") and 100
//
//	or m_max(m_min((modDB:Override(nil, "enemyCritChance") or env.configInput["enemyCritChance"]
var enemyCritChanceBuilds = []string{
	"0004-trickster-lightning-tendrils-of-eccentricity.xml",
	"0013-inquisitor-divine-ire.xml",
}

func TestEnemyCritChanceUsesTheConfigPlaceholder(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range enemyCritChanceBuilds {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			want, ok := rec.Output["EnemyCritChance"]
			if !ok {
				t.Fatalf("REFUSING: PoB reports no EnemyCritChance for %s", build)
			}
			if want == 0 {
				t.Fatalf("REFUSING: PoB reports EnemyCritChance 0 for %s; the old code returned 0 "+
					"too, so this case cannot discriminate", build)
			}
			if ev := rec.Output["ConfiguredEvadeChance"]; ev != 0 {
				t.Fatalf("REFUSING: PoB's ConfiguredEvadeChance is %v for %s, and nothing in this "+
					"tree produces that key, so the mismatch would not be about the placeholder", ev, build)
			}

			env := corpusEnvForLoadedData(t, build)

			if _, fromInput := env.Build.GetNumberOptionOK("enemyCritChance"); fromInput {
				t.Fatalf("REFUSING: %s now carries an explicit enemyCritChance <Input>, so it no "+
					"longer exercises the configPlaceholder fallback this fix restored", build)
			}

			got := env.Player.Output["EnemyCritChance"]
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: EnemyCritChance = %v, PoB reports %v.\n"+
					"  got 0 means the key is not published at all; got a value ignoring the "+
					"placeholder means the 3.20 <Input>-only read is back", build, got, want)
			}
		})
	}
}
