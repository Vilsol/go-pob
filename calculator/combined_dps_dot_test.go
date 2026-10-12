package calculator

import (
	"math"
	"os"
	"testing"
)

// TestCombinedDPSFoldsDotOnceViaTotalDotDPS pins CalcOffence.lua:6131-6133 — the single place upstream adds damage over time to CombinedDPS.
//
//	if not skillData.showAverage then
//	end
func TestCombinedDPSFoldsDotOnceViaTotalDotDPS(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build                    string
		combinedDPS, totalDotDPS float64
		showAverage              bool
	}{
		{"0123-elementalist-corrupting-fever.xml", 177760.92337481, 177760.92337481, false},
		{"0022-saboteur-blade-trap-of-greatswords.xml", 25413.2659625, 2233.6719355392, true},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			out := env.Player.Output

			if tc.totalDotDPS == 0 {
				t.Fatal("fixture is useless: PoB's TotalDotDPS is 0, so folding it changes nothing")
			}
			if got := out["TotalDotDPS"]; math.Abs(got-tc.totalDotDPS) > math.Abs(tc.totalDotDPS)*5e-3 {
				t.Fatalf("TotalDotDPS = %v, want %v (PoB) — the input moved, so this fixture no "+
					"longer isolates how CombinedDPS consumes it", got, tc.totalDotDPS)
			}
			if got := out["CombinedDPS"]; math.Abs(got-tc.combinedDPS) > math.Abs(tc.combinedDPS)*5e-3 {
				t.Errorf("CombinedDPS = %v, want %v (PoB) — CalcOffence.lua:6131-6133 adds "+
					"TotalDotDPS exactly once and only when showAverage is false (here %v)",
					got, tc.combinedDPS, tc.showAverage)
			}
		})
	}
}
