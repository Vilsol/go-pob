package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestMaddeningPresenceLowersEnemyDamage pins ModParser.lua:4162-4163 together with their consumers at CalcPerform.lua:928-934.
func TestMaddeningPresenceLowersEnemyDamage(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const (
		affected = "0040-trickster-exsanguinate-of-transmission.xml"
		control  = "0103-elementalist-mana-infused-staff.xml"
	)

	if a, c := oracle[affected], oracle[control]; a.Output["PhysicalEnemyDamageMult"] == c.Output["PhysicalEnemyDamageMult"] {
		t.Fatalf("REFUSING: both fixtures now report PhysicalEnemyDamageMult %v; the affected build "+
			"and the control have stopped differing", a.Output["PhysicalEnemyDamageMult"])
	}

	for _, name := range []string{affected, control} {
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
			"PhysicalEnemyDamageMult",
			"FireEnemyDamageMult",
			"ChaosEnemyDamageMult",
			"PhysicalEnemyDamage",
			"totalEnemyDamage",
			"EnemyCritEffect",
		} {
			want := rec.Output[key]
			got := env.Player.Output[key]
			if math.Abs(got-want) > math.Abs(want)*1e-9 {
				t.Errorf("%s: %s = %v, want %v (PoB)", name, key, got, want)
			}
		}
	}
}
