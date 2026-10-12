package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestKineticFusillade covers act_int.lua:11009-11045 — the sequential-projectile scaling on Kinetic Fusillade and its two transfigured variants.
func TestKineticFusillade(t *testing.T) {
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
		keys  []string
	}{
		{"kinetic-fusillade", "0185-hierophant-kinetic-fusillade.xml",
			[]string{"KineticFusilladeAvgMoreMult"}},
		{"other-skill-control", "0077-pathfinder-toxic-rain-2.xml",
			[]string{"AverageDamage"}},
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

			if tc.name == "kinetic-fusillade" {
				wantAvg := rec.Output["AverageDamage"]
				if wantAvg == 0 {
					t.Fatalf("REFUSING: %s publishes no AverageDamage in the oracle blob; the "+
						"fixture has stopped exercising the damage half", tc.build)
				}
				if got := env.Player.Output["AverageDamage"]; math.Abs(got-wantAvg) > wantAvg*1e-9 {
					t.Errorf("%s: AverageDamage = %v, want %v (PoB)", tc.build, got, wantAvg)
				}
			}

			for _, key := range tc.keys {
				want := rec.Output[key]
				if want == 0 {
					t.Fatalf("REFUSING: %s publishes no %s; the fixture has stopped exercising it",
						tc.build, key)
				}
				got := env.Player.Output[key]
				if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
					t.Errorf("%s: %s = %v, want %v (PoB)", tc.build, key, got, want)
				}
			}
		})
	}
}
