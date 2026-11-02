package calculator2

import (
	"slices"
	"sort"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/poe2skills"
	"github.com/Vilsol/go-pob/pob2"
)

// The Skills tab's edits. PoB-PoE2 edits skillsTab.socketGroupList, the list the last calculation filled in, and
// saves it; here the build file is the list, so an edit first writes the groups the calculation generated or
// dropped into it (SyncSocketGroups), then edits the saved entry the way the control edits the group. Group and
// gem indexes are 1-based places in socketGroupList and gemList.

type skillsEditError string

func (e skillsEditError) Error() string { return string(e) }

const (
	errNoSocketGroup skillsEditError = "no such socket group"
	errNoGem         skillsEditError = "no such gem"
	errGeneratedGem  skillsEditError = "the gem is granted by the group's source"
	errSourceGroup   skillsEditError = "this socket group cannot be deleted as it is created by an equipped item"
	errUnknownGem    skillsEditError = "unknown gem"
)

// savedSkillList is the active skill set's saved <Skill> list, by reference, removed groups included.
//
//pob2:block:src/Classes/SkillsTab.lua:1776-1801
func savedSkillList(build *pob2.PathOfBuilding2) *[]pob2.Skill {
	skills := &build.Skills
	if len(skills.SkillSets) == 0 {
		return &skills.Skills
	}
	list := &skills.SkillSets[0].Skills
	for i := range skills.SkillSets {
		if skills.SkillSets[i].ID == skills.ActiveSkillSet {
			list = &skills.SkillSets[i].Skills
		}
	}
	return list
}

func isRemoved(s *pob2.Skill) bool { return s.Removed != nil && *s.Removed }

// savedPositions is the slice position of each saved group that is not removed, in order.
func savedPositions(list []pob2.Skill) []int {
	var out []int
	for i := range list {
		if !isRemoved(&list[i]) {
			out = append(out, i)
		}
	}
	return out
}

func strPtr(s string) *string { return &s }

// saveGeneratedGem is SkillsTab:Save's <Gem> for a gem the calculation made or rewrote, over the saved one it
// rewrote.
//
//pob2:block:src/Classes/SkillsTab.lua:497-526
//pob2:block:src/Modules/CalcSetup.lua:1943-1965
func saveGeneratedGem(gemInstance *GemInstance, gem pob2.Gem) pob2.Gem {
	gem.NameSpec = gemInstance.NameSpec
	gem.SkillID, gem.GemID, gem.VariantID = nil, nil, nil
	gem.Level = pob2.Nil[int]{V: gemInstance.Level}
	gem.Quality = pob2.Nil[int]{V: int(gemInstance.Quality)}
	gem.Enabled = gemInstance.Enabled
	gem.EnableGlobal1 = pob2.Nil[bool]{V: gemInstance.EnableGlobal1}
	gem.EnableGlobal2 = pob2.Nil[bool]{V: gemInstance.EnableGlobal2}
	gem.Count = pob2.Nil[int]{V: int(gemInstance.Count)}
	gem.Corrupted = pob2.Nil[bool]{V: gemInstance.Corrupted}
	gem.CorruptLevel = pob2.Nil[int]{V: gemInstance.CorruptLevel}
	if gemInstance.SkillID != "" {
		gem.SkillID = strPtr(gemInstance.SkillID)
	}
	if gemInstance.GemData != nil {
		gem.GemID = strPtr(gemInstance.GemData.Data.GameID)
		gem.VariantID = strPtr(gemInstance.GemData.Data.VariantID)
	}
	return gem
}

