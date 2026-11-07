package calculator2

import (
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2skills"
	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob2"
)

// GemName and GemTags give calcLib.gemIsType the effect's gemData; Corrupted is the effect's own flag.
func (e *GemEffect) GemName() string          { return e.GemData.Data.Name }
func (e *GemEffect) GemTags() map[string]bool { return gemTags(e.GemData.Data) }
func (e *GemEffect) GemCorrupted() bool       { return e.Corrupted }

// loadData is env.data: data.skills and its gem lookups.
func (env *Environment) loadData() *skillData {
	if env.data == nil {
		d, err := loadSkillData()
		if err != nil {
			panic("skill data: " + err.Error())
		}
		env.data = d
	}
	return env.data
}

func (env *Environment) effectFromItem(ge *poe2skills.GrantedEffect) bool {
	return ge != nil && (ge.Overlay != nil && ge.Overlay.FromItem || env.fromItemEffects[ge])
}

// loadSocketGroups is SkillsTab:Load for the active skill set: each group as LoadSkill reads it, then
// ProcessSocketGroup. A gem saved with only a nameSpec is not looked up by FindSkillGem.
//
//pob2:src/Classes/SkillsTab.lua:288-399
func loadSocketGroups(env *Environment) []*SocketGroup {
	saved := activeSkillSet(env)
	list := make([]*SocketGroup, 0, len(saved))
	for i := range saved {
		node := &saved[i]
		group := &SocketGroup{savedIndex: i + 1}
		group.Enabled = (node.Active != nil && *node.Active) || node.Enabled
		group.IncludeInFullDPS = !node.IncludeInFullDPS.IsNil && node.IncludeInFullDPS.V
		if node.GroupCount != nil {
			group.GroupCount, group.HasGroupCount = float64(*node.GroupCount), true
		}
		if node.Label != nil {
			group.Label = *node.Label
		}
		if node.Source != nil {
			group.Source = *node.Source
		}
		if node.Slot != nil {
			group.Slot = *node.Slot
		}
		group.Set1 = node.Set1
		group.Set2 = node.Set2
		group.MainActiveSkill = 1
		if !node.MainActiveSkill.IsNil {
			group.MainActiveSkill = node.MainActiveSkill.V
		}
		group.MainActiveSkillCalcs = 1
		if !node.MainActiveSkillCalcs.IsNil {
			group.MainActiveSkillCalcs = node.MainActiveSkillCalcs.V
		}
		for j := range node.Gems {
			group.GemList = append(group.GemList, loadGemInstance(env, &node.Gems[j]))
		}
		processSocketGroup(env, group)
		list = append(list, group)
	}
	return list
}

//pob2:block:src/Classes/SkillsTab.lua:308-386
func loadGemInstance(env *Environment, child *pob2.Gem) *GemInstance {
	gemInstance := &GemInstance{NameSpec: sanitiseText(child.NameSpec)}
	switch {
	case child.GemID != nil:
		var gemData *poe2skills.Gem
		if possibleVariants := env.data.gemsByGameID[*child.GemID]; possibleVariants != nil {
			if child.VariantID != nil && possibleVariants[*child.VariantID] != nil {
				// New save format from 3.23 that stores the specific variation (transfiguration)
				gemData = possibleVariants[*child.VariantID]
			} else {
				// If a gem has changed names between updates, assumed it's the first gem in the list
				variants := make([]string, 0, len(possibleVariants))
				for v := range possibleVariants {
					variants = append(variants, v)
				}
				sort.Strings(variants)
				gemData = possibleVariants[variants[0]]
			}
		}
		if gemData != nil {
			gemInstance.GemID = gemData.Data.ID
			gemInstance.SkillID = gemData.Data.GrantedEffectID
		}
	case child.SkillID != nil:
		if grantedEffect := env.data.effect(*child.SkillID); grantedEffect != nil {
			gemInstance.GemID = env.data.gemForSkill[grantedEffect]
			gemInstance.SkillID = grantedEffect.ID
			gemInstance.NameSpec = grantedEffect.Name
		}
	}
	if !child.Level.IsNil {
		gemInstance.Level = child.Level.V
	}
	if !child.Quality.IsNil {
		gemInstance.Quality = float64(child.Quality.V)
	}
	gemInstance.Enabled = child.Enabled
	gemInstance.EnableGlobal1 = child.EnableGlobal1.IsNil || child.EnableGlobal1.V
	gemInstance.EnableGlobal2 = !child.EnableGlobal2.IsNil && child.EnableGlobal2.V
	gemInstance.Count = 1
	if !child.Count.IsNil {
		gemInstance.Count = float64(child.Count.V)
	}
	gemInstance.Corrupted = !child.Corrupted.IsNil && child.Corrupted.V
	if !child.CorruptLevel.IsNil {
		gemInstance.CorruptLevel = child.CorruptLevel.V
	}
	gemInstance.StatSet = map[string]int{}
	gemInstance.StatSetCalcs = map[string]int{}
	for _, s := range child.StatSetIndexes {
		if s.GrantedEffect != "" && !s.Index.IsNil {
			gemInstance.StatSet[s.GrantedEffect] = s.Index.V
		}
	}
	for _, s := range child.StatSetCalcsIndexes {
		if s.GrantedEffect != "" && !s.Index.IsNil {
			gemInstance.StatSetCalcs[s.GrantedEffect] = s.Index.V
		}
	}
	//pob2:block:src/Classes/SkillsTab.lua:357-360
	if child.SkillMinion != nil {
		gemInstance.SkillMinion = *child.SkillMinion
	}
	if child.SkillMinionCalcs != nil {
		gemInstance.SkillMinionCalcs = *child.SkillMinionCalcs
	}
	if child.SkillMinionItemSet != nil {
		gemInstance.SkillMinionItemSet = *child.SkillMinionItemSet
	}
	if child.SkillMinionItemSetCalcs != nil {
		gemInstance.SkillMinionItemSetCalcs = *child.SkillMinionItemSetCalcs
	}
	return gemInstance
}

// processSocketGroup fills in the gem data, levels and requirements of a group's gems, and its weapon
// sets. A gem's level is not reset to a changed naturalMaxLevel (no gem here has a previous default).
//
//pob2:src/Classes/SkillsTab.lua:1296-1391
func processSocketGroup(env *Environment, socketGroup *SocketGroup) {
	for _, gemInstance := range socketGroup.GemList {
		gemInstance.GemData, gemInstance.GrantedEffect = nil, nil
		switch {
		case gemInstance.GemID != "":
			// Specified by gem ID
			// Used for skills granted by skill gems
			gemInstance.GemData = env.data.gem(gemInstance.GemID)
			if gemInstance.GemData != nil {
				if !strings.HasPrefix(gemInstance.NameSpec, "Companion:") && !strings.HasPrefix(gemInstance.NameSpec, "Spectre:") {
					gemInstance.NameSpec = gemInstance.GemData.Data.Name
				}
				gemInstance.SkillID = gemInstance.GemData.Data.GrantedEffectID
			}
		case gemInstance.SkillID != "":
			// Specified by skill ID
			// Used for skills granted by items and passive tree nodes
			grantedEffect := env.data.effect(gemInstance.SkillID)
			if gemID, ok := env.data.gemForSkill[grantedEffect]; ok && grantedEffect != nil {
				gemInstance.GemData = env.data.gem(gemID)
				gemInstance.NameSpec = gemInstance.GemData.Data.Name
			} else {
				gemInstance.GrantedEffect = grantedEffect
			}
		}
		if gemInstance.GemData != nil && gemInstance.GemData.GrantedEffect.Overlay.HideFromSideBar {
			gemInstance.GemData = nil
		}
		if gemInstance.GemData != nil || gemInstance.GrantedEffect != nil {
			grantedEffect := gemInstance.GrantedEffect
			if grantedEffect == nil {
				grantedEffect = gemInstance.GemData.GrantedEffect
			}
			gemInstance.Level = validateGemLevel(gemInstance.GrantedEffect, gemInstance.GemData, gemInstance.Level)
			if gemInstance.GemData != nil {
				gemInstance.ReqLevel, _ = levelRequirement(grantedEffect, gemInstance.Level)
				gemInstance.ReqStr = getGemStatRequirement(gemInstance.ReqLevel, gemInstance.GemData.Data.ReqStr, grantedEffect.Support)
				gemInstance.ReqDex = getGemStatRequirement(gemInstance.ReqLevel, gemInstance.GemData.Data.ReqDex, grantedEffect.Support)
				gemInstance.ReqInt = getGemStatRequirement(gemInstance.ReqLevel, gemInstance.GemData.Data.ReqInt, grantedEffect.Support)
			}
		}
	}
	var sourceSlot *ItemSlot
	if (socketGroup.SourceItem != nil || socketGroup.Source == "Default Attack") && socketGroup.Slot != "" && env.items != nil {
		sourceSlot = env.items.Slots[socketGroup.Slot]
	}
	if sourceSlot != nil && sourceSlot.WeaponSet != 0 {
		socketGroup.Set1 = sourceSlot.WeaponSet != 2
		socketGroup.Set2 = sourceSlot.WeaponSet != 1
	}
	if !socketGroup.Set1 && !socketGroup.Set2 {
		socketGroup.Set1 = true
		socketGroup.Set2 = true
	}
}

