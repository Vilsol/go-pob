package calculator2

import (
	"slices"
	"strconv"
	"strings"

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

// SkillSelectControls is what RefreshSkillSelectControls leaves on the Active Skill, Stat Set, Skill Part, Skill Stages,
// Active Mines, Minion, Minion Skill and Minion Skill Stat Set controls and the Manage Spectres and Manage Beasts
// buttons. Indexes are 1-based; a buf is the edit box's text.
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

	Minions                   []string
	MinionSkills              []string
	MinionSkillStatSets       []string
	Minion                    int
	MinionSkill               int
	MinionSkillStatSet        int
	MinionShown               bool
	MinionEnabled             bool
	MinionLibraryShown        bool
	BeastLibraryShown         bool
	MinionSkillShown          bool
	MinionSkillEnabled        bool
	MinionSkillStatSetShown   bool
	MinionSkillStatSetEnabled bool
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
// group's skills; calcs is the "Calcs" suffix, whose controls carry the Calcs tab's Show Minion Stats check box.
// PoE2's skill data has no multipart skill, so the Skill Part dropdown never shows.
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
	skillSelectMinion(build, env, &controls, activeSkill, gem, calcs)
	return controls
}

// minionEntry is one mainSkillMinion list entry: an item set or a minion.
type minionEntry struct {
	label     string
	minionID  string
	itemSetID int
}

// skillMinionEntries is the Minion dropdown's list for activeSkill, and whether the dropdown shows at all.
//
//pob2:block:src/Modules/Build.lua:2132-2167
func skillMinionEntries(env *Environment, activeSkill *ActiveSkill) ([]minionEntry, bool) {
	activeEffect := activeSkill.ActiveEffect
	overlay := activeEffect.GrantedEffect.Overlay
	minionList := activeSkill.MinionList
	if minionList == nil {
		minionList = overlay.MinionList
	}
	var skillFlags map[string]bool
	if activeEffect.StatSet != nil {
		skillFlags = activeEffect.StatSet.SkillFlags
	}
	if skillFlags["disable"] || (overlay.MinionList == nil && len(minionList) == 0) {
		return nil, false
	}
	var entries []minionEntry
	if overlay.MinionHasItemSet {
		if tab := env.items; tab != nil {
			for _, itemSetID := range tab.ItemSetOrderList {
				label := "Default Item Set"
				if set := tab.ItemSets[itemSetID]; set != nil && set.Title != "" {
					label = set.Title
				}
				entries = append(entries, minionEntry{label: label, itemSetID: itemSetID})
			}
		}
		return entries, true
	}
	minions := minionData()
	for _, minionID := range minionList {
		if minion := minions[minionID]; minion != nil {
			entries = append(entries, minionEntry{label: minion.Name, minionID: minionID})
		}
	}
	return entries, true
}

// skillMinionSkillIndex is srcInstance.skillMinionSkill..suffix as createMinionSkills leaves it: clamped to the
// minion's skills.
//
//pob2:block:src/Modules/CalcActiveSkill.lua:1194-1203
func skillMinionSkillIndex(gem *pob2.Gem, calcs bool, count int) int {
	index := 1
	saved := gem.SkillMinionSkill
	if calcs {
		saved = gem.SkillMinionSkillCalcs
	}
	if saved != nil {
		index = *saved
	}
	return max(min(index, count), 1)
}

// minionSkillStatSetLookup is srcInstance.skillMinionSkillStatSetIndexLookup..suffix.
func minionSkillStatSetLookup(gem *pob2.Gem, calcs bool) *[]pob2.MinionSkillIndexLookup {
	if calcs {
		return &gem.MinionSkillIndexLookupCalcs
	}
	return &gem.MinionSkillIndexLookups
}

