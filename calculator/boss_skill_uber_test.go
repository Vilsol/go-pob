package calculator

import (
	"math"
	"testing"
)

// TestBossSkillUberDamageMultiplier: an Uber boss preset scales its damage placeholder by UberDamageMultiplier (ConfigOptions.lua:2249-2266).
func TestBossSkillUberDamageMultiplier(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1399-elementalist-eviscerate-10.xml")
	if got := env.Player.Output["totalEnemyDamageIn"]; got != 49866 {
		t.Errorf("totalEnemyDamageIn = %v, want 49866 (PoB)", got)
	}
	if got := env.Player.Output["totalEnemyDamage"]; math.Abs(got-107554.73) > 0.1 {
		t.Errorf("totalEnemyDamage = %v, want 107554.73 (PoB)", got)
	}
}
