package calculator

import (
	"os"
	"testing"
)

// TestResistBlockMatchesPoB pins CalcDefence.lua:578-619, the resist block.
func TestResistBlockMatchesPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		want  map[string]float64
	}{
		{
			build: "0045-hierophant-spark-of-the-nova.xml",
			want: map[string]float64{
				"ChaosResist": 15, "ChaosResistOverTime": 55,
			},
		},
		{
			build: "0052-juggernaut-cleave.xml",
			want: map[string]float64{
				"FireResistTotal": 72, "FireResist": 72, "MissingFireResist": 8,
			},
		},
		{
			build: "0001-necromancer-unknown.xml",
			want: map[string]float64{
				"FireResistTotal": 156,
			},
		},
		{
			build: "0045-hierophant-spark-of-the-nova.xml",
			want: map[string]float64{
				"MissingChaosResist": 60,
			},
		},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			for _, k := range sortedKeys(tc.want) {
				got, ok := env.Player.Output[k]
				if !ok {
					t.Errorf("%s ABSENT; PoB publishes it on every corpus build", k)
					continue
				}
				if got != tc.want[k] {
					t.Errorf("%s = %v, want %v (from the 3.29 oracle blob)", k, got, tc.want[k])
				}
			}
		})
	}
}
