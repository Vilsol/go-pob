package calculator2

import (
	"slices"
	"strconv"

	"github.com/Vilsol/go-pob/pob2"
)

// CalcsInputs is calcsTab.input as the Calcs tab reads it: the socket group it shows (1-based), its buff mode and
// whether it shows the minion.
type CalcsInputs struct {
	SkillNumber int
	BuffMode    string
	ShowMinion  bool
}

func calcsInput(build *pob2.PathOfBuilding2, name string) *pob2.Input {
	for i := range build.Calcs.Inputs {
		if build.Calcs.Inputs[i].Name == name {
			return &build.Calcs.Inputs[i]
		}
	}
	return nil
}

// readCalcsInputs is calcsTab.input after the tab's constructor and Load.
//
//pob2:block:src/Classes/CalcsTab.lua:31-33
//pob2:block:src/Classes/CalcsTab.lua:200-217
func readCalcsInputs(build *pob2.PathOfBuilding2) CalcsInputs {
	in := CalcsInputs{SkillNumber: 1, BuffMode: "EFFECTIVE"}
	if v := calcsInput(build, "skill_number"); v != nil && v.Number != nil {
		in.SkillNumber = int(*v.Number)
	}
	if v := calcsInput(build, "misc_buffMode"); v != nil && v.String != nil {
		in.BuffMode = *v.String
	}
	if v := calcsInput(build, "showMinion"); v != nil && v.Boolean != nil {
		in.ShowMinion = *v.Boolean
	}
	return in
}

func ReadCalcsInputs(build *pob2.PathOfBuilding2) CalcsInputs {
	return readCalcsInputs(build)
}

func setCalcsInput(build *pob2.PathOfBuilding2, in pob2.Input) {
	if v := calcsInput(build, in.Name); v != nil {
		*v = in
		return
	}
	build.Calcs.Inputs = append(build.Calcs.Inputs, in)
}

// SetCalcsInputNumber, SetCalcsInputString and SetCalcsInputBool are the tab's controls writing self.input[name].
//
//pob2:block:src/Classes/CalcsTab.lua:43-46
func SetCalcsInputNumber(build *pob2.PathOfBuilding2, name string, value float64) {
	setCalcsInput(build, pob2.Input{Name: name, Number: &value})
}

//pob2:block:src/Classes/CalcsTab.lua:165-168
func SetCalcsInputString(build *pob2.PathOfBuilding2, name string, value string) {
	setCalcsInput(build, pob2.Input{Name: name, String: &value})
}

//pob2:block:src/Classes/CalcsTab.lua:102-105
func SetCalcsInputBool(build *pob2.PathOfBuilding2, name string, value bool) {
	setCalcsInput(build, pob2.Input{Name: name, Boolean: &value})
}

// SetMainActiveSkillCalcs is the tab's Active Skill dropdown: the shown socket group's mainActiveSkillCalcs.
//
//pob2:block:src/Classes/CalcsTab.lua:58-63
func SetMainActiveSkillCalcs(build *pob2.PathOfBuilding2, env *Environment, index int) error {
	group, err := editGroup(build, env, readCalcsInputs(build).SkillNumber)
	if err != nil {
		return err
	}
	group.MainActiveSkillCalcs = pob2.Nil[int]{V: index}
	return nil
}

// BuildCalcsOutput is calcs.buildOutput(build, "CALCS") after the main pass: the Calcs tab's own calculation over the
// socket group list the main pass left, with the tab's lists. main and used are calcsTab.mainEnv and its tables.
//
//pob2:block:src/Classes/CalcsTab.lua:512-513
//pob2:block:src/Modules/Calcs.lua:518-531
func BuildCalcsOutput(main *Environment, used *UsedTables) *Environment {
	env := newWeaponSetEnv(main, "CALCS", calcOverride{})
	initEnv(env)
	perform(env)
	setFullDPS(env, calcFullDPS(env, "CALCULATOR", calcOverride{}))
	calcsBuffLists(env, used.MultipliersUsed)
	return env
}

// SkillSelectControls is what RefreshSkillSelectControls leaves on the Active Skill, Stat Set, Skill Part, Skill Stages
// and Active Mines controls. Indexes are 1-based; a buf is the edit box's text.
type SkillSelectControls struct {
	Skills          []string
	StatSets        []string
	StageCount      string
	MineCount       string
	MainActiveSkill int
	StatSet         int
	SkillShown      bool
	SkillEnabled    bool
	StatSetShown    bool
	StatSetEnabled  bool
	PartShown       bool
	StageShown      bool
	MineShown       bool
}

