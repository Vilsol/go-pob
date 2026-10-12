package calculator

import (
	"os"
	"testing"
)

// TestNetLifeRecoupOverTime covers CalcDefence.lua:3051-3054.
func TestNetLifeRecoupOverTime(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	keys := []string{"netLifeRecoupAndLossLostOverTimeMax", "netLifeRecoupAndLossLostOverTimeAvg"}
	seen := map[string]int{}
	agreed := map[string]int{}
	var builds, poolDependent int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		if _, has := rec.Output["netLifeRecoupAndLossLostOverTimeMax"]; !has {
			continue
		}
		builds++
		if _, hasLoss := rec.Output["LifeLossLostMax"]; hasLoss {
			poolDependent++
		}
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
		if builds >= 200 {
			break
		}
	}

	if builds == 0 {
		t.Fatal("REFUSING: no graded build publishes netLifeRecoupAndLossLostOverTimeMax; this " +
			"test no longer measures the mechanism it was written for")
	}
	for _, key := range keys {
		t.Logf("%-38s %d of %d", key, agreed[key], seen[key])
	}
	t.Logf("of %d sampled builds, %d depend on the unported pool simulation", builds, poolDependent)

	floors := map[string]int{
		"netLifeRecoupAndLossLostOverTimeMax": 90,
		"netLifeRecoupAndLossLostOverTimeAvg": 86,
	}
	for _, key := range keys {
		if agreed[key] < floors[key] {
			t.Errorf("%s agrees on %d of %d graded builds, below the floor of %d "+
				"(CalcDefence.lua:3051-3054)", key, agreed[key], seen[key], floors[key])
		}
	}
}
