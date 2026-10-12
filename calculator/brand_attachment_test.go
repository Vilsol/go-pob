package calculator

import (
	"math"
	"os"
	"testing"
)

// TestAttachedBrandCountIsPublished pins CalcOffence.lua:1428.
func TestAttachedBrandCountIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	distinct := map[float64]bool{}
	for _, build := range []string{
		"0063-occultist-wintertide-brand.xml",
		"0078-elementalist-penance-brand.xml",
		"0090-elementalist-penance-brand-3.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			want, ok := rec.Output["AttachedBrandCount"]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero AttachedBrandCount for %s; an "+
					"unwritten key agrees with 0, so this fixture cannot fail", build)
			}
			distinct[want] = true

			env := corpusEnvForLoadedData(t, build)
			got, present := env.Player.Output["AttachedBrandCount"]
			if !present {
				t.Errorf("%s: we never write AttachedBrandCount at all; PoB reports %v", build, want)
				return
			}
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: AttachedBrandCount = %v, PoB reports %v", build, got, want)
			}
		})
	}

	if len(distinct) < 2 {
		t.Errorf("REFUSING: the fixtures no longer carry distinct expected values (%v); this "+
			"test can no longer distinguish publishing the limit from publishing the right value",
			distinct)
	}
}
