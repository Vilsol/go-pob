package calculator

import (
	"os"
	"testing"
)

// TestKineticFusilladePostCritCapsTheRate covers act_int.lua:11046-11123.
func TestKineticFusilladePostCritCapsTheRate(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build   string
		wantAPS float64
	}{
		{"0462-hierophant-kinetic-fusillade-4", 2.7548209366391},
		{"0422-hierophant-kinetic-fusillade-3", 3.7878787878788},
		{"0185-hierophant-kinetic-fusillade", 3.030303030303},
	} {
		env := corpusEnvForLoadedData(t, tc.build)
		got, ok := env.Player.Output["KineticFusilladeMaxEffectiveAPS"]
		if !ok {
			t.Errorf("%s: KineticFusilladeMaxEffectiveAPS absent; the pinned PoB publishes %v",
				tc.build, tc.wantAPS)
			continue
		}
		if diff := got - tc.wantAPS; diff > 1e-9 || diff < -1e-9 {
			t.Errorf("%s: KineticFusilladeMaxEffectiveAPS = %v, want %v (the pinned PoB's value)",
				tc.build, got, tc.wantAPS)
		}
	}
}