// getSocketGroupWeaponSet is SkillsTab:GetSocketGroupWeaponSet.
//
//pob2:src/Classes/SkillsTab.lua:1398-1405
func getSocketGroupWeaponSet(env *Environment, socketGroup *SocketGroup) int {
	if socketGroup == nil || socketGroup.Set1 && socketGroup.Set2 {
		return useSecondWeaponSet(env)
	}
	if socketGroup.Set2 {
		return 2
	}
	return 1
}

//pob2:src/Modules/CalcSetup.lua:543-548
func getGemModList(env *Environment, groupCfg *moddb.ListCfg, socketColor string, socketNum int) []gemPropertyMod {
	gemCfg := *groupCfg
	gemCfg.SocketColor = socketColor
	gemCfg.SocketNum = socketNum
	return tabulateGemProperties(env, &gemCfg)
}

// gemPropertyMod is a modDB:Tabulate("LIST", cfg, "GemProperty") row: the evaluated value, and its mod's source
// and first tag.
type gemPropertyMod struct {
	Value  any
	Source mod.Source
	Tag    mod.Tag
}

// tabulateGemProperties pairs List's evaluated values with Tabulate's mods, which both walk in one order.
func tabulateGemProperties(env *Environment, cfg *moddb.ListCfg) []gemPropertyMod {
	values := env.ModDB.List(cfg, "GemProperty")
	rows := env.ModDB.Tabulate(mod.TypeList, cfg, "GemProperty")
	out := make([]gemPropertyMod, len(values))
	for i, v := range values {
		out[i].Value = v
		if i < len(rows) && len(rows) == len(values) {
			out[i].Source = rows[i].Mod.GetSource()
			if tags := rows[i].Mod.Tags(); len(tags) > 0 {
				out[i].Tag = tags[0]
			}
		}
	}
	return out
}

// gemPropertyKeyword is GemProperty's keyword, "" when it has none.
func gemPropertyKeyword(p mod.GemProperty) string {
	if p.Keyword == nil {
		return ""
	}
	return *p.Keyword
}

//pob2:src/Modules/CalcSetup.lua:550-575
func applyGemMods(effect *GemEffect, modList []gemPropertyMod) {
	for _, row := range modList {
		value, ok := row.Value.(mod.GemProperty)
		if !ok {
			continue
		}
		match := true
		// A cloned GemProperty carries an empty KeywordList where the parsed one had none.
		if len(value.KeywordList) > 0 {
			for _, keyword := range value.KeywordList {
				if !gemIsType(effect, keyword, true) {
					match = false
					break
				}
			}
		} else if !gemIsType(effect, gemPropertyKeyword(value), true) {
			match = false
		}
		for req, val := range value.GemRequirements {
			match = match && gemRequirement(effect.GemData, req) > val
		}
		if match {
			switch value.Key {
			case "level":
				effect.Level += int(value.Value)
			case "quality":
				effect.Quality += value.Value
			default:
				if effect.Values == nil {
					effect.Values = map[string]float64{}
				}
				effect.Values[value.Key] += value.Value
			}
			effect.GemPropertyInfo = append(effect.GemPropertyInfo, gemPropertyInfo{Value: value, Source: row.Source, Tag: row.Tag})
		}
	}
}

// gemRequirement is gemData[req] for the reqStr/reqDex/reqInt keys a GemProperty's gemRequirements names.
func gemRequirement(gem *poe2skills.Gem, req string) float64 {
	if gem == nil || gem.Data == nil {
		return 0
	}
	switch req {
	case "reqStr":
		return gem.Data.ReqStr
	case "reqDex":
		return gem.Data.ReqDex
	case "reqInt":
		return gem.Data.ReqInt
	}
	return 0
}

// socketCfg.skillGem has no PoE2 field on ListCfg; the gem is passed as GemData.
//
//pob2:src/Modules/CalcSetup.lua:577-584
func applySocketMods(env *Environment, gem *poe2skills.Gem, groupCfg *moddb.ListCfg, socketNum int, modSource string) {
	socketCfg := *groupCfg
	socketCfg.GemData = gem
	socketCfg.SocketNum = socketNum
	if modSource == "" {
		modSource = groupCfg.SlotName
	}
	for _, v := range env.ModDB.List(&socketCfg, "SocketProperty") {
		if p, ok := v.(mod.SocketProperty); ok {
			env.Player.ModDB.AddMod(setSource(p.Mod.Clone(), modSource))
		}
	}
}

// No PoE2 effect carries plusVersionOf, so its two branches never match.
//
//pob2:src/Modules/CalcSetup.lua:586-628
func addBestSupport(supportEffect *GemEffect, appliedSupportList *[]*GemEffect, mode string) {
	add := true
	for index, otherSupport := range *appliedSupportList {
		// Check if there's another better support already present
		if supportEffect.GrantedEffect == otherSupport.GrantedEffect {
			add = false
			if supportEffect.Level > otherSupport.Level || (supportEffect.Level == otherSupport.Level && supportEffect.Quality > otherSupport.Quality) {
				if mode == "MAIN" {
					otherSupport.Superseded = true
				}
				(*appliedSupportList)[index] = supportEffect
			} else {
				supportEffect.Superseded = true
			}
			break
		} else if supportEffect.GrantedEffect.GemFamily != nil && otherSupport.GrantedEffect.GemFamily != nil {
			for _, family := range supportEffect.GrantedEffect.GemFamily {
				for _, otherFamily := range otherSupport.GrantedEffect.GemFamily {
					if family == otherFamily {
						add = false
						if mode == "MAIN" {
							otherSupport.Superseded = true
						}
						(*appliedSupportList)[index] = supportEffect
						break
					}
				}
			}
		}
	}
	if add {
		*appliedSupportList = append(*appliedSupportList, supportEffect)
	}
}

// groupCfg is one groupCfgList entry: the cfg the group's gem lookups read, and its property list.
type groupCfg struct {
	cfg              *moddb.ListCfg
	propertyModList  []gemPropertyMod
	slotName         string
	intelligenceGems float64
	dexterityGems    float64
	strengthGems     float64
}

// socketedIn is playerItems[groupCfg.slotName].sockets[gemIndex]; PoE2 sockets carry no colour.
func socketedIn(env *Environment, slotName string, gemIndex int) bool {
	item := env.Player.ItemList[slotName]
	return item != nil && gemIndex >= 1 && gemIndex <= len(item.Sockets)
}