// saveGeneratedGroup is SkillsTab:Save's <Skill> for a group the calculation made.
//
//pob2:block:src/Classes/SkillsTab.lua:479-496
func saveGeneratedGroup(socketGroup *SocketGroup) pob2.Skill {
	skill := pob2.Skill{
		Enabled:              socketGroup.Enabled,
		IncludeInFullDPS:     pob2.Nil[bool]{IsNil: !socketGroup.IncludeInFullDPS, V: socketGroup.IncludeInFullDPS},
		Set1:                 socketGroup.Set1,
		Set2:                 socketGroup.Set2,
		MainActiveSkill:      pob2.Nil[int]{V: socketGroup.MainActiveSkill},
		MainActiveSkillCalcs: pob2.Nil[int]{V: socketGroup.MainActiveSkillCalcs},
	}
	skill.Label = strPtr(socketGroup.Label)
	if socketGroup.Slot != "" {
		skill.Slot = strPtr(socketGroup.Slot)
	}
	if socketGroup.Source != "" {
		skill.Source = strPtr(socketGroup.Source)
	}
	if socketGroup.HasGroupCount {
		n := int(socketGroup.GroupCount)
		skill.GroupCount = &n
	}
	for _, gemInstance := range socketGroup.GemList {
		skill.Gems = append(skill.Gems, saveGeneratedGem(gemInstance, pob2.Gem{}))
	}
	return skill
}

// SyncSocketGroups writes the calculation's socketGroupList over the active skill set's saved groups: a group it
// loaded keeps its entry, with the source, slot and gems the calculation gave it; a group it generated is saved
// after them; a saved group it dropped is gone. Removed groups stay. It writes once per calculation, as the edits
// after it change the list it would write over.
//
//pob2:block:src/Modules/CalcSetup.lua:1859-2100
func SyncSocketGroups(build *pob2.PathOfBuilding2, env *Environment) {
	if env == nil || env.socketGroupsSynced {
		return
	}
	env.socketGroupsSynced = true
	list := savedSkillList(build)
	positions := savedPositions(*list)
	groups := make([]pob2.Skill, 0, len(env.SocketGroupList))
	for _, socketGroup := range env.SocketGroupList {
		if socketGroup.savedIndex == 0 || socketGroup.savedIndex > len(positions) {
			groups = append(groups, saveGeneratedGroup(socketGroup))
			continue
		}
		skill := (*list)[positions[socketGroup.savedIndex-1]]
		skill.Gems = slices.Clone(skill.Gems)
		if socketGroup.Source != "" {
			skill.Source = strPtr(socketGroup.Source)
		}
		if socketGroup.Slot == "" {
			skill.Slot = nil
		} else {
			skill.Slot = strPtr(socketGroup.Slot)
		}
		// a group the calculation merged supports into, or gave a granted skill
		for i, gemInstance := range socketGroup.GemList {
			switch {
			case i >= len(skill.Gems):
				skill.Gems = append(skill.Gems, saveGeneratedGem(gemInstance, pob2.Gem{}))
			case i == 0 && socketGroup.Source != "":
				skill.Gems[0] = saveGeneratedGem(gemInstance, skill.Gems[0])
			}
		}
		groups = append(groups, skill)
	}
	var removed []pob2.Skill
	for i := range *list {
		if isRemoved(&(*list)[i]) {
			removed = append(removed, (*list)[i])
		}
	}
	*list = append(groups, removed...)
	for i, socketGroup := range env.SocketGroupList {
		socketGroup.savedIndex = i + 1
	}
}

// savedGroup is socketGroupList[index]'s saved entry after a sync.
func savedGroup(build *pob2.PathOfBuilding2, index int) *pob2.Skill {
	list := savedSkillList(build)
	positions := savedPositions(*list)
	if index < 1 || index > len(positions) {
		return nil
	}
	return &(*list)[positions[index-1]]
}

func groupSource(s *pob2.Skill) string {
	if s.Source == nil {
		return ""
	}
	return *s.Source
}

// calcsSkillNumber is calcsTab.input.skill_number, when the build saved one.
func calcsSkillNumber(build *pob2.PathOfBuilding2) *float64 {
	for i := range build.Calcs.Inputs {
		if build.Calcs.Inputs[i].Name == "skill_number" {
			return build.Calcs.Inputs[i].Number
		}
	}
	return nil
}

// NewSocketGroup is the socket group list's New button; it returns the new group's index.
//
//pob2:block:src/Classes/SkillListControl.lua:58-71
//pob2:block:src/Classes/SkillsTab.lua:1384-1390
func NewSocketGroup(build *pob2.PathOfBuilding2, env *Environment) int {
	SyncSocketGroups(build, env)
	list := savedSkillList(build)
	positions := savedPositions(*list)
	newGroup := pob2.Skill{
		Enabled:              true,
		Label:                strPtr(""),
		IncludeInFullDPS:     pob2.Nil[bool]{IsNil: true},
		Set1:                 true,
		Set2:                 true,
		MainActiveSkill:      pob2.Nil[int]{V: 1},
		MainActiveSkillCalcs: pob2.Nil[int]{V: 1},
	}
	*list = slices.Insert(*list, len(positions), newGroup)
	return len(positions) + 1
}

