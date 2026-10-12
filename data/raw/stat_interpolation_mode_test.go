package raw

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/storage"
)

// TestStatInterpolationModeSurvivesLevelBuild pins that a granted effect's levels keep the interpolation MODE the .dat gives them.
func TestStatInterpolationModeSurvivesLevelBuild(t *testing.T) {
	if err := poe.InitializeAll(DataContext(t.Context()), LatestVersion, AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	modes := map[int]int{}
	for _, effect := range poe.GrantedEffects {
		for _, level := range GetCalculatedGrantedEffect(effect).GetCalculatedLevels() {
			for _, m := range level.StatInterpolation {
				modes[m]++
			}
		}
	}

	if modes[2] == 0 {
		t.Fatal("REFUSING: no level carries interpolation mode 2, so the probe cannot tell a " +
			"working fix from a broken query; the mode-2 population has emptied")
	}
	if modes[3] == 0 {
		t.Errorf("interpolation mode 3 occurs 0 times across all built levels (modes seen: %v); "+
			"the effectiveness-interpolation arm of CalcTools is dead code", modes)
	}
}
