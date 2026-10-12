package calculator

import (
	"math"
	"os"
	"testing"
)

// TestTotalPoisonAverageDamageIsAPlainCopy pins CalcOffence.lua:4767-4769, which on a showAverage build is a plain copy.
func TestTotalPoisonAverageDamageIsAPlainCopy(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const (
		build       = "13142-ladder-blazerofl_5749.SummonarTreCucks.xml"
		wantAverage = 179.4587886
		wantCombine = 1429.972996486
	)

	env := corpusEnvForLoadedData(t, build)
	out := env.Player.Output

	if pc := out["PoisonChance"]; pc >= 100 {
		t.Fatalf("fixture is useless: PoisonChance is %v, where PoisonDamage and "+
			"HitChance/100*PoisonChance/100*PoisonDamage coincide — pick a build below 100", pc)
	}
	if got := out["TotalPoisonAverageDamage"]; math.Abs(got-wantAverage) > wantAverage*5e-3 {
		t.Errorf("TotalPoisonAverageDamage = %v, want %v (PoB) — CalcOffence.lua:4768 copies "+
			"PoisonDamage (%v) with no chance factors", got, wantAverage, out["PoisonDamage"])
	}
	if got := out["CombinedAvg"]; math.Abs(got-wantCombine) > wantCombine*5e-3 {
		t.Errorf("CombinedAvg = %v, want %v (PoB) — CalcOffence.lua:5993 adds "+
			"TotalPoisonAverageDamage, not PoisonDamage", got, wantCombine)
	}
}
