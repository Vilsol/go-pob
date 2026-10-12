package calculator

import (
	"math"
	"os"
	"testing"
)

// TestCannotRecoverLifeOutsideLeechGatesLifeRecovery pins the three CalcDefence.lua sites that read `CannotRecoverLifeOutsideLeech`.
func TestCannotRecoverLifeOutsideLeechGatesLifeRecovery(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		stat  string
		want  float64
		why   string
	}{
		{"1066-slayer-wild-strike.xml", "LifeRecoveryRateMod", 1,
			"CalcDefence.lua:1195 — the flag pins the rate mod at 1"},
		{"0832-juggernaut-heavy-strike-of-trarthus-2.xml", "LifeRecoveryRateMod", 1,
			"CalcDefence.lua:1195 — the flag pins the rate mod at 1"},

		{"1884-ladder-ShinX__4985.Dippiebeavermelon.xml", "LifeOnBlock", 0,
			"CalcDefence.lua:1525 — the flag zeroes recovery on block"},
		{"1884-ladder-ShinX__4985.Dippiebeavermelon.xml", "LifeRecoveryRateMod", 1,
			"CalcDefence.lua:1195 — the flag pins the rate mod at 1"},

		{"0041-pathfinder-vaal-fireball.xml", "LifeRecoveryRateMod", 1.34,
			"CONTROL: no flag, so the rate mod must NOT be pinned to 1"},
	} {
		t.Run(tc.build+"/"+tc.stat, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output[tc.stat]
			if !ok {
				t.Fatalf("%s ABSENT", tc.stat)
			}
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("%s = %v, want %v — %s", tc.stat, got, tc.want, tc.why)
			}
		})
	}
}
