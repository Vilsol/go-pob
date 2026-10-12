package calculator

import (
	"math"
	"os"
	"testing"
)

// TestKalandrasTouchMirrorsRingOne pins an ORDER dependency, not a missing feature.
func TestKalandrasTouchMirrorsRingOne(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		stat string
		want float64
	}{
		{"Str", 3386},
		{"Dex", 348},
		{"Int", 523},
	} {
		t.Run(tc.stat, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, "2044-ladder-shing829_1070.hsisbbq.xml")
			got := env.Player.Output[tc.stat]
			if math.Abs(got-tc.want) > math.Abs(tc.want)*0.005 {
				t.Errorf("%s = %v, want %v (PoB) — Kalandra's Touch in Ring 1 mirrors Ring 2", tc.stat, got, tc.want)
			}
		})
	}

	t.Run("CONTROL: a build without the ring must not move", func(t *testing.T) {
		env := corpusEnvForLoadedData(t, "0004-trickster-lightning-tendrils-of-eccentricity.xml")
		if got := env.Player.Output["Life"]; math.Abs(got-3666) > 3666*0.005 {
			t.Errorf("CONTROL Life = %v, want 3666 (ours, unchanged) — the two-pass split has "+
				"disturbed a slot-chain arm that is not Kalandra's Touch", got)
		}
	})
}
