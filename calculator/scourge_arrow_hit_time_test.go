package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestScourgeArrowWritesHitTimeMultiplier pins act_dex.lua:12987-12989.
func TestScourgeArrowWritesHitTimeMultiplier(t *testing.T) {
	fn, ok := skillFuncsFor("ScourgeArrow")["preDamageFunc"]
	if !ok {
		t.Fatalf("ScourgeArrow declares no preDamageFunc — act_dex.lua:12987 is a preDamageFunc, " +
			"and registering it as an initialFunc is the defect this test exists for")
	}
	if _, wrongHook := skillFuncsFor("ScourgeArrow")["initialFunc"]; wrongHook {
		t.Errorf("ScourgeArrow still declares an initialFunc — the pin declares only a preDamageFunc")
	}

	for _, c := range []struct {
		stages float64
		want   float64
	}{
		{0, 0.5},
		{1, 0.5},
		{2, 1.5},
		{10, 9.5},
	} {
		t.Run("", func(t *testing.T) {
			list := moddb.NewModList()
			if c.stages != 0 {
				list.AddMod(mod.NewFloat("Multiplier:ScourgeArrowStage", mod.TypeBase, c.stages).Source("Test"))
			}
			as := &ActiveSkill{SkillData: &SkillData{}, SkillModList: list}
			fn(as, map[string]float64{}, nil)

			if got := as.SkillData.HitTimeMultiplier; got != c.want {
				t.Errorf("%v stages: HitTimeMultiplier = %v, want %v — max(sum - 0.5, 0.5)",
					c.stages, got, c.want)
			}
			if as.SkillData.DpsMultiplier != 0 {
				t.Errorf("%v stages: DpsMultiplier = %v, want 0 — the pin writes no dpsMultiplier "+
					"for this skill, so writing one is an over-application",
					c.stages, as.SkillData.DpsMultiplier)
			}
		})
	}
}
