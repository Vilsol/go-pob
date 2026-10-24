package calculator

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

const (
	fullDotDPSAgreeFloor       = 96
	fullDotDPSZeroControlFloor = 8242
)

var _ = registerCorpusCheck("TestFullDotDPSAgrees", func() corpusCheck { return &fullDotDPSCheck{} })

type fullDotDPSCheck struct {
	mu                                       sync.Mutex
	nonzeroSeen, nonzeroOK, zeroSeen, zeroOK int
	controlLost                              []string
}

func (c *fullDotDPSCheck) visit(name string, rec oracleRecord, graded bool, env *Environment) {
	if !graded {
		return
	}
	want, has := rec.Output["FullDotDPS"]
	if !has {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	agrees := oracleClose(env.Player.Output["FullDotDPS"], want, oracleDumpTolerance)
	if want != 0 {
		c.nonzeroSeen++
		if agrees {
			c.nonzeroOK++
		}
	} else {
		c.zeroSeen++
		if agrees {
			c.zeroOK++
		} else if len(c.controlLost) < 8 {
			c.controlLost = append(c.controlLost, fmt.Sprintf("%s (ours %v, PoB 0)",
				name, env.Player.Output["FullDotDPS"]))
		}
	}
}

// TestFullDotDPSAgrees pins Calcs.lua:430 — `env.player.output.FullDotDPS = fullDPS.TotalDotDPS`.
func TestFullDotDPSAgrees(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*fullDotDPSCheck](t)
	nonzeroSeen, nonzeroOK, zeroSeen, zeroOK, controlLost := c.nonzeroSeen, c.nonzeroOK, c.zeroSeen, c.zeroOK, c.controlLost

	fmt.Printf("=== FullDotDPS === nonzero %d/%d agree, PoB-zero control %d/%d agree\n",
		nonzeroOK, nonzeroSeen, zeroOK, zeroSeen)

	if nonzeroSeen == 0 || zeroSeen == 0 {
		t.Fatalf("REFUSING: nonzero population %d, PoB-zero control %d", nonzeroSeen, zeroSeen)
	}
	if nonzeroOK < fullDotDPSAgreeFloor {
		t.Errorf("FullDotDPS agrees on %d of %d nonzero builds, floor %d",
			nonzeroOK, nonzeroSeen, fullDotDPSAgreeFloor)
	}
	if zeroOK < fullDotDPSZeroControlFloor {
		t.Errorf("the PoB-zero control REGRESSED: %d of %d agree, floor %d — a nonzero gain bought "+
			"out of builds that were already right. Known losers: %v",
			zeroOK, zeroSeen, fullDotDPSZeroControlFloor, controlLost)
	}
}
