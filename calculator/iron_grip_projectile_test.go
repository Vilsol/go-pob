package calculator

import (
	"math"
	"testing"
)

// TestIronGripNoDoubleDipForProjectileAttacks pins CalcOffence.lua:659-662 (upstream #10188).
func TestIronGripNoDoubleDipForProjectileAttacks(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "15877-saboteur-ghostly-artillery.xml")
	for k, w := range map[string]float64{"AverageDamage": 207129.7844, "TotalDPS": 1868769.0749268001} {
		if got := env.Player.Output[k]; math.Abs(got-w) > 1e-4*w {
			t.Errorf("%s = %v, want %v (PoB)", k, got, w)
		}
	}
}
