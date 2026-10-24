package calculator

import (
	"testing"
)

// TestSocketGroupOrderHoistsOnlyCrossLinkedSlots: CalcSetup.lua:998-1009 hoists a socket group to the front of the processing order only when the group's slot is the SOURCE SLOT of a live LinkedSupport mod.
func TestSocketGroupOrderHoistsOnlyCrossLinkedSlots(t *testing.T) {
	requireGameData(t, "3.29")

	t.Run("no linked support keeps file order", func(t *testing.T) {
		env := defenceEnv(t, "0005-occultist-vaal-double-strike.xml")
		wither := indexOfActiveSkill(t, env, "Wither")
		fleshAndStone := indexOfActiveSkill(t, env, "Flesh and Stone")
		if wither > fleshAndStone {
			t.Errorf("Wither (Helmet, group 3) is at index %d and Flesh and Stone (Weapon 2, group 6) at %d — a "+
				"socket group was hoisted on a build carrying no LinkedSupport mod", wither, fleshAndStone)
		}
	})

	t.Run("a cross-linked slot's supports reach the target slot's skill", func(t *testing.T) {
		// PoB's main skill here lists Awakened Lightning Penetration, socketed only in Weapon 2, whose item supports
		// skills from the main hand (Weapon 1). Its linked groups hold no active skill, so the hoisted order itself is not
		// observable on this build; the first subtest pins the order.
		env := defenceEnv(t, "1075-hierophant-ball-lightning-of-orbiting.xml")
		ball := env.Player.ActiveSkillList[indexOfActiveSkill(t, env, "Ball Lightning of Orbiting")]
		const want = "SupportAwakenedLightningPenetration"
		var have []string
		for _, support := range ball.SupportList {
			if raw := support.GrantedEffect.Raw; raw != nil {
				if raw.ID == want {
					return
				}
				have = append(have, raw.ID)
			}
		}
		t.Errorf("Ball Lightning of Orbiting is missing cross-linked support %q from the Weapon 2 item; supports: %v", want, have)
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
