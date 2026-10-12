package calculator

import (
	"math"
	"os"
	"testing"
)

// TestBurningGroundDPSIsPublished pins the two sites that write it.
//
//	m_max(skillModList:Max "IgniteDpsAsBurningGround",
func TestBurningGroundDPSIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range []string{
		"0100-elementalist-penance-brand-6.xml",
		"0169-elementalist-hexblast-of-contradiction.xml",
		"0226-elementalist-vaal-arc.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			want, ok := rec.Output["BurningGroundDPS"]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero BurningGroundDPS for %s", build)
			}

			env := corpusEnvForLoadedData(t, build)

			if wantIgnite, ok := rec.Output["IgniteDPS"]; ok && wantIgnite != 0 {
				gotIgnite := env.Player.Output["IgniteDPS"]
				if math.Abs(gotIgnite-wantIgnite) > math.Abs(wantIgnite)*oracleDumpTolerance {
					t.Fatalf("REFUSING: %s is contaminated — IgniteDPS is %v against PoB's %v, and "+
						"it shares baseVal/effectMod/rateMod with this key",
						build, gotIgnite, wantIgnite)
				}
			}

			got, present := env.Player.Output["BurningGroundDPS"]
			if !present {
				t.Errorf("%s: we never write BurningGroundDPS at all; PoB reports %v", build, want)
				return
			}
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: BurningGroundDPS = %v, PoB reports %v (ratio %.4f)",
					build, got, want, got/want)
			}
		})
	}
}
