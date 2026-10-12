package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestOathOfSummerScorchCanStack pins ModParser.lua:3027 ("You can inflict an additional Scorch on each Enemy", Warden's Oath of Summer, tree node 33645).
func TestOathOfSummerScorchCanStack(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	e := ParseMod("You can inflict an additional Scorch on each Enemy", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 2 || e.ModList[0].Name() != "ScorchCanStack" ||
		e.ModList[1].Name() != "ScorchStacksMax" || e.ModList[1].Value().Float() != 1 {
		t.Errorf("row: got %+v", e)
	}
	e = ParseMod("Hits that would Ignite instead Scorch", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 2 || e.ModList[0].Name() != "IgniteCanScorch" {
		t.Errorf("NEIGHBOUR CONTROL: got %+v", e)
	}

	build := loadCorpusBuild(t, "0048-warden-frost-blades.xml")
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	if got := env.Player.Output["CurrentScorch"]; got != 30 {
		t.Errorf("CurrentScorch = %v, PoB 30", got)
	}
}