// skillSelectSkill is socketGroupList[mainGroup]'s displaySkillList..suffix and mainActiveSkill..suffix; calcs is the
// "Calcs" suffix. displaySkillListCalcs is the group's socketGroupSkillList, CalcSetup.lua:2432-2434.
func skillSelectSkill(env *Environment, mainGroup int, calcs bool) ([]*ActiveSkill, int) {
	if mainGroup < 1 || mainGroup > len(env.SocketGroupList) {
		return nil, 0
	}
	group := env.SocketGroupList[mainGroup-1]
	mainActiveSkill := group.MainActiveSkill
	if calcs {
		mainActiveSkill = group.MainActiveSkillCalcs
	}
	if mainActiveSkill == 0 {
		mainActiveSkill = 1
	}
	return displaySkillList(env, group), mainActiveSkill
}

// SkillSelect is RefreshSkillSelectControls(controls, mainGroup, suffix) over env, the calculation that listed the
// group's skills; calcs is the "Calcs" suffix. The minion controls are not ported, and PoE2's skill data has no
// multipart skill, so the Skill Part dropdown never shows.
//
//pob2:src/Modules/Build.lua:2059-2195
func SkillSelect(build *pob2.PathOfBuilding2, env *Environment, mainGroup int, calcs bool) SkillSelectControls {
	controls := SkillSelectControls{Skills: []string{}, StatSets: []string{}}
	displaySkillList, mainActiveSkill := skillSelectSkill(env, mainGroup, calcs)
	if mainGroup < 1 || mainGroup > len(env.SocketGroupList) {
		return controls
	}
	for _, activeSkill := range displaySkillList {
		controls.Skills = append(controls.Skills, getActiveSkillDisplayName(activeSkill))
	}
	controls.SkillEnabled = len(displaySkillList) > 1
	controls.MainActiveSkill = mainActiveSkill
	controls.SkillShown = true
	if len(displaySkillList) == 0 || mainActiveSkill > len(displaySkillList) {
		return controls
	}
	activeSkill := displaySkillList[mainActiveSkill-1]
	activeEffect := activeSkill.ActiveEffect
	if activeEffect == nil || activeEffect.GrantedEffect == nil {
		return controls
	}
	for _, statSet := range activeEffect.GrantedEffect.StatSets {
		controls.StatSets = append(controls.StatSets, statSet.Label)
	}
	controls.StatSet = 1
	if src := activeEffect.SrcInstance; src != nil {
		statSets := src.StatSet
		if calcs {
			statSets = src.StatSetCalcs
		}
		if i, ok := statSets[activeEffect.GrantedEffect.ID]; ok {
			controls.StatSet = i
		}
	}
	controls.StatSetEnabled = len(controls.StatSets) > 1
	controls.StatSetShown = true
	var skillFlags map[string]bool
	if activeEffect.StatSet != nil {
		skillFlags = activeEffect.StatSet.SkillFlags
	}
	gem := skillSelectGem(build, env, mainGroup, activeEffect.SrcInstance)
	if skillFlags["mine"] {
		controls.MineShown = true
		if n := gemCount(gem, calcs, func(g *pob2.Gem) (*int, *int) { return g.SkillMineCount, g.SkillMineCountCalcs }); n != nil {
			controls.MineCount = strconv.Itoa(*n)
		}
	}
	if skillFlags["multiStage"] {
		controls.StageShown = true
		if n := gemCount(gem, calcs, func(g *pob2.Gem) (*int, *int) { return g.SkillStageCount, g.SkillStageCountCalcs }); n != nil {
			controls.StageCount = strconv.Itoa(*n)
		} else if v, ok := activeSkill.SkillData["stagesMin"].(float64); ok {
			controls.StageCount = strconv.FormatFloat(v, 'f', -1, 64)
		} else {
			controls.StageCount = "1"
		}
	}
	return controls
}

func gemCount(gem *pob2.Gem, calcs bool, field func(*pob2.Gem) (*int, *int)) *int {
	if gem == nil {
		return nil
	}
	main, calc := field(gem)
	if calcs {
		return calc
	}
	return main
}

// skillSelectGem is the saved entry of activeEffect.srcInstance, nil for a gem the build file does not hold.
func skillSelectGem(build *pob2.PathOfBuilding2, env *Environment, mainGroup int, src *GemInstance) *pob2.Gem {
	if src == nil || mainGroup < 1 || mainGroup > len(env.SocketGroupList) {
		return nil
	}
	group := env.SocketGroupList[mainGroup-1]
	saved := savedSocketGroup(build, group.savedIndex)
	i := slices.Index(group.GemList, src)
	if saved == nil || i < 0 || i >= len(saved.Gems) {
		return nil
	}
	return &saved.Gems[i]
}

