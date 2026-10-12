package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestSelfAilmentDurationDividesByExpirationRate pins CalcDefence.lua:1600-1628.
func TestSelfAilmentDurationDividesByExpirationRate(t *testing.T) {
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
		"0007-elementalist-storm-burst-of-repulsion.xml",
		"0003-guardian-unknown.xml",
	} {
		rec, ok := oracle[name]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", name)
		}
		if rec.Output["DebuffExpirationRate"] == 0 {
			t.Fatalf("REFUSING: %s has DebuffExpirationRate 0, so it cannot witness the divisor", name)
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

		for _, k := range []string{"SelfIgniteDuration", "SelfPoisonDuration", "SelfBlindDuration"} {
			want := rec.Output[k]
			got := env.Player.Output[k]
			if math.Abs(got-want) > 0.005*math.Abs(want) {
				t.Errorf("%s: %s = %v, want %v", name, k, got, want)
			}
		}
	}
}
