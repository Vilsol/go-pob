package calculator

import (
	"os"
	"sync"
	"testing"
)

var _ = registerCorpusCheck("TestVaalStoredUsesFeedsStoredUses", func() corpusCheck { return &vaalStoredUsesCheck{} })

type vaalStoredUsesCheck struct {
	mu                            sync.Mutex
	graded, starved, wrongNonzero int
}

func (c *vaalStoredUsesCheck) visit(_ string, rec oracleRecord, _ bool, env *Environment) {
	want, has := rec.Output["StoredUses"]
	if !has || want == 0 {
		return
	}
	if env == nil || env.Player == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.graded++
	got := env.Player.Output["StoredUses"]
	if oracleClose(got, want, oracleDumpTolerance) {
		return
	}
	if got == 0 {
		c.starved++
	} else {
		c.wrongNonzero++
	}
}

// TestVaalStoredUsesFeedsStoredUses: `StoredUses` is STARVED, not mis-factored, and that is what names its cause.
//
//	if level.storedUses then     activeSkill.skillData.storedUses = level.storedUses end
//	if level.vaalStoredUses then activeSkill.skillData.storedUses =
func TestVaalStoredUsesFeedsStoredUses(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*vaalStoredUsesCheck](t)
	graded, starved, wrongNonzero := c.graded, c.starved, c.wrongNonzero

	if graded == 0 {
		t.Fatal("REFUSING: no build carries a nonzero StoredUses in the oracle, so this sweep is " +
			"not reaching the stored-uses block at all")
	}

	t.Logf("StoredUses: %d graded, %d starved to exactly 0, %d wrong-nonzero", graded, starved, wrongNonzero)

	const starvedCeiling = 2
	if starved > starvedCeiling {
		t.Errorf("StoredUses is starved to exactly 0 on %d builds (ceiling %d) — the vaal arm of "+
			"the granted-effect level loop is the only producer for these", starved, starvedCeiling)
	}

	const wrongCeiling = 5
	if wrongNonzero > wrongCeiling {
		t.Errorf("StoredUses is wrong-but-nonzero on %d builds (ceiling %d) — that is a wrong "+
			"factor, not a starved producer", wrongNonzero, wrongCeiling)
	}
}
