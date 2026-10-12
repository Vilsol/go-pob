package calculator

import (
	"math"
	"os"
	"testing"
)

// TestPerSecondCostUnleashSpeedAndAccumulation covers CalcOffence.lua:5803-5824, the per-second cost block.
func TestPerSecondCostUnleashSpeedAndAccumulation(t *testing.T) {
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
		key   string
	}{
		{"0040-trickster-exsanguinate-of-transmission.xml", "LifePerSecondCost"},
		{"0701-assassin-eye-of-winter.xml", "ManaPerSecondCost"},
		{"0810-reliquarian-arc-of-oscillating-2.xml", "ManaPerSecondCost"},
		{"1179-gladiator-bladefall-of-volleys.xml", "ManaPerSecondCost"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			rec, ok := oracle[tc.build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", tc.build)
			}
			want, ok := rec.Output[tc.key]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero %s for %s — this fixture grades "+
					"nothing", tc.key, tc.build)
			}

			env := corpusEnvForLoadedData(t, tc.build)
			got := env.Player.Output[tc.key]
			if math.IsInf(got, 0) || math.IsNaN(got) {
				t.Fatalf("%s: %s is %v — the :5815 nil-check conjunct is missing again. Do NOT "+
					"guard the division; restore the conjunct.", tc.build, tc.key, got)
			}
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: %s = %v, PoB reports %v (ratio %.4f)",
					tc.build, tc.key, got, want, got/want)
			}
		})
	}
}
