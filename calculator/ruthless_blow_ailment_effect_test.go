package calculator

import (
	"fmt"
	"os"
	"testing"
)

const (
	ruthlessAilmentPoisonAgreeFloor = 17
	ruthlessAilmentControlFloor     = 385
)

// TestRuthlessBlowAilmentEffectReachesEveryAilment pins CalcOffence.lua:4721's RuthlessBlowAilmentEffect factor on the POISON expression.
func TestRuthlessBlowAilmentEffectReachesEveryAilment(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var ruthlessSeen, ruthlessOK, controlSeen, controlOK int
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		want, has := rec.MinionMain["PoisonDPS"]
		if !has || want == 0 {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		if env.Minion == nil || env.Minion.Actor == nil {
			continue
		}
		main := env.Minion.OutputTable[OutTableMainHand]
		if main == nil {
			continue
		}
		agrees := oracleClose(main["PoisonDPS"], want, oracleDumpTolerance)

		if main["RuthlessBlowAilmentEffect"] != 1 {
			ruthlessSeen++
			if agrees {
				ruthlessOK++
			}
		} else {
			controlSeen++
			if agrees {
				controlOK++
			}
		}
	}

	if ruthlessSeen == 0 || controlSeen == 0 {
		t.Fatalf("REFUSING: ruthless population %d, control population %d — a floor over an empty "+
			"set asserts nothing", ruthlessSeen, controlSeen)
	}
	fmt.Printf("=== RUTHLESS BLOW AILMENT EFFECT === ruthless %d/%d agree · control %d/%d agree\n",
		ruthlessOK, ruthlessSeen, controlOK, controlSeen)

	if ruthlessOK < ruthlessAilmentPoisonAgreeFloor {
		t.Errorf("Minion.MainHand.PoisonDPS agrees on %d of %d builds whose minion carries a "+
			"RuthlessBlowAilmentEffect != 1, floor %d — CalcOffence.lua:4721's factor regressed",
			ruthlessOK, ruthlessSeen, ruthlessAilmentPoisonAgreeFloor)
	}
	if controlOK < ruthlessAilmentControlFloor {
		t.Errorf("Minion.MainHand.PoisonDPS agrees on %d of %d builds whose minion has NO ruthless "+
			"blow effect, floor %d — the factor was applied where it is inert",
			controlOK, controlSeen, ruthlessAilmentControlFloor)
	}
}
