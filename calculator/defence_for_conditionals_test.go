package calculator

import (
	"math"
	"os"
	"testing"
)

// TestDefenceForConditionalsRunsAsItsOwnPass pins the two-pass structure upstream uses.
func TestDefenceForConditionalsRunsAsItsOwnPass(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, tc := range []struct {
		build string
		stats []string
	}{
		{"3598-ladder-lIlIlIlIl_1773.IkaIgnite.xml", []string{"EnergyShield", "Armour"}},
		{"0332-chieftain-righteous-fire-5.xml", []string{"Armour"}},
		{"1118-chieftain-frostblink-of-wintry-blast.xml", []string{"Armour"}},
		{"1193-juggernaut-bladefall-of-trarthus.xml", []string{"Armour"}},
	} {
		t.Run(tc.build, func(t *testing.T) {
			rec, ok := oracle[tc.build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", tc.build)
			}
			env := corpusEnvForLoadedData(t, tc.build)

			for _, stat := range tc.stats {
				want, ok := rec.Output[stat]
				if !ok || want == 0 {
					t.Fatalf("REFUSING: PoB reports no nonzero %s for %s", stat, tc.build)
				}
				got := env.Player.Output[stat]
				if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
					t.Errorf("%s: %s = %v, PoB reports %v (ratio %.4f)", tc.build, stat, got, want, got/want)
				}
			}
		})
	}
}
