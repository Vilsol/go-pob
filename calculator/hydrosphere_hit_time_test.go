package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/moddb"
)

// TestHydrosphereWritesHitTimeOnlyOnItsFirstThreeParts pins act_int.lua:8972-8975.
func TestHydrosphereWritesHitTimeOnlyOnItsFirstThreeParts(t *testing.T) {
	fn := skillFuncsFor("Hydrosphere")["preDamageFunc"]
	if fn == nil {
		t.Fatal("REFUSING: Hydrosphere has no preDamageFunc registered")
	}

	run := func(part int) float64 {
		skill := &ActiveSkill{
			SkillPart:    part,
			SkillModList: moddb.NewModList(),
			SkillCfg:     &moddb.ListCfg{},
			SkillData:    &SkillData{RepeatFrequency: 0.5},
		}
		fn(skill, map[string]float64{}, nil)
		return skill.SkillData.HitTimeOverride
	}

	for _, part := range []int{1, 2, 3} {
		if got := run(part); got != 0.5 {
			t.Fatalf("part %d: hitTimeOverride = %v, want 0.5 — this arm must still fire", part, got)
		}
	}

	for _, part := range []int{4, 5, 6, 7} {
		if got := run(part); got != 0 {
			t.Fatalf("part %d: hitTimeOverride = %v, want 0 — :8973 gates the write on parts 1-3 and "+
				"the skill declares seven, so this publishes a HitTime upstream withholds", part, got)
		}
	}
}
