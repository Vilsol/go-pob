package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestConvertWeaponModsSkipsSpellPoisonBleed: Item.lua:2289's Poison/Bleed arm carries a `mod.flags ~= ModFlag.Spell` conjunct that this port dropped.
//
//	elseif (mod.name == "PoisonChance" or mod.name == "BleedChance") and mod.flags ~= ModFlag.Spell
//	       and (not mod[1] or (mod[1].type == "Condition" and mod[1].var == "CriticalStrike" and not mod[2])) then
func TestConvertWeaponModsSkipsSpellPoisonBleed(t *testing.T) {
	hasHandCondition := func(m mod.Mod) bool {
		for _, raw := range m.Tags() {
			tag, ok := raw.(*mod.ConditionTag)
			if !ok {
				continue
			}
			for _, v := range tag.VarList {
				if v == "MainHandAttack" || v == "OffHandAttack" {
					return true
				}
			}
		}
		return false
	}

	for _, name := range []string{"PoisonChance", "BleedChance"} {
		t.Run(name+" spell-flagged is left alone", func(t *testing.T) {
			list := []mod.Mod{mod.NewFloat(name, mod.TypeBase, 10).Flag(mod.MFlagSpell)}
			convertWeaponModsToLocal(list, 1)
			testza.AssertFalse(t, hasHandCondition(list[0]))
		})

		t.Run(name+" unflagged still converts", func(t *testing.T) {
			list := []mod.Mod{mod.NewFloat(name, mod.TypeBase, 10)}
			convertWeaponModsToLocal(list, 1)
			testza.AssertTrue(t, hasHandCondition(list[0]))
		})

		t.Run(name+" attack-flagged still converts", func(t *testing.T) {
			list := []mod.Mod{mod.NewFloat(name, mod.TypeBase, 10).Flag(mod.MFlagAttack)}
			convertWeaponModsToLocal(list, 1)
			testza.AssertTrue(t, hasHandCondition(list[0]))
		})
	}
}