// skillSelectMinion fills the Minion, Minion Skill and Minion Skill Stat Set controls and the library buttons.
//
//pob2:block:src/Modules/Build.lua:2132-2191
func skillSelectMinion(build *pob2.PathOfBuilding2, env *Environment, controls *SkillSelectControls, activeSkill *ActiveSkill, gem *pob2.Gem, calcs bool) {
	entries, shown := skillMinionEntries(env, activeSkill)
	if !shown {
		return
	}
	activeEffect := activeSkill.ActiveEffect
	grantedEffect := activeEffect.GrantedEffect
	src := activeEffect.SrcInstance
	controls.Minions = []string{}
	controls.Minion = 1
	if grantedEffect.Overlay.MinionHasItemSet {
		itemSetID := 0
		if src != nil {
			itemSetID = src.SkillMinionItemSet
			if calcs {
				itemSetID = src.SkillMinionItemSetCalcs
			}
		}
		if itemSetID == 0 {
			itemSetID = 1
		}
		for i, e := range entries {
			if e.itemSetID == itemSetID {
				controls.Minion = i + 1
				break
			}
		}
	} else {
		// the Calcs tab's Show Minion Stats check box hides the library buttons
		libraryShown := grantedEffect.Overlay.MinionList != nil && len(grantedEffect.Overlay.MinionList) == 0 &&
			!(calcs && readCalcsInputs(build).ShowMinion)
		controls.MinionLibraryShown = libraryShown && strings.HasPrefix(grantedEffect.Name, "Spectre:")
		controls.BeastLibraryShown = libraryShown && strings.HasPrefix(grantedEffect.Name, "Companion:")
		minionID := ""
		if src != nil {
			minionID = src.SkillMinion
			if calcs {
				minionID = src.SkillMinionCalcs
			}
		}
		if minionID == "" && len(entries) > 0 {
			minionID = entries[0].minionID
		}
		for i, e := range entries {
			if e.minionID == minionID {
				controls.Minion = i + 1
				break
			}
		}
	}
	for _, e := range entries {
		controls.Minions = append(controls.Minions, e.label)
	}
	controls.MinionEnabled = len(entries) > 1
	controls.MinionShown = true
	minion := activeSkill.Minion
	if minion == nil {
		controls.Minions = append(controls.Minions, "<No spectres in build>")
		return
	}
	if len(minion.ActiveSkillList) == 0 || gem == nil {
		return
	}
	for _, minionSkill := range minion.ActiveSkillList {
		controls.MinionSkills = append(controls.MinionSkills, minionSkill.ActiveEffect.GrantedEffect.Name)
	}
	controls.MinionSkill = skillMinionSkillIndex(gem, calcs, len(minion.ActiveSkillList))
	controls.MinionSkillShown = true
	controls.MinionSkillEnabled = len(controls.MinionSkills) > 1
	controls.MinionSkillStatSets = []string{}
	for _, statSet := range minion.ActiveSkillList[controls.MinionSkill-1].ActiveEffect.GrantedEffect.StatSets {
		controls.MinionSkillStatSets = append(controls.MinionSkillStatSets, statSet.Label)
	}
	controls.MinionSkillStatSet = 1
	for _, lookup := range *minionSkillStatSetLookup(gem, calcs) {
		if lookup.GrantedEffect != grantedEffect.ID {
			continue
		}
		for _, m := range lookup.Maps {
			if !m.SkillIndex.IsNil && m.SkillIndex.V == controls.MinionSkill && !m.StatSetIndex.IsNil {
				controls.MinionSkillStatSet = m.StatSetIndex.V
			}
		}
	}
	controls.MinionSkillStatSetShown = true
	controls.MinionSkillStatSetEnabled = len(controls.MinionSkillStatSets) > 1
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
	activeSkill, gem, err := skillSelectActive(build, env, mainGroup, calcs)
	if err != nil {
		return nil, "", err
	}
	return gem, activeSkill.ActiveEffect.GrantedEffect.ID, nil
}

// skillSelectActive is displaySkillList..suffix[mainActiveSkill..suffix] and the saved gem of its srcInstance.
func skillSelectActive(build *pob2.PathOfBuilding2, env *Environment, mainGroup int, calcs bool) (*ActiveSkill, *pob2.Gem, error) {
	displaySkillList, mainActiveSkill := skillSelectSkill(env, mainGroup, calcs)
	if mainActiveSkill < 1 || mainActiveSkill > len(displaySkillList) || displaySkillList[mainActiveSkill-1].ActiveEffect == nil {
		return nil, nil, errNoSocketGroup
	}
	activeSkill := displaySkillList[mainActiveSkill-1]
	gem := skillSelectGem(build, env, mainGroup, activeSkill.ActiveEffect.SrcInstance)
	if gem == nil || activeSkill.ActiveEffect.GrantedEffect == nil {
		return nil, nil, errNoSocketGroup
	}
	return activeSkill, gem, nil
}

