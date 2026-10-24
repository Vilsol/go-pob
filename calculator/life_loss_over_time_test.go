package calculator

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
)

var lifeLossOverTimeFloors = map[string]int{
	"LifeBelowHalfLossLostOverTime": 456,
	"LifeLossLostAvg":               337,
	"LifeLossLostMax":               438,
}

var _ = registerCorpusCheck("TestLifeLossOverTimeTrackersAgree", func() corpusCheck {
	return &lifeLossOverTimeCheck{seen: map[string]int{}, agree: map[string]int{}}
})

type lifeLossOverTimeCheck struct {
	mu          sync.Mutex
	seen, agree map[string]int
}

func (c *lifeLossOverTimeCheck) visit(_ string, rec oracleRecord, graded bool, env *Environment) {
	if !graded || env.Player.Output["preventedLifeLossTotal"] <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range lifeLossOverTimeFloors {
		want, has := rec.Output[key]
		if !has {
			continue
		}
		c.seen[key]++
		if oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
			c.agree[key]++
		}
	}
}

// TestLifeLossOverTimeTrackersAgree pins the three sites that carry prevented life loss OUT of the EHP walk and into the outputs that read it.
func TestLifeLossOverTimeTrackersAgree(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*lifeLossOverTimeCheck](t)
	seen, agree := c.seen, c.agree

	keys := make([]string, 0, len(lifeLossOverTimeFloors))
	for k := range lifeLossOverTimeFloors {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if seen[k] == 0 {
			t.Fatalf("REFUSING: the oracle carries %s on 0 of the builds with "+
				"preventedLifeLossTotal > 0 — a floor over an empty set asserts nothing", k)
		}
		fmt.Printf("=== LIFE LOSS OVER TIME === %-30s %d / %d agree\n", k, agree[k], seen[k])
	}
	for _, k := range keys {
		if agree[k] < lifeLossOverTimeFloors[k] {
			t.Errorf("%s agrees on %d of %d petrified builds, floor %d", k, agree[k], seen[k],
				lifeLossOverTimeFloors[k])
		}
	}
}
