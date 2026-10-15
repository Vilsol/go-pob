package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

// TestBloodShrines pins CalcSetup.lua:107-111, CalcPerform.lua:379-396, ConfigOptions.lua:1763-1777 and ModParser.lua:5664 (upstream #10186).
func TestBloodShrines(t *testing.T) {
	requireGameData(t, "3.29")
	entry := ParseMod("Gain a random Blood Shrine buff every 10 seconds", false)
	if strings.TrimSpace(entry.Extra) != "" || len(entry.ModList) != 1 || entry.ModList[0].Name() != "Condition:CanHaveBloodShrines" {
		t.Errorf("blood shrine row: extra %q, %d mods", entry.Extra, len(entry.ModList))
	}

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	corpusDir, _ := corpusNamesForLoadedData(t)
	file, err := os.ReadFile(filepath.Join(corpusDir, "0006-elementalist-storm-burst.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatal(err)
	}
	on := true
	build.Config.Inputs = append(build.Config.Inputs, pob.Input{Name: "buffBloodShrineOfCrows", Boolean: &on}, pob.Input{Name: "buffBloodShrineOfBats", Boolean: &on})
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	for name, want := range map[string]float64{"ChaosResist": 50, "PhysicalDamageFromHitsTakenAsChaos": 10, "EnemyCurseLimit": 1} {
		got, ok := 0.0, false
		for _, m := range env.ModDB.Mods[name] {
			if s := string(m.GetSource()); s == "Blood Shrine of Crows" || s == "Blood Shrine of Bats" {
				got, ok = ourModValue(m), true
			}
		}
		if !ok || got < want {
			t.Errorf("%s from the blood shrine = %v (%v), want at least %v", name, got, ok, want)
		}
	}
	if len(env.ModDB.Mods["ExtraCurse"]) == 0 {
		t.Error("no ExtraCurse from Blood Shrine of Bats")
	}
}