// countSocketedGem keeps track of the gem count for each colour socketed in a group, once per gem.
func countSocketedGem(env *Environment, gemInstance *GemInstance, gc *groupCfg, gemIndex int, processedSockets map[*GemInstance]bool) {
	if processedSockets[gemInstance] {
		return
	}
	processedSockets[gemInstance] = true
	itemName := ""
	if item := env.Player.ItemList[gc.slotName]; item != nil {
		itemName = item.Name
	}
	applySocketMods(env, gemInstance.GemData, gc.cfg, gemIndex, itemName)
	tags := gemTags(gemInstance.GemData.Data)
	if tags["intelligence"] {
		gc.intelligenceGems++
	}
	if tags["dexterity"] {
		gc.dexterityGems++
	}
	if tags["strength"] {
		gc.strengthGems++
	}
	gc.cfg.IntelligenceGems, gc.cfg.DexterityGems, gc.cfg.StrengthGems = gc.intelligenceGems, gc.dexterityGems, gc.strengthGems
}

// gemModsFor is the property list a gem at gemIndex reads.
func gemModsFor(env *Environment, gc *groupCfg, gemIndex int) []gemPropertyMod {
	if socketedIn(env, gc.slotName, gemIndex) {
		return getGemModList(env, gc.cfg, "", gemIndex)
	}
	return gc.propertyModList
}

//pob2:src/Modules/CalcSetup.lua:630-666
func processGrantedEffect(grantedEffect *poe2skills.GrantedEffect, gemInstance *GemInstance, env *Environment, gc *groupCfg, gemIndex int, processedSockets map[*GemInstance]bool, targetListList []*[]*GemEffect) {
	if grantedEffect == nil || !grantedEffect.Support {
		return
	}
	supportEffect := &GemEffect{
		GrantedEffect: grantedEffect,
		Level:         gemInstance.Level,
		Quality:       gemInstance.Quality,
		SrcInstance:   gemInstance,
		GemData:       gemInstance.GemData,
		IsSupporting:  map[*GemInstance]bool{},
	}
	if env.Mode == "MAIN" {
		gemInstance.DisplayEffect = supportEffect
		gemInstance.SupportEffect = supportEffect
	}
	if gemInstance.GemData != nil {
		applyGemMods(supportEffect, gemModsFor(env, gc, gemIndex))
		countSocketedGem(env, gemInstance, gc, gemIndex, processedSockets)
	}
	// Validate support gem level in case there is no active skill (and no full calculation)
	supportEffect.Level = validateGemLevel(supportEffect.GrantedEffect, supportEffect.GemData, supportEffect.Level)

	for _, targetList := range targetListList {
		addBestSupport(supportEffect, targetList, env.Mode)
	}
}

//pob2:src/Modules/CalcSetup.lua:667-676
func getNormalizedSkillLevel(env *Environment, grantedSkill *grantedSkill) int {
	// Levels in socketGroup.gemList[1].level are normalized
	// grantedSkill.level is not causing group match miss which causes all things that rely on group order to fail
	return validateGemLevel(env.data.effect(grantedSkill.SkillID), nil, grantedSkill.Level)
}

//pob2:src/Modules/CalcSetup.lua:678-703
func getGrantedSkillLevel(gemData *poe2skills.Gem, maxLevel int, characterLevel float64, modDB *moddb.ModDB) int {
	var str, dex, int_ float64
	if modDB != nil {
		str = math.Max(round(calcLibVal(modDB, "Str", nil)), 0)
		dex = math.Max(round(calcLibVal(modDB, "Dex", nil)), 0)
		int_ = math.Max(round(calcLibVal(modDB, "Int", nil)), 0)
	}
	for level := maxLevel; level >= 1; level-- {
		levelRequirement, ok := levelRequirement(gemData.GrantedEffect, level)
		if ok && levelRequirement <= characterLevel && (modDB == nil ||
			getGemStatRequirement(levelRequirement, gemData.Data.ReqStr, false) <= str &&
				getGemStatRequirement(levelRequirement, gemData.Data.ReqDex, false) <= dex &&
				getGemStatRequirement(levelRequirement, gemData.Data.ReqInt, false) <= int_) {
			return level
		}
	}
	// Some granted skills only define higher levels (Pinnacle of Power has only 20)
	if lowest := lowestLevel(gemData.GrantedEffect); lowest != 0 {
		return lowest
	}
	return 1
}

var thornsStats = []string{"PhysicalMin", "PhysicalMax", "FireMin", "FireMax", "ColdMin", "ColdMax", "LightningMin", "LightningMax", "ChaosMin", "ChaosMax"}

//pob2:src/Modules/CalcSetup.lua:721-734
func modDBHasThornsDamage(modDB *moddb.ModDB) bool {
	for _, stat := range thornsStats {
		for _, m := range modDB.Mods[stat] {
			if m.Type() == mod.TypeBase && m.Flags()&ModFlagThorns != 0 {
				return true
			}
		}
	}
	parent, ok := modDB.Parent.(*moddb.ModDB)
	return ok && parent != nil && modDBHasThornsDamage(parent)
}

//pob2:src/Modules/CalcSetup.lua:774-783
func socketGroupHasGem(socketGroup *SocketGroup, gemInstance *GemInstance) bool {
	for _, existing := range socketGroup.GemList {
		if gemInstance.GemID != "" && existing.GemID == gemInstance.GemID ||
			gemInstance.SkillID != "" && existing.SkillID == gemInstance.SkillID ||
			gemInstance.GemID == "" && gemInstance.SkillID == "" && existing.NameSpec == gemInstance.NameSpec {
			return true
		}
	}
	return false
}

// copyGemInstance is copyTable(gemInstance, true).
func copyGemInstance(g *GemInstance) *GemInstance {
	c := *g
	return &c
}

//pob2:src/Modules/CalcSetup.lua:785-797
func migrateLegacySlotSupports(socketGroup *SocketGroup, legacyGroups []*SocketGroup, migratedGroups map[*SocketGroup]bool) {
	for _, legacyGroup := range legacyGroups {
		for _, gemInstance := range legacyGroup.GemList {
			grantedEffect := gemInstance.GrantedEffect
			if grantedEffect == nil && gemInstance.GemData != nil {
				grantedEffect = gemInstance.GemData.GrantedEffect
			}
			if grantedEffect != nil && grantedEffect.Support {
				migratedGroups[legacyGroup] = true
				if !socketGroupHasGem(socketGroup, gemInstance) {
					socketGroup.GemList = append(socketGroup.GemList, copyGemInstance(gemInstance))
				}
			}
		}
	}
}

// nodeGrantedSkills is node.grantedSkills: the ExtraSkill list of the node's own mods. The jewel and
// PassiveSkillHasOtherEffect rewrites buildModListForNode applies first are not seen here.
//
//pob2:block:src/Modules/CalcSetup.lua:321-335
func nodeGrantedSkills(env *Environment, node *tree2.SpecNode) []*grantedSkill {
	var out []*grantedSkill
	for _, v := range env.nodeModList(node).List(nil, "ExtraSkill") {
		skill, ok := v.(mod.ExtraSkill)
		if p, isPtr := v.(*mod.ExtraSkill); isPtr {
			skill, ok = *p, true
		}
		if !ok || skill.SkillName == "Unknown" {
			continue
		}
		out = append(out, &grantedSkill{SkillID: skill.SkillID, Level: skill.Level, NoSupports: skill.NoSupports, Source: "Tree:" + strconv.Itoa(node.ID)})
	}
	return out
}

var defaultAttackSlots = [][2]string{
	{"Weapon 1", "Weapon 2"},
	{"Weapon 1 Swap", "Weapon 2 Swap"},
}

