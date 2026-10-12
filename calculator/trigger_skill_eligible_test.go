package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/moddb"
)

// TestTriggerSkillEligibleDropsTheThreeConjuncts pins CalcTriggers.lua:82.
func TestTriggerSkillEligibleDropsTheThreeConjuncts(t *testing.T) {
	newSkill := func() *ActiveSkill {
		return &ActiveSkill{
			SkillFlags: map[SkillFlag]bool{},
			SkillTypes: map[data.SkillType]bool{},
			SkillCfg:   &moddb.ListCfg{SkillCond: map[string]bool{}},
		}
	}
	if !triggerSkillEligible(newSkill()) {
		t.Fatal("REFUSING: a plain skill is ineligible — the fixture reaches no conjunct")
	}

	disabled := newSkill()
	disabled.SkillFlags[SkillFlagDisable] = true
	if triggerSkillEligible(disabled) {
		t.Error("a disabled skill is eligible — skillFlags.disable is not read")
	}

	mirage := newSkill()
	mirage.SkillCfg.SkillCond["usedByMirage"] = true
	if triggerSkillEligible(mirage) {
		t.Error("a mirage copy is eligible — skillCond[\"usedByMirage\"] is not read")
	}

	otherThing := newSkill()
	otherThing.SkillTypes[data.SkillTypeOtherThingUsesSkill] = true
	if triggerSkillEligible(otherThing) {
		t.Error("an OtherThingUsesSkill skill is eligible — the skill type is not read")
	}

	bare := &ActiveSkill{}
	if !triggerSkillEligible(bare) {
		t.Error("a skill with no flags and no cfg is ineligible — PoB's `and` guards make it eligible")
	}
}
