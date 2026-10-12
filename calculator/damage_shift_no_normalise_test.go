package calculator

import (
	"os"
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

// TestDamageShiftTableIsNotNormalised pins CalcDefence.lua:1820-1823.
func TestDamageShiftTableIsNotNormalised(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	_, names := corpusNamesForLoadedData(t)

	checked, selfZero := 0, 0
	for _, name := range names {
		env := corpusEnvForLoadedData(t, name)
		if env.Player.DamageShiftTable == nil {
			t.Fatal("REFUSING: DamageShiftTable is nil; the producer moved")
		}
		for _, dt := range data.DamageType("").Values() {
			total := shiftDestTotal(env, dt)
			if total <= 100 {
				continue
			}
			checked++
			got := float64(0)
			for _, dest := range data.DamageType("").Values() {
				if dest != dt {
					got += env.Player.DamageShiftTable[dt][dest]
				}
			}
			if !oracleClose(got, total, 1e-9) {
				t.Errorf("%s %s: destination shares sum to %.4f, want the RAW %.4f "+
					"(a sum of exactly 100 is the normalisation this test exists to refuse)",
					name, dt, got, total)
			}
			if env.Player.DamageShiftTable[dt][dt] != 0 {
				t.Errorf("%s %s: self share %.4f, want 0 — :1822 clamps it at 0",
					name, dt, env.Player.DamageShiftTable[dt][dt])
			} else {
				selfZero++
			}
		}
	}
	if checked == 0 {
		t.Fatal("REFUSING: no corpus build reaches destTotal > 100, so this test exercised the " +
			"clamp on nothing. Empty must mean 'I looked and found nothing', never 'I did not look'")
	}
	t.Logf("destTotal > 100 on %d (damage type, build) pairs; self share 0 on %d", checked, selfZero)
}
