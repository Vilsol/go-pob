package calculator

import "testing"

// TestEnemySkillPhysicalToChaosConversion pins CalcDefence.lua:1793-1799.
func TestEnemySkillPhysicalToChaosConversion(t *testing.T) {
	assertPlayerOutputs(t, "1399-elementalist-eviscerate-10.xml",
		map[string]float64{"ChaosEnemyDamage": 10755.472874999999, "totalEnemyDamage": 107554.72874999999})
}
