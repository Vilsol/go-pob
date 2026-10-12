package calculator

import (
	"fmt"
	"os"
	"testing"
)

const (
	blockChanceAgreeFloor            = 8622
	effectiveAvgBlockAgreeFloor      = 8601
	configuredDamageChanceAgreeFloor = 8596
)

// TestBlockChanceSourcesAgree pins the two producers of baseBlockChance/BlockChance that this tree had no row for at all.
func TestBlockChanceSourcesAgree(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	stats := []string{"BlockChance", "EffectiveAverageBlockChance", "ConfiguredDamageChance"}
	seen := map[string]int{}
	agree := map[string]int{}
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		for _, s := range stats {
			want, has := rec.Output[s]
			if !has {
				continue
			}
			seen[s]++
			if oracleClose(env.Player.Output[s], want, oracleDumpTolerance) {
				agree[s]++
			}
		}
	}
	for _, s := range stats {
		if seen[s] == 0 {
			t.Fatalf("REFUSING: %s has no graded builds — a floor over an empty set asserts nothing", s)
		}
		fmt.Printf("=== BLOCK CHANCE SOURCES === %-28s %d/%d agree\n", s, agree[s], seen[s])
	}

	if agree["BlockChance"] < blockChanceAgreeFloor {
		t.Errorf("BlockChance agrees on %d of %d, floor %d — CalcDefence.lua:678 / ModParser.lua:2846 "+
			"or :5566 regressed", agree["BlockChance"], seen["BlockChance"], blockChanceAgreeFloor)
	}
	if agree["EffectiveAverageBlockChance"] < effectiveAvgBlockAgreeFloor {
		t.Errorf("EffectiveAverageBlockChance agrees on %d of %d, floor %d",
			agree["EffectiveAverageBlockChance"], seen["EffectiveAverageBlockChance"], effectiveAvgBlockAgreeFloor)
	}
	if agree["ConfiguredDamageChance"] < configuredDamageChanceAgreeFloor {
		t.Errorf("ConfiguredDamageChance agrees on %d of %d, floor %d — CalcDefence.lua:2839",
			agree["ConfiguredDamageChance"], seen["ConfiguredDamageChance"], configuredDamageChanceAgreeFloor)
	}
}
