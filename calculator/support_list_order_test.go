package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestCrossLinkedSupportsFollowOwnGems pins CalcSetup.lua:1688-1731: a group's own supports come first and the cross-linked slot's are appended after them.
func TestCrossLinkedSupportsFollowOwnGems(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	build := loadCorpusBuild(t, "7129-ladder-dwh_4650.BREACH_DELIRIUM.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
	skill := env.Player.MainSkill
	if skill == nil || skill.ActiveEffect.GrantedEffect.ID() != "LightningStrikeAltX" {
		t.Fatal("CONTROL: the main skill is not Lightning Strike of Arcing")
	}

	last := skill.EffectList[len(skill.EffectList)-1].GrantedEffect.ID()
	if last != "SupportEnduranceChargeOnMeleeStun" {
		t.Errorf("last support = %s, want the cross-linked SupportEnduranceChargeOnMeleeStun", last)
	}
	if got := skill.SkillModList.More(skill.SkillCfg, "SupportManaMultiplier"); got != 7.8623 {
		t.Errorf("SupportManaMultiplier MORE = %.17g, want 7.8623", got)
	}
}
