package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/moddb"
)

// TestRotationImpactKeepsAZeroChance: CalcTriggers.lua:110 is `chance or 100`, so an explicit 0 stays 0 and the rate
// is 0. The wants are the pinned Lua function run under luajit: rotation {cd 0.5, icdr 1}, source rate 4, trigger
// cooldown 0.25.
//
//pob:src/Modules/CalcTriggers.lua:105-148
func TestRotationImpactKeepsAZeroChance(t *testing.T) {
	mainSkill := &ActiveSkill{SkillModList: moddb.NewModList(), SkillCfg: &moddb.ListCfg{}, SkillData: &SkillData{}, SkillTypes: map[data.SkillType]bool{}}
	env := &Environment{Player: &Actor{MainSkill: mainSkill}}
	uuid := cacheSkillUUID(env, mainSkill)
	for _, tc := range []struct {
		chance, want float64
	}{
		{0, 0},
		{50, 1.0674486803519061},
		{100, 1.456},
	} {
		got, _ := calcMultiSpellRotationImpact329(env, []*triggerRotationSkill{{uuid: uuid, cd: 0.5, icdr: 1}}, 4, 0.25, tc.chance, nil)
		if math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("chance %v: rate = %v, want %v", tc.chance, got, tc.want)
		}
	}
}
