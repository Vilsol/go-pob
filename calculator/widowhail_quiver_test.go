package calculator

import (
	"math"
	"os"
	"testing"
)

// TestWidowhailScalesQuiverBonuses pins CalcSetup.lua:1126-1134.
func TestWidowhailScalesQuiverBonuses(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		stat  string
		want  float64
		tol   float64
		why   string
	}{
		{"0043-pathfinder-toxic-rain.xml", "Life", 5004, 0,
			"Widowhail build; the quiver's Life bonus is multiplied"},
		{"0220-pathfinder-toxic-rain-5.xml", "Life", 6922, 0,
			"second Widowhail build"},

		{"0004-trickster-lightning-tendrils-of-eccentricity.xml", "Life", 3666, 0,
			"CONTROL: no quiver; must not move"},
	} {
		t.Run(tc.build+"/"+tc.stat, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output[tc.stat]
			if !ok {
				t.Fatalf("%s ABSENT", tc.stat)
			}
			if math.Abs(got-tc.want) > tc.tol {
				t.Errorf("%s = %v, want %v (PoB) — %s", tc.stat, got, tc.want, tc.why)
			}
		})
	}
}
