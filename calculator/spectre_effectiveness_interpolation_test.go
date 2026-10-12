package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestSpectreAuraScalesWithActorLevel pins CalcTools.lua:196-205 — the arm that scales a stat by the ACTOR's level rather than by the gem's.
//
//	if level.statInterpolation[index] == 3 then
func TestSpectreAuraScalesWithActorLevel(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "1375-necromancer-raise-spectre-19.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	if rec.Output["Armour"] == 0 {
		t.Fatalf("REFUSING: %s publishes no Armour; the fixture has moved", name)
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

	const source = "Spectre:Determination - "
	var base float64
	found := false
	for _, m := range env.Player.ModDB.Mods["Armour"] {
		if strings.HasPrefix(string(m.GetSource()), source) && m.Type() == mod.TypeBase {
			base, found = ourModValue(m), true
		}
	}
	if !found {
		t.Fatalf("REFUSING: %s grants no %s BASE Armour modifier; the fixture's spectre has "+
			"changed and the test measures nothing", name, source)
	}

	if base != 1235 {
		t.Errorf("%s: %s BASE Armour = %v, want 1235 (18 is the same expression at actorLevel 1)",
			name, source, base)
	}
}

// TestMinionDamageScalesWithActorLevel is the OUTPUT-level half.
func TestMinionDamageScalesWithActorLevel(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "0664-necromancer-raise-spectre-4.xml"
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
	if env.Minion == nil {
		t.Fatalf("REFUSING: %s grants no minion actor; the fixture has changed", name)
	}

	for _, key := range []string{"TotalMin", "TotalMax"} {
		want := rec.Minion[key]
		if want == 0 {
			t.Fatalf("REFUSING: %s publishes no Minion.%s; the fixture has moved", name, key)
		}
		got := env.Minion.Output[key]
		if math.Abs(got-want) > 0.005*math.Abs(want) {
			t.Errorf("%s: Minion.%s = %v, want %v (ratio %.4f)", name, key, got, want, got/want)
		}
	}
}
