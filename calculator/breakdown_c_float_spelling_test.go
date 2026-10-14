package calculator

import (
	"slices"
	"testing"
)

// TestBreakdownCFloatSpelling: LuaJIT's string.format spells infinities and NaN as C does; the lines are the pin's CALCS pass.
func TestBreakdownCFloatSpelling(t *testing.T) {
	requireGameData(t, "3.29")
	for _, tc := range []struct {
		build, key string
		want       []string
	}{
		{"0521-elementalist-penance-brand-15.xml", "EHPSurvivalTime", []string{
			"inf ^8(total average number of hits you can take)", "x 0.70 ^8enemy attack/cast time",
			"= inf seconds ^8(total time it would take to die)",
		}},
		{"10691-ladder-LaerMM37_9637.xXxNotMinionxXx.xml", "LifeRecoupRecoveryMax", []string{
			"= 0 ^8(total damage recoup amount)", "/ 4.00 ^8(over 4 seconds)", "= nan per second ^8",
		}},
	} {
		c := &Calculator{PoB: loadCorpusBuild(t, tc.build)}
		cache := NewEnvironmentCache()
		BuildOutputWithCache(c, OutputModeMain, cache)
		env := BuildOutputWithCache(c, OutputModeCalcs, cache)
		var got []string
		if e := env.Player.Breakdown.Data[tc.key]; e != nil {
			got = e.Lines
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s %s:\n got  %q\n want %q", tc.build, tc.key, got, tc.want)
		}
	}
}
