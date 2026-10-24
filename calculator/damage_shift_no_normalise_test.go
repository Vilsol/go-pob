package calculator

import (
	"os"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

func shiftDestTotal(env *Environment, damageType data.DamageType) float64 {
	isEle := map[data.DamageType]bool{data.DamageTypeFire: true, data.DamageTypeCold: true, data.DamageTypeLightning: true}
	total := float64(0)
	for _, dest := range data.DamageType("").Values() {
		if dest == damageType {
			continue
		}
		ele, eleHits := "", ""
		if isEle[damageType] {
			ele = string("ElementalDamageTakenAs" + dest)
			eleHits = string("ElementalDamageFromHitsTakenAs" + dest)
		}
		total += env.Player.ModDB.Sum(mod.TypeBase, nil, string(damageType+"DamageTakenAs"+dest), ele)
		total += env.Player.ModDB.Sum(mod.TypeBase, nil, string(damageType+"DamageFromHitsTakenAs"+dest), eleHits)
	}
	return total
}

var _ = registerCorpusCheck("TestDamageShiftTableIsNotNormalised", func() corpusCheck { return &damageShiftNoNormaliseCheck{} })

type damageShiftNoNormaliseCheck struct {
	mu                sync.Mutex
	checked, selfZero int
	log               corpusCheckLog
}

func (c *damageShiftNoNormaliseCheck) visit(name string, _ oracleRecord, _ bool, env *Environment) {
	if env == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if env.Player.DamageShiftTable == nil {
		c.log.fatalf(name, "REFUSING: DamageShiftTable is nil; the producer moved")
		return
	}
	for _, dt := range data.DamageType("").Values() {
		total := shiftDestTotal(env, dt)
		if total <= 100 {
			continue
		}
		c.checked++
		got := float64(0)
		for _, dest := range data.DamageType("").Values() {
			if dest != dt {
				got += env.Player.DamageShiftTable[dt][dest]
			}
		}
		if !oracleClose(got, total, 1e-9) {
			c.log.errorf(name, "%s %s: destination shares sum to %.4f, want the RAW %.4f "+
				"(a sum of exactly 100 is the normalisation this test exists to refuse)",
				name, dt, got, total)
		}
		if env.Player.DamageShiftTable[dt][dt] != 0 {
			c.log.errorf(name, "%s %s: self share %.4f, want 0 — :1822 clamps it at 0",
				name, dt, env.Player.DamageShiftTable[dt][dt])
		} else {
			c.selfZero++
		}
	}
}

// TestDamageShiftTableIsNotNormalised pins CalcDefence.lua:1820-1823.
func TestDamageShiftTableIsNotNormalised(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	c := corpusWalkCheck[*damageShiftNoNormaliseCheck](t)
	c.log.report(t)
	checked, selfZero := c.checked, c.selfZero
	if checked == 0 {
		t.Fatal("REFUSING: no corpus build reaches destTotal > 100, so this test exercised the " +
			"clamp on nothing. Empty must mean 'I looked and found nothing', never 'I did not look'")
	}
	t.Logf("destTotal > 100 on %d (damage type, build) pairs; self share 0 on %d", checked, selfZero)
}
