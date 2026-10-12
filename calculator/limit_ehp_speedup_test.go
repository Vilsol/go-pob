package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestLimitEHPSpeedup covers CalcDefence.lua:2682, :2690 and :2837 — the flag that halves the EHP walk's acceleration whenever a per-hit tracker is armed.
func TestLimitEHPSpeedup(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "0479-hierophant-kinetic-fusillade-5.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
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

	for _, key := range []string{
		"NumberOfMitigatedDamagingHits",
		"NumberOfDamagingHits",
	} {
		want := rec.Output[key]
		if want == 0 {
			t.Fatalf("REFUSING: %s publishes no %s; the fixture has moved", name, key)
		}
		got := env.Player.Output[key]
		if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
			t.Errorf("%s: %s = %v, want %v (PoB)", name, key, got, want)
		}
	}
}
