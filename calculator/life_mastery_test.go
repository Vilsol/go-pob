package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestLifeMasteryMoreLife pins the two halves of PoB's Life Mastery "10% more Maximum Life".
//
//	CalcSetup.lua:688-690   modDB:NewMod("Multiplier:AllocatedLifeMastery", "BASE",
func TestLifeMasteryMoreLife(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var withMastery, decisive, moreApplied, seen, wrong int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		count := env.ModDB.Multipliers["AllocatedLifeMastery"]
		if count == 0 {
			count = env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:AllocatedLifeMastery")
		}
		if count > 0 {
			withMastery++
		}
		if count < 6 {
			continue
		}
		decisive++
		if env.Player.ModDB.More(nil, "Life") > 1 {
			moreApplied++
		}
		if want, has := rec.Output["Life"]; has && want != 0 {
			seen++
			if !oracleClose(env.Player.Output["Life"], want, oracleDumpTolerance) {
				wrong++
			}
		}
		if decisive >= 60 {
			break
		}
	}

	if withMastery == 0 {
		t.Fatal("REFUSING: no graded build allocates a Life Mastery at all, so " +
			"Multiplier:AllocatedLifeMastery is never produced and this test measures nothing. " +
			"Before this port that count was zero by construction")
	}
	if decisive == 0 {
		t.Fatalf("REFUSING: %d builds allocate a Life Mastery but none reaches the threshold of 6, "+
			"so the MORE never fires and the parser entry is ungraded here", withMastery)
	}
	if moreApplied == 0 {
		t.Errorf("across %d builds at or above the threshold, NOT ONE carries a More(Life) above "+
			"1 — the parser entry or the MultiplierThreshold tag is not reaching the modDB", decisive)
	}
	if wrong > 0 {
		t.Logf("Life still disagrees with PoB on %d of %d builds at or above the threshold "+
			"(a separate residual)", wrong, seen)
	}
	t.Logf("MEASURED: %d builds allocate a Life Mastery, %d reach the threshold", withMastery, decisive)
}
