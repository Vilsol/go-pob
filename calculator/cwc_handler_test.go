package calculator

import (
	"math"
	"os"
	"testing"
)

// TestCWCTriggerRateRunsTheRotation pins CalcTriggers.lua:219-387 on a build whose Cast while Channelling group triggers Berserk alongside other cooldown spells.
func TestCWCTriggerRateRunsTheRotation(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	build := loadCorpusBuild(t, "4956-ladder-Icebloom008_0770.СаблимНаНищету.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
	out := env.Player.Output

	for _, tc := range []struct {
		key  string
		want float64
	}{
		{"SkillTriggerRate", 0.2375},
		{"TriggerRateCap", 0.24038461538462},
		{"ChannelTimeToTrigger", 0.32},
	} {
		if got := out[tc.key]; math.Abs(got-tc.want) > tc.want*1e-6 {
			t.Errorf("%s = %v, want %v (PoB's value, to 1e-6)", tc.key, got, tc.want)
		}
	}
}
