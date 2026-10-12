package calculator

import (
	"os"
	"testing"
)

// TestItemGrantedNoSupportsSkillTakesNoSupports pins CalcSetup.lua:1396-1397: "Curse Enemies with Vulnerability on Hit" grants its skill with noSupports.
func TestItemGrantedNoSupportsSkillTakesNoSupports(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	build := loadCorpusBuild(t, "4956-ladder-Icebloom008_0770.СаблимНаНищету.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)

	found := false
	for _, skill := range env.Player.ActiveSkillList {
		if skill.ActiveEffect.GrantedEffect.Name() != "Vulnerability" {
			continue
		}
		found = true
		if len(skill.EffectList) != 1 {
			t.Errorf("item-granted Vulnerability carries %d effects, want 1 (no supports)", len(skill.EffectList))
		}
		if skill.SkillData.TriggeredWhileChannelling {
			t.Error("item-granted Vulnerability must not be supported by the weapon's Cast while Channelling")
		}
	}
	if !found {
		t.Fatal("REFUSING: no Vulnerability active skill, so this fixture proves nothing")
	}
}
