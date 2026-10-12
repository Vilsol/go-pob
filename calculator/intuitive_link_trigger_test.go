package calculator

import (
	"math"
	"os"
	"testing"
)

// TestIntuitiveLinkTriggerEntry pins CalcTriggers.lua:1483-1497.
func TestIntuitiveLinkTriggerEntry(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	for _, c := range []struct {
		build string
		want  map[string]float64
	}{
		{"0648-luminary-blade-vortex-of-the-scythe.xml", map[string]float64{"EffectiveSourceRate": 1000, "TriggerRateCap": 1.8939393939394, "SkillTriggerRate": 2}},
		{"1540-luminary-blade-vortex-of-the-scythe-2.xml", map[string]float64{"EffectiveSourceRate": 1000, "TriggerRateCap": 1.8939393939394, "SkillTriggerRate": 2}},
		{"0482-luminary-intuitive-link.xml", map[string]float64{"EffectiveSourceRate": 5, "TriggerRateCap": 1.8939393939394, "SkillTriggerRate": 1.67}},
		{"1474-luminary-void-sphere-of-rending.xml", map[string]float64{"EffectiveSourceRate": 2, "TriggerRateCap": 1.8939393939394, "SkillTriggerRate": 1.092}},
		{"1479-luminary-discharge.xml", map[string]float64{"EffectiveSourceRate": 2, "TriggerRateCap": 1.1223344556678, "SkillTriggerRate": 1}},
	} {
		env := corpusEnvForLoadedData(t, c.build)
		for key, want := range c.want {
			if got := env.Player.Output[key]; math.Abs(got-want) > want*1e-6 {
				t.Errorf("%s %s = %v, PoB has %v", c.build, key, got, want)
			}
		}
	}
}
