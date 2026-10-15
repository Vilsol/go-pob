package calculator

import "testing"

// TestConsecratedGroundHalvesCursesOnSelf pins CalcPerform.lua:3145-3148 (upstream #10177): the -50% curse effect on self lands before the curse loop reads it.
func TestConsecratedGroundHalvesCursesOnSelf(t *testing.T) {
	assertPlayerOutputs(t, "0607-luminary-raise-spectre-4.xml", map[string]float64{"ActionSpeedMod": 0.96, "Speed": 1.888})
}
