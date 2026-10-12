package calculator

import (
	"math"
	"os"
	"testing"
)

// TestCalcLifeHitPoolWithLossPrevention pins CalcDefence.lua:177-181 as a pure function.
//
//	local function calcLifeHitPoolWithLossPrevention(life, maxLife, lifeLossPrevented, lifeLossBelowHalfPrevented)
//		local halfLife = maxLife * 0.5
//		local aboveLow = m_max(life - halfLife, 0)
//			+ m_min(life, halfLife) / (1 - lifeLossBelowHalfPrevented / 100) / (1 - lifeLossPrevented / 100)
func TestCalcLifeHitPoolWithLossPrevention(t *testing.T) {
	cases := []struct {
		name                    string
		life, maxLife           float64
		prevented, belowHalfRaw float64
		want                    float64
	}{
		{"below-half only", 4064, 4064, 0, 40, 5418.6666666667},
		{"flat only", 2967, 2967, 25, 0, 3956},
		{"both", 3880, 3880, 42, 40, 8919.5402298851},
		{"pool below half of max life", 2000, 6000, 0, 40, 3333.3333333333},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := calcLifeHitPoolWithLossPrevention(tc.life, tc.maxLife, tc.prevented, tc.belowHalfRaw)
			if math.Abs(got-tc.want) > math.Abs(tc.want)*1e-10 {
				t.Errorf("calcLifeHitPoolWithLossPrevention(%v, %v, %v, %v) = %v, PoB %v",
					tc.life, tc.maxLife, tc.prevented, tc.belowHalfRaw, got, tc.want)
			}
		})
	}
}

// TestLifeHitPoolChainIsPublished covers CalcDefence.lua:2230-2245.
func TestLifeHitPoolChainIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, build := range []string{
		"0001-necromancer-unknown.xml",
		"0287-pathfinder-toxic-rain-7.xml",
	} {
		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			out := env.Player.Output

			for _, name := range []string{
				"LifeHitPool", "preventedLifeLoss", "preventedLifeLossBelowHalf", "preventedLifeLossTotal",
			} {
				if _, ok := out[name]; !ok {
					t.Errorf("%s absent; PoB publishes it on every corpus build", name)
				}
			}

			if out["preventedLifeLoss"] != 0 || out["preventedLifeLossBelowHalf"] != 0 {
				t.Skipf("prevention is no longer 0 here (%v / %v) — a producer landed; re-pick "+
					"this fixture or pin the full formula instead",
					out["preventedLifeLoss"], out["preventedLifeLossBelowHalf"])
			}
			if got, want := out["LifeHitPool"], out["LifeRecoverable"]; got != want {
				t.Errorf("LifeHitPool = %v, want LifeRecoverable %v — with both prevention terms "+
					"at 0 the formula collapses to aboveLow + m_min(life, halfLife) = life", got, want)
			}
			if got := out["preventedLifeLossTotal"]; got != 0 {
				t.Errorf("preventedLifeLossTotal = %v, want 0", got)
			}
		})
	}
}
