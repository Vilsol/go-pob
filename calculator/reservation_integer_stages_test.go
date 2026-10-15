package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/utils"
)

// TestReservationIntegerStages pins CalcPerform.lua:2016-2036.
func TestReservationIntegerStages(t *testing.T) {
	requireGameData(t, "3.29")
	cases := []struct {
		build, skill, pool string
		percent            float64
	}{
		{"0021-assassin-blade-flurry.xml", "Blood and Sand", "Life", 20.09},
		{"14379-ladder-oottesuklaata_0678.silkki_mimosa.xml", "Malevolence", "Mana", 53.66},
		{"13267-ladder-crazystyle11_3482.LillyBadTripAF.xml", "Determination", "Mana", 0},
	}
	for _, c := range cases {
		t.Run(c.build+"/"+c.skill, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, c.build)
			var found bool
			for _, as := range env.Player.ActiveSkillList {
				if as.ActiveEffect.GrantedEffect.Name() != c.skill {
					continue
				}
				found = true
				rn := reservationNames[c.pool]
				if got := utils.GetOr(as.SkillData, rn.reservedPercent, float64(0)); got != c.percent {
					t.Errorf("%s %s = %v, want %v", c.skill, rn.reservedPercent, got, c.percent)
				}
				if got := utils.GetOr(as.SkillData, rn.reservedBase, float64(0)); got != got {
					t.Errorf("%s %s is NaN", c.skill, rn.reservedBase)
				}
			}
			if !found {
				t.Fatalf("REFUSING: %s has no %s", c.build, c.skill)
			}
		})
	}
}
