package calculator

import (
	"testing"
)

// TestCalcsSkillControlsAreTheTabsOwn: The Calcs tab's own selection.
func TestCalcsSkillControlsAreTheTabsOwn(t *testing.T) {
	build := parseCorpusBuild(t, "0030-saboteur-cyclone-2.xml")
	set := build.ActiveSkillSetIndex()
	build.SetCalcsInputNumber("skill_number", 2)
	build.SetMainActiveSkillCalcs(set, 1, 2)
	build.SetGemSkillMinionItemSetCalcs(set, 1, 1, 2)
	build.Skills.SkillSets[set].Skills[1].Gems[1].SkillMinionItemSet = 1

	env := (&Calculator{PoB: build}).BuildOutput(OutputModeCalcs)
	c := env.CalcsSkillControls()
	if c == nil || c.SocketGroup != 1 || c.MainActiveSkill != 2 || c.GemIndex != 1 {
		t.Fatalf("controls %+v", c)
	}
	if !c.MinionShown || c.Minion != 2 || c.Minions[1].Label != "AG" {
		t.Fatalf("minion select %d of %+v", c.Minion, c.Minions)
	}
	minion := env.Player.MainSkill.Minion
	if minion == nil || minion.ItemSet == nil || minion.ItemSet.ID != "2" {
		t.Fatalf("the CALCS pass's minion item set: %+v", minion)
	}

	main := (&Calculator{PoB: build}).BuildOutput(OutputModeMain).MainSkillControls()
	if main.SocketGroup != build.Build.MainSocketGroup-1 {
		t.Fatalf("the sidebar moved: group %d", main.SocketGroup)
	}
}
