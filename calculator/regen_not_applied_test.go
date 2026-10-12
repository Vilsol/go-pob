package calculator

import "testing"

// TestRecoveryFromRegenerationNotApplied pins ModParser.lua:2192 (Chainbreaker via Lethal Pride).
func TestRecoveryFromRegenerationNotApplied(t *testing.T) {
	assertBuildStats(t, "5026-ladder-Jaxinator_3709.SickLaunchYouRestarts.xml", []string{"NetManaRegen", "TotalNetRegen"}, nil)
}
