package calculator

import (
	"os"
	"testing"
)

// TestTriggeredSkillGetsNoSelfCastFlag pins the conjunct on CalcActiveSkill.lua:421.
func TestTriggeredSkillGetsNoSelfCastFlag(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, c := range []struct {
		build string
		want  bool
		why   string
	}{
		{"0011-saboteur-cyclone", false, "PoB skillFlags: triggered=true, selfCast absent"},
		{"0027-saboteur-spark-of-the-nova", false, "PoB skillFlags: triggered=true, selfCast absent"},
		{"0001-necromancer-unknown", true, "the control — PoB skillFlags: selfCast=true"},
	} {
		t.Run(c.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, c.build)
			skill := env.Player.MainSkill
			if skill == nil {
				t.Fatalf("REFUSING: %s has no main skill, so this fixture measures nothing", c.build)
			}
			if got := skill.SkillFlags[SkillFlagSelfCast]; got != c.want {
				t.Errorf("%s: selfCast = %v, want %v — %s", c.build, got, c.want, c.why)
			}
		})
	}
}
