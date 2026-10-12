package calculator

import (
	"os"
	"testing"
)

// TestChargeDurationOutputs pins CalcPerform.lua:954/961/968/976 and CalcSetup.lua:30.
func TestChargeDurationOutputs(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	type want struct {
		power, frenzy, endurance, brine float64
	}

	for _, tc := range []struct {
		build  string
		player want
	}{
		{
			build:  "0001-necromancer-unknown.xml",
			player: want{power: 10, frenzy: 10, endurance: 10, brine: 3},
		},
		{
			build:  "0068-reliquarian-poisonous-concoction-of-bouncing.xml",
			player: want{power: 20, frenzy: 22, endurance: 20, brine: 3},
		},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)

			for _, c := range []struct {
				name string
				want float64
			}{
				{"PowerChargesDuration", tc.player.power},
				{"FrenzyChargesDuration", tc.player.frenzy},
				{"EnduranceChargesDuration", tc.player.endurance},
				{"BrineChargesMax", tc.player.brine},
			} {
				got, ok := env.Player.Output[c.name]
				if !ok {
					t.Errorf("player %s ABSENT; PoB publishes it on every corpus build", c.name)
					continue
				}
				if got != c.want {
					t.Errorf("player %s = %v, want %v (from the 3.29 oracle blob)", c.name, got, c.want)
				}
			}

			for _, c := range []struct {
				name string
				want float64
			}{
				{"PowerChargesDuration", 10},
				{"FrenzyChargesDuration", 10},
				{"EnduranceChargesDuration", 10},
				{"BrineChargesMax", 3},
			} {
				got, ok := env.Player.Enemy.Output[c.name]
				if !ok {
					t.Errorf("enemy %s ABSENT; PoB publishes it on every corpus build", c.name)
					continue
				}
				if got != c.want {
					t.Errorf("enemy %s = %v, want %v (from the 3.29 oracle blob)", c.name, got, c.want)
				}
			}
		})
	}
}
