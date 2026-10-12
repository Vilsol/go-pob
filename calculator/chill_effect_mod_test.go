package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestChillEffectModIsAMultiplier pins CalcOffence.lua:5256-5262, the ChillingArea / NonHitChill branch.
func TestChillEffectModIsAMultiplier(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	for _, name := range []string{
		"0063-occultist-wintertide-brand.xml",
	} {
		rec, ok := oracle[name]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob; this test cannot check anything", name)
		}
		want, ok := rec.Output["ChillEffectMod"]
		if !ok || want == 0 {
			t.Fatalf("REFUSING: %s publishes no ChillEffectMod; the fixture has moved", name)
		}
		file, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		got := env.Player.Output["ChillEffectMod"]
		if math.Abs(got-want) > 0.005*math.Abs(want) {
			t.Errorf("%s: ChillEffectMod = %v, want %v (a MULTIPLIER, not a percentage)", name, got, want)
		}
	}
}