// SetSkillMinion is the Minion dropdown: either suffix writes both the minion (or its item set) and its Calcs twin,
// and renames a Spectre or Companion gem after its pick.
//
//pob2:block:src/Modules/Build.lua:421-443
//pob2:block:src/Classes/CalcsTab.lua:107-132
func SetSkillMinion(build *pob2.PathOfBuilding2, env *Environment, mainGroup int, calcs bool, index int) error {
	activeSkill, gem, err := skillSelectActive(build, env, mainGroup, calcs)
	if err != nil {
		return err
	}
	entries, shown := skillMinionEntries(env, activeSkill)
	if !shown || index < 1 || index > len(entries) {
		return errNoSocketGroup
	}
	value := entries[index-1]
	if value.itemSetID != 0 {
		gem.SkillMinionItemSet = &value.itemSetID
		gem.SkillMinionItemSetCalcs = &value.itemSetID
	} else {
		gem.SkillMinion = &value.minionID
		gem.SkillMinionCalcs = &value.minionID
	}
	if strings.HasPrefix(gem.NameSpec, "Spectre:") {
		gem.NameSpec = "Spectre: " + value.label
	} else if strings.HasPrefix(gem.NameSpec, "Companion:") {
		gem.NameSpec = "Companion: " + value.label
	}
	return nil
}

// SetCompareCalcsMinion is the Compare tab's Calcs header Minion dropdown: unlike the Calcs tab's, it writes only the
// Calcs minion (or item set) and keeps the gem's name.
//
//pob2:block:src/Classes/CompareTab.lua:556-573
func SetCompareCalcsMinion(build *pob2.PathOfBuilding2, env *Environment, mainGroup int, index int) error {
	activeSkill, gem, err := skillSelectActive(build, env, mainGroup, true)
	if err != nil {
		return err
	}
	entries, shown := skillMinionEntries(env, activeSkill)
	if !shown || index < 1 || index > len(entries) {
		return errNoSocketGroup
	}
	if selected := entries[index-1]; selected.itemSetID != 0 {
		gem.SkillMinionItemSetCalcs = &selected.itemSetID
	} else if selected.minionID != "" {
		gem.SkillMinionCalcs = &selected.minionID
	}
	return nil
}

// SetSkillMinionSkill is the Minion Skill dropdown: srcInstance.skillMinionSkill..suffix = index.
//
//pob2:block:src/Modules/Build.lua:468-474
//pob2:block:src/Classes/CalcsTab.lua:143-151
func SetSkillMinionSkill(build *pob2.PathOfBuilding2, env *Environment, mainGroup int, calcs bool, index int) error {
	_, gem, err := skillSelectActive(build, env, mainGroup, calcs)
	if err != nil {
		return err
	}
	if calcs {
		gem.SkillMinionSkillCalcs = &index
	} else {
		gem.SkillMinionSkill = &index
	}
	return nil
}

// SetSkillMinionSkillStatSet is the Minion Skill Stat Set dropdown: the stat set of the selected minion skill, kept
// per granted effect in srcInstance.skillMinionSkillStatSetIndexLookup..suffix.
//
//pob2:block:src/Modules/Build.lua:475-483
//pob2:block:src/Classes/CalcsTab.lua:152-162
func SetSkillMinionSkillStatSet(build *pob2.PathOfBuilding2, env *Environment, mainGroup int, calcs bool, index int) error {
	activeSkill, gem, err := skillSelectActive(build, env, mainGroup, calcs)
	if err != nil {
		return err
	}
	if activeSkill.Minion == nil || len(activeSkill.Minion.ActiveSkillList) == 0 {
		return errNoSocketGroup
	}
	grantedEffectID := activeSkill.ActiveEffect.GrantedEffect.ID
	skillIndex := skillMinionSkillIndex(gem, calcs, len(activeSkill.Minion.ActiveSkillList))
	lookups := minionSkillStatSetLookup(gem, calcs)
	i := slices.IndexFunc(*lookups, func(l pob2.MinionSkillIndexLookup) bool { return l.GrantedEffect == grantedEffectID })
	if i < 0 {
		*lookups = append(*lookups, pob2.MinionSkillIndexLookup{GrantedEffect: grantedEffectID})
		i = len(*lookups) - 1
	}
	lookup := &(*lookups)[i]
	for j := range lookup.Maps {
		if m := &lookup.Maps[j]; !m.SkillIndex.IsNil && m.SkillIndex.V == skillIndex {
			m.StatSetIndex = pob2.Nil[int]{V: index}
			return nil
		}
	}
	lookup.Maps = append(lookup.Maps, pob2.MinionSkillIndexMap{SkillIndex: pob2.Nil[int]{V: skillIndex}, StatSetIndex: pob2.Nil[int]{V: index}})
	return nil
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
