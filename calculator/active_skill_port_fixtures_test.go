package calculator

import "testing"

// TestDualWieldRequiresDifferentTypes pins CalcActiveSkill.lua:346-349.
func TestDualWieldRequiresDifferentTypes(t *testing.T) {
	same := portFixtureEnv(t, "dual-wield-same-type", OutputModeMain)
	if ms := same.Player.MainSkill; ms == nil || !ms.SkillFlags[SkillFlagDisable] || ms.DisableReason != "Weapon Types Need to be Different" {
		t.Errorf("dual-wield-same-type: Cleave of Rage with two swords is not disabled for differing weapon types")
	}
	wantAbsent(t, "dual-wield-same-type", same.Player.Output, "TotalDPS")

	diff := portFixtureEnv(t, "dual-wield-different-type", OutputModeMain)
	wantStat(t, "dual-wield-different-type", diff.Player.Output, "TotalDPS", 3.8871048393443)
}

// TestForceSourceWeaponUsesTheGrantingHand pins CalcActiveSkill.lua:308-313 and 321.
func TestForceSourceWeaponUsesTheGrantingHand(t *testing.T) {
	const fixture = "ghostly-artillery-off-hand"
	env := portFixtureEnv(t, fixture, OutputModeMain)
	ms := env.Player.MainSkill
	if ms == nil || ms.ActiveEffect.GrantedEffect.Raw.ID != "GhostCannons" {
		t.Fatalf("REFUSING: %s's main skill is not Ghostly Artillery", fixture)
	}
	if ms.SkillFlags[SkillFlagWeapon1Attack] || !ms.SkillFlags[SkillFlagWeapon2Attack] {
		t.Errorf("%s: weapon1Attack=%v weapon2Attack=%v, PoB attacks with the granting off hand only",
			fixture, ms.SkillFlags[SkillFlagWeapon1Attack], ms.SkillFlags[SkillFlagWeapon2Attack])
	}
	wantStat(t, fixture, env.Player.Output, "TotalDPS", 1.543707)
	wantStat(t, fixture, env.Player.Output, "Speed", 1.705)
}
