package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob-data/raw"
)

// TestMinionSkillSupportReadsOnlyTheSummonTypes: CalcTools.lua:99-107 takes only the skill types from a minion
// skill's summonSkill; the from-item refusal still reads the minion skill itself. So an item-granted support can
// support a minion of an item-granted summon. The want is the pinned function run under luajit on the same shape.
//
//pob:src/Modules/CalcTools.lua:86-161
func TestMinionSkillSupportReadsOnlyTheSummonTypes(t *testing.T) {
	ge := func(id string) *GrantedEffect {
		return &GrantedEffect{Raw: &poe.GrantedEffect{GrantedEffect: raw.GrantedEffect{ID: id}}}
	}
	support, summonEffect, minionEffect := ge("SupportAddedLightningDamage"), ge("ApproachingFlames"), ge("Fireball")
	support.fromItem = true
	if !support.FromItem() || !summonEffect.FromItem() || minionEffect.FromItem() {
		t.Fatal("premise broken: the fixture's from-item flags moved")
	}

	player, minion := &Actor{}, &Actor{}
	enemy := &Actor{Player: player}
	player.Enemy, minion.Enemy, player.Player, minion.Player = enemy, enemy, player, player
	summon := &ActiveSkill{Actor: player, SkillTypes: typeGateOpen(), ActiveEffect: &GemEffect{GrantedEffect: summonEffect}}
	minionSkill := &ActiveSkill{Actor: minion, SkillTypes: typeGateOpen(), SummonSkill: summon, ActiveEffect: &GemEffect{GrantedEffect: minionEffect}}

	if !CalcCanGrantedEffectSupportActiveSkill(support, minionSkill, false) {
		t.Error("the item support is refused on the minion skill, PoB allows it")
	}
}
