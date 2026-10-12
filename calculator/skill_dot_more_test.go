package calculator

import "testing"

// TestSkillDotMoreIsNotRoundedTwice pins CalcOffence.lua:5720-5723: the skill dot's MORE product is used as ModList:More returns it, without a second rounding.
func TestSkillDotMoreIsNotRoundedTwice(t *testing.T) {
	for _, name := range []string{"13779-ladder-iceemanzero_2877.DeezNutsEnyaMouth.xml", "9455-ladder-Cryomatrix5_6949.Selma_Warpriest.xml"} {
		t.Run(name, func(t *testing.T) {
			assertBuildStats(t, name, []string{"ChaosDot", "TotalDotDPS"}, nil)
		})
	}
}
