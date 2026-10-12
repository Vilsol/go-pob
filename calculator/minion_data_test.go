package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestMinionDataIsResolvable asserts the two halves of the minion tables agree with each other and with the game data.
func TestMinionDataIsResolvable(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	for skillID, minionList := range raw.SkillMinionLists {
		for _, minionType := range minionList {
			if raw.Minions[minionType] == nil {
				t.Errorf("SkillMinionLists[%q] names minion type %q, which is not in Minions", skillID, minionType)
			}
		}
	}

	for skillID, minionList := range raw.SupportAddMinionLists {
		for _, minionType := range minionList {
			if raw.Minions[minionType] == nil {
				t.Errorf("SupportAddMinionLists[%q] names minion type %q, which is not in Minions", skillID, minionType)
			}
		}
	}

	for minionType, minionData := range raw.Minions {
		if len(minionData.SkillList) == 0 {
			t.Errorf("minion %q has an empty skillList", minionType)
		}
		spectre := raw.Spectres[minionType] != nil
		for _, skillID := range minionData.SkillList {
			if poe.GrantedEffectByID(skillID) != nil {
				continue
			}
			if spectre {
				t.Logf("spectre %q lists skill %q, which go-pob-data does not have "+
					"(Data/Skills/spectre.lua is not shipped) — it is dropped from the minion's skill list",
					minionType, skillID)
				continue
			}
			t.Errorf("minion %q lists skill %q, which go-pob-data does not have", minionType, skillID)
		}
	}
}
