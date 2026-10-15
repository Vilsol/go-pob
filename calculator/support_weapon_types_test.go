package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob-data/raw"

	"github.com/Vilsol/go-pob/data"
)

// TestSupportWeaponTypesGate pins CalcTools.lua:119-159, including the support-granted branch (:127-135, #10287).
func TestSupportWeaponTypesGate(t *testing.T) {
	ge := func(id string) *GrantedEffect {
		return &GrantedEffect{Raw: &poe.GrantedEffect{GrantedEffect: raw.GrantedEffect{ID: id}}}
	}
	for _, id := range []string{"SupportArrowNova", "SupportSacredWisps", "SupportNightblade",
		"ArtilleryBallista", "AnimateGuardianSmite", "Smite"} {
		if len(ge(id).WeaponTypes()) == 0 {
			t.Fatalf("fixture is inert: %s declares no weaponTypes", id)
		}
	}

	player := &Actor{}
	enemy := &Actor{}
	player.Enemy, enemy.Enemy, player.Player, enemy.Player = enemy, player, player, player

	varunastra := &Actor{Enemy: enemy, WeaponData1: &SkillData{CountsAsAll1H: true}}
	clawEquipped := &Actor{Enemy: enemy, WeaponData1: &SkillData{Type: data.Claw}}

	skillOn := func(actor *Actor, activeID string) *ActiveSkill {
		return &ActiveSkill{
			Actor:        actor,
			SkillTypes:   typeGateOpen(),
			ActiveEffect: &GemEffect{GrantedEffect: ge(activeID)},
		}
	}

	for _, c := range []struct {
		name    string
		support string
		actor   *Actor
		active  string
		want    bool
	}{
		{"bow support on bow skill", "SupportArrowNova", player, "ArtilleryBallista", true},
		{"bow support on melee skill", "SupportArrowNova", player, "AnimateGuardianSmite", false},
		{"bow support on typeless skill", "SupportArrowNova", player, "PlayerMelee", false},
		{"wand support on melee skill, varunastra", "SupportSacredWisps", varunastra, "AnimateGuardianSmite", false},
		{"bow support on bow skill, varunastra", "SupportArrowNova", varunastra, "ArtilleryBallista", true},
		{"claw support on smite, no varunastra", "SupportNightblade", player, "Smite", false},
		{"claw support on smite, varunastra", "SupportNightblade", varunastra, "Smite", true},
		{"claw support on support-granted windburst, claw equipped", "SupportNightblade", clawEquipped, "TriggeredSupportWindburst", true},
		{"wand support on support-granted windburst, claw equipped", "SupportSacredWisps", clawEquipped, "TriggeredSupportWindburst", false},
		{"claw support on support-granted windburst, unarmed", "SupportNightblade", player, "TriggeredSupportWindburst", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			skill := skillOn(c.actor, c.active)
			if c.active == "TriggeredSupportWindburst" {
				skill.SkillTypes[data.SkillType("SkillGrantedBySupport")] = true
			}
			if got := CalcCanGrantedEffectSupportActiveSkill(ge(c.support), skill, false); got != c.want {
				t.Fatalf("CalcCanGrantedEffectSupportActiveSkill = %v, want %v", got, c.want)
			}
		})
	}
}

func typeGateOpen() map[data.SkillType]bool {
	return map[data.SkillType]bool{
		data.SkillTypeAttack: true, data.SkillTypeDamage: true, data.SkillTypeSpell: true,
		data.SkillTypeLightning: true, data.SkillTypeTriggerable: true, data.SkillTypeProjectile: true,
		data.SkillTypeRangedAttack: true, data.SkillTypeProjectilesFromUser: true,
	}
}
