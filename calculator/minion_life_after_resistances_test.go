package calculator

import (
	"math"
	"testing"
)

// TestMinionLifeRecomputedAfterResistances: a minion's life/mana is recomputed once its resistances exist.
func TestMinionLifeRecomputedAfterResistances(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "15406-variant-15400.toxic-rain-sporeburst.xml")
	if env.Minion == nil {
		t.Fatalf("fixture premise moved: no minion")
	}
	if env.Minion.Output["FireResistOverCap"] <= 0 {
		t.Fatalf("fixture premise moved: minion FireResistOverCap = %v", env.Minion.Output["FireResistOverCap"])
	}
	const want = 3503.0
	if got := env.Minion.Output["Life"]; math.Abs(got-want) > 0.5 {
		t.Errorf("minion Life = %v, want %v (PoB)", got, want)
	}
}
