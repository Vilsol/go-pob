package calculator

import (
	"math"
	"os"
	"testing"
)

// TestEffectiveDodgeChanceIsPublished pins CalcDefence.lua:1169-1176.
func TestEffectiveDodgeChanceIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	cases := map[string]map[string]float64{
		"6371-ladder-Toribey_5636.Dokkism_badbuild":       {"SpellDodgeChance": 5, "EffectiveSpellDodgeChance": 5, "EffectiveAttackDodgeChance": 0},
		"2584-ladder-StartledGamer_7343.CurseOfTheLoginQ": {"SpellDodgeChance": 9.5, "EffectiveSpellDodgeChance": 9.5, "EffectiveAttackDodgeChance": 0},
		"6012-ladder-Scccxd_6309.Ya_Doldayob":             {"SpellDodgeChance": 71, "EffectiveSpellDodgeChance": 71, "EffectiveAttackDodgeChance": 0},
		"0006-elementalist-storm-burst":                   {"SpellDodgeChance": 0, "EffectiveSpellDodgeChance": 0, "EffectiveAttackDodgeChance": 0},
	}

	for build, want := range cases {
		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			for key, w := range want {
				got, ok := env.Player.Output[key]
				if !ok {
					t.Errorf("%s is not published at all", key)
					continue
				}
				if math.Abs(got-w) > 1e-6 {
					t.Errorf("%s = %v, want %v", key, got, w)
				}
			}
		})
	}
}
