package calculator

import "testing"

const defence320UnchilledBuild = "0011-champion-cyclone.xml"

// TestEnemySkillTimeIsExposed pins CalcDefence.lua:2083-2085.
func TestEnemySkillTimeIsExposed(t *testing.T) {
	env := defence320Env(t, defence320UnchilledBuild)
	assertMatches320Oracle(t, env, defence320UnchilledBuild, "enemySkillTime")
}

// TestMeleeAndProjectileEvasionAreExposed pins CalcDefence.lua:438-439.
func TestMeleeAndProjectileEvasionAreExposed(t *testing.T) {
	env := defence320Env(t, defence320Build)
	assertMatches320Oracle(t, env, defence320Build, "MeleeEvasion")
	assertMatches320Oracle(t, env, defence320Build, "ProjectileEvasion")
	assertMatches320Oracle(t, env, defence320Build, "MeleeEvadeChance")
	assertMatches320Oracle(t, env, defence320Build, "ProjectileEvadeChance")
}

// TestResourceRegenRecoveryIsExposed pins CalcDefence.lua:628-705 -- the 3.20 loop that publishes six keys for each of Mana / Life / Energy Shield / Rage.
func TestResourceRegenRecoveryIsExposed(t *testing.T) {
	env := defence320Env(t, defence320Build)
	for _, resource := range []string{"Mana", "Life", "EnergyShield", "Rage"} {
		assertMatches320Oracle(t, env, defence320Build, resource+"RegenRecovery")
		assertMatches320Oracle(t, env, defence320Build, resource+"Regen")
		assertMatches320Oracle(t, env, defence320Build, resource+"RegenInc")
		assertMatches320Oracle(t, env, defence320Build, resource+"Degen")
		assertMatches320Oracle(t, env, defence320Build, resource+"Recovery")
	}
	for _, resource := range []string{"Mana", "Life", "EnergyShield"} {
		assertMatches320Oracle(t, env, defence320Build, resource+"RegenPercent")
	}
}