//pob2:src/Modules/CalcSetup.lua:710-719
func getCharacterMeleeSkillClass(item *Item) string {
	if item != nil && item.Base != nil {
		if slices.Contains(item.Base.Tags, "buckler") {
			return "Buckler"
		} else if item.Base.Weapon != nil {
			if item.Base.SubType != "" {
				return item.Base.SubType
			}
			return item.Base.Type
		}
	}
	return "Unarmed"
}

// setupSkills is the skill half of calcs.initEnv: the granted skill groups, the supports, the active
// skills and the main skill.
//
//pob2:block:src/Modules/CalcSetup.lua:1823-2462
func setupSkills(env *Environment) bool {
	env.loadData()
	if env.SocketGroupList == nil {
		env.SocketGroupList = loadSocketGroups(env)
	}
	if env.fromItemEffects == nil {
		env.fromItemEffects = map[*poe2skills.GrantedEffect]bool{}
	}
	build := env.Build
	modDB := env.ModDB
	characterLevel := float64(build.Build.Level)

	// Find skills granted by tree nodes
	for _, id := range sortedNodeIDs(env.AllocNodes) {
		node := env.AllocNodes[id]
		for _, skill := range nodeGrantedSkills(env, node) {
			skill.SourceNode = node
			env.grantedSkillsNodes = append(env.grantedSkillsNodes, skill)
		}
	}

	// Merge Granted Skills Tables
	grantedSkills := append(append([]*grantedSkill{}, env.grantedSkillsNodes...), env.grantedSkillsItems...)

	virtuousMoteSkillCount := map[string]float64{"Str": 3, "Dex": 3, "Int": 3}
	virtuousMoteSkillCounted := map[*GemInstance]bool{}

	if env.Mode == "MAIN" {
		// Keep both sets available, independently of the Items tab's selected set.
		//pob2:block:src/Modules/CalcSetup.lua:1845-1858
		selItem := func(slotName string) *Item {
			if env.items == nil || env.items.Slots[slotName] == nil {
				return nil
			}
			return env.items.Items[env.items.Slots[slotName].SelItemID]
		}
		for _, slots := range defaultAttackSlots {
			mainItem, offItem := selItem(slots[0]), selItem(slots[1])
			skillsForMainHand := env.data.characterMeleeSkills[getCharacterMeleeSkillClass(mainItem)]
			var defaultGems []*poe2skills.Gem
			if skillsForMainHand != nil {
				defaultGems = skillsForMainHand[getCharacterMeleeSkillClass(offItem)]
			}
			for _, gemData := range defaultGems {
				if gemData != nil {
					grantedSkills = append(grantedSkills, &grantedSkill{
						SkillID: gemData.Data.GrantedEffectID, NameSpec: gemData.Data.Name, SlotName: slots[0], Source: "Default Attack",
						Level: getGrantedSkillLevel(gemData, int(gemData.Data.NaturalMaxLevel), characterLevel, nil),
					})
				}
			}
		}
		//pob2:block:src/Modules/CalcSetup.lua:1859-1976
		markList := map[*SocketGroup]bool{}
		var mainGroup *SocketGroup
		if i := build.Build.MainSocketGroup; i >= 1 && i <= len(env.SocketGroupList) {
			mainGroup = env.SocketGroupList[i-1]
		}
		mergedGroups := map[*SocketGroup]bool{}
		legacySupportGroups := map[string][]*SocketGroup{}
		for _, socketGroup := range env.SocketGroupList {
			if socketGroup.Source == "" && socketGroup.Slot != "" {
				legacySupportGroups[socketGroup.Slot] = append(legacySupportGroups[socketGroup.Slot], socketGroup)
			}
		}
		var migratedLegacyGroups map[*SocketGroup]bool
		for _, grantedSkill := range grantedSkills {
			if grantedSkill.SourceNode != nil {
				grantedEffect := env.data.effect(grantedSkill.SkillID)
				if gemData := env.data.gem(env.data.gemForSkill[grantedEffect]); gemData != nil {
					grantedSkill.Level = getGrantedSkillLevel(gemData, int(gemData.Data.NaturalMaxLevel), characterLevel, nil)
				}
			}
			normalizedSkillLevel, hasNormalized := 0, false
			// Check if a matching group already exists
			var group, legacyGroup *SocketGroup
			for _, socketGroup := range env.SocketGroupList {
				if len(socketGroup.GemList) == 0 {
					continue
				}
				gemInstance := socketGroup.GemList[0]
				if gemInstance.SkillID != grantedSkill.SkillID {
					continue
				}
				if socketGroup.Source == grantedSkill.Source && socketGroup.Slot == grantedSkill.SlotName {
					group = socketGroup
				} else if legacyGroup == nil && !markList[socketGroup] && socketGroup.Source == "" &&
					(grantedSkill.Source != "Default Attack" || socketGroup.Set1 && socketGroup.Set2 ||
						env.items != nil && env.items.Slots[grantedSkill.SlotName] != nil && getSocketGroupWeaponSet(env, socketGroup) == env.items.Slots[grantedSkill.SlotName].WeaponSet) {
					matchingLevel := grantedSkill.Source == "Default Attack" || gemInstance.Level == grantedSkill.Level
					if !matchingLevel {
						if !hasNormalized {
							normalizedSkillLevel, hasNormalized = getNormalizedSkillLevel(env, grantedSkill), true
						}
						matchingLevel = gemInstance.Level == normalizedSkillLevel
					}
					grantedEffect := gemInstance.GrantedEffect
					if grantedEffect == nil && gemInstance.GemData != nil {
						grantedEffect = gemInstance.GemData.GrantedEffect
					}
					if matchingLevel && grantedEffect != nil && (grantedSkill.SourceItem != nil && grantedEffect.Overlay.FromItem || grantedSkill.SourceNode != nil && grantedEffect.Overlay.FromTree ||
						grantedSkill.Source == "Default Attack" && grantedEffect.Overlay.FromItem) {
						legacyGroup = socketGroup
					}
				}
			}
			if group != nil && legacyGroup != nil {
				// Keep the user's group, including its settings and any other active gems.
				for _, gem := range group.GemList[1:] {
					if !socketGroupHasGem(legacyGroup, gem) {
						legacyGroup.GemList = append(legacyGroup.GemList, copyGemInstance(gem))
					}
				}
				mergedGroups[group] = true
				if mainGroup == group {
					mainGroup = legacyGroup
				}
			}
			if legacyGroup != nil {
				group = legacyGroup
			}
			if group == nil {
				// Create a new group for this skill
				group = &SocketGroup{Enabled: true, Source: grantedSkill.Source, Slot: grantedSkill.SlotName, MainActiveSkill: 1, MainActiveSkillCalcs: 1}
				env.SocketGroupList = append(env.SocketGroupList, group)
			}
			markList[group] = true

			// Update the group
			group.Source = grantedSkill.Source
			group.Slot = grantedSkill.SlotName
			group.SourceItem = grantedSkill.SourceItem
			group.SourceNode = grantedSkill.SourceNode
			var activeGemInstance *GemInstance
			if len(group.GemList) > 0 {
				activeGemInstance = group.GemList[0]
			} else {
				activeGemInstance = &GemInstance{SkillID: grantedSkill.SkillID, NameSpec: grantedSkill.NameSpec}
			}
			activeGemInstance.FromItem = grantedSkill.SourceItem != nil || grantedSkill.Source == "Default Attack"
			activeGemInstance.FromTree = grantedSkill.SourceNode != nil
			activeGemInstance.SourceLevel, activeGemInstance.HasSourceLevel = 0, false
			if grantedSkill.SourceItem != nil {
				activeGemInstance.SourceLevel, activeGemInstance.HasSourceLevel = grantedSkill.Level, true
			}
			activeGemInstance.SkillID = grantedSkill.SkillID
			activeGemInstance.GemID = ""
			activeGemInstance.Level = grantedSkill.Level
			activeGemInstance.Quality = 0
			activeGemInstance.Enabled = true
			activeGemInstance.Count = 1
			activeGemInstance.Corrupted = false
			activeGemInstance.CorruptLevel = 0
			activeGemInstance.EnableGlobal1 = true
			activeGemInstance.EnableGlobal2 = true
			activeGemInstance.NoSupports = grantedSkill.NoSupports
			group.NoSupports = grantedSkill.NoSupports
			activeGemInstance.NoReservation = grantedSkill.NoReservation
			activeGemInstance.Triggered = grantedSkill.Triggered
			activeGemInstance.TriggerChance = grantedSkill.TriggerChance
			if activeGemInstance.StatSet == nil {
				activeGemInstance.StatSet = map[string]int{}
				activeGemInstance.StatSetCalcs = map[string]int{}
			}
			if len(group.GemList) == 0 {
				group.GemList = append(group.GemList, activeGemInstance)
			}
			if grantedSkill.SourceItem != nil && grantedSkill.SlotName != "" && legacySupportGroups[grantedSkill.SlotName] != nil {
				if migratedLegacyGroups == nil {
					migratedLegacyGroups = map[*SocketGroup]bool{}
				}
				migrateLegacySlotSupports(group, legacySupportGroups[grantedSkill.SlotName], migratedLegacyGroups)
			}
			processSocketGroup(env, group)
		}
		for legacyGroup := range migratedLegacyGroups {
			legacyGroup.Slot = ""
		}

		//pob2:block:src/Modules/CalcSetup.lua:2026-2076
		hasThornsDamage := modDBHasThornsDamage(env.ModDB)
		if !hasThornsDamage {
		groups:
			for _, socketGroup := range env.SocketGroupList {
				if socketGroup.Source != "Thorns" && socketGroup.Enabled {
					for _, gem := range socketGroup.GemList {
						if !gem.Enabled {
							continue
						}
						if grantedEffect := env.data.effect(gem.SkillID); grantedEffect != nil && grantedEffect.Overlay.GrantsThornsDamage {
							hasThornsDamage = true
							break groups
						}
					}
				}
			}
		}
		//pob2:block:src/Modules/CalcSetup.lua:1978-2024
		if len(env.explodeSources) != 0 {
			// Check if a matching group already exists
			var group *SocketGroup
			for _, socketGroup := range env.SocketGroupList {
				if socketGroup.Source == "Explode" {
					group = socketGroup
					break
				}
			}
			if group == nil {
				// Create a new group for this skill
				group = &SocketGroup{Label: "On Kill Monster Explosion", Enabled: true, Source: "Explode", NoSupports: true, MainActiveSkill: 1, MainActiveSkillCalcs: 1}
				env.SocketGroupList = append(env.SocketGroupList, group)
			}
			// Hack to remove duplicates
			// pairs() order over explodeBySource is first-seen order here.
			explodeBySource := map[string]any{}
			var explodeKeys []string
			for _, explodeSource := range env.explodeSources {
				key := explodeSourceKey(explodeSource)
				if _, ok := explodeBySource[key]; !ok {
					explodeKeys = append(explodeKeys, key)
				}
				explodeBySource[key] = explodeSource
			}
			// Update the group
			gemsBySource := map[string]*GemInstance{}
			for _, gem := range group.GemList {
				if gem.ExplodeSource != nil {
					gemsBySource[explodeSourceKey(gem.ExplodeSource)] = gem
				}
			}
			group.GemList = nil
			for _, key := range explodeKeys {
				activeGemInstance := gemsBySource[key]
				if activeGemInstance == nil {
					activeGemInstance = &GemInstance{
						SkillID:       "EnemyExplode",
						Enabled:       true,
						Level:         1,
						Triggered:     true,
						ExplodeSource: explodeBySource[key],
						StatSet:       map[string]int{},
						StatSetCalcs:  map[string]int{},
					}
				}
				group.GemList = append(group.GemList, activeGemInstance)
			}
			markList[group] = true
			processSocketGroup(env, group)
		}

		var thornsGroup *SocketGroup
		for _, socketGroup := range env.SocketGroupList {
			if socketGroup.Source == "Thorns" {
				thornsGroup = socketGroup
				break
			}
		}
		if !hasThornsDamage {
			if thornsGroup != nil {
				thornsGroup.GemList = nil
			}
		} else {
			if thornsGroup == nil {
				thornsGroup = &SocketGroup{Label: "Thorns", Enabled: true, Source: "Thorns", NoSupports: true, MainActiveSkill: 1, MainActiveSkillCalcs: 1}
				env.SocketGroupList = append(env.SocketGroupList, thornsGroup)
			}
			thornsGroup.GemList = []*GemInstance{{SkillID: "ThornsPlayer", Enabled: true, Level: 1, Triggered: true, StatSet: map[string]int{}, StatSetCalcs: map[string]int{}}}
			markList[thornsGroup] = true
			processSocketGroup(env, thornsGroup)
		}

		// Remove any socket groups that no longer have a matching item
		//pob2:block:src/Modules/CalcSetup.lua:2079-2100
		kept := env.SocketGroupList[:0]
		for _, socketGroup := range env.SocketGroupList {
			if mergedGroups[socketGroup] || socketGroup.Source != "" && !markList[socketGroup] {
				continue
			}
			kept = append(kept, socketGroup)
			if socketGroup == mainGroup {
				build.Build.MainSocketGroup = len(kept)
			}
		}
		env.SocketGroupList = kept

		// Generated groups can become the selected main group, most notably when an
		// imported build contains only a skill granted by a swapped weapon. Restart
		// before weapon data and modifiers are consumed if that changes the context.
		//pob2:block:src/Modules/CalcSetup.lua:2105-2116
		if env.override.WeaponSet == 0 {
			mainSocketGroup := build.Build.MainSocketGroup
			if env.override.MainSocketGroup != 0 {
				mainSocketGroup = env.override.MainSocketGroup
			}
			resolvedMainSocketGroup := min(max(len(env.SocketGroupList), 1), mainSocketGroup)
			var resolvedGroup *SocketGroup
			if resolvedMainSocketGroup >= 1 && resolvedMainSocketGroup <= len(env.SocketGroupList) {
				resolvedGroup = env.SocketGroupList[resolvedMainSocketGroup-1]
			}
			resolvedWeaponSet := getSocketGroupWeaponSet(env, resolvedGroup)
			if resolvedMainSocketGroup != env.MainSocketGroup || resolvedWeaponSet != env.WeaponSet {
				build.Build.MainSocketGroup = resolvedMainSocketGroup
				resolvedOverride := env.override
				resolvedOverride.MainSocketGroup = resolvedMainSocketGroup
				resolvedOverride.WeaponSet = resolvedWeaponSet
				weaponSetChanged := resolvedWeaponSet != env.WeaponSet
				restarted := newWeaponSetEnv(env, env.Mode, resolvedOverride)
				initEnv(restarted)
				*env = *restarted
				env.weaponSetRestart = weaponSetChanged
				return true
			}
		}
	}

	//pob2:block:src/Modules/CalcSetup.lua:2155-2245
	supportLists := map[*SocketGroup]*[]*GemEffect{}
	groupCfgList := map[*SocketGroup]*groupCfg{}
	processedSockets := map[*GemInstance]bool{}
	groupCfgFor := func(group *SocketGroup) *groupCfg {
		if gc := groupCfgList[group]; gc != nil {
			return gc
		}
		slotName := strings.ReplaceAll(group.Slot, " Swap", "")
		gc := &groupCfg{cfg: &moddb.ListCfg{SlotName: slotName}, slotName: slotName}
		gc.propertyModList = tabulateGemProperties(env, &moddb.ListCfg{SlotName: slotName})
		groupCfgList[group] = gc
		return gc
	}
	// Process supports and put them into the correct buckets
	for index, group := range env.SocketGroupList {
		group.UsingSkillSet = getSocketGroupWeaponSet(env, group)
		var sourceGem *GemInstance
		if len(group.GemList) > 0 {
			sourceGem = group.GemList[0]
		}
		if sourceGem != nil && sourceGem.FromItem && sourceGem.HasSourceLevel && sourceGem.GemData != nil && group.UsingSkillSet == env.WeaponSet {
			level := getGrantedSkillLevel(sourceGem.GemData, sourceGem.SourceLevel, characterLevel, env.ModDB)
			if sourceGem.Level != level {
				sourceGem.Level = level
				grantedEffect := sourceGem.GemData.GrantedEffect
				sourceGem.ReqLevel, _ = levelRequirement(grantedEffect, level)
				sourceGem.ReqStr = getGemStatRequirement(sourceGem.ReqLevel, sourceGem.GemData.Data.ReqStr, false)
				sourceGem.ReqDex = getGemStatRequirement(sourceGem.ReqLevel, sourceGem.GemData.Data.ReqDex, false)
				sourceGem.ReqInt = getGemStatRequirement(sourceGem.ReqLevel, sourceGem.GemData.Data.ReqInt, false)
			}
		}
		// if group is main skill or group is enabled
		if index+1 != env.MainSocketGroup && !group.Enabled {
			continue
		}
		gc := groupCfgFor(group)
		// One list per group: a slotted group's list is keyed by slot and group, an unslotted one by group.
		if supportLists[group] == nil {
			supportLists[group] = &[]*GemEffect{}
		}
		targetListList := []*[]*GemEffect{supportLists[group]}

		// if not unique item that provides skills
		if group.Source == "" {
			// Add extra supports from the item this group is socketed in
			for _, v := range env.ModDB.List(gc.cfg, "ExtraSupport") {
				value, ok := v.(mod.ExtraSupport)
				if !ok {
					continue
				}
				grantedEffect := env.data.effect(value.SkillID)
				// Some skill gems share the same name as support gems, e.g. Barrage.
				// Since a support gem is expected here, if the first lookup returns a skill, then
				// prepending "Support" to the skillId will find the support version of the gem.
				if grantedEffect != nil && !grantedEffect.Support {
					grantedEffect = env.data.effect("Support" + value.SkillID)
				}
				if grantedEffect == nil {
					continue
				}
				env.fromItemEffects[grantedEffect] = true
				gemData := env.data.gemForBaseName(grantedEffect.Name)
				if gemData == nil {
					gemData = env.data.gemForBaseName(grantedEffect.Name + " Support")
				}
				for _, targetList := range targetListList {
					*targetList = append(*targetList, &GemEffect{
						GrantedEffect: grantedEffect,
						GemData:       gemData,
						Level:         value.Level,
						Enabled:       true,
						IsSupporting:  map[*GemInstance]bool{},
					})
				}
			}
		}

		for i, gemInstance := range group.GemList {
			gemIndex := i + 1
			// Add support gems from this group
			if env.Mode == "MAIN" {
				gemInstance.DisplayEffect = nil
				gemInstance.SupportEffect = nil
			}
			if !gemInstance.Enabled {
				continue
			}
			if gemInstance.GemData != nil {
				processGrantedEffect(gemInstance.GemData.GrantedEffect, gemInstance, env, gc, gemIndex, processedSockets, targetListList)
				for _, additional := range gemInstance.GemData.AdditionalGrantedEffects {
					processGrantedEffect(additional, gemInstance, env, gc, gemIndex, processedSockets, targetListList)
				}
			} else {
				processGrantedEffect(gemInstance.GrantedEffect, gemInstance, env, gc, gemIndex, processedSockets, targetListList)
			}
		}
	}

	// Process active skills adding the applicable supports
	//pob2:block:src/Modules/CalcSetup.lua:2247-2372
	socketGroupSkillListList := map[*SocketGroup][]*ActiveSkill{}
	for index, group := range env.SocketGroupList {
		if index+1 != env.MainSocketGroup && !group.Enabled {
			continue
		}
		gc := groupCfgFor(group)
		slotHasActiveSkill := false

		// Create active skills
		for i, gemInstance := range group.GemList {
			gemIndex := i + 1
			if !gemInstance.Enabled || gemInstance.GemData == nil && gemInstance.GrantedEffect == nil {
				continue
			}
			grantedEffectList := []*poe2skills.GrantedEffect{gemInstance.GrantedEffect}
			if gemInstance.GemData != nil {
				grantedEffectList = gemInstance.GemData.GrantedEffectList
			}
			for effectIndex, grantedEffect := range grantedEffectList {
				if grantedEffect == nil {
					break
				}
				if grantedEffect.Support || grantedEffect.Overlay.HideFromSideBar || grantedEffect.GlobalEffect() && !gemInstance.enableGlobal(effectIndex+1) {
					continue
				}
				slotHasActiveSkill = true
				if gemInstance.GemData != nil && !virtuousMoteSkillCounted[gemInstance] && !(gemInstance.FromTree || gemInstance.FromItem) {
					virtuousMoteSkillCounted[gemInstance] = true
					var requiredAttributes []string
					if gemInstance.GemData.Data.ReqStr > 0 {
						requiredAttributes = append(requiredAttributes, "Str")
					}
					if gemInstance.GemData.Data.ReqDex > 0 {
						requiredAttributes = append(requiredAttributes, "Dex")
					}
					if gemInstance.GemData.Data.ReqInt > 0 {
						requiredAttributes = append(requiredAttributes, "Int")
					}
					moteValue := 1.0
					if len(requiredAttributes) == 1 {
						moteValue = 2
					}
					for _, attr := range requiredAttributes {
						virtuousMoteSkillCount[attr] += moteValue
					}
				}
				activeEffect := &GemEffect{
					GrantedEffect: grantedEffect,
					Level:         gemInstance.Level,
					Quality:       gemInstance.Quality,
					QualityID:     gemInstance.QualityID,
					Corrupted:     gemInstance.Corrupted,
					CorruptLevel:  gemInstance.CorruptLevel,
					SrcInstance:   gemInstance,
					GemData:       gemInstance.GemData,
				}
				activeEffect.Level = max(activeEffect.Level+activeEffect.CorruptLevel, 1)
				statSets := gemInstance.StatSet
				if env.Mode == "CALCS" {
					statSets = gemInstance.StatSetCalcs
				}
				activeEffect.StatSet = &StatSetChoice{Index: 1}
				if idx, ok := statSets[grantedEffect.ID]; ok {
					activeEffect.StatSet.Index = idx
				}
				if gemInstance.GemData != nil {
					applyGemMods(activeEffect, gemModsFor(env, gc, gemIndex))
					countSocketedGem(env, gemInstance, gc, gemIndex, processedSockets)
				}
				if env.Mode == "MAIN" {
					gemInstance.DisplayEffect = activeEffect
				}
				var appliedSupportList []*GemEffect
				if !group.NoSupports {
					if l := supportLists[group]; l != nil {
						appliedSupportList = append([]*GemEffect{}, *l...)
					}
					// add displayGemList for tooltip to display all gems linked to active skills
					group.DisplayGemList = append([]*GemInstance{}, group.GemList...)
					// TODO: crossLinkedSupportGroups — no PoE2 mod line produces LinkedSupport yet.
				}
				activeSkill := createActiveSkill(activeEffect, appliedSupportList, env, env.Player, group, nil)
				if gemInstance.GemData != nil {
					activeSkill.SlotName = gc.slotName
				}
				socketGroupSkillListList[group] = append(socketGroupSkillListList[group], activeSkill)
				env.Player.ActiveSkillList = append(env.Player.ActiveSkillList, activeSkill)
			}
			if gemInstance.GemData != nil && !(gemInstance.FromTree || gemInstance.FromItem) {
				env.requirementsTableGems = append(env.requirementsTableGems, requirementSource{
					source: "Gem", sourceGem: gemInstance,
					Str: gemInstance.ReqStr, Dex: gemInstance.ReqDex, Int: gemInstance.ReqInt,
				})
			}
		}

		if !slotHasActiveSkill && group.DisplayGemList != nil {
			group.DisplayGemList = nil
		}
	}

	// Process calculated active skill lists
	//pob2:block:src/Modules/CalcSetup.lua:2374-2404
	for index, group := range env.SocketGroupList {
		socketGroupSkillList := socketGroupSkillListList[group]
		if index+1 != env.MainSocketGroup && !group.Enabled {
			continue
		}
		gc := groupCfgFor(group)
		for _, v := range env.ModDB.List(gc.cfg, "GroupProperty") {
			if p, ok := v.(mod.GroupProperty); ok {
				env.Player.ModDB.AddMod(setSource(p.Mod.Clone(), gc.slotName))
			}
		}

		if index+1 == env.MainSocketGroup && len(socketGroupSkillList) > 0 {
			// Select the main skill from this socket group
			var activeSkillIndex int
			if env.Mode == "CALCS" {
				group.MainActiveSkillCalcs = min(len(socketGroupSkillList), group.MainActiveSkillCalcs)
				activeSkillIndex = group.MainActiveSkillCalcs
			} else {
				activeSkillIndex = min(len(socketGroupSkillList), group.MainActiveSkill)
				if env.Mode == "MAIN" {
					group.MainActiveSkill = activeSkillIndex
				}
			}
			env.Player.MainSkill = socketGroupSkillList[activeSkillIndex-1]
		}
	}
	// TODO: the Energy Blade restart (CalcSetup.lua:2436-2446) is not ported.

	// UPDATE THIS TO ADD DEFAULT SKILLS FOR EACH WEAPON
	//pob2:block:src/Modules/CalcSetup.lua:2450-2462
	if env.Player.MainSkill == nil {
		// Add a default main skill if none are specified
		defaultEffect := &GemEffect{
			GrantedEffect: env.data.effect("MeleeUnarmedPlayer"),
			Level:         1,
			Enabled:       true,
			StatSet:       &StatSetChoice{Index: 1},
		}
		env.Player.MainSkill = createActiveSkill(defaultEffect, nil, env, env.Player, nil, nil)
		env.Player.ActiveSkillList = append(env.Player.ActiveSkillList, env.Player.MainSkill)
	}

	setupWeaponData(env)
	//pob2:block:src/Modules/CalcSetup.lua:2120-2153
	env.Player.WeaponData1 = weaponData(env, env.Player, 1)
	env.Player.WeaponData2 = weaponData(env, env.Player, 2)

	// some active skills, like those granted by support gems depend on the stats of other gems,
	// which means we have to do them last.
	//pob2:block:src/Modules/CalcSetup.lua:2464-2477
	var deferredActiveSkills []*ActiveSkill
	// Build skill modifier lists
	for _, activeSkill := range env.Player.ActiveSkillList {
		if activeSkill.ActiveEffect.GemData != nil && activeSkill.ActiveEffect.GemData.GrantedEffect.Support {
			deferredActiveSkills = append(deferredActiveSkills, activeSkill)
		} else {
			buildActiveSkillModList(env, activeSkill)
		}
	}
	for _, activeSkill := range deferredActiveSkills {
		buildActiveSkillModList(env, activeSkill)
	}

	// Rebuild auxiliary groups assigned exclusively to the other weapon set in
	// that set's actor context. Their skill modifier lists retain that context
	// when their buffs and debuffs are applied to the selected main skill.
	//pob2:block:src/Modules/CalcSetup.lua:2479-2509
	if !env.override.SkipWeaponSetContexts {
		otherSet, otherSetMainGroup := 0, 0
		for index, group := range env.SocketGroupList {
			// Bare default attacks have no auxiliary effects; evaluate their set when selected.
			if group.Enabled && group.UsingSkillSet != env.WeaponSet &&
				(group.Source != "Default Attack" || len(group.GemList) > 1 || group.IncludeInFullDPS) {
				otherSet = group.UsingSkillSet
				otherSetMainGroup = index + 1
				break
			}
		}
		if otherSet != 0 {
			contextOverride := env.override
			contextOverride.WeaponSet = otherSet
			contextOverride.MainSocketGroup = otherSetMainGroup
			contextOverride.SkipWeaponSetContexts = true
			contextMode := "CALCULATOR"
			if env.Mode == "CALCS" {
				contextMode = "CALCS"
			}
			weaponSetEnv := newWeaponSetEnv(env, contextMode, contextOverride)
			initEnv(weaponSetEnv)
			env.WeaponSetEnvs = map[int]*Environment{otherSet: weaponSetEnv}
			// Preserve the uncomposed lists for checkbox validation; the composed lists
			// below intentionally replace groups with their assigned-set versions.
			env.WeaponSetValidationSkillList = env.Player.ActiveSkillList
			weaponSetEnv.WeaponSetValidationSkillList = weaponSetEnv.Player.ActiveSkillList
			replaceWeaponSetActiveSkills(env, weaponSetEnv, otherSet)
			replaceWeaponSetActiveSkills(weaponSetEnv, env, env.WeaponSet)
		}
	}

	//pob2:block:src/Modules/CalcSetup.lua:2535-2538
	env.VirtuousMoteSkillCount = virtuousMoteSkillCount
	modDB.Multipliers["StrengthMoteSkillCount"] = virtuousMoteSkillCount["Str"]
	modDB.Multipliers["DexterityMoteSkillCount"] = virtuousMoteSkillCount["Dex"]
	modDB.Multipliers["IntelligenceMoteSkillCount"] = virtuousMoteSkillCount["Int"]
	return false
}

