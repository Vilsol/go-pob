package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestGeneralsCryCooldownFloorsAtAttackTime pins CalcMirages.lua:391-394.
func TestGeneralsCryCooldownFloorsAtAttackTime(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		build   string
		wantDPS float64
	}{
		{"0324-champion-autoexertion-2.xml", 0.96},
		{"0188-champion-generals-cry.xml", 0.58},
		{"1303-slayer-spectral-throw.xml", 0.43},
		{"11002-ladder-Millen_1846.MillenT.xml", 0.45},
		{"10376-ladder-Inxn_3755.inxm.xml", 0.69},
	}
	for _, c := range cases {
		build := loadCorpusBuild(t, c.build)
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		sd := env.Player.MainSkill.SkillData
		if !sd.TriggeredByGeneralsCry {
			t.Fatalf("%s: CONTROL IS VACUOUS — main skill is not triggeredByGeneralsCry", c.build)
		}
		if math.Abs(sd.DpsMultiplier-c.wantDPS) > 1e-9 {
			t.Errorf("%s: dpsMultiplier = %v, PoB %v", c.build, sd.DpsMultiplier, c.wantDPS)
		}
	}
}
