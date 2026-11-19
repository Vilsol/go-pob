package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/moddb"
)

// TestTriggerRateCapWithoutActionCooldownIsHuge: CalcTriggers.lua:583-586: with no action cooldown the cap stays m_huge.
func TestTriggerRateCapWithoutActionCooldownIsHuge(t *testing.T) {
	mainSkill := &ActiveSkill{SkillModList: moddb.NewModList(), SkillCfg: &moddb.ListCfg{}, SkillData: &SkillData{}, SkillTypes: map[data.SkillType]bool{}}
	source := &ActiveSkill{SkillModList: moddb.NewModList(), SkillCfg: &moddb.ListCfg{}, SkillData: &SkillData{}, SkillTypes: map[data.SkillType]bool{}}
	env := &Environment{Player: &Actor{MainSkill: mainSkill}}

	output := map[string]float64{}
	calcActualTriggerRate(env, source, nil, []*triggerRotationSkill{{}}, output, 100)

	if got := output["TriggerRateCap"]; !math.IsInf(got, 1) {
		t.Fatalf("TriggerRateCap = %v, want +Inf (m_huge)", got)
	}
}
