package calculator

import (
	"math"
	"os"
	"testing"
)

// TestRightmostMagicUtilityFlasksApply pins CalcPerform.lua:1544-1565's RIGHT arm and the parser row that feeds it (ModParser.lua:5223).
func TestRightmostMagicUtilityFlasksApply(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		stat  string
		want  float64
		why   string
	}{
		{"0180-elementalist-winter-orb-11.xml", "ChaosResist", 75,
			"a rightmost-applied magic utility flask grants Chaos Resistance"},

		{"0002-pathfinder-unknown.xml", "MovementSpeedMod", 1.656,
			"CONTROL: the LEFT (Mageblood) arm, at PoB's value (testdata/oracle-3.29)"},
	} {
		t.Run(tc.build+"/"+tc.stat, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output[tc.stat]
			if !ok {
				t.Fatalf("%s ABSENT", tc.stat)
			}
			if math.Abs(got-tc.want) > math.Abs(tc.want)*0.005 {
				t.Errorf("%s = %v, want %v — %s", tc.stat, got, tc.want, tc.why)
			}
		})
	}
}
