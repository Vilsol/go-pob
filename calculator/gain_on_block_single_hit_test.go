package calculator

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

const (
	singleHitGainAgreeFloor = 171
	multiHitGainAgreeFloor  = 1970
)

var gainOnBlockKeys = []string{"LifeOnBlock", "ManaOnBlock", "EnergyShieldOnBlock",
	"EnergyShieldOnSpellBlock", "EnergyShieldOnSuppress", "LifeOnSuppress"}

var _ = registerCorpusCheck("TestGainOnBlockNeedsMoreThanOneHit", func() corpusCheck { return &gainOnBlockSingleHitCheck{} })

type gainOnBlockSingleHitCheck struct {
	mu                                       sync.Mutex
	singleSeen, singleOK, multiSeen, multiOK int
}

func (c *gainOnBlockSingleHitCheck) visit(_ string, rec oracleRecord, graded bool, env *Environment) {
	if !graded {
		return
	}
	want, has := rec.Output["NumberOfMitigatedDamagingHits"]
	if !has {
		return
	}
	gains := false
	for _, k := range gainOnBlockKeys {
		if env.Player.Output[k] != 0 {
			gains = true
		}
	}
	if !gains {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	agrees := oracleClose(env.Player.Output["NumberOfMitigatedDamagingHits"], want, oracleDumpTolerance)
	if rec.Output["NumberOfDamagingHits"] <= 1 {
		c.singleSeen++
		if agrees {
			c.singleOK++
		}
	} else {
		c.multiSeen++
		if agrees {
			c.multiOK++
		}
	}
}

// TestGainOnBlockNeedsMoreThanOneHit pins the second conjunct of CalcDefence.lua:2760 and :2793, which this port dropped from both.
//
//	if not env.configInput.DisableEHPGainOnBlock and (output["NumberOfDamagingHits"] > 1) then
func TestGainOnBlockNeedsMoreThanOneHit(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*gainOnBlockSingleHitCheck](t)
	singleSeen, singleOK, multiSeen, multiOK := c.singleSeen, c.singleOK, c.multiSeen, c.multiOK

	if singleSeen == 0 || multiSeen == 0 {
		t.Fatalf("REFUSING: single-hit population %d, multi-hit control %d — a floor over an empty "+
			"set asserts nothing", singleSeen, multiSeen)
	}
	fmt.Printf("=== GAIN ON BLOCK / SINGLE HIT === dies-to-one-hit %d/%d agree · control %d/%d agree\n",
		singleOK, singleSeen, multiOK, multiSeen)

	if singleOK < singleHitGainAgreeFloor {
		t.Errorf("NumberOfMitigatedDamagingHits agrees on %d of %d builds PoB kills in one hit, "+
			"floor %d — the :2760 / :2793 conjunct regressed", singleOK, singleSeen,
			singleHitGainAgreeFloor)
	}
	if multiOK < multiHitGainAgreeFloor {
		t.Errorf("NumberOfMitigatedDamagingHits agrees on %d of %d control builds, floor %d — the "+
			"conjunct was paid for out of the builds that satisfy it", multiOK, multiSeen,
			multiHitGainAgreeFloor)
	}
}
