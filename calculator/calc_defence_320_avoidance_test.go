package calculator

import (
	"testing"
)

const avoidance320Build = "1268-assassin-cyclone-2.xml"

func TestAvoidAllDamageFromHitsChanceIsExposed(t *testing.T) {
	env := defence320Env(t, avoidance320Build)
	assertMatches320Oracle(t, env, avoidance320Build, "AvoidAllDamageFromHitsChance")
}

func TestNotHitChanceIncludesAvoidAllDamageFromHits(t *testing.T) {
	env := defence320Env(t, avoidance320Build)
	for _, category := range []string{"Melee", "Projectile", "Spell", "SpellProjectile", "Average"} {
		assertMatches320Oracle(t, env, avoidance320Build, category+"NotHitChance")
	}
}

const (
	avoidanceProjectileOnlyBuild = "0005-trickster-vaal-flicker-strike.xml"
	avoidanceSpecificTypeBuild   = "0804-elementalist-frostblink-2.xml"
)

func TestProjectileAvoidanceEntersNotHitChanceWithoutSpecificTypeAvoidance(t *testing.T) {
	env := defence320Env(t, avoidanceProjectileOnlyBuild)
	for _, stat := range []string{"ProjectileNotHitChance", "SpellProjectileNotHitChance", "AverageNotHitChance", "ConfiguredNotHitChance", "ConfiguredDamageChance"} {
		assertMatches320Oracle(t, env, avoidanceProjectileOnlyBuild, stat)
	}
}

func TestSpecificTypeAvoidanceSuppressesTheProjectileNotHitFactor(t *testing.T) {
	env := defence320Env(t, avoidanceSpecificTypeBuild)
	for _, stat := range []string{"ProjectileNotHitChance", "SpellProjectileNotHitChance", "AverageNotHitChance", "ConfiguredDamageChance"} {
		assertMatches320Oracle(t, env, avoidanceSpecificTypeBuild, stat)
	}
}