// explodeSourceKey is explodeSource.modSource or explodeSource.id: an item's modSource, a node's id.
func explodeSourceKey(explodeSource any) string {
	switch src := explodeSource.(type) {
	case *Item:
		return src.modSourceName
	case *tree2.SpecNode:
		return strconv.Itoa(src.ID)
	}
	return ""
}

//pob2:src/Modules/CalcSetup.lua:745-772
func replaceWeaponSetActiveSkills(targetEnv, sourceEnv *Environment, sourceSet int) {
	replacements := map[*SocketGroup][]*ActiveSkill{}
	for _, activeSkill := range sourceEnv.Player.ActiveSkillList {
		group := activeSkill.SocketGroup
		if group != nil && group.UsingSkillSet == sourceSet {
			replacements[group] = append(replacements[group], activeSkill)
		}
	}
	replaced := map[*SocketGroup]bool{}
	var activeSkillList []*ActiveSkill
	for _, activeSkill := range targetEnv.Player.ActiveSkillList {
		group := activeSkill.SocketGroup
		if replacements[group] != nil {
			if !replaced[group] {
				replaced[group] = true
				activeSkillList = append(activeSkillList, replacements[group]...)
			}
		} else {
			activeSkillList = append(activeSkillList, activeSkill)
		}
	}
	targetEnv.Player.ActiveSkillList = activeSkillList
}

