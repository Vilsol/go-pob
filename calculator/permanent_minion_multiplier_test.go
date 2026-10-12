package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestPermanentMinionMultiplier: CalcPerform.lua:1419-1421 accumulates counts.permanent and :1463-1465 emits it as Multiplier:PermanentMinion.
func TestPermanentMinionMultiplier(t *testing.T) {
	for _, tc := range []struct {
		build          string
		permanent      float64
		nonVaalControl float64
	}{
		{"0243-necromancer-raise-spectre", 8, 8},
		{"0008-necromancer-vaal-summon-skeletons", 14, 25},
	} {
		t.Run(tc.build, func(t *testing.T) {
			xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), tc.build+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := builds.ParseBuild(xml)
			if err != nil {
				t.Fatal(err)
			}
			env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)

			if got := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:NonVaalSummonedMinion"); got != tc.nonVaalControl {
				t.Fatalf("CONTROL Multiplier:NonVaalSummonedMinion = %g, PoB %g — the loop or the "+
					"Condition:Combat tag is broken, so the PermanentMinion result below means nothing",
					got, tc.nonVaalControl)
			}
			if got := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:PermanentMinion"); got != tc.permanent {
				t.Errorf("Multiplier:PermanentMinion = %g, PoB %g", got, tc.permanent)
			}
		})
	}
}
