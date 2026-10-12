package calculator

import (
	"os"
	"testing"
)

// TestBleedStacksAndRollAverage covers CalcOffence.lua:4308-4340, :4363-4372 and :4389-4410.
func TestBleedStacksAndRollAverage(t *testing.T) {
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
		"BleedRollAverage", "BleedStackPotential", "BleedStacks", "BleedStacksMax",
		"BleedDamage", "BleedDPS", "BleedDuration",
	}
	seen := map[string]int{}
	agreed := map[string]int{}
	var builds int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		if _, has := rec.Output["BleedRollAverage"]; !has {
			continue
		}
		builds++
		env := corpusEnvForLoadedData(t, name)
		for _, key := range keys {
			want, has := rec.Output[key]
			if !has || want == 0 {
				continue
			}
			seen[key]++
			if oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
				agreed[key]++
			}
		}
		if builds >= 150 {
			break
		}
	}

	if builds == 0 {
		t.Fatal("REFUSING: no graded build publishes BleedRollAverage; this test no longer " +
			"measures the mechanism it was written for")
	}
	for _, key := range keys {
		t.Logf("%-22s %d of %d", key, agreed[key], seen[key])
	}

	floors := map[string]int{
		"BleedRollAverage":    86,
		"BleedStackPotential": 65,
		"BleedStacks":         65,
		"BleedStacksMax":      138,
		"BleedDamage":         48,
		"BleedDPS":            48,
		"BleedDuration":       132,
	}
	for _, key := range keys {
		if agreed[key] < floors[key] {
			t.Errorf("%s agrees on %d of %d graded builds, below the floor of %d "+
				"(CalcOffence.lua:4308-4410)", key, agreed[key], seen[key], floors[key])
		}
	}
}
