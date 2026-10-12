package calculator

import (
	"github.com/Vilsol/go-pob-data/poe"

	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

type GemSortRow struct {
	CanSupport bool
	Pending    bool
}

//pob:block:src/Classes/GemSelectControl.lua:294-359
func (env *Environment) GemSortCache(group int, gems []pob.Gem) []GemSortRow {
	out := make([]GemSortRow, len(gems))
	if env == nil || env.Build == nil {
		return out
	}
	skillSet := activeSkillSetIndex(env.Build.Skills)
	if skillSet < 0 || group < 0 || group >= len(env.Build.Skills.SkillSets[skillSet].Skills) {
		return out
	}
	groups := env.Build.Skills.SkillSets[skillSet].Skills
	displayGroup := &groups[group]

	var skillLists [][]*ActiveSkill
	if list, _ := pob.DisplaySkillListOf(displayGroup).([]*ActiveSkill); len(list) > 0 {
		skillLists = append(skillLists, list)
	} else if displayGroup.Slot != "" {
		for i := range groups {
			other := &groups[i]
			if other.SourceAttr == nil || other.Slot == "" || other.Slot != displayGroup.Slot {
				continue
			}
			if list, _ := pob.DisplaySkillListOf(other).([]*ActiveSkill); len(list) > 0 {
				skillLists = append(skillLists, list)
			}
		}
	}

	for i, gem := range gems {
		_, row, ok := gemTooltipRow(gem)
		if !ok {
			continue
		}
		effect := poe.GrantedEffectByID(row.GrantedEffectID)
		if effect == nil {
			continue
		}
		support := raw2.PoBSkills[row.GrantedEffectID].Support
		if support {
			wrapped := &GrantedEffect{Raw: effect, fromItem: env.itemSupports[effect.ID]}
		lists:
			for _, list := range skillLists {
				for _, activeSkill := range list {
					if CalcCanGrantedEffectSupportActiveSkill(wrapped, activeSkill) {
						out[i].CanSupport = true
						break lists
					}
				}
			}
		}
		out[i].Pending = out[i].CanSupport || (raw2.SkillHasGlobalEffect(row.GrantedEffectID) && !support)
	}
	return out
}

func (env *Environment) markItemSupport(id string) {
	env.itemSupportSet()[id] = true
}

func (env *Environment) itemSupportSet() map[string]bool {
	if env.itemSupports == nil {
		env.itemSupports = map[string]bool{}
	}
	return env.itemSupports
}
