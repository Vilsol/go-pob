package calculator

import (
	"math"
	"testing"
)

// TestTriggerCastOnBlockUniqueCalcsPass: The CALCS pass's main skill is the one a Svalinn / Festering Resentment "Cast on Block" support triggers.
//
//pob:src/Modules/CalcTriggers.lua:1498-1511
func TestTriggerCastOnBlockUniqueCalcsPass(t *testing.T) {
	requireGameData(t, "3.29")
	for _, tc := range []struct {
		build string
		want  map[string]float64
	}{
		{"8595-ladder-zibazmeypro_7731.OldGameIsDed.xml", map[string]float64{
			"SkillTriggerRate": 0.65744400527009, "EffectiveSourceRate": 0.65876152832675,
			"TriggerRateCap": 0.65876152832675, "Speed": 0.65744400527009,
		}},
		{"12800-ladder-ZaphodTwo_3286.ArtOfNotGivingFuck.xml", map[string]float64{
			"SkillTriggerRate": 0.083250083250083, "EffectiveSourceRate": 0.16650016650017,
			"TriggerRateCap": 0.16650016650017, "Speed": 0.083250083250083,
		}},
	} {
		t.Run(tc.build, func(t *testing.T) {
			c := &Calculator{PoB: loadCorpusBuild(t, tc.build)}
			cache := NewEnvironmentCache()
			BuildOutputWithCache(c, OutputModeMain, cache)
			env := BuildOutputWithCache(c, OutputModeCalcs, cache)
			for key, want := range tc.want {
				if got := env.Player.Output[key]; math.Abs(got-want) > 1e-9*want {
					t.Errorf("%s = %v, PoB %v", key, got, want)
				}
			}
		})
	}
}
