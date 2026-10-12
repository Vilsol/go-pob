package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	raw2 "github.com/Vilsol/go-pob/data/raw"
)

// TestTriggeredItemGrantedSkillCostsNothing pins SkillsTab.lua:1161-1165.
func TestTriggeredItemGrantedSkillCostsNothing(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "4047-ladder-Anime_8347.cmcws.xml")
	if got := env.Player.Output["LifeCost"]; got != 0 {
		t.Errorf("LifeCost = %v, PoB 0: a triggered item-granted skill must cost nothing", got)
	}
	if !env.Player.MainSkill.ActiveEffect.CostEmptied {
		t.Error("the main skill is the triggered Enfeeble; CostEmptied must be set")
	}

	shared := raw2.GetCalculatedGrantedEffect(poe.GrantedEffectByID("Enfeeble")).GetCalculatedLevels()[1]
	if len(shared.Cost) == 0 {
		t.Error("the SHARED Enfeeble level-1 cost was emptied — the per-effect copy leaked into the global table")
	}
}
