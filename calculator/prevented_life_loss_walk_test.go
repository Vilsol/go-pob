package calculator

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

const (
	preventedWalkAgreeFloor   = 500
	unpreventedWalkAgreeFloor = 6202
)

var _ = registerCorpusCheck("TestPreventedLifeLossWalkAgrees", func() corpusCheck { return &preventedLifeLossWalkCheck{} })

type preventedLifeLossWalkCheck struct {
	mu                                                 sync.Mutex
	preventedSeen, preventedOK, controlSeen, controlOK int
}

func (c *preventedLifeLossWalkCheck) visit(_ string, rec oracleRecord, graded bool, env *Environment) {
	if !graded {
		return
	}
	want, has := rec.Output["NumberOfDamagingHits"]
	if !has {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	agrees := oracleClose(env.Player.Output["NumberOfDamagingHits"], want, oracleDumpTolerance)
	if env.Player.Output["preventedLifeLossTotal"] > 0 {
		c.preventedSeen++
		if agrees {
			c.preventedOK++
		}
	} else {
		c.controlSeen++
		if agrees {
			c.controlOK++
		}
	}
}

// TestPreventedLifeLossWalkAgrees pins CalcDefence.lua:348-392, the prevented-life-loss arm of calcs.reducePoolsByDamage, as it is threaded through the EHP walk.
func TestPreventedLifeLossWalkAgrees(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*preventedLifeLossWalkCheck](t)
	preventedSeen, preventedOK, controlSeen, controlOK := c.preventedSeen, c.preventedOK, c.controlSeen, c.controlOK

	if preventedSeen == 0 || controlSeen == 0 {
		t.Fatalf("REFUSING: prevented population %d, control population %d — a floor over an empty "+
			"set asserts nothing", preventedSeen, controlSeen)
	}
	fmt.Printf("=== PREVENTED LIFE LOSS WALK === petrified %d/%d agree · control %d/%d agree\n",
		preventedOK, preventedSeen, controlOK, controlSeen)

	if preventedOK < preventedWalkAgreeFloor {
		t.Errorf("NumberOfDamagingHits agrees on %d of %d builds with preventedLifeLossTotal > 0, "+
			"floor %d — the CalcDefence.lua:348-392 drain regressed", preventedOK, preventedSeen,
			preventedWalkAgreeFloor)
	}
	if controlOK < unpreventedWalkAgreeFloor {
		t.Errorf("NumberOfDamagingHits agrees on %d of %d builds with NO prevented life loss, floor "+
			"%d — the petrified branch was paid for out of the builds it does not apply to",
			controlOK, controlSeen, unpreventedWalkAgreeFloor)
	}
}
