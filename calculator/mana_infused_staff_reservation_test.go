package calculator

import "testing"

// TestManaInfusedStaffNeedsMana: with no Mana, CalcPerform.lua:2013-2016 disables a requiresManaReservation skill
// and Blood Magic's reserve-life rule skips it (:2032). The wants are the pinned PoB's dump with "Removes all Mana"
// and "Skills Reserve Life instead of Mana" added: the skill is disabled and reserves nothing.
//
//pob:block:src/Modules/CalcPerform.lua:2013-2016,src/Modules/CalcPerform.lua:2032
func TestManaInfusedStaffNeedsMana(t *testing.T) {
	old := "94% increased Mana Regeneration Rate\nItems and Gems have 25% reduced Attribute Requirements\n"
	env := corpusEnvWithEdit(t, "0103-elementalist-mana-infused-staff", old,
		old+"Removes all Mana\nSkills Reserve Life instead of Mana\n")
	ms := env.Player.MainSkill
	if ms.ActiveEffect.GrantedEffect.Name() != "Mana-Infused Staff" {
		t.Fatal("premise broken: the main skill is not Mana-Infused Staff")
	}
	if !ms.SkillData.RequiresManaReservation {
		t.Error("skillData.requiresManaReservation is unset, PoB's dump carries true")
	}
	if !ms.SkillFlags[SkillFlagDisable] {
		t.Error("the skill is enabled, PoB disables it")
	}
	if got := env.Player.Output["LifeReserved"]; got != 0 {
		t.Errorf("LifeReserved = %v, PoB 0", got)
	}
	if got := env.Player.Output["LifeUnreserved"]; got != 2974 {
		t.Errorf("LifeUnreserved = %v, PoB 2974", got)
	}
}
