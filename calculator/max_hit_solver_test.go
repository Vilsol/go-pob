package calculator

import (
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
	"testing"
)

const maxHitSolverAgreeFloor = 46017

var _ = registerCorpusCheck("TestMaxHitSolverAgrees", func() corpusCheck {
	return &maxHitSolverCheck{perStat: map[string][2]int{}}
})

type maxHitSolverCheck struct {
	mu                     sync.Mutex
	builds                 int
	nonzeroSeen, nonzeroOK int
	perStat                map[string][2]int
	nanBuilds              []string
}

func (c *maxHitSolverCheck) visit(name string, rec oracleRecord, graded bool, env *Environment) {
	if !graded {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.builds++
	for _, stat := range maxHitSolverStats {
		want, has := rec.Output[stat]
		if !has || want == 0 {
			continue
		}
		got := env.Player.Output[stat]
		if math.IsNaN(got) || math.IsInf(got, 0) {
			if len(c.nanBuilds) < 8 {
				c.nanBuilds = append(c.nanBuilds, fmt.Sprintf("%s %s=%v", name, stat, got))
			}
		}
		c.nonzeroSeen++
		pc := c.perStat[stat]
		pc[0]++
		if oracleClose(got, want, oracleDumpTolerance) {
			c.nonzeroOK++
			pc[1]++
		}
		c.perStat[stat] = pc
	}
}

var maxHitSolverStats = []string{
	"PhysicalMaximumHitTaken", "LightningMaximumHitTaken", "ColdMaximumHitTaken",
	"FireMaximumHitTaken", "ChaosMaximumHitTaken", "SecondMinimalMaximumHitTaken",
}

// TestMaxHitSolverAgrees pins CalcDefence.lua:3165-3222.
func TestMaxHitSolverAgrees(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*maxHitSolverCheck](t)
	nonzeroSeen, nonzeroOK, perStat, nanBuilds := c.nonzeroSeen, c.nonzeroOK, c.perStat, c.nanBuilds

	keys := make([]string, 0, len(perStat))
	for k := range perStat {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Printf("=== MAX HIT SOLVER === %d/%d nonzero pairs agree\n", nonzeroOK, nonzeroSeen)
	for _, k := range keys {
		fmt.Printf("      %-30s %d/%d\n", k, perStat[k][1], perStat[k][0])
	}

	if nonzeroSeen == 0 {
		t.Fatalf("REFUSING: 0 nonzero max-hit pairs across %d builds and %d stats",
			c.builds, len(maxHitSolverStats))
	}
	for _, s := range maxHitSolverStats {
		if perStat[s][0] == 0 {
			t.Errorf("REFUSING: %s has an EMPTY nonzero population — it is no longer being compared", s)
		}
	}
	if len(nanBuilds) > 0 {
		t.Errorf("the solver produced NaN/Inf on %d+ pairs — CalcDefence.lua:3212 guards `a ~= 0` "+
			"and clamps the discriminant at 0 for exactly this: %v", len(nanBuilds), nanBuilds)
	}
	if nonzeroOK < maxHitSolverAgreeFloor {
		t.Errorf("max-hit solver agrees on %d of %d nonzero pairs, floor %d",
			nonzeroOK, nonzeroSeen, maxHitSolverAgreeFloor)
	}
}
