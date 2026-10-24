package calculator

import (
	"testing"
)

const elusive320Build = "0149-warden-frost-blades-2.xml"

func TestElusive320EffectIsHalved(t *testing.T) {
	env := defenceEnv(t, elusive320Build)
	assertMatchesOracle(t, env, elusive320Build, "ElusiveEffectMod")
}

func TestElusive320GrantsAvoidAllRatherThanPerType(t *testing.T) {
	env := defenceEnv(t, elusive320Build)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatchesOracle(t, env, elusive320Build, "Avoid"+damageType+"DamageChance")
	}
}
