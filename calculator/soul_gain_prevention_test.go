package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestSoulGainPreventionDuration pins CalcOffence.lua:1230-1244.
func TestSoulGainPreventionDuration(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS!=true")
	}
	if raw.LatestVersion != "3.29" {
		t.Skipf("SKIP: this test pins 3.29 values but the process loaded %s game data", raw.LatestVersion)
	}

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}

	builds3 := []string{
		"12671-ladder-WcEendVis_5386.dkfljgjkdfhg.xml",
		"0430-reliquarian-vaal-cold-snap.xml",
		"6919-ladder-borpasnorpglorpin_6384.deletiiiiiiii.xml",
	}

	for _, build := range builds3 {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", build)
		}
		want, ok := rec.Output["SoulGainPreventionDuration"]
		if !ok {
			t.Fatalf("REFUSING: PoB emits no SoulGainPreventionDuration for %s — wrong fixture", build)
		}

		xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build))
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := builds.ParseBuild(xml)
		if err != nil {
			t.Fatal(err)
		}
		env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)

		got, ok := env.Player.Output["SoulGainPreventionDuration"]
		switch {
		case !ok:
			t.Errorf("%s: we never write SoulGainPreventionDuration; PoB has %g", build, want)
		case !oracleClose(got, want, oracleDumpTolerance):
			t.Errorf("%s: SoulGainPreventionDuration = %g, PoB %g", build, got, want)
		}
	}
}
