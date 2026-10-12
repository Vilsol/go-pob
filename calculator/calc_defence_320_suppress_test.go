package calculator

import (
	"testing"
)

const (
	suppress320CappedBuild  = "0030-saboteur-hexblast.xml"
	suppress320PartialBuild = "0041-saboteur-lightning-trap-5.xml"
)

// TestSuppressionRecoveryOutputsAreExposed: The values come from ModParser.lua:3257-3258.
func TestSuppressionRecoveryOutputsAreExposed(t *testing.T) {
	for _, file := range []string{suppress320CappedBuild, suppress320PartialBuild} {
		env := defence320Env(t, file)
		assertMatches320Oracle(t, env, file, "EnergyShieldOnSuppress")
		assertMatches320Oracle(t, env, file, "LifeOnSuppress")
	}
}

// TestSuppressionEffectIsNeutralAtFullSuppression: At 100% suppression PoB leaves suppressionEffect at 1 and lets damage reduction carry the whole effect.
func TestSuppressionEffectIsNeutralAtFullSuppression(t *testing.T) {
	env := defence320Env(t, suppress320CappedBuild)
	assertMatches320Oracle(t, env, suppress320CappedBuild, "ConfiguredDamageChance")
}
