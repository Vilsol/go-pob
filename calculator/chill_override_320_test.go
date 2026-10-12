package calculator

import (
	"testing"
)

const chillOverride320Build = "0003-hierophant-storm-burst.xml"

func TestChillingSkillsGrantChillOverride(t *testing.T) {
	env := defence320Env(t, chillOverride320Build)
	assertMatches320Oracle(t, env, chillOverride320Build, "CurrentChill")
	assertMatches320Oracle(t, env, chillOverride320Build, "MaximumChill")
}
