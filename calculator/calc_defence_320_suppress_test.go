package calculator

import (
	"testing"
)

const (
	suppress320CappedBuild      = "0077-pathfinder-toxic-rain-2.xml"
	suppressLifeOnSuppressBuild = "15854-pathfinder-poisonous-concoction-of-bouncing.xml"
	suppressESOnSuppressBuild   = "3665-ladder-nonghoop_1966.NongHoop_ALLF.xml"
)

// TestSuppressionRecoveryOutputsAreExposed: The values come from ModParser.lua:3257-3258.
func TestSuppressionRecoveryOutputsAreExposed(t *testing.T) {
	for _, file := range []string{suppressLifeOnSuppressBuild, suppressESOnSuppressBuild} {
		env := defenceEnv(t, file)
		assertMatchesOracle(t, env, file, "EnergyShieldOnSuppress")
		assertMatchesOracle(t, env, file, "LifeOnSuppress")
	}
}

// TestSuppressionEffectIsNeutralAtFullSuppression: At 100% suppression PoB leaves suppressionEffect at 1 and lets damage reduction carry the whole effect.
func TestSuppressionEffectIsNeutralAtFullSuppression(t *testing.T) {
	env := defenceEnv(t, suppress320CappedBuild)
	assertMatchesOracle(t, env, suppress320CappedBuild, "ConfiguredDamageChance")
}
