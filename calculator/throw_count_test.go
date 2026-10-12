package calculator

import (
	"os"
	"testing"
)

// TestTrapAndMineThrowCount covers CalcSetup.lua:527-528 and CalcOffence.lua:1230-1237 / :1319-1336.
func TestTrapAndMineThrowCount(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var seenTrap, okTrap, seenMine, okMine int
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		wantTrap, hasTrap := rec.Output["TrapThrowCount"]
		wantMine, hasMine := rec.Output["MineThrowCount"]
		if !hasTrap && !hasMine {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		if hasTrap {
			seenTrap++
			if oracleClose(env.Player.Output["TrapThrowCount"], wantTrap, oracleDumpTolerance) {
				okTrap++
			}
		}
		if hasMine {
			seenMine++
			if oracleClose(env.Player.Output["MineThrowCount"], wantMine, oracleDumpTolerance) {
				okMine++
			}
		}
	}

	if seenTrap == 0 || seenMine == 0 {
		t.Fatalf("REFUSING: PoB publishes TrapThrowCount on %d graded builds and MineThrowCount on "+
			"%d; this test no longer measures the mechanism it was written for", seenTrap, seenMine)
	}
	t.Logf("TrapThrowCount %d/%d, MineThrowCount %d/%d", okTrap, seenTrap, okMine, seenMine)

	const (
		trapFloor = 44
		mineFloor = 94
	)
	if okTrap < trapFloor {
		t.Errorf("TrapThrowCount agrees on %d of %d graded builds, below the floor of %d measured "+
			"when this landed (CalcOffence.lua:1234)", okTrap, seenTrap, trapFloor)
	}
	if okMine < mineFloor {
		t.Errorf("MineThrowCount agrees on %d of %d graded builds, below the floor of %d measured "+
			"when this landed (CalcOffence.lua:1323)", okMine, seenMine, mineFloor)
	}
}
