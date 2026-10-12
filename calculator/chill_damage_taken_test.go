package calculator

import (
	"math"
	"os"
	"testing"
)

// TestChillEffectIncreasesDamageTaken pins CalcPerform.lua:3436-3450.
func TestChillEffectIncreasesDamageTaken(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		stat  string
		want  float64
		why   string
	}{
		{"0075-elementalist-winter-orb.xml", "TotalDPS", 90033.4,
			"Cold Mastery 'Enemies Chilled by your Hits have Cold Damage taken increased by Chill Effect'"},

		{"0004-trickster-lightning-tendrils-of-eccentricity.xml", "Life", 3666,
			"CONTROL: no chill-effect flag; must not move"},
	} {
		t.Run(tc.build+"/"+tc.stat, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output[tc.stat]
			if !ok {
				t.Fatalf("%s ABSENT", tc.stat)
			}
			if math.Abs(got-tc.want) > math.Abs(tc.want)*0.005 {
				t.Errorf("%s = %v, want %v (PoB) — %s", tc.stat, got, tc.want, tc.why)
			}
		})
	}
}
