package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestSummonedMinionMultiplier covers CalcPerform.lua:1401-1425 and :1456-1466 — the block that counts active minions and publishes Multiplier:SummonedMinion.
func TestSummonedMinionMultiplier(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	for _, tc := range []struct {
		name  string
		build string
	}{
		{"minion-build", "0168-necromancer-animate-weapon-of-ranged-arms-3.xml"},
		{"no-minions-control", "0231-berserker-flicker-strike.xml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec, ok := oracle[tc.build]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", tc.build)
			}
			file, err := os.ReadFile(filepath.Join(corpusDir, tc.build))
			if err != nil {
				t.Fatalf("%v", err)
			}
			build, err := builds.ParseBuild(file)
			if err != nil {
				t.Fatalf("%v", err)
			}
			env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

			for _, key := range []string{
				"BasePhysicalDamageReduction",
				"BasePhysicalDamageReductionWhenHit",
			} {
				want := rec.Output[key]
				got := env.Player.Output[key]
				if math.Abs(got-want) > 1e-9 {
					t.Errorf("%s: %s = %v, want %v (PoB)", tc.build, key, got, want)
				}
			}
		})
	}
}
