package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestEHPLoopReadsEffectiveSuppression pins CalcDefence.lua:2770-2773.
func TestEHPLoopReadsEffectiveSuppression(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "0022-saboteur-blade-trap-of-greatswords.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}

	plain := rec.Output["SpellSuppressionChance"]
	effective := rec.Output["EffectiveSpellSuppressionChance"]
	if math.Abs(plain-effective) < 1e-9 {
		t.Fatalf("REFUSING: %s no longer distinguishes the two keys (both %v); pick another build "+
			"from the 168 where PoB rerolls suppression", name, plain)
	}

	file, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(string(file), "Suppress") {
		t.Fatalf("REFUSING: %s carries no suppression text at all", name)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	if got := env.Player.Output["EffectiveSpellSuppressionChance"]; math.Abs(got-effective) > 0.005*effective {
		t.Fatalf("REFUSING: our EffectiveSpellSuppressionChance is %v against PoB's %v; the "+
			"producer disagrees, so this test cannot isolate the consumer", got, effective)
	}

	for _, k := range []string{"TotalNumberOfHits", "EHPSurvivalTime"} {
		want := rec.Output[k]
		got := env.Player.Output[k]
		if math.Abs(got-want) > 0.005*math.Abs(want)+1e-9 {
			t.Errorf("%s: %s = %v, want %v (the EHP loop must read EffectiveSpellSuppressionChance, "+
				"not the pre-reroll SpellSuppressionChance)", name, k, got, want)
		}
	}
}