// DeleteSocketGroup is the socket group list's Delete, confirmed.
//
//pob2:src/Classes/SkillListControl.lua:139-173
func DeleteSocketGroup(build *pob2.PathOfBuilding2, env *Environment, index int) error {
	SyncSocketGroups(build, env)
	socketGroup := savedGroup(build, index)
	if socketGroup == nil {
		return errNoSocketGroup
	}
	if socketGroup.Source != nil {
		return errSourceGroup
	}
	list := savedSkillList(build)
	*list = slices.Delete(*list, savedPositions(*list)[index-1], savedPositions(*list)[index-1]+1)
	if build.Build.MainSocketGroup > index {
		build.Build.MainSocketGroup--
	}
	if n := calcsSkillNumber(build); n != nil && int(*n) > index {
		*n--
	}
	return nil
}

// DeleteAllSocketGroups is the socket group list's Delete All, confirmed.
//
//pob2:block:src/Classes/SkillListControl.lua:45-54
func DeleteAllSocketGroups(build *pob2.PathOfBuilding2) {
	list := savedSkillList(build)
	kept := (*list)[:0]
	for _, s := range *list {
		if isRemoved(&s) {
			kept = append(kept, s)
		}
	}
	*list = kept
}

// MoveSocketGroup is a drag in the socket group list: the group at from ends up at to, and the main group and
// the Calcs tab's skill follow it.
//
//pob2:block:src/Classes/ListControl.lua:440-448
//pob2:src/Classes/SkillListControl.lua:108-127
func MoveSocketGroup(build *pob2.PathOfBuilding2, env *Environment, selIndex, selDragIndex int) error {
	SyncSocketGroups(build, env)
	list := savedSkillList(build)
	positions := savedPositions(*list)
	if selIndex < 1 || selIndex > len(positions) || selDragIndex < 1 || selDragIndex > len(positions) {
		return errNoSocketGroup
	}
	if selIndex == selDragIndex {
		return nil
	}
	groups := make([]pob2.Skill, len(positions))
	for i, p := range positions {
		groups[i] = (*list)[p]
	}
	moved := groups[selIndex-1]
	groups = slices.Delete(groups, selIndex-1, selIndex)
	groups = slices.Insert(groups, selDragIndex-1, moved)
	for i, p := range positions {
		(*list)[p] = groups[i]
	}
	follow := func(index int) int {
		switch {
		case index == selIndex:
			return selDragIndex
		case index > selIndex && index <= selDragIndex:
			return index - 1
		case index < selIndex && index >= selDragIndex:
			return index + 1
		}
		return index
	}
	build.Build.MainSocketGroup = follow(build.Build.MainSocketGroup)
	if n := calcsSkillNumber(build); n != nil {
		*n = float64(follow(int(*n)))
	}
	return nil
}

// editGroup syncs and finds socketGroupList[index]'s saved entry.
func editGroup(build *pob2.PathOfBuilding2, env *Environment, index int) (*pob2.Skill, error) {
	SyncSocketGroups(build, env)
	socketGroup := savedGroup(build, index)
	if socketGroup == nil {
		return nil, errNoSocketGroup
	}
	return socketGroup, nil
}

// SetSocketGroupLabel is the group's Label box.
//
//pob2:block:src/Classes/SkillsTab.lua:151-156
func SetSocketGroupLabel(build *pob2.PathOfBuilding2, env *Environment, index int, label string) error {
	socketGroup, err := editGroup(build, env, index)
	if err != nil {
		return err
	}
	socketGroup.Label = strPtr(label)
	return nil
}

