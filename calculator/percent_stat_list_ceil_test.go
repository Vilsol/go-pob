package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestPercentStatCeilsATableValue: ModStore.lua:518-530 ceil a PercentStat-scaled table value.
func TestPercentStatCeilsATableValue(t *testing.T) {
	requireGameData(t, "3.29")
	raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), "0543-luminary-mana-infused-staff.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	for stat, want := range map[string]float64{"LightningMinBase": 1218, "LightningMaxBase": 1695} {
		if got := env.Player.Output[stat]; got != want {
			t.Errorf("%s = %v, PoB %v", stat, got, want)
		}
	}
}

// TestPercentStatDividesThePercentFirst: ModStore.lua:505 divides the percent first.
func TestPercentStatDividesThePercentFirst(t *testing.T) {
	requireGameData(t, "3.29")
	raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), "6603-ladder-XiledOne_4259.NizNecroKBoC.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	for stat, want := range map[string]float64{"PhysicalMinBase": 717, "PhysicalMaxBase": 742} {
		if got := env.Player.OutputTable[OutTableMainHand][stat]; got != want {
			t.Errorf("MainHand.%s = %v, PoB %v", stat, got, want)
		}
	}
}
