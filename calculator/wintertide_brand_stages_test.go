package calculator

import (
	"math"
	"os"
	"testing"
)

// TestWintertideBrandStagesReachColdDot pins act_int.lua:21110-21129.
//
//pob:block:src/Data/Skills/act_int.lua:21110-21129
func TestWintertideBrandStagesReachColdDot(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	builds := []string{
		"0063-occultist-wintertide-brand.xml",
		"0958-elementalist-wintertide-brand-10.xml",
		"0760-elementalist-wintertide-brand-9.xml",
	}

	graded := 0
	for _, build := range builds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		for _, key := range []string{"ColdDot", "TotalDotInstance"} {
			want, ok := rec.Output[key]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero %s for %s — a 0-vs-0 fixture scores "+
					"ok and cannot see the branch being deleted", key, build)
			}
		}
		graded++

		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			for _, key := range []string{"ColdDot", "TotalDotInstance"} {
				want := rec.Output[key]
				got := env.Player.Output[key]
				if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
					t.Errorf("%s: %s = %v, PoB reports %v (ratio %.5f)", build, key, got, want, got/want)
				}
			}
		})
	}

	if graded != len(builds) {
		t.Fatalf("REFUSING: graded %d of %d fixtures", graded, len(builds))
	}
}
