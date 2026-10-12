package calculator

import (
	"math"
	"os"
	"testing"
)

var removableChargeBuilds = []string{
	"0001-necromancer-unknown.xml",
	"0002-pathfinder-unknown.xml",
	"0004-trickster-lightning-tendrils-of-eccentricity.xml",
	"0005-occultist-vaal-double-strike.xml",
}

func TestRemovableTotalChargesIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	sawNonzero := false
	for _, build := range removableChargeBuilds {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			want, ok := rec.Output["RemovableTotalCharges"]
			if !ok {
				t.Fatalf("REFUSING: PoB reports no RemovableTotalCharges for %s", build)
			}
			if want == 0 {
				t.Fatalf("REFUSING: PoB reports 0 for %s; an unwritten key agrees with that and "+
					"the case cannot discriminate", build)
			}
			sawNonzero = true

			env := corpusEnvForLoadedData(t, build)
			got, present := env.Player.Output["RemovableTotalCharges"]
			if !present {
				t.Errorf("%s: we never write RemovableTotalCharges at all; PoB reports %v", build, want)
				return
			}
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: RemovableTotalCharges = %v, PoB reports %v", build, got, want)
			}

			sum := env.Player.Output["RemovableEnduranceCharges"] +
				env.Player.Output["RemovableFrenzyCharges"] +
				env.Player.Output["RemovablePowerCharges"]
			if math.Abs(got-sum) > 1e-9 {
				t.Errorf("%s: RemovableTotalCharges = %v but the three summands total %v", build, got, sum)
			}
		})
	}
	if !sawNonzero {
		t.Fatal("REFUSING: every fixture reports 0, so none of them can distinguish this from an " +
			"absent producer")
	}
}
