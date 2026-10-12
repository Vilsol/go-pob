package calculator

import (
	"math"
	"os"
	"testing"
)

// TestTotemDurationOutputs pins CalcOffence.lua:1590-1593.
func TestTotemDurationOutputs(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build    string
		mod      float64
		duration float64
	}{
		{"0001-necromancer-unknown.xml", 1, 0},
		{"0045-hierophant-spark-of-the-nova.xml", 2.5, 20.031},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)

			got, ok := env.Player.Output["TotemDurationMod"]
			if !ok {
				t.Fatal("TotemDurationMod ABSENT; PoB publishes it on every corpus build")
			}
			if math.Abs(got-tc.mod) > 0.0005 {
				t.Errorf("TotemDurationMod = %v, want %v (from the 3.29 oracle blob)", got, tc.mod)
			}

			dur, ok := env.Player.Output["TotemDuration"]
			if !ok {
				t.Fatal("TotemDuration ABSENT; PoB publishes it on every corpus build")
			}
			if math.Abs(dur-tc.duration) > 0.0005 {
				t.Errorf("TotemDuration = %v, want %v (from the 3.29 oracle blob)", dur, tc.duration)
			}
		})
	}
}
