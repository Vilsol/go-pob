package calculator

import (
	"os"
	"testing"
)

// TestGroundDotZeroesTotalDot covers CalcOffence.lua:5763-5782.
func TestGroundDotZeroesTotalDot(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	var checked int
	for _, build := range []string{
		"0015-occultist-vaal-caustic-arrow.xml",
		"0123-elementalist-corrupting-fever.xml",
		"0139-elementalist-corrupting-fever-2.xml",
		"0313-pathfinder-vaal-caustic-arrow.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok || rec.TreeSources < oracleMinTreeSources {
				t.Fatalf("REFUSING: %s is not graded", build)
			}
			if rec.Output["TotalDot"] != 0 || rec.Output["TotalDotCalcSection"] == 0 {
				t.Fatalf("REFUSING: PoB reports TotalDot %v and TotalDotCalcSection %v for %s; "+
					"this fixture is not on a ground-dot branch",
					rec.Output["TotalDot"], rec.Output["TotalDotCalcSection"], build)
			}

			env := corpusEnvForLoadedData(t, build)
			if got := env.Player.Output["TotalDot"]; got != 0 {
				t.Errorf("TotalDot = %v, want 0 on a ground dot (CalcOffence.lua:5764)", got)
			}
			if got, want := env.Player.Output["TotalDotCalcSection"], env.Player.Output["TotalDotInstance"]; got != want {
				t.Errorf("TotalDotCalcSection = %v, want our own TotalDotInstance %v "+
					"(CalcOffence.lua:5765)", got, want)
			}
			checked++
		})
	}
	if checked == 0 {
		t.Error("REFUSING: no fixture exercised a ground-dot branch")
	}
}
