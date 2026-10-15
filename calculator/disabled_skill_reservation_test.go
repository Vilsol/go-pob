package calculator

import "testing"

// TestDisabledSkillsReserveNothing pins CalcPerform.lua:1979 and CalcActiveSkill.lua:278-282.
func TestDisabledSkillsReserveNothing(t *testing.T) {
	assertPlayerOutputs(t, "12226-ladder-TG1337_3331.eqwesstttttt.xml", map[string]float64{"ManaReserved": 666})
	assertPlayerOutputs(t, "10001-ladder-Flynbull05_6233.EAZEY_SCUBA.xml", map[string]float64{"ManaReserved": 729})
}
