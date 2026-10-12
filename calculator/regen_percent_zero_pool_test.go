package calculator

import (
	"math"
	"os"
	"testing"
)

// TestRegenPercentIsZeroNotNaNOnAnEmptyPool pins CalcDefence.lua:1299.
//
//	output[resource.."RegenPercent"] = pool > 0 and round(... / pool * 100, 1) or 0
func TestRegenPercentIsZeroNotNaNOnAnEmptyPool(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0001-necromancer-unknown.xml")

	for _, name := range []string{"RageRegenPercent", "ManaRegenPercent", "EnergyShieldRegenPercent"} {
		got, ok := env.Player.Output[name]
		if !ok {
			t.Errorf("%s absent; PoB publishes it on every corpus build", name)
			continue
		}
		if math.IsNaN(got) || math.IsInf(got, 0) {
			t.Errorf("%s = %v — a zero pool must yield 0, not a poisoned value. "+
				"CalcDefence.lua:1299 guards the division with `pool > 0 and ... or 0`. A NaN "+
				"propagates into every consumer and compares equal to nothing, so it cannot be "+
				"graded or bisected; that is why the oracle counts it as its own defect class.",
				name, got)
		}
	}
}
