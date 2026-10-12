package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestBannerValourScalesTheBannerAura pins CalcPerform.lua:1298-1303 and the two inputs it needs.
func TestBannerValourScalesTheBannerAura(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0070-champion-frostblink.xml")

	if got := env.ModDB.Sum(mod.TypeBase, nil, "MaximumValour"); got != 105 {
		t.Errorf("MaximumValour = %v, want 105 (50 base + 55 from tree)", got)
	}
	if got := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:BannerValour"); got != 105 {
		t.Errorf("Multiplier:BannerValour = %v, want 105", got)
	}
	if got := env.Player.Output["EffectiveSpellSuppressionChance"]; got != 100 {
		t.Errorf("EffectiveSpellSuppressionChance = %v, want 100", got)
	}
	for _, tc := range []struct {
		key  string
		want float64
	}{
		{"PhysicalAfterReductionTakenHitMulti", 0.544},
		{"PhysicalTakenHit", 135.123049152},
	} {
		if got := env.Player.Output[tc.key]; math.Abs(got-tc.want) > math.Abs(tc.want)*1e-9 {
			t.Errorf("%s = %v, want %v (PoB)", tc.key, got, tc.want)
		}
	}

	ctrl := corpusEnvForLoadedData(t, "0004-trickster-lightning-tendrils-of-eccentricity.xml")
	if got := ctrl.ModDB.Sum(mod.TypeBase, nil, "Multiplier:BannerValour"); got != 0 {
		t.Errorf("CONTROL Multiplier:BannerValour = %v, want 0 (no banner planted)", got)
	}
}
