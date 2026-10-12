package calculator

import (
	"math"
	"os"
	"testing"
)

// TestCriticalCullUsesHitRate pins CalcOffence.lua:3214-3217.
//
//	local criticalCull = skillModList:Max(cfg, "CriticalCullPercent") or 0
//	if criticalCull > 0 then
//		criticalCull = m_min(criticalCull, criticalCull * (1 - (1 - output.CritChance / 100) ^ hitRate))
//	end
func TestCriticalCullUsesHitRate(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0208-hierophant-lightning-tendrils-of-eccentricity.xml")

	if got := env.Player.Output["CullPercent"]; got == 0 {
		t.Fatal("fixture is useless: CullPercent is 0, so the critical-cull branch never ran")
	}

	for _, tc := range []struct {
		stat string
		want float64
	}{
		{"CullPercent", 9.6414169920536},
		{"CullMultiplier", 1.1067017285033},
	} {
		got := env.Player.Output[tc.stat]
		if math.Abs(got-tc.want) > tc.want*1e-6 {
			t.Errorf("%s = %v, want %v (PoB) — CalcOffence.lua:3216 raises (1 - critChance) to the "+
				"HIT RATE; a bare critChance multiply gives %v", tc.stat, got, tc.want,
				10*env.Player.Output["CritChance"]/100)
		}
	}
}
