package raw

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/storage"
)

// TestStatRowsComeFromStatSet1: skills.lua reads an effect's stat rows by its StatSet1 column, not by the rows that name the effect.
//
//pob:block:src/Export/Scripts/skills.lua:256
func TestStatRowsComeFromStatSet1(t *testing.T) {
	if err := poe.InitializeAll(DataContext(t.Context()), LatestVersion, AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, ge := range poe.GrantedEffects {
		calc := &CalculatedGrantedEffect{GrantedEffect: ge}
		if set := calc.GetStatSet(); set == nil || set.Key != ge.GrantedEffectStatSets {
			t.Errorf("%s: stat set %v, StatSet1 is %d", ge.ID, set, ge.GrantedEffectStatSets)
		}
		_, rows := rowListsForEffect(ge)
		for _, row := range rows {
			if row.StatSet != ge.GrantedEffectStatSets {
				t.Errorf("%s: stat row of set %d, StatSet1 is %d", ge.ID, row.StatSet, ge.GrantedEffectStatSets)
			}
		}
		checked++
	}
	ge := poe.GrantedEffectByID("AncestralMeleeKnockbackStun")
	if ge == nil || poe.GrantedEffectStatSets[ge.GrantedEffectStatSets].ID != "AncestralMeleeStun" {
		t.Fatalf("REFUSING: the control effect no longer differs (%v)", ge)
	}
	if checked < 1000 {
		t.Fatalf("only %d effects", checked)
	}
}
