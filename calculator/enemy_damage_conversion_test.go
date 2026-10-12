package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestEnemyDamageConversion pins CalcDefence.lua:1721-1777.
func TestEnemyDamageConversion(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "0103-elementalist-mana-infused-staff.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}

	if rec.Output["PhysicalEnemyDamage"] == rec.Output["FireEnemyDamage"] {
		t.Fatalf("REFUSING: %s no longer splits PhysicalEnemyDamage from FireEnemyDamage in the "+
			"oracle (both %v); the fixture has stopped exercising enemy conversion",
			name, rec.Output["PhysicalEnemyDamage"])
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
		"PhysicalEnemyDamage",
		"FireEnemyDamage",
		"ColdEnemyDamage",
		"LightningEnemyDamage",
		"totalEnemyDamageIn",
	} {
		want := rec.Output[key]
		got := env.Player.Output[key]
		if math.Abs(got-want) > math.Abs(want)*1e-9 {
			t.Errorf("%s: %s = %v, want %v (PoB)", name, key, got, want)
		}
	}
}
