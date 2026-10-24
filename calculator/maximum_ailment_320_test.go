package calculator

import (
	"testing"
)

const maximumAilment320Build = "0307-ascendant-lightning-tendrils-of-escalation.xml"

func TestMaximumAilmentIncludesBaseSkillModListTerm(t *testing.T) {
	env := defenceEnv(t, maximumAilment320Build)
	assertMatchesOracle(t, env, maximumAilment320Build, "MaximumShock")
}
