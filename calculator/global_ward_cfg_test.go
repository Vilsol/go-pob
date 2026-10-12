package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestGlobalWardIgnoresTheLastSlotCfg: CalcDefence.lua:931-932 total the global Ward with no slot cfg.
func TestGlobalWardIgnoresTheLastSlotCfg(t *testing.T) {
	requireGameData(t, "3.29")
	raw, err := os.ReadFile("../testdata/corpus-3.29/1606-chieftain-treeskill-4.xml")
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	if got := env.Player.Output["Ward"]; got != 35009 {
		t.Errorf("Ward = %v, PoB 35009", got)
	}
}
