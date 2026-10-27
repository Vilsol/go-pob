package calculator2

import (
	"maps"
)

// createActiveSkill creates an active skill using the given active gem and list of support gems.
// It will determine the base flag set, and check which of the support gems can support this skill.
//
//pob2:src/Modules/CalcActiveSkill.lua:142-234
func createActiveSkill(activeEffect *GemEffect, supportList []*GemEffect, env *Environment, actor *Actor, socketGroup *SocketGroup, summonSkill *ActiveSkill) *ActiveSkill {
	activeSkill := &ActiveSkill{
		ActiveEffect: activeEffect,
		SupportList:  supportList,
		Actor:        actor,
		SummonSkill:  summonSkill,
		SocketGroup:  socketGroup,
		SkillData:    map[string]any{},
	}

	activeGrantedEffect := activeEffect.GrantedEffect

	// Initialise skill types
	activeSkill.SkillTypes = maps.Clone(activeGrantedEffect.SkillTypes)
	if activeSkill.SkillTypes == nil {
		activeSkill.SkillTypes = map[string]bool{}
	}
	if activeGrantedEffect.MinionSkillTypes != nil {
		activeSkill.MinionSkillTypes = maps.Clone(activeGrantedEffect.MinionSkillTypes)
	}

	// Initialise skill flag set ('attack', 'projectile', etc)
	// CALCS mode reads activeEffect.statSetCalcs, which StatSet stands for here.
	var skillFlags map[string]bool
	index := activeEffect.StatSet.Index
	if index >= 1 && index <= len(activeGrantedEffect.StatSets) {
		statSet := activeGrantedEffect.StatSets[index-1]
		activeEffect.StatSet.StatSet = statSet
		skillFlags = maps.Clone(statSet.BaseFlags)
	}
	if skillFlags == nil {
		skillFlags = map[string]bool{}
	}
	activeEffect.StatSet.SkillFlags = skillFlags
	skillFlags["hit"] = skillFlags["hit"] || activeSkill.SkillTypes["Attack"] || activeSkill.SkillTypes["Damage"] || activeSkill.SkillTypes["Projectile"]
	if !skillFlags["hit"] {
		delete(skillFlags, "hit")
	}
	activeSkill.SkillFlags = skillFlags

	// Process support skills
	activeSkill.EffectList = []*GemEffect{activeEffect}
	var rejectedSupportsIndices []int

	for index, supportEffect := range supportList {
		// Pass 1: Add skill types from compatible supports
		if canGrantedEffectSupportActiveSkill(env, supportEffect.GrantedEffect, activeSkill) {
			for _, skillType := range supportEffect.GrantedEffect.AddSkillTypes {
				activeSkill.SkillTypes[skillType] = true
			}
		} else {
			rejectedSupportsIndices = append(rejectedSupportsIndices, index)
		}
	}

	// loop over rejected supports until none are added.
	// Makes sure that all skillType flags that should be added are added regardless of support gem order in group
	// ipairs stops at the first index the loop set to nil, as it does here at the first -1.
	for {
		notAddedNewSupport := true
		for i, supportEffectIndex := range rejectedSupportsIndices {
			if supportEffectIndex < 0 {
				break
			}
			supportEffect := supportList[supportEffectIndex]
			if canGrantedEffectSupportActiveSkill(env, supportEffect.GrantedEffect, activeSkill) {
				notAddedNewSupport = false
				rejectedSupportsIndices[i] = -1
				for _, skillType := range supportEffect.GrantedEffect.AddSkillTypes {
					activeSkill.SkillTypes[skillType] = true
				}
			}
		}
		if notAddedNewSupport {
			break
		}
	}

	for _, supportEffect := range supportList {
		// Pass 2: Add all compatible supports
		if canGrantedEffectSupportActiveSkill(env, supportEffect.GrantedEffect, activeSkill) {
			activeSkill.EffectList = append(activeSkill.EffectList, supportEffect)
			// Track how many active skills are supported by this support effect
			if supportEffect.IsSupporting != nil && activeEffect.SrcInstance != nil {
				supportEffect.IsSupporting[activeEffect.SrcInstance] = true
				if supportEffect.SrcInstance != activeEffect.SrcInstance || !(activeEffect.GemData != nil && activeEffect.GemData.GrantedEffect.Support) {
					supportEffect.ActiveSkillLevel = activeEffect.Level
				}
			}
			if supportEffect.GrantedEffect.Overlay.AddFlags != nil && summonSkill == nil {
				// Support skill adds flags to supported skills (eg. Remote Mine adds 'mine')
				for _, k := range supportEffect.GrantedEffect.Overlay.AddFlags {
					skillFlags[k] = true
				}
			}
		}
	}

	return activeSkill
}
