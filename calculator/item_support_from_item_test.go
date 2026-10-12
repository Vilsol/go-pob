package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestItemSupportFlagReachesSocketedGem pins CalcSetup.lua:1525-1536 with CalcTools.lua:94.
func TestItemSupportFlagReachesSocketedGem(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	build := loadCorpusBuild(t, "1767-ladder-Eramyr_0756.chebat.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
	if !env.itemSupports["SupportGenerosity"] {
		t.Fatal("CONTROL: no item ExtraSupport flagged SupportGenerosity")
	}

	envy := 0
	for _, activeSkill := range env.Player.ActiveSkillList {
		if activeSkill.ActiveEffect.GrantedEffect.ID() != "Envy" {
			continue
		}
		envy++
		for _, effect := range activeSkill.EffectList[1:] {
			if effect.GrantedEffect.ID() == "SupportGenerosity" {
				t.Errorf("Envy is supported by Generosity (fromItem %v)", effect.GrantedEffect.FromItem())
			}
		}
	}
	if envy != 1 {
		t.Fatalf("CONTROL: %d Envy active skills, want 1", envy)
	}
	if got := env.Player.Output["ChaosMinBase"]; got != 301 {
		t.Errorf("ChaosMinBase = %v, want 301", got)
	}
}
