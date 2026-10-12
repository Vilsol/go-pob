package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestPoBOnlySkillTypesMatchThePin pins the hand-transcribed shells in pobOnlyGrantedEffects against src/Data/Skills/*.lua at the CURRENT pin.
func TestPoBOnlySkillTypesMatchThePin(t *testing.T) {
	want := map[string]struct {
		types []data.SkillType
		lua   string
	}{
		"InfernalLegion": {
			types: []data.SkillType{data.SkillTypeDamageOverTime, data.SkillTypeCausesBurning},
			lua:   "src/Data/Skills/minion.lua:1826",
		},
		"MinionInstability": {
			types: []data.SkillType{data.SkillTypeDamage, data.SkillTypeArea},
			lua:   "src/Data/Skills/minion.lua:1756",
		},
		"DelveMonsterEnfeebleOnHit": {
			types: []data.SkillType{
				data.SkillTypeSpell, data.SkillTypeArea, data.SkillTypeDuration,
				data.SkillTypeTrappable, data.SkillTypeTotemable, data.SkillTypeMineable,
				data.SkillTypeMulticastable, data.SkillTypeTriggerable, data.SkillTypeCascadable,
				data.SkillTypeAppliesCurse, data.SkillTypeCanRapidFire, data.SkillTypeAreaSpell,
				data.SkillTypeInstantNoRepeatWhenHeld, data.SkillTypeInstantShiftAttackForLeftMouse,
				data.SkillTypeHex,
			},
			lua: "src/Data/Skills/spectre.lua:6089-6144",
		},
		"EnemyExplode": {
			types: []data.SkillType{data.SkillTypeDamage, data.SkillTypeArea},
			lua:   "src/Data/Skills/other.lua:6063",
		},
		"HarvestNessaCrabScreechDebuff": {
			types: []data.SkillType{
				data.SkillTypeSpell, data.SkillTypeArea, data.SkillTypeDuration,
				data.SkillTypeTriggerable, data.SkillTypeAreaSpell,
			},
			lua: "src/Data/Skills/spectre.lua:4840",
		},
	}

	if len(pobOnlyGrantedEffects) != len(want) {
		t.Fatalf("pobOnlyGrantedEffects holds %d rows and this test knows %d — a new shell must "+
			"bring its own expectation, or it inherits no drift check at all",
			len(pobOnlyGrantedEffects), len(want))
	}

	for id, w := range want {
		t.Run(id, func(t *testing.T) {
			got, ok := pobOnlyGrantedEffects[id]
			if !ok {
				t.Fatalf("%s is missing from pobOnlyGrantedEffects", id)
			}
			set := make(map[data.SkillType]bool, len(got.SkillTypes))
			for _, ty := range got.SkillTypes {
				set[ty] = true
			}
			for _, ty := range w.types {
				if !set[ty] {
					t.Errorf("%s is missing SkillType %s — %s declares it", id, ty, w.lua)
				}
				delete(set, ty)
			}
			for ty := range set {
				t.Errorf("%s carries SkillType %s, which %s does not declare", id, ty, w.lua)
			}
		})
	}
}
