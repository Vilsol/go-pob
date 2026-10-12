package calculator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestEnergyShieldLeechOnFullReachesTheCondition pins ModParser.lua:3000 and :3013 together with their consumer at CalcPerform.lua:877-880.
func TestEnergyShieldLeechOnFullReachesTheCondition(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	corpusDir, _ := corpusNamesForLoadedData(t)

	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	const itemPhrase = "not removed at Full Energy Shield"

	soulDrinker := regexp.MustCompile(`(^|[^0-9])48999([^0-9]|$)`)

	carriers, fromItemText, conditionSet := 0, 0, 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		hasItemPhrase := strings.Contains(string(raw), itemPhrase)
		if !hasItemPhrase && !soulDrinker.Match(raw) {
			continue
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.ModDB == nil {
			continue
		}
		if !env.ModDB.Flag(nil, "CanLeechEnergyShieldOnFullEnergyShield") {
			continue
		}
		carriers++
		if hasItemPhrase {
			fromItemText++
		}
		if env.ModDB.Conditions["Leeching"] && env.ModDB.Conditions["LeechingEnergyShield"] {
			conditionSet++
		}
	}

	if carriers < 130 {
		t.Fatalf("REFUSING: only %d builds carry CanLeechEnergyShieldOnFullEnergyShield; the row or the population has moved", carriers)
	}
	if conditionSet != carriers {
		t.Errorf("CalcPerform.lua:877-880 did not fire on every carrier: %d of %d set Leeching+LeechingEnergyShield", conditionSet, carriers)
	}
	if fromItemText < 100 {
		t.Errorf("ModParser.lua:3000's row REGRESSED: only %d of %d carriers carry its item phrase", fromItemText, carriers)
	}
	if carriers-fromItemText < 20 {
		t.Errorf("the NON-item-text carriers REGRESSED to %d: ModParser.lua:3013's tree-sourced row is the only thing that produces them", carriers-fromItemText)
	}
}