// SetSocketGroupEnabled is the group's Enabled box, and Ctrl+click in the list.
//
//pob2:block:src/Classes/SkillsTab.lua:202-206
func SetSocketGroupEnabled(build *pob2.PathOfBuilding2, env *Environment, index int, state bool) error {
	socketGroup, err := editGroup(build, env, index)
	if err != nil {
		return err
	}
	socketGroup.Enabled, socketGroup.Active = state, nil
	return nil
}

// SetSocketGroupIncludeInFullDPS is the group's Include in Full DPS box, and Ctrl+right click in the list.
//
//pob2:block:src/Classes/SkillsTab.lua:220-224
func SetSocketGroupIncludeInFullDPS(build *pob2.PathOfBuilding2, env *Environment, index int, state bool) error {
	socketGroup, err := editGroup(build, env, index)
	if err != nil {
		return err
	}
	socketGroup.IncludeInFullDPS = pob2.Nil[bool]{V: state}
	return nil
}

// SetSocketGroupCount is the Count box of a group with a source.
//
//pob2:block:src/Classes/SkillsTab.lua:229-233
func SetSocketGroupCount(build *pob2.PathOfBuilding2, env *Environment, index, count int) error {
	socketGroup, err := editGroup(build, env, index)
	if err != nil {
		return err
	}
	if count == 0 {
		count = 1
	}
	socketGroup.GroupCount = &count
	return nil
}

// SetSocketGroupWeaponSet is the group's Set 1 or Set 2 box: a group stays in at least one weapon set.
//
//pob2:block:src/Classes/SkillsTab.lua:157-165
func SetSocketGroupWeaponSet(build *pob2.PathOfBuilding2, env *Environment, index, set int, state bool) error {
	socketGroup, err := editGroup(build, env, index)
	if err != nil {
		return err
	}
	if !state && (set == 1 && !socketGroup.Set2 || set == 2 && !socketGroup.Set1) {
		return nil
	}
	if set == 1 {
		socketGroup.Set1 = state
	} else {
		socketGroup.Set2 = state
	}
	return nil
}

// editGem syncs and finds gemList[gem] of socketGroupList[group]; a gem the group's source grants is not editable.
//
//pob2:block:src/Classes/SkillsTab.lua:764-766
func editGem(build *pob2.PathOfBuilding2, env *Environment, group, gem int) (*pob2.Gem, error) {
	socketGroup, err := editGroup(build, env, group)
	if err != nil {
		return nil, err
	}
	if gem < 1 || gem > len(socketGroup.Gems) {
		return nil, errNoGem
	}
	if gem == 1 && socketGroup.Source != nil {
		return nil, errGeneratedGem
	}
	return &socketGroup.Gems[gem-1], nil
}

func defaultGemQuality(build *pob2.PathOfBuilding2) int {
	if q := build.Skills.DefaultGemQuality; !q.IsNil {
		return max(min(q.V, 23), 0)
	}
	return 0
}

// defaultGemLevel is skillsTab.defaultGemLevel as Load reads it.
//
//pob2:block:src/Classes/SkillsTab.lua:405-413
func defaultGemLevel(build *pob2.PathOfBuilding2) string {
	skills := &build.Skills
	if skills.MatchGemLevelToCharacterLevel != nil && *skills.MatchGemLevelToCharacterLevel {
		return "characterLevel"
	}
	if skills.DefaultGemLevel != nil {
		switch *skills.DefaultGemLevel {
		case "normalMaximum", "corruptedMaximum", "characterLevel":
			return *skills.DefaultGemLevel
		}
	}
	return "normalMaximum"
}

// processGemLevel is SkillsTab:ProcessGemLevel: the level a gem picked in the gem list starts at, and whether the
// default corrupts it.
//
//pob2:src/Classes/SkillsTab.lua:1259-1294
func processGemLevel(build *pob2.PathOfBuilding2, gemData *poe2skills.Gem) (int, bool) {
	if gemData == nil {
		return 1, false
	}
	grantedEffect := gemData.GrantedEffect
	naturalMaxLevel := int(gemData.Data.NaturalMaxLevel)
	switch defaultGemLevel(build) {
	case "corruptedMaximum":
		// no PoE2 skill has a plusVersionOf
		return naturalMaxLevel, true
	case "normalMaximum":
		return naturalMaxLevel, false
	}
	maxGemLevel := naturalMaxLevel
	if grantedEffect.Levels[maxGemLevel] == nil {
		maxGemLevel = levelCount(grantedEffect)
	}
	characterLevel := float64(max(build.Build.Level, 1))
	for gemLevel := maxGemLevel; gemLevel >= 1; gemLevel-- {
		if req, _ := levelRequirement(grantedEffect, gemLevel); req <= characterLevel {
			return gemLevel, false
		}
	}
	return 1, false
}

