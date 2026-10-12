package calculator

import (
	"os"
	"testing"
)

// TestInherentRageLossOutputs covers CalcPerform.lua:899-900 and ModParser.lua:5472-5474.
func TestInherentRageLossOutputs(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	keys := []string{"InherentRageLoss", "InherentRageLossDelay"}
	seen := map[string]int{}
	agreed := map[string]int{}
	var builds int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		if _, has := rec.Output["InherentRageLoss"]; !has {
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
		t.Fatal("REFUSING: no graded build publishes InherentRageLoss; this test no longer " +
			"measures the mechanism it was written for")
	}
	for _, key := range keys {
		t.Logf("%-24s %d of %d", key, agreed[key], seen[key])
	}

	floors := map[string]int{"InherentRageLoss": 113, "InherentRageLossDelay": 113}
	for _, key := range keys {
		if agreed[key] < floors[key] {
			t.Errorf("%s agrees on %d of %d graded builds, below the floor of %d measured when this "+
				"landed (CalcPerform.lua:899-900)", key, agreed[key], seen[key], floors[key])
		}
	}
}
