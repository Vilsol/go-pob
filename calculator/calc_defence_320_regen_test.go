package calculator

import "testing"

const (
	defence320UnchilledBuild = "4243-ladder-BoBricek_5410.BoBricekScion.xml"
	defenceEvasionBuild      = "5650-ladder-ODINTOPPISOS_0104.Ya_Sosal_Menya_Kachumba.xml"
	defenceRegenBuild        = "13938-ladder-katanakid_4789.HERALD_OF_TFT.xml"
)

// TestEnemySkillTimeIsExposed pins CalcDefence.lua:2083-2085.
func TestEnemySkillTimeIsExposed(t *testing.T) {
	env := defenceEnv(t, defence320UnchilledBuild)
	assertMatchesOracle(t, env, defence320UnchilledBuild, "enemySkillTime")
}

// TestMeleeAndProjectileEvasionAreExposed pins CalcDefence.lua:438-439.
func TestMeleeAndProjectileEvasionAreExposed(t *testing.T) {
	env := defenceEnv(t, defenceEvasionBuild)
	assertMatchesOracle(t, env, defenceEvasionBuild, "MeleeEvasion")
	assertMatchesOracle(t, env, defenceEvasionBuild, "ProjectileEvasion")
	assertMatchesOracle(t, env, defenceEvasionBuild, "MeleeEvadeChance")
	assertMatchesOracle(t, env, defenceEvasionBuild, "ProjectileEvadeChance")
}

// TestResourceRegenRecoveryIsExposed pins CalcDefence.lua:628-705 -- the 3.20 loop that publishes six keys for each of Mana / Life / Energy Shield / Rage.
func TestResourceRegenRecoveryIsExposed(t *testing.T) {
	env := defenceEnv(t, defenceRegenBuild)
	for _, resource := range []string{"Mana", "Life", "EnergyShield", "Rage"} {
		assertMatchesOracle(t, env, defenceRegenBuild, resource+"RegenRecovery")
		assertMatchesOracle(t, env, defenceRegenBuild, resource+"Regen")
		assertMatchesOracle(t, env, defenceRegenBuild, resource+"RegenInc")
		assertMatchesOracle(t, env, defenceRegenBuild, resource+"Degen")
		assertMatchesOracle(t, env, defenceRegenBuild, resource+"Recovery")
	}
	for _, resource := range []string{"Mana", "Life", "EnergyShield"} {
		assertMatchesOracle(t, env, defenceRegenBuild, resource+"RegenPercent")
	}
}
