package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func assertMainHandStats(t *testing.T, oracle map[string]oracleRecord, name string, stats ...string) {
	t.Helper()
	rec, ok := oracle[name+".xml"]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	env := corpusEnvForLoadedData(t, name)
	mine := env.Player.OutputTable[OutTableMainHand]
	for _, k := range stats {
		want := rec.MainHand[k]
		if want == 0 {
			t.Fatalf("REFUSING: PoB reports 0 for MainHand.%s on %s; a 0-vs-0 agreement proves nothing", k, name)
		}
		if got := mine[k]; math.Abs(got-want) > 0.005*math.Abs(want)+1e-9 {
			t.Errorf("%s: MainHand.%s = %v, want %v", name, k, got, want)
		}
	}
}

// TestIgnoreEnemyPhysicalDamageReduction pins CalcOffence.lua:3423-3435, the chance-to-ignore arm of the enemy physical damage-reduction calculation.
func TestIgnoreEnemyPhysicalDamageReduction(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}

	t.Run("a chance at or above 100 zeroes the reduction", func(t *testing.T) {
		assertMainHandStats(t, oracle, "0429-slayer-precision", "PhysicalHitAverage", "TotalMin")
	})

	t.Run("a partial chance scales the reduction", func(t *testing.T) {
		assertMainHandStats(t, oracle, "0071-slayer-flicker-strike", "PhysicalHitAverage", "TotalMin")
	})

	t.Run("a non-carrier is untouched", func(t *testing.T) {
		const name = "0183-pathfinder-toxic-rain-3"
		env := corpusEnvForLoadedData(t, name)
		ms := env.Player.MainSkill
		if ms == nil {
			t.Fatal("control is useless: no main skill")
		}
		if got := ms.SkillModList.Sum(mod.TypeBase, ms.SkillCfg, "ChanceToIgnoreEnemyPhysicalDamageReduction"); got != 0 {
			t.Fatalf("control is useless: the build now carries a chance of %v, so it is a "+
				"carrier and no longer controls for anything", got)
		}
		assertMainHandStats(t, oracle, name, "PhysicalHitAverage", "TotalMin")
	})
}