// SetGem is a gem slot's name box picking gemID: the slot past the last gem adds one.
//
//pob2:block:src/Classes/SkillsTab.lua:815-875
//pob2:block:src/Classes/SkillsTab.lua:1305-1317
func SetGem(build *pob2.PathOfBuilding2, env *Environment, group, gem int, gemID string) error {
	socketGroup, err := editGroup(build, env, group)
	if err != nil {
		return err
	}
	if gem == 1 && socketGroup.Source != nil {
		return errGeneratedGem
	}
	if gem < 1 || gem > len(socketGroup.Gems)+1 {
		return errNoGem
	}
	data, err := loadSkillData()
	if err != nil {
		return err
	}
	gemData := data.gem(gemID)
	if gemData == nil || gemData.GrantedEffect == nil {
		return errUnknownGem
	}
	if gem == len(socketGroup.Gems)+1 {
		socketGroup.Gems = append(socketGroup.Gems, pob2.Gem{
			Level:         pob2.Nil[int]{V: 1},
			Quality:       pob2.Nil[int]{V: defaultGemQuality(build)},
			Enabled:       true,
			EnableGlobal1: pob2.Nil[bool]{V: true},
			EnableGlobal2: pob2.Nil[bool]{V: true},
			Count:         pob2.Nil[int]{V: 1},
			Corrupted:     pob2.Nil[bool]{V: false},
			CorruptLevel:  pob2.Nil[int]{V: 0},
		})
	} else if g := &socketGroup.Gems[gem-1]; g.GemID != nil && g.VariantID != nil && *g.GemID == gemData.Data.GameID && *g.VariantID == gemData.Data.VariantID {
		return nil
	}
	gemInstance := &socketGroup.Gems[gem-1]
	gemInstance.NameSpec = gemData.Data.Name
	gemInstance.SkillID = strPtr(gemData.Data.GrantedEffectID)
	gemInstance.GemID = strPtr(gemData.Data.GameID)
	gemInstance.VariantID = strPtr(gemData.Data.VariantID)
	level, corrupt := processGemLevel(build, gemData)
	gemInstance.Level = pob2.Nil[int]{V: level}
	if corrupt {
		gemInstance.Corrupted = pob2.Nil[bool]{V: true}
		gemInstance.CorruptLevel = pob2.Nil[int]{V: 1}
	}
	return nil
}

// RemoveGem is a gem slot's x button.
//
//pob2:block:src/Classes/SkillsTab.lua:773-789
func RemoveGem(build *pob2.PathOfBuilding2, env *Environment, group, gem int) error {
	if _, err := editGem(build, env, group, gem); err != nil {
		return err
	}
	socketGroup := savedGroup(build, group)
	socketGroup.Gems = slices.Delete(socketGroup.Gems, gem-1, gem)
	return nil
}

// savedGemData is the gem data LoadSkill and ProcessSocketGroup resolve a saved gem to.
func savedGemData(data *skillData, g *pob2.Gem) (*poe2skills.Gem, *poe2skills.GrantedEffect) {
	gemInstance := loadGemInstance(&Environment{data: data}, g)
	switch {
	case gemInstance.GemID != "":
		if gemData := data.gem(gemInstance.GemID); gemData != nil {
			return gemData, gemData.GrantedEffect
		}
	case gemInstance.SkillID != "":
		grantedEffect := data.effect(gemInstance.SkillID)
		if gemID, ok := data.gemForSkill[grantedEffect]; ok && grantedEffect != nil {
			gemData := data.gem(gemID)
			return gemData, gemData.GrantedEffect
		}
		return nil, grantedEffect
	}
	return nil, nil
}

