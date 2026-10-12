package calculator

import "testing"

// TestFlaskHaveBaseCondition: every equipped flask sets Condition:Have<BaseName> (CalcSetup.lua:927), which Master Distiller's HaveDiamondFlask crit reads.
func TestFlaskHaveBaseCondition(t *testing.T) {
	assertPlayerOutputs(t, "1880-ladder-Seaborgium_0_4103.VODOLAZ_KB.xml", map[string]float64{"CritChance": 43.38})
}
