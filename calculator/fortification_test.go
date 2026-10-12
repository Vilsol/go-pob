package calculator

import (
	"os"
	"testing"
)

// TestFortificationOutputs covers CalcPerform.lua:627-655.
func TestFortificationOutputs(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	keys := []string{
		"MaximumFortification", "MinimumFortification", "RemovableFortification",
		"FortificationStacks", "FortificationStacksOver20", "FortifyDuration", "FortificationEffect",
	}
	seen := map[string]int{}
	agreed := map[string]int{}
	var builds int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		if _, has := rec.Output["MaximumFortification"]; !has {
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
		t.Fatal("REFUSING: no graded build publishes MaximumFortification; this test no longer " +
			"measures the mechanism it was written for")
	}
	for _, key := range keys {
		t.Logf("%-26s %d of %d", key, agreed[key], seen[key])
	}

	floors := map[string]int{
		"MaximumFortification":      120,
		"MinimumFortification":      120,
		"RemovableFortification":    120,
		"FortificationStacks":       120,
		"FortificationStacksOver20": 120,
		"FortifyDuration":           120,
		"FortificationEffect":       120,
	}
	for _, key := range keys {
		if agreed[key] < floors[key] {
			t.Errorf("%s agrees on %d of %d graded builds, below the floor of %d "+
				"(CalcPerform.lua:640-648)", key, agreed[key], seen[key], floors[key])
		}
	}
}
