package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestConsecratedGroundRunsForEveryActor pins CalcPerform.lua:770-775's POSITION, not its arithmetic.
func TestConsecratedGroundRunsForEveryActor(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var seen, wrong int
	var wrongNames []string
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		if env.Minion == nil || env.Minion.Actor == nil {
			continue
		}
		if !env.Minion.ModDB.Flag(nil, "Condition:OnConsecratedGround") {
			continue
		}
		seen++

		if got := env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf"); got >= 0 {
			t.Errorf("%s: the minion's CurseEffectOnSelf INC is %v, want negative — "+
				"CalcPerform.lua:773 did not run for this actor", name, got)
		}

		if got := env.Minion.ModDB.Sum(mod.TypeBase, nil, "LifeRegenPercent"); got < 5 {
			t.Errorf("%s: the minion's LifeRegenPercent BASE sum is %v, want at least the 5 "+
				"CalcPerform.lua:772 grants", name, got)
		}

		if want, has := rec.Minion["CurseEffectOnSelf"]; has &&
			!oracleClose(env.Minion.Output["CurseEffectOnSelf"], want, oracleDumpTolerance) {
			wrong++
			if len(wrongNames) < 5 {
				wrongNames = append(wrongNames, name)
			}
		}
		if seen >= 25 {
			break
		}
	}

	if seen == 0 {
		t.Fatal("REFUSING: no graded corpus build has a minion on consecrated ground, so the " +
			"per-actor position decides nothing and this test measured nothing")
	}
	if wrong > 0 {
		t.Errorf("Minion.CurseEffectOnSelf disagrees with PoB on %d of %d builds with a minion "+
			"on consecrated ground: %v", wrong, seen, wrongNames)
	}
}
