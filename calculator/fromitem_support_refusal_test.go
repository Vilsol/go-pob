package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob-data/raw"
)

// TestFromItemSupportRefusal pins CalcTools.lua:93-96.
func TestFromItemSupportRefusal(t *testing.T) {
	ge := func(id string) *GrantedEffect {
		return &GrantedEffect{Raw: &poe.GrantedEffect{GrantedEffect: raw.GrantedEffect{ID: id}}}
	}

	const (
		itemSupport = "SupportUniqueMjolnerLightningSpellsCastOnHit"
		gemSupport  = "SupportAddedLightningDamage"
		itemActive  = "ApproachingFlames"
		gemActive   = "Fireball"
	)

	for _, c := range []struct {
		id       string
		fromItem bool
		support  bool
	}{
		{itemSupport, true, true},
		{gemSupport, false, true},
		{itemActive, true, false},
		{gemActive, false, false},
	} {
		if got := ge(c.id).FromItem(); got != c.fromItem {
			t.Fatalf("fixture is inert: %s FromItem = %v, want %v", c.id, got, c.fromItem)
		}
		if got := ge(c.id).IsSupport(); got != c.support {
			t.Fatalf("fixture is inert: %s IsSupport = %v, want %v", c.id, got, c.support)
		}
	}

	player := &Actor{}
	enemy := &Actor{}
	player.Enemy, enemy.Enemy, player.Player, enemy.Player = enemy, player, player, player

	skill := func(activeID string) *ActiveSkill {
		return &ActiveSkill{
			Actor:        player,
			SkillTypes:   typeGateOpen(),
			ActiveEffect: &GemEffect{GrantedEffect: ge(activeID)},
		}
	}

	for _, c := range []struct {
		name    string
		support string
		active  string
		want    bool
	}{
		{"item support on item skill", itemSupport, itemActive, false},
		{"item support on gem skill", itemSupport, gemActive, true},
		{"gem support on item skill", gemSupport, itemActive, true},
		{"gem support on gem skill", gemSupport, gemActive, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := CalcCanGrantedEffectSupportActiveSkill(ge(c.support), skill(c.active)); got != c.want {
				t.Fatalf("CalcCanGrantedEffectSupportActiveSkill = %v, want %v", got, c.want)
			}
		})
	}
}