// skillSelectSrc is the saved gem the Stat Set, Skill Stages and Active Mines controls write: displaySkillList..suffix
// [mainActiveSkill..suffix].activeEffect.srcInstance, and that effect's granted effect id.
func skillSelectSrc(build *pob2.PathOfBuilding2, env *Environment, mainGroup int, calcs bool) (*pob2.Gem, string, error) {
	displaySkillList, mainActiveSkill := skillSelectSkill(env, mainGroup, calcs)
	if mainActiveSkill < 1 || mainActiveSkill > len(displaySkillList) || displaySkillList[mainActiveSkill-1].ActiveEffect == nil {
		return nil, "", errNoSocketGroup
	}
	activeEffect := displaySkillList[mainActiveSkill-1].ActiveEffect
	gem := skillSelectGem(build, env, mainGroup, activeEffect.SrcInstance)
	if gem == nil || activeEffect.GrantedEffect == nil {
		return nil, "", errNoSocketGroup
	}
	return gem, activeEffect.GrantedEffect.ID, nil
}

// SetSkillStatSet is the Stat Set dropdown: srcInstance.statSet..suffix[grantedEffectId] = index.
//
//pob2:block:src/Modules/Build.lua:382-389
//pob2:block:src/Classes/CalcsTab.lua:65-74
func SetSkillStatSet(build *pob2.PathOfBuilding2, env *Environment, mainGroup int, calcs bool, index int) error {
	gem, id, err := skillSelectSrc(build, env, mainGroup, calcs)
	if err != nil {
		return err
	}
	list := &gem.StatSetIndexes
	if calcs {
		list = &gem.StatSetCalcsIndexes
	}
	for i := range *list {
		if (*list)[i].GrantedEffect == id {
			(*list)[i].Index = pob2.Nil[int]{V: index}
			return nil
		}
	}
	*list = append(*list, pob2.StatSetIndex{GrantedEffect: id, Index: pob2.Nil[int]{V: index}})
	return nil
}

// SetSkillStageCount and SetSkillMineCount are the Skill Stages and Active Mines boxes: srcInstance.skillStageCount..
// suffix = tonumber(buf); nil clears it.
//
//pob2:block:src/Modules/Build.lua:402-420
//pob2:block:src/Classes/CalcsTab.lua:83-100
func SetSkillStageCount(build *pob2.PathOfBuilding2, env *Environment, mainGroup int, calcs bool, count *int) error {
	gem, _, err := skillSelectSrc(build, env, mainGroup, calcs)
	if err != nil {
		return err
	}
	if calcs {
		gem.SkillStageCountCalcs = count
	} else {
		gem.SkillStageCount = count
	}
	return nil
}

func SetSkillMineCount(build *pob2.PathOfBuilding2, env *Environment, mainGroup int, calcs bool, count *int) error {
	gem, _, err := skillSelectSrc(build, env, mainGroup, calcs)
	if err != nil {
		return err
	}
	if calcs {
		gem.SkillMineCountCalcs = count
	} else {
		gem.SkillMineCount = count
	}
	return nil
}

// SyncCalcsSkillSelection points a compared build's Calcs tab at its main socket group and active skill, as loading it
// does.
//
//pob2:block:src/Classes/CompareEntry.lua:232-238
func SyncCalcsSkillSelection(build *pob2.PathOfBuilding2) {
	SetCalcsInputNumber(build, "skill_number", float64(build.Build.MainSocketGroup))
	mainGroup := savedSocketGroup(build, build.Build.MainSocketGroup)
	if mainGroup == nil {
		return
	}
	mainGroup.MainActiveSkillCalcs = mainGroup.MainActiveSkill
}

// SetCalcsSectionCollapsed is a section's +/- toggle, kept as Save writes subSection.collapsed.
//
//pob2:block:src/Classes/CalcSectionControl.lua:50-55
//pob2:block:src/Classes/CalcsTab.lua:252-260
func SetCalcsSectionCollapsed(build *pob2.PathOfBuilding2, id, subsection string, collapsed bool) {
	for i := range build.Calcs.Sections {
		if s := &build.Calcs.Sections[i]; s.ID == id && s.Subsection == subsection {
			s.Collapsed = collapsed
			return
		}
	}
	build.Calcs.Sections = append(build.Calcs.Sections, pob2.Section{ID: id, Subsection: subsection, Collapsed: collapsed})
}

// CalcsSectionCollapsed is the collapsed state Load gives a subsection, and whether the build saved one.
//
//pob2:block:src/Classes/CalcsTab.lua:218-233
func CalcsSectionCollapsed(build *pob2.PathOfBuilding2, id, subsection string) (bool, bool) {
	for _, s := range build.Calcs.Sections {
		if s.ID == id && s.Subsection == subsection {
			return s.Collapsed, true
		}
	}
	return false, false
}
