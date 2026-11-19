package calculator

import "testing"

// TestSacredWispsSetsWispFlag pins CalcMirages.lua:331: the main skill carries the wisp flag that shows Total Wisp DPS.
func TestSacredWispsSetsWispFlag(t *testing.T) {
	env := corpusOnlyEnv(t, "0195-elementalist-kinetic-rain-of-impact")
	// PoB's dump sets it.
	if !env.Player.MainSkill.SkillFlags[SkillFlag("wisp")] {
		t.Errorf("wisp flag not set")
	}
}
