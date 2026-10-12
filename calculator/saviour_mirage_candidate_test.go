package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestSaviourMirageWarriorCandidateDropsTheTwoConjuncts pins CalcMirages.lua:123.
func TestSaviourMirageWarriorCandidateDropsTheTwoConjuncts(t *testing.T) {
	swordOneHand := mod.MFlagSword | mod.MFlagWeapon1H

	newSkill := func() *ActiveSkill {
		flags := swordOneHand
		return &ActiveSkill{
			SkillTypes: map[data.SkillType]bool{data.SkillTypeAttack: true},
			SkillCfg:   &moddb.ListCfg{Flags: &flags, SkillCond: map[string]bool{}},
		}
	}
	main := newSkill()

	if !saviourMirageWarriorCandidate(newSkill(), main) {
		t.Fatal("REFUSING: a plain 1H sword attack is not a candidate — the fixture reaches no conjunct")
	}

	if saviourMirageWarriorCandidate(main, main) {
		t.Error("the main skill is its own candidate — `skill ~= env.player.mainSkill` is not read")
	}

	notAttack := newSkill()
	delete(notAttack.SkillTypes, data.SkillTypeAttack)
	if saviourMirageWarriorCandidate(notAttack, main) {
		t.Error("a non-attack is a candidate — SkillType.Attack is not read")
	}

	summonsTotem := newSkill()
	summonsTotem.SkillTypes[data.SkillTypeSummonsTotem] = true
	if saviourMirageWarriorCandidate(summonsTotem, main) {
		t.Error("a totem-summoning sword attack is a candidate — SkillType.SummonsTotem is not read")
	}

	marked := newSkill()
	marked.Marked = true
	if saviourMirageWarriorCandidate(marked, main) {
		t.Error("a mirage copy is a candidate — Marked (skillCond[\"usedByMirage\"]) is not read")
	}

	condMirage := newSkill()
	condMirage.SkillCfg.SkillCond["usedByMirage"] = true
	if saviourMirageWarriorCandidate(condMirage, main) {
		t.Error("a skillCond-marked copy is a candidate — skillCond[\"usedByMirage\"] is not read")
	}

	twoHand := newSkill()
	th := mod.MFlagSword
	twoHand.SkillCfg.Flags = &th
	if saviourMirageWarriorCandidate(twoHand, main) {
		t.Error("a sword with no Weapon1H flag is a candidate — the flag mask is not read")
	}

	notSword := newSkill()
	ns := mod.MFlagWeapon1H
	notSword.SkillCfg.Flags = &ns
	if saviourMirageWarriorCandidate(notSword, main) {
		t.Error("a non-sword 1H weapon is a candidate — the flag mask is not read")
	}

	noCfg := newSkill()
	noCfg.SkillCfg = nil
	if saviourMirageWarriorCandidate(noCfg, main) {
		t.Error("a skill with no SkillCfg is a candidate")
	}
}