// setupGemCounts counts the socketed support gems by colour, adds their requirement, merges the
// requirement tables, and sets the socketed-support conditions.
//
//pob2:block:src/Modules/CalcSetup.lua:2540-2601
func setupGemCounts(env *Environment) {
	// Currently PoB2 doesn't associate weapon skill supports with the actual weapon sets
	// so it ends up counting all support gems when it should only take into account the active weapon set
	var r, g, b float64
	// Loop through socket groups to calculate number of socketed gems
	for _, socketGroup := range env.SocketGroupList {
		if !socketGroup.Enabled {
			continue
		}
		for _, gem := range socketGroup.GemList {
			if gem.SupportEffect != nil && gem.SupportEffect.GrantedEffect != nil && !gem.SupportEffect.GrantedEffect.Hidden {
				switch gem.SupportEffect.GrantedEffect.Color {
				case 1:
					r++
				case 2:
					g++
				case 3:
					b++
				}
			}
		}
	}
	modDB := env.ModDB
	modDB.Multipliers["RedSupportGems"] += r
	modDB.Multipliers["GreenSupportGems"] += g
	modDB.Multipliers["BlueSupportGems"] += b
	env.requirementsTableGems = append(env.requirementsTableGems, requirementSource{
		source: "Support Gems",
		Str:    r * 5,
		Dex:    g * 5,
		Int:    b * 5,
	})

	// Merge Requirements Tables
	env.requirementsTable = append(append([]requirementSource{}, env.requirementsTableItems...), env.requirementsTableGems...)

	// Crystallised Immunity notable support
	switch {
	case r > g && r > b:
		modDB.Conditions["MajorityRedSocketedSupports"] = true
	case g > r && g > b:
		modDB.Conditions["MajorityGreenSocketedSupports"] = true
	case b > r && b > g:
		modDB.Conditions["MajorityBlueSocketedSupports"] = true
	}

	// Gem Studded, Gemling notable support
	switch {
	case r >= g && r >= b:
		modDB.Conditions["MostNumerousRedSocketedSupports"] = true
	case g >= r && g >= b:
		modDB.Conditions["MostNumerousGreenSocketedSupports"] = true
	case b >= r && b >= g:
		modDB.Conditions["MostNumerousBlueSocketedSupports"] = true
	}
}

