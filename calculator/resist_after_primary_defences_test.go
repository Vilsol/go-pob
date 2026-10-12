package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestResistSeesPerSlotDefences pins the ORDER of two sections of CalculateDefence.
func TestResistSeesPerSlotDefences(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "1148-occultist-storm-burst.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	if rec.Output["FireResist"] != rec.Output["ColdResist"] || rec.Output["FireResist"] != rec.Output["LightningResist"] {
		t.Fatalf("REFUSING: %s no longer caps all three elements together (fire %v cold %v "+
			"lightning %v); it is not exercising Melding of the Flesh any more",
			name, rec.Output["FireResist"], rec.Output["ColdResist"], rec.Output["LightningResist"])
	}
	file, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("%v", err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	for _, k := range []string{"FireResist", "ColdResist", "LightningResist"} {
		want := rec.Output[k]
		got := env.Player.Output[k]
		if math.Abs(got-want) > 0.005*math.Abs(want) {
			t.Errorf("%s: %s = %v, want %v", name, k, got, want)
		}
	}
}
