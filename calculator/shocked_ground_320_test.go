package calculator

import (
	"testing"
)

// local effect = skillModList:Sum("BASE", nil, "ShockedGroundEffect")
const shockedGround320Build = "2504-pathfinder-herald-of-thunder.xml"

func TestShockedGroundScalesByEnemyShockEffect(t *testing.T) {
	env := defence320Env(t, shockedGround320Build)
	assertMatches320Oracle(t, env, shockedGround320Build, "CurrentShock")
}