// SetGemLevel is a gem slot's Level box; 0 is an empty box. ProcessSocketGroup then holds the level to one the
// gem has.
//
//pob2:block:src/Classes/SkillsTab.lua:883-898
//pob2:block:src/Classes/SkillsTab.lua:1370
func SetGemLevel(build *pob2.PathOfBuilding2, env *Environment, group, gem, level int) error {
	gemInstance, err := editGem(build, env, group, gem)
	if err != nil {
		return err
	}
	data, err := loadSkillData()
	if err != nil {
		return err
	}
	gemData, grantedEffect := savedGemData(data, gemInstance)
	if level == 0 {
		level = 20
		if gemData != nil {
			level = int(gemData.Data.NaturalMaxLevel)
		}
	}
	if grantedEffect != nil {
		level = validateGemLevel(grantedEffect, gemData, level)
	}
	gemInstance.Level = pob2.Nil[int]{V: level}
	return nil
}

// SetGemQuality is a gem slot's Quality box; a negative quality is an empty box.
//
//pob2:block:src/Classes/SkillsTab.lua:909-924
func SetGemQuality(build *pob2.PathOfBuilding2, env *Environment, group, gem, quality int) error {
	gemInstance, err := editGem(build, env, group, gem)
	if err != nil {
		return err
	}
	if quality < 0 {
		quality = defaultGemQuality(build)
	}
	gemInstance.Quality = pob2.Nil[int]{V: quality}
	return nil
}

// SetGemEnabled is a gem slot's Enabled box; PoE2 has no Vaal gems, so both global effects are enabled with it.
//
//pob2:block:src/Classes/SkillsTab.lua:1027-1047
func SetGemEnabled(build *pob2.PathOfBuilding2, env *Environment, group, gem int, state bool) error {
	gemInstance, err := editGem(build, env, group, gem)
	if err != nil {
		return err
	}
	gemInstance.EnableGlobal1 = pob2.Nil[bool]{V: true}
	gemInstance.EnableGlobal2 = pob2.Nil[bool]{V: true}
	gemInstance.Enabled = state
	return nil
}

// SetGemCount is a gem slot's Count box; 0 is an empty box.
//
//pob2:block:src/Classes/SkillsTab.lua:1070-1085
func SetGemCount(build *pob2.PathOfBuilding2, env *Environment, group, gem, count int) error {
	gemInstance, err := editGem(build, env, group, gem)
	if err != nil {
		return err
	}
	if count == 0 {
		count = 1
	}
	gemInstance.Count = pob2.Nil[int]{V: count}
	return nil
}

// CorruptOptions are the Corrupt dropdown's entries.
//
//pob2:block:src/Classes/SkillsTab.lua:44-49
var CorruptOptions = []struct {
	Label string
	Level int
}{
	{"Not Corrupted", 0},
	{"+1 to Gem Level", 1},
	{"Corrupted", 0},
	{"-1 to Gem Level", -1},
}

// SetGemCorruption is a gem slot's Corrupt dropdown picking CorruptOptions[option-1].
//
//pob2:block:src/Classes/SkillsTab.lua:1116-1132
func SetGemCorruption(build *pob2.PathOfBuilding2, env *Environment, group, gem, option int) error {
	gemInstance, err := editGem(build, env, group, gem)
	if err != nil {
		return err
	}
	if option < 1 || option > len(CorruptOptions) {
		return errNoGem
	}
	value := CorruptOptions[option-1]
	gemInstance.CorruptLevel = pob2.Nil[int]{V: value.Level}
	gemInstance.Corrupted = pob2.Nil[bool]{V: value.Label != "Not Corrupted"}
	return nil
}

// CorruptIndex is SkillsTab:GetCorruptIndex: the Corrupt dropdown's entry for a gem.
//
//pob2:src/Classes/SkillsTab.lua:276-286
func CorruptIndex(gemInstance *GemInstance) int {
	switch {
	case gemInstance.CorruptLevel == 1:
		return 2
	case gemInstance.CorruptLevel == -1:
		return 4
	case gemInstance.Corrupted:
		return 3
	}
	return 1
}

