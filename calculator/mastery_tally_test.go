package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestMasteryTallyCountsRunegraftsSeparately pins PassiveSpec.lua:1393-1420's mastery tally against the pinned PoB's own answer for three corpus builds.
func TestMasteryTallyCountsRunegraftsSeparately(t *testing.T) {
	requireGameData(t, "3.29")
	for _, tc := range []struct {
		build     string
		mastery   float64
		types     float64
		lifeCount float64
	}{
		{"0007-elementalist-storm-burst-of-repulsion", 10, 9, 0},
		{"0029-inquisitor-spark-of-the-nova", 10, 7, 1},
		{"0040-trickster-exsanguinate-of-transmission", 7, 6, 2},
	} {
		t.Run(tc.build, func(t *testing.T) {
			file, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), tc.build+".xml"))
			if err != nil {
				t.Fatalf("read build: %v", err)
			}
			build, err := builds.ParseBuild(file)
			if err != nil {
				t.Fatalf("parse build: %v", err)
			}
			env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
			got := func(name string) float64 { return env.ModDB.Sum(mod.TypeBase, nil, name) }
			if v := got("Multiplier:AllocatedMastery"); v != tc.mastery {
				t.Errorf("Multiplier:AllocatedMastery = %v, pin says %v", v, tc.mastery)
			}
			if v := got("Multiplier:AllocatedMasteryType"); v != tc.types {
				t.Errorf("Multiplier:AllocatedMasteryType = %v, pin says %v", v, tc.types)
			}
			if v := got("Multiplier:AllocatedLifeMastery"); v != tc.lifeCount {
				t.Errorf("Multiplier:AllocatedLifeMastery = %v, pin says %v", v, tc.lifeCount)
			}
		})
	}
}
