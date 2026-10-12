package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestSecondaryGrantedEffectIsSupported pins CalcSetup.lua:1620-1625.
func TestSecondaryGrantedEffectIsSupported(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	for _, name := range []string{
		"0138-berserker-earthshatter-3.xml",
		"0486-berserker-earthshatter-12.xml",
	} {
		rec, ok := oracle[name]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", name)
		}
		if rec.Output["ManaReservedPercent"] == 0 {
			t.Fatalf("REFUSING: %s reserves no mana in the oracle; the fixture has moved", name)
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

		want := rec.Output["ManaReservedPercent"]
		got := env.Player.Output["ManaReservedPercent"]
		if math.Abs(got-want) > 0.005*math.Abs(want) {
			t.Errorf("%s: ManaReservedPercent = %v, want %v", name, got, want)
		}
	}
}
