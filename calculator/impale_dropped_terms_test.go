package calculator

import (
	"math"
	"os"
	"testing"
)

// TestImpaleAdditionalDurationChanceScalesTheStackCap is the positive successor to the tripwire that stood here until 2026-08-30.
func TestImpaleAdditionalDurationChanceScalesTheStackCap(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	builds := []string{
		"0016-gladiator-dual-strike-of-ambidexterity.xml",
		"0845-champion-double-strike-of-impaling.xml",
		"0101-champion-lancing-steel.xml",
	}

	graded := 0
	for _, build := range builds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		want, ok := rec.Output["ImpaleStacksMax"]
		if !ok || want == 0 {
			t.Fatalf("REFUSING: PoB reports no nonzero ImpaleStacksMax for %s -- this fixture "+
				"grades nothing", build)
		}

		env := corpusEnvForLoadedData(t, build)
		got := env.Player.Output["ImpaleStacksMax"]
		if got == 0 {
			t.Fatalf("REFUSING: %s produces no ImpaleStacksMax at all; this test is about the "+
				"1.1x factor, not about the base cap", build)
		}
		graded++

		if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
			t.Errorf("%s: ImpaleStacksMax = %v against PoB's %v, ratio %.4f. A ratio of 0.9091 "+
				"= 1/1.1 means the ImpaleAdditionalDurationChance parser row "+
				"(ModParser.lua:3859) has been lost again.", build, got, want, got/want)
		}
	}

	if graded == 0 {
		t.Fatal("REFUSING: no fixture graded ImpaleStacksMax -- the population emptied and this " +
			"test would otherwise have passed vacuously")
	}
}
