package calculator

import (
	"math"
	"testing"
)

// TestExplosiveTrapRadiusMargin: radiusTertiaryBaseMargin averages the small-explosion radius.
func TestExplosiveTrapRadiusMargin(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "2363-ladder-smrnov2025_5399.jdudodgeram.xml")
	for k, want := range map[string]float64{"AreaOfEffectRadiusTertiary": 18.508196721311, "OverlapChance": 68.304262295082} {
		if got := env.Player.Output[k]; math.Abs(got-want) > 1e-9*want {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}
