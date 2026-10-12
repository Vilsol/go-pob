package calculator

import (
	"math"
	"os"
	"testing"
)

// TestDoomBlastDefaultSourceAndOverlaps pins ConfigOptions.lua:408's defaultIndex.
func TestDoomBlastDefaultSourceAndOverlaps(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	for _, c := range []struct {
		build, key string
		want       float64
	}{
		{"0271-occultist-doom-blast.xml", "EffectiveSourceRate", 2.4},
		{"0271-occultist-doom-blast.xml", "SkillTriggerRate", 4.8},
		{"1332-occultist-doom-blast-4.xml", "SkillTriggerRate", 10.933333333333},
		{"1338-occultist-doom-blast-5.xml", "SkillTriggerRate", 10.933333333333},
		{"1307-occultist-doom-blast-2.xml", "SkillTriggerRate", 4.88488},
	} {
		env := corpusEnvForLoadedData(t, c.build)
		if got := env.Player.Output[c.key]; math.Abs(got-c.want) > c.want*1e-6 {
			t.Errorf("%s %s = %v, PoB has %v", c.build, c.key, got, c.want)
		}
	}
}
