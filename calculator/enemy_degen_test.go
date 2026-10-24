package calculator

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

var enemyDegenOwned = []string{
	"TotalDegen",
	"PhysicalEnemyDegen", "FireEnemyDegen", "ColdEnemyDegen", "LightningEnemyDegen", "ChaosEnemyDegen",
	"ComprehensiveNetLifeRegen", "ComprehensiveNetManaRegen",
	"ComprehensiveNetEnergyShieldRegen", "ComprehensiveTotalNetRegen",
}

var _ = registerCorpusCheck("TestEnemyDegenMatchesOracle", func() corpusCheck { return &enemyDegenCheck{} })

type enemyDegenCheck struct {
	mu                          sync.Mutex
	seen, agree, missing, extra int
	worst                       corpusFirst
}

func (c *enemyDegenCheck) visit(name string, rec oracleRecord, graded bool, env *Environment) {
	if !graded {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, key := range enemyDegenOwned {
		want, pobHas := rec.Output[key]
		got, ourHas := env.Player.Output[key]
		switch {
		case pobHas && !ourHas:
			c.seen++
			c.missing++
			c.worst.offer(name, fmt.Sprintf("%s: %s absent from ours, PoB %.4f", name, key, want))
		case pobHas && ourHas:
			c.seen++
			if oracleClose(got, want, oracleDumpTolerance) {
				c.agree++
			} else {
				c.worst.offer(name, fmt.Sprintf("%s: %s ours %.4f, PoB %.4f", name, key, got, want))
			}
		case !pobHas && ourHas:
			c.extra++
			c.worst.offer(name, fmt.Sprintf("%s: %s ours %.4f, PoB does not emit it", name, key, got))
		}
	}
}

// TestEnemyDegenMatchesOracle pins CalcDefence.lua:3487-3742 — the ailment/enemy-damage degen block and the Comprehensive* net-regen keys that hang off it.
func TestEnemyDegenMatchesOracle(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*enemyDegenCheck](t)
	seen, agree, missing, extra, worst := c.seen, c.agree, c.missing, c.extra, c.worst.msg

	if seen == 0 {
		t.Fatalf("REFUSING: no build in the corpus carries any key of this block — the population " +
			"is gone, so this test is vacuous rather than passing")
	}

	fmt.Printf("=== ENEMY DEGEN === %d/%d graded pairs agree · %d absent from ours · %d emitted where PoB does not\n",
		agree, seen, missing, extra)
	if worst != "" {
		fmt.Printf("    first divergence: %s\n", worst)
	}

	if extra != 0 {
		t.Errorf("emitted %d key(s) PoB does not emit — CalcDefence.lua:3633-3634 nils TotalDegen "+
			"and skips the Comprehensive* block when only build degen contributed; first: %s",
			extra, worst)
	}

	const enemyDegenAgreeFloor = 124
	if agree < enemyDegenAgreeFloor {
		t.Errorf("only %d of %d pairs agree, floor is %d; first divergence: %s",
			agree, seen, enemyDegenAgreeFloor, worst)
	}
}
