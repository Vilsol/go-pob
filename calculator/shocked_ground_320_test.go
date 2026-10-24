package calculator

import (
	"testing"
)

// local effect = skillModList:Sum("BASE", nil, "ShockedGroundEffect")
const shockedGround320Build = "15827-gladiator-dual-strike-of-ambidexterity.xml"

func TestShockedGroundScalesByEnemyShockEffect(t *testing.T) {
	env := defenceEnv(t, shockedGround320Build)
	assertMatchesOracle(t, env, shockedGround320Build, "CurrentShock")
}
