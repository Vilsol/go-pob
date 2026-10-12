package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob-data/raw"
)

// TestTriggerSupportRefusesNonPlayerActor pins CalcTools.lua:113.
func TestTriggerSupportRefusesNonPlayerActor(t *testing.T) {
	player := &Actor{}
	enemy := &Actor{}
	player.Enemy = enemy
	enemy.Enemy = player
	player.Player = player
	enemy.Player = player

	minion := &Actor{Enemy: enemy}

	trigger := &GrantedEffect{Raw: &poe.GrantedEffect{
		GrantedEffect: raw.GrantedEffect{ID: "SupportCastOnCritTriggered", IsSupport: true},
	}}
	if !grantedEffectIsTrigger(trigger) {
		t.Fatal("fixture is inert: SupportCastOnCritTriggered is not being read as a trigger")
	}

	plain := &GrantedEffect{Raw: &poe.GrantedEffect{
		GrantedEffect: raw.GrantedEffect{ID: "SupportAddedFireDamage", IsSupport: true},
	}}
	if grantedEffectIsTrigger(plain) {
		t.Fatal("fixture is inert: SupportAddedFireDamage is being read as a trigger")
	}

	for _, c := range []struct {
		name  string
		actor *Actor
		ge    *GrantedEffect
		want  bool
	}{
		{"trigger on player", player, trigger, true},
		{"trigger on minion", minion, trigger, false},
		{"plain support on minion", minion, plain, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			as := &ActiveSkill{
				Actor:      c.actor,
				SkillTypes: typeGateOpen(),
				ActiveEffect: &GemEffect{GrantedEffect: &GrantedEffect{
					Raw: &poe.GrantedEffect{GrantedEffect: raw.GrantedEffect{ID: "PlayerMelee"}},
				}},
			}
			if got := CalcCanGrantedEffectSupportActiveSkill(c.ge, as); got != c.want {
				t.Fatalf("CalcCanGrantedEffectSupportActiveSkill = %v, want %v", got, c.want)
			}
		})
	}
}
