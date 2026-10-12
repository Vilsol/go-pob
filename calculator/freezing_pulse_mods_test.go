package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestFreezingPulseAddsExactlyThePinsTwoMods pins act_int.lua:7686-7689.
func TestFreezingPulseAddsExactlyThePinsTwoMods(t *testing.T) {
	fn := skillFuncsFor("FreezingPulse")["preDamageFunc"]
	if fn == nil {
		t.Fatal("REFUSING: FreezingPulse has no preDamageFunc registered; an empty mod list would " +
			"then satisfy every count below")
	}

	list := moddb.NewModList()
	skill := &ActiveSkill{
		SkillModList: list,
		SkillCfg:     &moddb.ListCfg{},
		SkillData:    &SkillData{},
	}
	fn(skill, map[string]float64{"ProjectileSpeedMod": 1}, nil)

	byName := map[string]int{}
	for _, m := range list.Mods() {
		byName[m.Name()]++
	}

	if got := byName["Damage"]; got != 1 {
		t.Fatalf("Damage mods = %d, want exactly 1 — :7687 declares one `Damage MORE -50`; a second "+
			"one is `FreezingPulseRamp`, a name the pinned PoB does not contain", got)
	}
	if got := byName["EnemyFreezeChance"]; got != 1 {
		t.Fatalf("EnemyFreezeChance mods = %d, want exactly 1 (:7688)", got)
	}
	if got := len(list.Mods()); got != 2 {
		t.Fatalf("total mods = %d, want 2 — the pin's callback is two NewMod calls and nothing else", got)
	}

	var damage mod.Mod
	for _, m := range list.Mods() {
		if m.Name() == "Damage" {
			damage = m
		}
	}
	if damage.Type() != mod.TypeMore || damage.Value().Float() != -50 {
		t.Fatalf("Damage mod = %v %v, want MORE -50", damage.Type(), damage.Value().Float())
	}
}
