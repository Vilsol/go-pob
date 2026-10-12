package calculator

import (
	"math"
	"os"
	"testing"
)

// TestGenericReflectedDamageTakenIsRead pins CalcDefence.lua:1906-1907, which sums BOTH "ReflectedDamageTaken" and "<Type>ReflectedDamageTaken".
func TestGenericReflectedDamageTakenIsRead(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	cases := []struct {
		build string
		want  map[string]float64
	}{
		{
			build: "1678-ladder-Sergohero_2699.SergoheroGaz",
			want:  map[string]float64{"PhysicalTakenReflect": 0, "FireTakenReflect": 0, "ChaosTakenReflect": 0},
		},
		{
			build: "1706-ladder-Akzzhol_1275.KineticBlyadina",
			want:  map[string]float64{"PhysicalTakenReflect": 0, "FireTakenReflect": 0, "ChaosTakenReflect": 0},
		},
		{
			build: "1042-occultist-ice-nova-of-frostbolts",
			want:  map[string]float64{"PhysicalTakenReflect": 1, "FireTakenReflect": 1},
		},
		{
			build: "8098-ladder-rozaywastaken_6515.RozKBOC",
			want:  map[string]float64{"PhysicalTakenReflect": 0.73, "FireTakenReflect": 0.73},
		},
	}

	for _, c := range cases {
		t.Run(c.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, c.build)
			for key, want := range c.want {
				got := env.Player.Output[key]
				if math.Abs(got-want) > 1e-6 {
					t.Errorf("%s = %v, want %v", key, got, want)
				}
			}
		})
	}
}
