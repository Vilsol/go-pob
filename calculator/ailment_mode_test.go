package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestAilmentModeConfigAndFlag pins CalcOffence.lua:4165-4179.
func TestAilmentModeConfigAndFlag(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	e := ParseMod("Non-Critical Strikes cannot inflict Ailments", false)
	if e == nil || strings.TrimSpace(modExtra(e)) != "" {
		t.Errorf("`Non-Critical Strikes cannot inflict Ailments` does not parse cleanly")
	} else {
		got := make([]string, 0, len(e.ModList))
		for _, m := range e.ModList {
			got = append(got, m.Name())
		}
		if !contains(got, "AilmentsOnlyFromCrit") {
			t.Errorf("produced %v, want AilmentsOnlyFromCrit", got)
		}
	}

	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "1078-elementalist-fireball-3.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	file, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(string(file), `name="ailmentMode" string="CRIT"`) {
		t.Fatalf("REFUSING: %s no longer sets ailmentMode=CRIT", name)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	for _, k := range []string{"IgniteChanceOnHit", "ShockChanceOnHit", "FreezeChanceOnHit"} {
		want := rec.Output[k]
		got := env.Player.Output[k]
		if math.Abs(got-want) > 0.005*math.Abs(want)+1e-9 {
			t.Errorf("%s: %s = %v, want %v", name, k, got, want)
		}
	}
}
