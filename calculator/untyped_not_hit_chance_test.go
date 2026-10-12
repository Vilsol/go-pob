package calculator

import (
	"math"
	"os"
	"testing"
)

// TestUntypedNotHitChance pins CalcDefence.lua:1652, the one member of the NotHitChance family this port never wrote.
func TestUntypedNotHitChance(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0021-assassin-blade-flurry.xml")
	out := env.Player.Output

	for _, tc := range []struct {
		key  string
		want float64
		role string
	}{
		{"UntypedNotHitChance", 25, "the subject: 100 - (1 - AvoidAllDamageFromHitsChance/100) * 100"},
		{"AvoidAllDamageFromHitsChance", 25, "control: the subject's only input, unchanged"},
	} {
		if got := out[tc.key]; math.Abs(got-tc.want) > 0.005 {
			t.Errorf("%s = %g, want %g (%s)", tc.key, got, tc.want, tc.role)
		}
	}
	if _, present := out["UntypedNotHitChance"]; !present {
		t.Error("UntypedNotHitChance was never written; an absent key and a zero are the same value here")
	}
}
