package calculator

import (
	"math"
	"os"
	"testing"
)

// TestIgniteDPSUsesTheComputedStacks pins CalcOffence.lua:5090.
//
//	local activeIgnites = m_min(igniteStacks, maxStacks)
//	local IgniteDPSUncapped = baseVal * effectMod * rateMod * activeIgnites * effMult
func TestIgniteDPSUsesTheComputedStacks(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, tc := range []struct {
		build string
		arm   string
	}{
		{"1555-luminary-tempest-shield.xml", "the overwrite actually binds here"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			rec, ok := oracle[tc.build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", tc.build)
			}
			want, ok := rec.Output["IgniteDPS"]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero IgniteDPS for %s", tc.build)
			}

			env := corpusEnvForLoadedData(t, tc.build)

			if wantEff, ok := rec.Output["IgniteEffMult"]; ok && wantEff != 0 {
				gotEff := env.Player.Output["IgniteEffMult"]
				if math.Abs(gotEff-wantEff) > math.Abs(wantEff)*oracleDumpTolerance {
					t.Fatalf("REFUSING: %s is contaminated — IgniteEffMult is %v against PoB's %v, "+
						"so IgniteDPS cannot be attributed to the stacks overwrite",
						tc.build, gotEff, wantEff)
				}
			}

			got := env.Player.Output["IgniteDPS"]
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s [%s]: IgniteDPS = %v, PoB reports %v (ratio %.4f)",
					tc.build, tc.arm, got, want, got/want)
			}
		})
	}
}