// GemOptions are the Skills tab's gem options as Load reads them.
type GemOptions struct {
	DefaultGemLevel     string
	DefaultGemQuality   int
	ShowSupportGemTypes string
	ShowLegacyGems      bool
	SortGemsByDPS       bool
	SortGemsByDPSField  string
}

// ReadGemOptions is the gem options Load reads.
//
//pob2:block:src/Classes/SkillsTab.lua:405-427
func ReadGemOptions(build *pob2.PathOfBuilding2) GemOptions {
	skills := &build.Skills
	out := GemOptions{
		DefaultGemLevel:     defaultGemLevel(build),
		DefaultGemQuality:   defaultGemQuality(build),
		ShowSupportGemTypes: "ALL",
		ShowLegacyGems:      skills.ShowLegacyGems,
		SortGemsByDPS:       true,
		SortGemsByDPSField:  "CombinedDPS",
	}
	if v := skills.ShowSupportGemTypes; v != nil && (*v == "ALL" || *v == "LINEAGE" || *v == "NORMAL") {
		out.ShowSupportGemTypes = *v
	}
	if v := skills.SortGemsByDPSField; v != nil && *v != "" {
		out.SortGemsByDPSField = *v
	}
	if _, ok := skills.ExtraAttrib["sortGemsByDPS"]; ok || skills.SortGemsByDPS {
		out.SortGemsByDPS = skills.SortGemsByDPS
	}
	return out
}

// SetGemOptions is the gem option controls; Save writes every option.
//
//pob2:block:src/Classes/SkillsTab.lua:114-140
//pob2:block:src/Classes/SkillsTab.lua:452-460
func SetGemOptions(build *pob2.PathOfBuilding2, options GemOptions) {
	skills := &build.Skills
	skills.DefaultGemLevel = strPtr(options.DefaultGemLevel)
	skills.MatchGemLevelToCharacterLevel = nil
	skills.DefaultGemQuality = pob2.Nil[int]{V: max(min(options.DefaultGemQuality, 23), 0)}
	skills.ShowSupportGemTypes = strPtr(options.ShowSupportGemTypes)
	skills.ShowLegacyGems = options.ShowLegacyGems
	skills.SortGemsByDPS = options.SortGemsByDPS
	skills.SortGemsByDPSField = strPtr(options.SortGemsByDPSField)
}

// GemListEntry is one gem the gem list can offer, with what its search and filters read.
type GemListEntry struct {
	ID        string
	Name      string
	Tags      []string
	Support   bool
	IsLineage bool
	Legacy    bool
	// LevelRequirement is the level requirement of the gem's first level.
	LevelRequirement float64
	// Color is grantedEffect.color: 1 strength, 2 dexterity, 3 intelligence.
	Color int
}

// GemList is data.gems less the gems an item or the tree grants, sorted by id; the gem list's own filters and
// search run over it.
//
//pob2:block:src/Classes/GemSelectControl.lua:104-122
func GemList() ([]GemListEntry, error) {
	data, err := loadSkillData()
	if err != nil {
		return nil, err
	}
	var out []GemListEntry
	ids := make([]string, len(poe2.Gems))
	for i := range poe2.Gems {
		ids[i] = poe2.Gems[i].ID
	}
	sort.Strings(ids)
	for _, id := range ids {
		gemData := data.gem(id)
		if gemData == nil {
			continue
		}
		grantedEffect := gemData.GrantedEffect
		if grantedEffect == nil || grantedEffect.Overlay != nil && (grantedEffect.Overlay.FromItem || grantedEffect.Overlay.FromTree) {
			continue
		}
		levelReq := 1.0
		if req, ok := levelRequirement(grantedEffect, 1); ok {
			levelReq = req
		}
		tags := make([]string, 0, len(gemData.Data.Tags))
		tags = append(tags, gemData.Data.Tags...)
		sort.Strings(tags)
		out = append(out, GemListEntry{
			ID:               id,
			Name:             gemData.Data.Name,
			Tags:             tags,
			Support:          grantedEffect.Support,
			IsLineage:        grantedEffect.IsLineage,
			Legacy:           grantedEffect.Overlay != nil && grantedEffect.Overlay.Legacy,
			LevelRequirement: levelReq,
			Color:            grantedEffect.Color,
		})
	}
	return out, nil
}
