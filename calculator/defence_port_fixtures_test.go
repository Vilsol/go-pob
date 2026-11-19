package calculator

import "testing"

// TestShowDebuffExpirationModifier pins CalcDefence.lua:1670.
func TestShowDebuffExpirationModifier(t *testing.T) {
	const fixture = "debuff-expiration"
	env := portFixtureEnv(t, fixture, OutputModeMain)
	wantStat(t, fixture, env.Player.Output, "showDebuffExpirationModifier", 1)
	wantStat(t, fixture, env.Player.Output, "DebuffExpirationModifier", 83.333333333333)
}

// TestZeroBuildDegenIsUnset pins CalcDefence.lua:3415 and 3466-3467.
func TestZeroBuildDegenIsUnset(t *testing.T) {
	const fixture = "degen-fully-reduced"
	env := portFixtureEnv(t, fixture, OutputModeMain)
	wantStat(t, fixture, env.Player.Output, "ChaosTakenDotMult", 0)
	wantStat(t, fixture, env.Player.Output, "ChaosBuildDegen", 0)
	wantAbsent(t, fixture, env.Player.Output, "TotalBuildDegen")
	wantAbsent(t, fixture, env.Player.Output, "NetLifeRegen")
}
