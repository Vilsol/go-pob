package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestEnemyModifierRows pins nine specialModList rows that put a modifier on the ENEMY.
func TestEnemyModifierRows(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	if e := ParseMod("With at least one nearby corpse, you and nearby Allies deal 10% more Damage", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the ported ALLY half of this passive no longer parses; this test " +
			"cannot attribute a failure to the missing enemy half")
	}

	for _, line := range []string{
		"With at least one nearby corpse, nearby Enemies deal 10% reduced Damage",
		"Nearby Chilled Enemies deal 10% reduced Damage with Hits",
		"Nearby Hindered Enemies deal 10% reduced Damage over Time",
		"Nearby Enemies cannot deal Critical Strikes",
		"Enemies Poisoned by you cannot deal Critical Strikes",
	} {
		e := ParseMod(line, false)
		if e == nil {
			t.Errorf("no entry: %s", line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q (line is DROPPED WHOLE): %s", extra, line)
			continue
		}
		var sawEnemyMod bool
		for _, m := range e.ModList {
			if m.Name() == "EnemyModifier" {
				sawEnemyMod = true
			}
		}
		if !sawEnemyMod {
			t.Errorf("no EnemyModifier produced: %s", line)
		}
	}

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)
	const name = "0008-necromancer-vaal-summon-skeletons.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	if rec.Output["PhysicalEnemyDamageMult"] == 1 {
		t.Fatalf("REFUSING: %s no longer reduces enemy damage, so it cannot witness this", name)
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
	for _, k := range []string{"PhysicalEnemyDamageMult", "FireEnemyDamageMult"} {
		want := rec.Output[k]
		got := env.Player.Output[k]
		if math.Abs(got-want) > 0.005*math.Abs(want) {
			t.Errorf("%s: %s = %v, want %v", name, k, got, want)
		}
	}
}
