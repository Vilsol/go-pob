package calculator

import "testing"

// TestRepeatModifiersNeedARepeat pins CalcOffence.lua:783-869: repeat-mode modifiers.
func TestRepeatModifiersNeedARepeat(t *testing.T) {
	for _, name := range []string{"6030-ladder-Sedu18_2697.AllFlame_Ogho.xml", "9974-ladder-Finnfal_5019.FinnDB.xml"} {
		t.Run(name, func(t *testing.T) {
			assertBuildStats(t, name, []string{"AreaOfEffectMod", "AreaOfEffectRadius"}, nil)
		})
	}
}
