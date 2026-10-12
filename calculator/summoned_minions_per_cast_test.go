package calculator

import (
	"os"
	"testing"
)

// TestSummonedMinionsPerCast covers CalcOffence.lua:1027-1030.
func TestSummonedMinionsPerCast(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var seen, agreed, nonTrivial int
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		want, has := rec.Output["SummonedMinionsPerCast"]
		if !has {
			continue
		}
		seen++
		if want != 1 {
			nonTrivial++
		}
		env := corpusEnvForLoadedData(t, name)
		if oracleClose(env.Player.Output["SummonedMinionsPerCast"], want, oracleDumpTolerance) {
			agreed++
		}
		if seen >= 120 {
			break
		}
	}

	if seen == 0 {
		t.Fatal("REFUSING: no graded build publishes SummonedMinionsPerCast; this test no longer " +
			"measures the mechanism it was written for")
	}
	t.Logf("SummonedMinionsPerCast %d of %d (%d of them not the fallback 1)", agreed, seen, nonTrivial)

	const floor = 118
	if agreed < floor {
		t.Errorf("SummonedMinionsPerCast agrees on %d of %d graded builds, below the floor of %d "+
			"(CalcOffence.lua:1027-1030)", agreed, seen, floor)
	}
}
