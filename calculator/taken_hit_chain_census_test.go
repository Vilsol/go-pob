package calculator

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
)

var takenHitExonerated = []string{
	"ArmourDefense", "DamageReductionMax", "PhysicalResist", "PhysicalEnemyPen",
	"PhysicalEnemyOverwhelm",
}

const takenHitArisesHereCeiling = 150

func takenHitChainRungs() []string {
	rungs := []string{
		"PhysicalTakenDamage", "PhysicalEnemyDamage", "totalEnemyDamage", "PhysicalDamageReduction",
		"Armour", "Evasion", "ConfiguredEvadeChance", "EnemyCritChance", "EnemyCritEffect",
		"PhysicalEnemyDamageMult", "PhysicalTakenHitMult",
	}
	seenRung := map[string]bool{}
	for _, k := range rungs {
		seenRung[k] = true
	}
	for _, k := range takenHitExonerated {
		if !seenRung[k] {
			seenRung[k] = true
			rungs = append(rungs, k)
		}
	}
	return rungs
}

var _ = registerCorpusCheck("TestTakenHitChainCensus", func() corpusCheck {
	return &takenHitChainCheck{rungs: takenHitChainRungs(), counts: map[string]int{}}
})

type takenHitChainCheck struct {
	mu                      sync.Mutex
	rungs                   []string
	counts                  map[string]int
	seen, wrong, arisesHere int
}

func (c *takenHitChainCheck) visit(_ string, rec oracleRecord, graded bool, env *Environment) {
	if !graded {
		return
	}
	wrongAt := func(key string) (bool, bool) {
		want, has := rec.Output[key]
		if !has {
			return false, false
		}
		return !oracleClose(env.Player.Output[key], want, oracleDumpTolerance), true
	}
	bad, present := wrongAt("PhysicalTakenHit")
	if !present {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen++
	if !bad {
		return
	}
	c.wrong++
	for _, k := range c.rungs {
		if w, p := wrongAt(k); p && w {
			c.counts[k]++
		}
	}
	dmgBad, _ := wrongAt("PhysicalTakenDamage")
	drBad, _ := wrongAt("PhysicalDamageReduction")
	if !dmgBad && !drBad {
		c.arisesHere++
	}
}

// TestTakenHitChainCensus answers "where in the incoming-damage chain does <type>TakenHit go wrong".
func TestTakenHitChainCensus(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*takenHitChainCheck](t)
	rungs, counts, seen, wrong, arisesHere := takenHitChainRungs(), c.counts, c.seen, c.wrong, c.arisesHere

	if seen == 0 || wrong == 0 {
		t.Fatalf("REFUSING: PhysicalTakenHit graded on %d builds, %d wrong — a census over an empty "+
			"set describes nothing", seen, wrong)
	}

	fmt.Printf("\n=== TakenHit CHAIN CENSUS === PhysicalTakenHit %d graded, %d wrong\n", seen, wrong)
	fmt.Printf("  %-28s %5d  %5.1f%%   <- arises in CalcDefence.lua:2041 itself\n",
		"INPUTS AGREE", arisesHere, 100*float64(arisesHere)/float64(wrong))
	keys := append([]string(nil), rungs...)
	sort.Slice(keys, func(a, b int) bool { return counts[keys[a]] > counts[keys[b]] })
	for _, k := range keys {
		fmt.Printf("  %-28s %5d  %5.1f%%\n", k, counts[k], 100*float64(counts[k])/float64(wrong))
	}

	if arisesHere > takenHitArisesHereCeiling {
		t.Errorf("%d of %d PhysicalTakenHit failures have both inputs AGREEING, ceiling %d — the "+
			"defect has moved INTO this step and the chain ranking above no longer describes it",
			arisesHere, wrong, takenHitArisesHereCeiling)
	}
	for _, k := range takenHitExonerated {
		if counts[k] != 0 {
			t.Errorf("%s overlaps %d PhysicalTakenHit failures, measured 0 — this rung was "+
				"exonerated and is now implicated; re-rank before trusting the census", k, counts[k])
		}
	}
}
