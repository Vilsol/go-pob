package calculator

import (
	"os"
	"testing"
)

// TestMeleeRangeAndStrikeTargets covers CalcOffence.lua:1138-1147.
func TestMeleeRangeAndStrikeTargets(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	keys := []string{"WeaponRangeMetre", "StrikeTargets"}
	seen := map[string]int{}
	agreed := map[string]int{}
	var builds int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		if _, has := rec.Output["StrikeTargets"]; !has {
			continue
		}
		builds++
		env := corpusEnvForLoadedData(t, name)
		for _, key := range keys {
			want, has := rec.Output[key]
			if !has {
				continue
			}
			seen[key]++
			if oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
				agreed[key]++
			}
		}
		if builds >= 120 {
			break
		}
	}

	if builds == 0 {
		t.Fatal("REFUSING: no graded build publishes StrikeTargets; this test no longer measures " +
			"the mechanism it was written for")
	}
	for _, key := range keys {
		t.Logf("%-20s %d of %d", key, agreed[key], seen[key])
	}

	floors := map[string]int{
		"WeaponRangeMetre": 113,
		"StrikeTargets":    120,
	}
	for _, key := range keys {
		if agreed[key] < floors[key] {
			t.Errorf("%s agrees on %d of %d graded builds, below the floor of %d "+
				"(CalcOffence.lua:1138-1147)", key, agreed[key], seen[key], floors[key])
		}
	}
}
