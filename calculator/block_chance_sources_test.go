package calculator

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

const (
	blockChanceAgreeFloor            = 8622
	effectiveAvgBlockAgreeFloor      = 8601
	configuredDamageChanceAgreeFloor = 8596
)

var blockChanceSourcesStats = []string{"BlockChance", "EffectiveAverageBlockChance", "ConfiguredDamageChance"}

var _ = registerCorpusCheck("TestBlockChanceSourcesAgree", func() corpusCheck {
	return &blockChanceSourcesCheck{seen: map[string]int{}, agree: map[string]int{}}
})

type blockChanceSourcesCheck struct {
	mu          sync.Mutex
	seen, agree map[string]int
}

func (c *blockChanceSourcesCheck) visit(_ string, rec oracleRecord, graded bool, env *Environment) {
	if !graded {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range blockChanceSourcesStats {
		want, has := rec.Output[s]
		if !has {
			continue
		}
		c.seen[s]++
		if oracleClose(env.Player.Output[s], want, oracleDumpTolerance) {
			c.agree[s]++
		}
	}
}

// TestBlockChanceSourcesAgree pins the two producers of baseBlockChance/BlockChance that this tree had no row for at all.
func TestBlockChanceSourcesAgree(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*blockChanceSourcesCheck](t)
	stats, seen, agree := blockChanceSourcesStats, c.seen, c.agree
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
