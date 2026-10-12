package calculator

import (
	"math"
	"os"
	"testing"
)

// TestTotalDotDPSCarriesTheGroundAndCorruptingBloodTerms pins CalcOffence.lua:6126.
func TestTotalDotDPSCarriesTheGroundAndCorruptingBloodTerms(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	builds := []string{
		"0123-elementalist-corrupting-fever.xml",
		"0139-elementalist-corrupting-fever-2.xml",
		"0456-elementalist-corrupting-fever-3.xml",
	}

	graded := 0
	for _, build := range builds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		want, ok := rec.Output["TotalDotDPS"]
		if !ok || want == 0 {
			t.Fatalf("REFUSING: PoB reports no nonzero TotalDotDPS for %s", build)
		}
		cb := rec.Output["CorruptingBloodDPS"]
		if math.Abs(cb-want) > math.Abs(want)*1e-9 {
			t.Fatalf("REFUSING: %s is not a clean fixture — PoB's TotalDotDPS %v is not its "+
				"CorruptingBloodDPS %v, so this build grades a sum rather than the term",
				build, want, cb)
		}
		graded++

		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			gotCB := env.Player.Output["CorruptingBloodDPS"]
			if math.Abs(gotCB-cb) > math.Abs(cb)*oracleDumpTolerance {
				t.Fatalf("REFUSING: %s is contaminated — our CorruptingBloodDPS is %v against "+
					"PoB's %v, so TotalDotDPS cannot be attributed to the sum", build, gotCB, cb)
			}
			got := env.Player.Output["TotalDotDPS"]
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: TotalDotDPS = %v, PoB reports %v (ratio %.4f)",
					build, got, want, got/want)
			}
		})
	}

	if graded == 0 {
		t.Fatal("REFUSING: no fixture graded TotalDotDPS — the population emptied rather than " +
			"this test passing")
	}
}
