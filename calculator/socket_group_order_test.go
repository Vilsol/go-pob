package calculator

import (
	"testing"
)

// TestSocketGroupOrderHoistsOnlyCrossLinkedSlots: CalcSetup.lua:998-1009 hoists a socket group to the front of the processing order only when the group's slot is the SOURCE SLOT of a live LinkedSupport mod.
func TestSocketGroupOrderHoistsOnlyCrossLinkedSlots(t *testing.T) {
	t.Run("no linked support keeps file order", func(t *testing.T) {
		env := defence320Env(t, "0277-juggernaut-righteous-fire.xml")
		skitterbots := indexOfActiveSkill(t, env, "Summon Skitterbots")
		frostblink := indexOfActiveSkill(t, env, "Frostblink")
		if skitterbots > frostblink {
			t.Errorf("Summon Skitterbots (Weapon 1, group 5) is at index %d and Frostblink "+
				"(Weapon 2, group 9) at %d — a socket group was hoisted on a build carrying no "+
				"LinkedSupport mod", skitterbots, frostblink)
		}
	})

	t.Run("linked support still hoists its source slot", func(t *testing.T) {
		env := defence320Env(t, "4783-elementalist-ethereal-knives-118.xml")
		knives := env.Player.ActiveSkillList[indexOfActiveSkill(t, env, "Ethereal Knives")]
		const want = "SupportSwiftAfflictionPlus"
		for _, support := range knives.SupportList {
			if raw := support.GrantedEffect.Raw; raw != nil && raw.ID == want {
				return
			}
		}
		t.Errorf("Ethereal Knives is missing cross-linked support %q from Uul-Netol's Vow — the "+
			"Amulet group (file index 1) was not hoisted ahead of Body Armour (file index 0)", want)
	})
}

func indexOfActiveSkill(t *testing.T, env *Environment, displayedName string) int {
	t.Helper()
	for i, activeSkill := range env.Player.ActiveSkillList {
		if as := activeSkill.ActiveEffect.GrantedEffect.Raw.GetActiveSkill(); as != nil && as.DisplayedName == displayedName {
			return i
		}
	}
	t.Fatalf("REFUSING: %q is not in the active skill list", displayedName)
	return -1
}
