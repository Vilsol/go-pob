package calculator

import (
	"testing"
)

const chillOverride320Build = "0430-reliquarian-vaal-cold-snap.xml"

func TestChillingSkillsGrantChillOverride(t *testing.T) {
	env := defenceEnv(t, chillOverride320Build)
	assertMatchesOracle(t, env, chillOverride320Build, "CurrentChill")
	assertMatchesOracle(t, env, chillOverride320Build, "MaximumChill")
}
