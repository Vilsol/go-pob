package calculator

import "testing"

// TestItemGrantedTriggerChance pins ModParser.lua:2062, :3333.
func TestItemGrantedTriggerChance(t *testing.T) {
	assertBuildStats(t, "1460-necromancer-flesh-offering.xml", []string{"SkillTriggerRate", "TriggerTime"}, nil)
}
