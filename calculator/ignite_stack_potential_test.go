package calculator

import (
	"math"
	"os"
	"testing"
)

// TestIgniteStackPotentialIsFiniteAndMatchesPoB closes the ENTIRE NaN population of the authoritative gate.
func TestIgniteStackPotentialIsFiniteAndMatchesPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var seen, nan, agreed int
	var nanBuilds []string
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		want, ok := rec.Output["IgniteStackPotential"]
		if !ok {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		got := env.Player.Output["IgniteStackPotential"]
		seen++
		switch {
		case math.IsNaN(got) || math.IsInf(got, 0):
			nan++
			if len(nanBuilds) < 8 {
				nanBuilds = append(nanBuilds, name)
			}
		case oracleClose(got, want, oracleDumpTolerance):
			agreed++
		}
		if seen >= 120 {
			break
		}
	}

	if seen == 0 {
		t.Fatal("REFUSING: no graded build reports IgniteStackPotential; this test no longer " +
			"exercises the mechanism it was written for")
	}
	t.Logf("IgniteStackPotential over %d sampled builds: %d agree, %d NaN/Inf", seen, agreed, nan)

	if nan > 0 {
		t.Errorf("IgniteStackPotential is NaN/Inf on %d of %d sampled builds (e.g. %v). "+
			"CalcOffence.lua:4909-4916 never divides by an unguarded Time.", nan, seen, nanBuilds)
	}
	const agreementFloor = 14
	if agreed < agreementFloor {
		t.Errorf("IgniteStackPotential agrees with PoB on %d of %d sampled builds, below the "+
			"floor of %d measured when the weighted-average chain landed", agreed, seen, agreementFloor)
	}
}
