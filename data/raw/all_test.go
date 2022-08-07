package raw

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/utils"
)

func init() {
	config.InitLogging(false)
}

func TestAll(t *testing.T) {
	err := InitializeAll(LatestVersion)
	testza.AssertNoError(t, err)

	effect := GrantedEffectByID("PlayerMelee")
	testza.AssertEqual(t, false, effect.IsSupport)
	testza.AssertEqual(t, 1000, effect.CastTime)
	testza.AssertEqual(t, 4, effect.Attribute)
	testza.AssertEqual(t, utils.Ptr(58), effect.ActiveSkill)

	skill := effect.GetActiveSkill()
	testza.AssertEqual(t, "melee", skill.ID)
	testza.AssertEqual(t, true, skill.IsManuallyCasted)
	testza.AssertEqual(t, "Default Attack", skill.DisplayedName)

	types := skill.GetActiveSkillTypes()
	testza.AssertEqual(t, 8, len(types))
}
