package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestTotalDPSReadsDpsMultiplierAfterPostCritFunc pins CalcOffence.lua:3712.
func TestTotalDPSReadsDpsMultiplierAfterPostCritFunc(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		build string
		want  float64
	}{
		{"0479-hierophant-kinetic-fusillade-5.xml", 514887.98029394},
		{"8300-ladder-tappg_2733.TheStinkyDiver.xml", 3005236.4951818},
	}
	for _, c := range cases {
		build := loadCorpusBuild(t, c.build)
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		got := env.Player.OutputTable[OutTableMainHand]["TotalDPS"]
		if math.Abs(got/c.want-1) > 1e-9 {
			t.Errorf("%s: MainHand.TotalDPS = %v, PoB %v (ratio %.4f)", c.build, got, c.want, got/c.want)
		}
	}
}
