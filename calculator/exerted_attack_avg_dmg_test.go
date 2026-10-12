package calculator

import (
	"math"
	"os"
	"testing"
)

// TestExertedAttackAvgDmgReadsAllThreeNames pins CalcOffence.lua:2789-2790.
func TestExertedAttackAvgDmgReadsAllThreeNames(t *testing.T) {
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
		arm   string
	}{
		{"0113-chieftain-ice-crash.xml", "overexertion only (1.10)"},
		{"0268-juggernaut-boneshatter-of-complex-trauma-3.xml", "overexertion and echoes"},
		{"0290-berserker-earthshatter-6.xml", "overexertion and echoes"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			rec, ok := oracle[tc.build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", tc.build)
			}
			want, ok := rec.Output["ExertedAttackAvgDmg"]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero ExertedAttackAvgDmg for %s", tc.build)
			}

			env := corpusEnvForLoadedData(t, tc.build)

			if wantUptime, ok := rec.Output["ExertedAttackUptimeRatio"]; ok {
				gotUptime := env.Player.Output["ExertedAttackUptimeRatio"]
				if math.Abs(gotUptime-wantUptime) > math.Abs(wantUptime)*oracleDumpTolerance {
					t.Fatalf("REFUSING: %s is contaminated — ExertedAttackUptimeRatio is %v "+
						"against PoB's %v, so ExertedAttackAvgDmg cannot be attributed to the "+
						"missing names", tc.build, gotUptime, wantUptime)
				}
			}

			got, present := env.Player.Output["ExertedAttackAvgDmg"]
			if !present {
				t.Errorf("%s: we never write ExertedAttackAvgDmg; PoB reports %v", tc.build, want)
				return
			}
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s [%s]: ExertedAttackAvgDmg = %v, PoB reports %v (ratio %.4f)",
					tc.build, tc.arm, got, want, got/want)
			}
		})
	}
}
