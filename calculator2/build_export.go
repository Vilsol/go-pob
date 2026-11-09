package calculator2

import (
	"slices"
	"strconv"

	"github.com/Vilsol/go-pob/data/poe2skills"
	"github.com/Vilsol/go-pob/pob2"
)

// BuildPlannerSkill is a skills entry of the in-game build planner's .build file; a support is its id alone
// when it has no text.
type BuildPlannerSkill struct {
	ID             string `json:"id"`
	AdditionalText string `json:"additional_text,omitempty"`
	SupportSkills  []any  `json:"support_skills,omitempty"`
}

type BuildPlannerSupport struct {
	ID             string `json:"id"`
	AdditionalText string `json:"additional_text"`
}

//pob2:src/Modules/BuildExportPoE2.lua:101-116
func gemAdditionalText(gem *GemInstance, note string, isSupport bool) string {
	if gem == nil {
		return ""
	}
	if note != "" {
		return note
	}
	level := gem.Level
	if level <= 0 {
		return ""
	}
	quality := gem.Quality
	if isSupport && level == 1 && quality == 0 {
		return ""
	}
	if quality > 0 {
		return "Level " + strconv.Itoa(level) + ", " + strconv.FormatFloat(quality, 'f', -1, 64) + "% Quality"
	}
	return "Level " + strconv.Itoa(level)
}

//pob2:src/Modules/BuildExportPoE2.lua:118-132
func activeSkillGems(group *SocketGroup) []*GemInstance {
	var out []*GemInstance
	for _, gemInstance := range group.GemList {
		if gemInstance.Enabled && (gemInstance.GemData != nil || gemInstance.GrantedEffect != nil) {
			grantedEffectList := []*poe2skills.GrantedEffect{gemInstance.GrantedEffect}
			if gemInstance.GemData != nil {
				grantedEffectList = gemInstance.GemData.GrantedEffectList
			}
			for index, grantedEffect := range grantedEffectList {
				if grantedEffect == nil {
					break
				}
				if !grantedEffect.Support && !grantedEffect.Overlay.HideFromSideBar && (!grantedEffect.GlobalEffect() || gemInstance.enableGlobal(index+1)) {
					out = append(out, gemInstance)
				}
			}
		}
	}
	return out
}

// BuildPlannerSkills is the skills list of a .build file for one skill set. The active set's groups are
// skillsTab.socketGroupList as the last calculation left it; another set's are loaded from the build as SkillsTab:Load
// loads them. A gem's note is read from the saved gem it was loaded from.
//
//pob2:src/Modules/BuildExportPoE2.lua:133-178
func BuildPlannerSkills(env *Environment, skillSetID int) []BuildPlannerSkill {
	out := []BuildPlannerSkill{}
	if env == nil || env.Build == nil {
		return out
	}
	skills := normalSkillSets(&pob2.PathOfBuilding2{Skills: pob2.Skills{
		ActiveSkillSet: env.Build.Skills.ActiveSkillSet,
		Skills:         env.Build.Skills.Skills,
		SkillSets:      slices.Clone(env.Build.Skills.SkillSets),
	}})
	setIndex := skillSetIndex(skills, skillSetID)
	if setIndex < 0 {
		return out
	}
	active := skillSetID == skills.ActiveSkillSet
	var saved []pob2.Skill
	var groups []*SocketGroup
	if active {
		saved = activeSkillSet(env)
		groups = env.SocketGroupList
	} else {
		for _, s := range skills.SkillSets[setIndex].Skills {
			if s.Removed == nil || !*s.Removed {
				saved = append(saved, s)
			}
		}
		groups = loadSavedSocketGroups(env, saved)
	}
	note := func(group *SocketGroup, gem *GemInstance) string {
		if group.savedIndex < 1 || group.savedIndex > len(saved) {
			return ""
		}
		gems := saved[group.savedIndex-1].Gems
		if i := slices.Index(group.GemList, gem); i >= 0 && i < len(gems) && gems[i].Note != nil {
			return *gems[i].Note
		}
		return ""
	}
	for _, group := range groups {
		if !group.Enabled || len(group.GemList) == 0 {
			continue
		}
		activeIdx := group.MainActiveSkill
		if activeIdx < 1 {
			activeIdx = 1
		}
		// a group on the other weapon set keeps the displaySkillList an earlier calculation gave it in PoB-PoE2; its
		// active gems stand in for that list here
		var activeGem *GemInstance
		if active && group.UsingSkillSet == env.WeaponSet {
			if list := displaySkillList(env, group); activeIdx <= len(list) && list[activeIdx-1].ActiveEffect != nil {
				activeGem = list[activeIdx-1].ActiveEffect.SrcInstance
			}
		} else if list := activeSkillGems(group); activeIdx <= len(list) {
			activeGem = list[activeIdx-1]
		}
		if activeGem == nil || activeGem.GemData == nil || activeGem.GemData.Data.GameID == "" {
			continue
		}
		entry := BuildPlannerSkill{ID: activeGem.GemData.Data.GameID, AdditionalText: gemAdditionalText(activeGem, note(group, activeGem), false)}
		for _, gem := range group.GemList {
			if gem != activeGem && gem.Enabled && gem.GemData != nil && gem.GemData.GrantedEffect.Support && gem.GemData.Data.GameID != "" {
				if supText := gemAdditionalText(gem, note(group, gem), true); supText != "" {
					entry.SupportSkills = append(entry.SupportSkills, BuildPlannerSupport{ID: gem.GemData.Data.GameID, AdditionalText: supText})
				} else {
					entry.SupportSkills = append(entry.SupportSkills, gem.GemData.Data.GameID)
				}
			}
		}
		out = append(out, entry)
	}
	return out
}

// loadSavedSocketGroups is loadSocketGroups over a skill set that is not the active one.
//
//pob2:block:src/Classes/SkillsTab.lua:288-399
func loadSavedSocketGroups(env *Environment, saved []pob2.Skill) []*SocketGroup {
	list := make([]*SocketGroup, 0, len(saved))
	for i := range saved {
		node := &saved[i]
		group := &SocketGroup{savedIndex: i + 1}
		group.Enabled = (node.Active != nil && *node.Active) || node.Enabled
		group.MainActiveSkill = 1
		if !node.MainActiveSkill.IsNil {
			group.MainActiveSkill = node.MainActiveSkill.V
		}
		group.Set1 = node.Set1
		group.Set2 = node.Set2
		for j := range node.Gems {
			group.GemList = append(group.GemList, loadGemInstance(env, &node.Gems[j]))
		}
		processSocketGroup(env, group)
		list = append(list, group)
	}
	return list
}
