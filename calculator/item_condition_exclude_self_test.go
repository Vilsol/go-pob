package calculator

import (
	"math"
	"os"
	"testing"
)

// TestItemConditionExcludeSelfDropsTheTagsOwnSlot pins ModStore.lua:673.
//
//	if (not tag.allSlots or tag.allSlots and (item.type ~= "Jewel" and item.type ~= "Graft"))
//	   and slot ~= itemSlot or not tag.excludeSelf then
func TestItemConditionExcludeSelfDropsTheTagsOwnSlot(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		life  float64
	}{
		{"1028-assassin-tornado-of-elemental-turbulence.xml", 2777},
		{"11594-ladder-RaP1D_0286.RapidAllflameFOUR.xml", 2935},
		{"11952-ladder-ShinX__4985.HCDipppppppppie.xml", 4620},
		{"14211-ladder-metalheaddragonlord_3566.AllFlameWoody.xml", 7098},
		{"14801-ladder-starter7874_2530.Allflame_cs.xml", 2927},
		{"2587-ladder-TT515842056_1727.Curse_Flooodbeast.xml", 4983},
		{"2902-ladder-Clad777_5174.Double_Corrupting_Rings.xml", 6130},
		{"5466-ladder-MickPlaYer_3911.IcecreamAssassin.xml", 2677},
		{"6993-ladder-chukadeaz_4449.COTA_chukadea.xml", 3507},
		{"8843-ladder-Agion_POE_7046.Sea_Templar.xml", 6530},
		{"9322-ladder-Cavalorth_5118.Cavaaa.xml", 2866},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got := env.Player.Output["Life"]
			if math.Abs(got-tc.life) > math.Abs(tc.life)*0.005 {
				t.Errorf("Life = %v, want %v (oracle)", got, tc.life)
			}
		})
	}
}