// weaponSetValidity is SkillsTab's weaponSetValidityCache, weaponSetValidityInProgress and each
// group's forcedBoth, for one output revision.
type weaponSetValidity struct {
	cache      map[*SocketGroup]map[int]bool
	forcedBoth map[*SocketGroup]bool
	mainEnv    *Environment
	inProgress bool
}

//pob2:src/Classes/SkillsTab.lua:1393-1396
func isSocketGroupWeaponSetLocked(env *Environment, socketGroup *SocketGroup) bool {
	var sourceSlot *ItemSlot
	if socketGroup != nil && (socketGroup.SourceItem != nil || socketGroup.Source == "Default Attack") && socketGroup.Slot != "" && env.items != nil {
		sourceSlot = env.items.Slots[socketGroup.Slot]
	}
	return sourceSlot != nil && sourceSlot.WeaponSet != 0
}

//pob2:src/Classes/SkillsTab.lua:1416-1437
func (v *weaponSetValidity) cacheWeaponSetContext(context *Environment) {
	if context == nil {
		return
	}
	groupSkillIndex := map[*SocketGroup]int{}
	activeSkillList := context.WeaponSetValidationSkillList
	if activeSkillList == nil {
		activeSkillList = context.Player.ActiveSkillList
	}
	for _, activeSkill := range activeSkillList {
		socketGroup := activeSkill.SocketGroup
		if socketGroup != nil {
			if truthy(activeSkill.SkillData["reservesInAllWeaponSets"]) {
				v.forcedBoth[socketGroup] = true
			}
			groupSkillIndex[socketGroup]++
			selectedIndex := socketGroup.MainActiveSkill
			if context.Mode == "CALCS" {
				selectedIndex = socketGroup.MainActiveSkillCalcs
			}
			if groupSkillIndex[socketGroup] == selectedIndex {
				var flags map[string]bool
				if activeSkill.ActiveEffect != nil && activeSkill.ActiveEffect.StatSet != nil {
					flags = activeSkill.ActiveEffect.StatSet.SkillFlags
				}
				if v.cache[socketGroup] == nil {
					v.cache[socketGroup] = map[int]bool{}
				}
				v.cache[socketGroup][context.WeaponSet] = !(flags != nil && flags["disable"])
			}
		}
	}
}

