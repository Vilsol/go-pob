package calculator

import (
	"math"
	"os"
	"testing"
)

// TestAverageBurstReadsRepeatsAndSealPenalty pins two lines that were both absent.
//
//	if skillData.averageBurstHits then output.AverageBurstHits = skillData.averageBurstHits
//	elseif output.Repeats and output.Repeats > 1 then output.AverageBurstHits = output.Repeats end
func TestAverageBurstReadsRepeatsAndSealPenalty(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build             string
		hits, burst, avg  float64
		wantPenaltyActive bool
	}{
		{"0046-elementalist-arc.xml", 3, 4839667.3138826, 1613222.4379609, false},
		{"0040-trickster-exsanguinate-of-transmission.xml", 6, 469039.20459451, 97716.50095719, true},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			out := env.Player.Output

			if got := out["AverageBurstHits"]; math.Abs(got-tc.hits) > 1e-9 {
				t.Errorf("AverageBurstHits = %v, want %v (PoB) — CalcOffence.lua:2115-2119", got, tc.hits)
			}
			plain := tc.avg * tc.hits
			penaltyMatters := math.Abs(plain-tc.burst) > math.Abs(tc.burst)*1e-6
			if penaltyMatters != tc.wantPenaltyActive {
				t.Fatalf("fixture no longer exercises what it claims: AverageDamage*AverageBurstHits "+
					"= %v against PoB's AverageBurstDamage %v (penalty active = %v, want %v)",
					plain, tc.burst, penaltyMatters, tc.wantPenaltyActive)
			}
			wantRatio := tc.burst / tc.avg
			if got := out["AverageBurstDamage"] / out["AverageDamage"]; math.Abs(got-wantRatio) > 1e-9 {
				t.Errorf("AverageBurstDamage/AverageDamage = %v, want %v (PoB) — "+
					"CalcOffence.lua:3709-3710 is AverageDamage * (1 + (hits-1)*repeatPenalty)",
					got, wantRatio)
			}
			if !tc.wantPenaltyActive {
				if got := out["AverageBurstDamage"]; math.Abs(got-tc.burst) > math.Abs(tc.burst)*1e-9 {
					t.Errorf("AverageBurstDamage = %v, want %v (PoB)", got, tc.burst)
				}
			}
		})
	}
}
