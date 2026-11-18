package calculator

import "testing"

// TestPvpScalingSetsTheSkillFlags: with PvpScaling on, CalcOffence.lua:3912-3917 copies the attack flags into their
// PvP twins. The wants are the pinned PoB's dumped skillFlags for a corpus build with PvpScaling set.
//
//pob:block:src/Modules/CalcOffence.lua:3904-3917
func TestPvpScalingSetsTheSkillFlags(t *testing.T) {
	env := corpusEnvForLoadedData(t, "15349-juggernaut-static-strike-of-gathering-lightning")
	if !env.Build.GetBooleanOption("PvpScaling") {
		t.Fatal("premise broken: the build does not set PvpScaling")
	}
	flags := env.Player.MainSkill.SkillFlags
	for flag, want := range map[SkillFlag]bool{
		SkillFlagIsPvP:            true,
		SkillFlagAttackPvP:        true,
		SkillFlagNotAttackPvP:     false,
		SkillFlagWeapon1AttackPvP: true,
		SkillFlagWeapon2AttackPvP: true,
		SkillFlagNotAveragePvP:    true,
	} {
		if got := flags[flag]; got != want {
			t.Errorf("%s = %v, PoB %v", flag, got, want)
		}
	}
}
