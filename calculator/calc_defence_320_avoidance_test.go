package calculator

import (
	"testing"
)

const avoidance320Build = "0149-warden-frost-blades-2.xml"

func TestAvoidAllDamageFromHitsChanceIsExposed(t *testing.T) {
	env := defenceEnv(t, avoidance320Build)
	assertMatchesOracle(t, env, avoidance320Build, "AvoidAllDamageFromHitsChance")
}

func TestNotHitChanceIncludesAvoidAllDamageFromHits(t *testing.T) {
	env := defenceEnv(t, avoidance320Build)
	for _, category := range []string{"Melee", "Projectile", "Spell", "SpellProjectile", "Average"} {
		assertMatchesOracle(t, env, avoidance320Build, category+"NotHitChance")
	}
}

const (
	avoidanceProjectileOnlyBuild = "0101-champion-lancing-steel.xml"
	avoidanceSpecificTypeBuild   = "1549-luminary-animate-guardian-of-smiting-5.xml"
)

func TestProjectileAvoidanceEntersNotHitChanceWithoutSpecificTypeAvoidance(t *testing.T) {
	env := defenceEnv(t, avoidanceProjectileOnlyBuild)
	for _, stat := range []string{"ProjectileNotHitChance", "SpellProjectileNotHitChance", "AverageNotHitChance", "ConfiguredNotHitChance", "ConfiguredDamageChance"} {
		assertMatchesOracle(t, env, avoidanceProjectileOnlyBuild, stat)
	}
}

func TestSpecificTypeAvoidanceSuppressesTheProjectileNotHitFactor(t *testing.T) {
	env := defenceEnv(t, avoidanceSpecificTypeBuild)
	for _, stat := range []string{"ProjectileNotHitChance", "SpellProjectileNotHitChance", "AverageNotHitChance", "ConfiguredDamageChance"} {
		assertMatchesOracle(t, env, avoidanceSpecificTypeBuild, stat)
	}
}
