package calculator

import (
	"math"
	"os"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

var _ = registerCorpusCheck("TestDamageTotalSumsTheRawHit", func() corpusCheck { return &damageTotalRawCheck{} })

type damageTotalRawCheck struct {
	mu                                             sync.Mutex
	shortSeen, shortOK, allSeen, allOK, fractional int
}

func (c *damageTotalRawCheck) visit(_ string, rec oracleRecord, graded bool, env *Environment) {
	if !graded {
		return
	}
	want, has := rec.Output["NumberOfDamagingHits"]
	if !has {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, dt := range data.DamageType("").Values() {
		if v := rec.Output[string(dt)+"TakenHit"]; v > 0 && v != math.Trunc(v) {
			c.fractional++
			break
		}
	}
	agrees := oracleClose(env.Player.Output["NumberOfDamagingHits"], want, oracleDumpTolerance)
	c.allSeen++
	if agrees {
		c.allOK++
	}
	if want <= 1 {
		c.shortSeen++
		if agrees {
			c.shortOK++
		}
	}
}

// TestDamageTotalSumsTheRawHit is the red-green for CalcDefence.lua:2643-2645.
//
//	local damage = DamageIn[damageType] or 0
func TestDamageTotalSumsTheRawHit(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*damageTotalRawCheck](t)
	shortSeen, shortOK, allSeen, allOK, fractional := c.shortSeen, c.shortOK, c.allSeen, c.allOK, c.fractional

	t.Logf("=== damageTotal === all: %d of %d agree | single-pass slice: %d of %d | "+
		"builds with fractional incoming damage: %d of %d",
		allOK, allSeen, shortOK, shortSeen, fractional, allSeen)

	if shortSeen == 0 {
		t.Fatal("REFUSING: the single-pass slice is empty, so this fixture cannot reach the " +
			"overkill correction it grades")
	}
	if fractional != allSeen {
		t.Errorf("%d of %d builds carry fractional incoming damage — an integral-damage control "+
			"is now REACHABLE and this fixture should use it as its arithmetic-identity control",
			fractional, allSeen)
	}
	const shortFloor = 840
	if shortOK < shortFloor {
		t.Errorf("single-pass agreement %d of %d, want >= %d — damageTotal is summing the "+
			"ceiled/multiplied hit again (CalcDefence.lua:2645 sums the raw one)",
			shortOK, shortSeen, shortFloor)
	}
	const allFloor = 7940
	if allOK < allFloor {
		t.Errorf("corpus-wide NumberOfDamagingHits agreement %d of %d, want >= %d",
			allOK, allSeen, allFloor)
	}
}
