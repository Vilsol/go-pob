package calculator

import (
	"os"
	"testing"
)

// TestLifeFlaskChargesReachesTheOracle pins CalcOffence.lua:533, which sits ONE LINE below the :532 this file already ported as LifeFlaskRecovery.
func TestLifeFlaskChargesReachesTheOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var checked, agreed int
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		want, ok := rec.Output["LifeFlaskCharges"]
		if !ok || want == 0 {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		got := env.Player.Output["LifeFlaskCharges"]
		checked++
		if got == want {
			agreed++
		}
		if checked >= 40 {
			break
		}
	}

	if checked == 0 {
		t.Fatal("REFUSING: not one graded build reports a nonzero LifeFlaskCharges; this test no " +
			"longer exercises the mechanism it was written for")
	}
	t.Logf("LifeFlaskCharges: %d of %d sampled builds agree with PoB exactly", agreed, checked)
	if agreed != checked {
		t.Errorf("LifeFlaskCharges agrees on %d of %d builds; CalcOffence.lua:533 publishes the "+
			"highest chargesMax among equipped Life flasks (CalcSetup.lua:936-941)", agreed, checked)
	}
}
