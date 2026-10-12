package calculator

import (
	"testing"
)

const elusive320Build = "1268-assassin-cyclone-2.xml"

func TestElusive320EffectIsHalved(t *testing.T) {
	env := defence320Env(t, elusive320Build)
	assertMatches320Oracle(t, env, elusive320Build, "ElusiveEffectMod")
}

func TestElusive320GrantsAvoidAllRatherThanPerType(t *testing.T) {
	env := defence320Env(t, elusive320Build)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatches320Oracle(t, env, elusive320Build, "Avoid"+damageType+"DamageChance")
	}
}
