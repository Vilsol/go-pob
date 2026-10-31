package poe2skills

import (
	"slices"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
)

// Gem is the part of data.gems[ID] setupGem links to data.skills: the effect it grants, the effects it
// adds, and both in the order the gem displays them.
type Gem struct {
	Data                     *poe2.Gem
	GrantedEffect            *GrantedEffect
	AdditionalGrantedEffects []*GrantedEffect
	GrantedEffectList        []*GrantedEffect
}

// Gem is data.gems[id] with its effects linked.
func (s *Skills) Gem(id string) (*Gem, bool) {
	g, ok := s.gems[id]
	return g, ok
}

// setupGem links one gem to its granted effects.
//
//pob2:src/Modules/Data.lua:980-1051
func (s *Skills) setupGem(gem *poe2.Gem) *Gem {
	g := &Gem{Data: gem, GrantedEffect: s.byID[gem.GrantedEffectID]}
	g.AdditionalGrantedEffects = []*GrantedEffect{}
	g.GrantedEffectList = []*GrantedEffect{g.GrantedEffect}
	for _, abilityID := range gem.AdditionalGrantedEffectIDs {
		skill := s.byID[abilityID]
		if skill != nil && skill.Name == "Command: {0} " {
			skill.Name = "Command"
		}
		g.GrantedEffectList = append(g.GrantedEffectList, skill)
		g.AdditionalGrantedEffects = append(g.AdditionalGrantedEffects, skill)
	}
	if strings.HasPrefix(gem.Name, "Barbs I") {
		additionalGrantedEffectID := "TriggeredBarbsPlayer"
		if gem.Name == "Barbs II" {
			additionalGrantedEffectID += "Two"
		} else if gem.Name == "Barbs III" {
			additionalGrantedEffectID += "Three"
		}
		// go-pob-data already lists the effect PoB-PoE2's Gems.lua leaves out; append it only once.
		if !slices.Contains(gem.AdditionalGrantedEffectIDs, additionalGrantedEffectID) {
			g.GrantedEffectList = append(g.GrantedEffectList, s.byID[additionalGrantedEffectID])
			g.AdditionalGrantedEffects = append(g.AdditionalGrantedEffects, s.byID[additionalGrantedEffectID])
		}
	}
	if gem.GrantedEffectDisplayOrder != nil {
		list := g.GrantedEffectList
		temp := make([]*GrantedEffect, len(gem.GrantedEffectDisplayOrder))
		used := map[int]bool{}
		for i, order := range gem.GrantedEffectDisplayOrder {
			index := int(order)
			if index < len(list) && list[index] != nil && !used[index] {
				temp[i] = list[index]
				used[index] = true
			} else {
				if i < len(list) {
					temp[i] = list[i]
				}
				used[i] = true
			}
		}
		for i, effect := range list {
			if !used[i] {
				temp = append(temp, effect)
			}
		}
		g.GrantedEffectList = temp
	}
	return g
}

func (s *Skills) setupGems() {
	s.gems = make(map[string]*Gem, len(poe2.Gems))
	for i := range poe2.Gems {
		gem := &poe2.Gems[i]
		s.gems[gem.ID] = s.setupGem(gem)
	}
}