// cacheSocketGroupWeaponSetValidity is SkillsTab:CacheSocketGroupWeaponSetValidity. Every env of one
// build output shares its revision, so the cache is reset only when it is created.
//
//pob2:src/Classes/SkillsTab.lua:1439-1456
func (v *weaponSetValidity) cacheSocketGroupWeaponSetValidity(env *Environment) {
	if env == nil {
		return
	}
	v.cacheWeaponSetContext(env)
	for _, set := range []int{1, 2} {
		if context := env.WeaponSetEnvs[set]; context != nil {
			v.cacheWeaponSetContext(context)
		}
	}
}

//pob2:src/Classes/SkillsTab.lua:1458-1470
func applySocketGroupWeaponSetValidity(socketGroup *SocketGroup, set1Valid, set2Valid bool) bool {
	var set1, set2 bool
	switch {
	case set1Valid != set2Valid:
		set1, set2 = set1Valid, set2Valid
	case !set1Valid || !socketGroup.Set1 && !socketGroup.Set2:
		set1, set2 = true, true
	default:
		return false
	}
	changed := socketGroup.Set1 != set1 || socketGroup.Set2 != set2
	socketGroup.Set1, socketGroup.Set2 = set1, set2
	return changed
}

// reconcileSocketGroupWeaponSets is SkillsTab:ReconcileSocketGroupWeaponSets with no group to validate,
// as CalcsTab:BuildOutput calls it on a frame's rebuild.
//
//pob2:src/Classes/SkillsTab.lua:1472-1504
func reconcileSocketGroupWeaponSets(env *Environment) bool {
	v := &weaponSetValidity{cache: map[*SocketGroup]map[int]bool{}, forcedBoth: map[*SocketGroup]bool{}, mainEnv: env}
	v.cacheSocketGroupWeaponSetValidity(env)
	changed := false
	for _, socketGroup := range env.SocketGroupList {
		if socketGroup.Enabled && !isSocketGroupWeaponSetLocked(env, socketGroup) {
			if v.forcedBoth[socketGroup] {
				changed = !socketGroup.Set1 || !socketGroup.Set2 || changed
				socketGroup.Set1, socketGroup.Set2 = true, true
			} else {
				validity := v.cache[socketGroup]
				if valid, ok := validity[env.WeaponSet]; ok && !valid {
					otherSet := 1
					if env.WeaponSet == 1 {
						otherSet = 2
					}
					v.isSocketGroupWeaponSetValid(socketGroup, otherSet)
					validity = v.cache[socketGroup]
				}
				set1Valid, ok1 := validity[1]
				set2Valid, ok2 := validity[2]
				if ok1 && ok2 {
					changed = applySocketGroupWeaponSetValidity(socketGroup, set1Valid, set2Valid) || changed
				}
			}
		}
	}
	return changed
}

//pob2:src/Classes/SkillsTab.lua:1506-1535
func (v *weaponSetValidity) isSocketGroupWeaponSetValid(socketGroup *SocketGroup, weaponSet int) bool {
	if socketGroup == nil || v.inProgress {
		return true
	}
	v.cacheSocketGroupWeaponSetValidity(v.mainEnv)
	if v.cache[socketGroup] == nil {
		v.cache[socketGroup] = map[int]bool{}
	}
	if valid, ok := v.cache[socketGroup][weaponSet]; ok {
		return valid
	}
	groupIndex := slices.Index(v.mainEnv.SocketGroupList, socketGroup) + 1
	if groupIndex == 0 {
		return true
	}
	v.inProgress = true
	valid := true
	env := newWeaponSetEnv(v.mainEnv, "CALCULATOR", calcOverride{
		WeaponSet:             weaponSet,
		MainSocketGroup:       groupIndex,
		SkipWeaponSetContexts: true,
	})
	ok := false
	func() {
		defer func() { _ = recover() }()
		initEnv(env)
		ok = true
	}()
	if ok && env.Player != nil && env.Player.MainSkill != nil {
		v.cacheSocketGroupWeaponSetValidity(env)
		if cached, ok := v.cache[socketGroup][weaponSet]; ok {
			valid = cached
		}
	}
	v.inProgress = false
	v.cache[socketGroup][weaponSet] = valid
	return valid
}
