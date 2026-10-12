package calculator

import (
	"testing"
)

const maximumAilment320Build = "0025-elementalist-lightning-conduit.xml"

func TestMaximumAilmentIncludesBaseSkillModListTerm(t *testing.T) {
	env := defence320Env(t, maximumAilment320Build)
	assertMatches320Oracle(t, env, maximumAilment320Build, "MaximumShock")
}
