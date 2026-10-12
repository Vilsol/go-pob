package calculator

import (
	"math"
	"os"
	"testing"
)

// TestCausticGroundFromPoisonHasAProducer pins CalcOffence.lua:4750-4755.
func TestCausticGroundFromPoisonHasAProducer(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		mh    float64
		oh    float64
		total float64
	}{
		{"0050-occultist-viper-strike-of-the-mamba.xml", 3977697.1, 4201548.7, 8179245.8},
		{"0469-pathfinder-viper-strike-of-the-mamba.xml", 4623999.2, 6211674.5, 10618960},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			for _, k := range []struct {
				key  string
				got  float64
				want float64
			}{
				{"MainHand.CausticGroundDPS", env.Player.OutputTable[OutTableMainHand]["CausticGroundDPS"], tc.mh},
				{"OffHand.CausticGroundDPS", env.Player.OutputTable[OutTableOffHand]["CausticGroundDPS"], tc.oh},
				{"CausticGroundDPS", env.Player.Output["CausticGroundDPS"], tc.total},
			} {
				if math.Abs(k.got-k.want) > math.Abs(k.want)*0.005 {
					t.Errorf("%s = %v, want %v (oracle)", k.key, k.got, k.want)
				}
			}
		})
	}
}
