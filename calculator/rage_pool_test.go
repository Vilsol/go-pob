package calculator

import "testing"

// TestRagePoolIsExposed pins CalcPerform.lua:1104, the last line of the `if` at :1100-1105.
func TestRagePoolIsExposed(t *testing.T) {
	const build = "0007-ascendant-hexblast.xml"
	env := defence320Env(t, build)
	assertMatches320Oracle(t, env, build, "Rage")
	assertMatches320Oracle(t, env, build, "MaximumRage")
}
