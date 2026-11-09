// Package calculator2 is the Path of Building (PoE2) calculation engine. The PoE1 engine never
// imports it.
package calculator2

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob2"
)

// Options carries what BuildOutput needs beside the build file.
type Options struct {
	// Tree is the build's passive tree, loaded by tree2.Load for the spec's treeVersion.
	Tree *tree2.PassiveTree
	// ModSource parses a mod line; nil is WithParser(ParseMod), PoB's modLib.parseMod.
	ModSource ModSource
	// Items is build.itemsTab, from NewItemsTab; nil means the build has no items.
	Items *ItemsTab
	// SpecItems is the passive spec's view of build.itemsTab.items (jewel radius and jewelData) that
	// allocation reads; nil is Items.SpecItem, or no item when Items is nil too.
	SpecItems func(itemID int) *tree2.Item
	// Party is the party tab's buffers as last applied; with PartyApplied unset the build's saved
	// ImportedBuffs are, as PartyTab:Load applies them.
	Party        []PartyBuffer
	PartyApplied bool
}

// WithParser is modLib.parseMod: PoB-PoE2's shipped ModCache first, and the parser for a line the
// cache does not hold.
//
//pob2:src/Modules/ModParser.lua:7435-7454
func WithParser(parse ModSource) ModSource {
	return func(line string) ([]mod.Mod, string) {
		if shippedCacheLookup(line) == nil {
			return parse(line)
		}
		return ModCache(line)
	}
}

// BuildOutput is calcs.buildOutput's first half for the "MAIN" mode: initEnv then perform. The returned
// env's Player.Output holds the computed stats.
//
//pob2:src/Modules/Calcs.lua:518-523
func BuildOutput(build *pob2.PathOfBuilding2, opts Options) (*Environment, error) {
	if opts.Tree == nil {
		return nil, fmt.Errorf("no passive tree")
	}
	env := &Environment{
		Build:        build,
		Mode:         "MAIN",
		ConfigInput:  configInput(build),
		modSource:    opts.ModSource,
		items:        opts.Items,
		nodeModLists: map[int]*moddb.ModList{},
	}
	if env.modSource == nil {
		env.modSource = WithParser(ParseMod)
	}
	specItems := opts.SpecItems
	if specItems == nil && opts.Items != nil {
		specItems = opts.Items.SpecItem
	}
	spec, err := loadSpec(build, opts.Tree, env.modSource, specItems)
	if err != nil {
		return nil, err
	}
	env.Spec = spec
	if opts.Items != nil {
		opts.Items.SetActiveSpec(spec)
	}
	if env.data, err = loadSkillData(); err != nil {
		return nil, fmt.Errorf("skill data: %w", err)
	}
	partyBuffers := opts.Party
	if !opts.PartyApplied {
		partyBuffers = ImportedPartyBuffers(build.Party)
	}
	data := env.data
	env.partyTab = loadPartyTab(partyBuffers, func(id string) bool { return data.effect(id) != nil })
	env.ClassID = spec.CurClassID
	if spec.Tree.Classes[env.ClassID] == nil {
		return nil, fmt.Errorf("class %d is not in the tree", env.ClassID)
	}
	initEnv(env)
	perform(env)
	// The rebuild keeps the socket group list, as build.skillsTab's persists across outputs. There is
	// no global cache to wipe.
	//pob2:block:src/Classes/CalcsTab.lua:505-510
	if reconcileSocketGroupWeaponSets(env) {
		env = newWeaponSetEnv(env, "MAIN", calcOverride{})
		initEnv(env)
		perform(env)
	}
	// Build output across all skills added to FullDPS skills
	//pob2:block:src/Modules/Calcs.lua:525-531
	setFullDPS(env, calcFullDPS(env, "CALCULATOR", calcOverride{}))
	buildOutputSpec(env)
	return env, nil
}

// buildOutputSpec is the part of calcs.buildOutput that sums the passive points and the tree's increases.
//
//pob2:block:src/Modules/Calcs.lua:576-587
func buildOutputSpec(env *Environment) {
	output := env.Player.Output
	output["ExtraPoints"] = env.ModDB.Sum(mod.TypeBase, nil, "ExtraPoints")
	output["WeaponSetPassivePoints"] = env.ModDB.Sum(mod.TypeBase, nil, "WeaponSetPassivePoints")
	output["PassivePointsToWeaponSetPoints"] = env.ModDB.Sum(mod.TypeBase, nil, "PassivePointsToWeaponSetPoints")

	source := mod.Source("Tree")
	specCfg := &moddb.ListCfg{Source: &source}
	output["Spec:LifeInc"] = env.ModDB.Sum(mod.TypeIncrease, nil, "Life")
	output["Spec:ManaInc"] = env.ModDB.Sum(mod.TypeIncrease, specCfg, "Mana")
	output["Spec:ArmourInc"] = env.ModDB.Sum(mod.TypeIncrease, specCfg, "Armour", "ArmourAndEvasion")
	output["Spec:EvasionInc"] = env.ModDB.Sum(mod.TypeIncrease, specCfg, "Evasion", "ArmourAndEvasion")
	output["Spec:EnergyShieldInc"] = env.ModDB.Sum(mod.TypeIncrease, specCfg, "EnergyShield")
}

// loadSpec is TreeTab:Load for the active spec.
//
//pob2:src/Classes/PassiveSpec.lua:117-379
func loadSpec(build *pob2.PathOfBuilding2, tree *tree2.PassiveTree, source ModSource, specItems func(int) *tree2.Item) (*tree2.PassiveSpec, error) {
	specs := build.Tree.Specs
	if len(specs) == 0 {
		return nil, fmt.Errorf("the build has no passive spec")
	}
	idx := build.Tree.ActiveSpec
	if idx < 1 || idx > len(specs) {
		idx = 1
	}
	s := specs[idx-1]
	in := tree2.SpecInput{
		Nodes:                s.Nodes,
		MasteryEffects:       s.MasteryEffects,
		AscendancyInternalID: s.AscendancyInternalID,
		WeaponSets:           map[int]string{},
	}
	if s.Title != nil {
		in.Title = *s.Title
	}
	if s.TreeVersion != nil {
		in.TreeVersion = *s.TreeVersion
	}
	itoa := func(v *int) *string {
		if v == nil {
			return nil
		}
		out := strconv.Itoa(*v)
		return &out
	}
	in.ClassID = itoa(s.ClassID)
	in.AscendClassID = itoa(s.AscendClassID)
	in.ClassInternalID = itoa(s.ClassInternalID)
	if !s.SecondaryAscendClassID.IsNil {
		in.SecondaryAscendClassID = itoa(&s.SecondaryAscendClassID.V)
	}
	if s.URL != nil {
		in.URL = &s.URL.Text
	}
	if s.WeaponSet1 != nil {
		in.WeaponSets[1] = s.WeaponSet1.Nodes
	}
	if s.WeaponSet2 != nil {
		in.WeaponSets[2] = s.WeaponSet2.Nodes
	}
	if s.Sockets != nil {
		for _, sock := range s.Sockets.Sockets {
			in.Sockets = append(in.Sockets, tree2.SocketInput{NodeID: strconv.Itoa(sock.NodeID), ItemID: strconv.Itoa(sock.ItemID)})
		}
	}
	if s.Overrides != nil {
		for _, o := range s.Overrides.AttributeOverrides {
			in.AttributeOverrides = append(in.AttributeOverrides, tree2.AttributeOverrideInput{StrNodes: o.StrNodes, DexNodes: o.DexNodes, IntNodes: o.IntNodes})
		}
	}
	spec := tree2.NewPassiveSpec(tree)
	spec.Items = specItems
	spec.AllocateFromNodeRadius = func(node *tree2.SpecNode) []tree2.NodeRadius {
		return allocateFromNodeRadius(processStats(source, node.ID, node.Type, node.Dn, node.Sd))
	}
	if err := spec.Load(in); err != nil {
		return nil, err
	}
	return spec, nil
}

// configInput is ConfigTab:Load's input table for the active config set, as setInputAndPlaceholder
// stores it; the per-option defaults CreateConfigSet seeds are applied where each option is read.
//
//pob2:src/Classes/ConfigTab.lua:878-978
func configInput(build *pob2.PathOfBuilding2) map[string]ConfigValue {
	inputs := build.Config.Inputs
	for _, set := range build.Config.ConfigSets {
		if set.ID == activeConfigSetID(build) {
			inputs = set.Inputs
		}
	}
	out := make(map[string]ConfigValue, len(inputs))
	for _, in := range inputs {
		var v ConfigValue
		switch {
		case in.Number != nil:
			v.Number, v.IsNumber = *in.Number, true
		case in.String != nil:
			v.String, v.IsString = *in.String, true
		case in.Boolean != nil:
			v.Boolean, v.IsBoolean = *in.Boolean, true
		}
		out[in.Name] = v
	}
	return out
}

func activeConfigSetID(build *pob2.PathOfBuilding2) int {
	if build.Config.ActiveConfigSet == 0 {
		return 1
	}
	return build.Config.ActiveConfigSet
}

// customModifierBlocks is the active config set's customModsList.
func customModifierBlocks(build *pob2.PathOfBuilding2) []pob2.CustomModifierBlock {
	blocks := build.Config.CustomModifierBlocks
	for _, set := range build.Config.ConfigSets {
		if set.ID == activeConfigSetID(build) {
			blocks = set.CustomModifierBlocks
		}
	}
	return blocks
}

// activeSkillSet is skillsTab.socketGroupList: the active skill set's groups, removed ones left out.
//
//pob2:block:src/Classes/SkillsTab.lua:393-398
func activeSkillSet(env *Environment) []pob2.Skill {
	skills := env.Build.Skills
	var list []pob2.Skill
	if len(skills.SkillSets) == 0 {
		list = skills.Skills
	} else {
		list = skills.SkillSets[0].Skills
		for _, set := range skills.SkillSets {
			if set.ID == skills.ActiveSkillSet {
				list = set.Skills
			}
		}
	}
	out := make([]pob2.Skill, 0, len(list))
	for _, s := range list {
		if s.Removed != nil && *s.Removed {
			continue
		}
		out = append(out, s)
	}
	return out
}

// useSecondWeaponSet is itemsTab.activeItemSet.useSecondWeaponSet and 2 or 1.
//
//pob2:block:src/Classes/SkillsTab.lua:1400
func useSecondWeaponSet(env *Environment) int {
	items := env.Build.Items
	for _, set := range items.ItemSets {
		if set.ID == items.ActiveItemSet {
			if !set.UseSecondWeaponSet.IsNil && set.UseSecondWeaponSet.V {
				return 2
			}
			return 1
		}
	}
	if !items.UseSecondWeaponSet.IsNil && items.UseSecondWeaponSet.V {
		return 2
	}
	return 1
}

//pob2:block:src/Classes/PassiveSpec.lua:1499-1503
func allocateFromNodeRadius(modList *moddb.ModList) []tree2.NodeRadius {
	var out []tree2.NodeRadius
	for _, v := range modList.List(nil, "AllocateFromNodeRadius") {
		t, ok := v.(LuaTable)
		if !ok {
			continue
		}
		r := tree2.NodeRadius{}
		r.From, _ = t["from"].(string)
		idx, _ := t["radiusIndex"].(float64)
		r.RadiusIndex = int(idx)
		to, _ := t["to"].([]any)
		for _, x := range to {
			if s, ok := x.(string); ok {
				r.To = append(r.To, s)
			}
		}
		out = append(out, r)
	}
	return out
}

// UsedTables is the env.skillsUsed and env.*Used tables calcs.buildOutput fills for the config tab.
type UsedTables struct {
	SkillsUsed           map[string]bool
	ConditionsUsed       map[string][]mod.Mod
	EnemyConditionsUsed  map[string][]mod.Mod
	MinionConditionsUsed map[string][]mod.Mod
	MultipliersUsed      map[string][]mod.Mod
	EnemyMultipliersUsed map[string][]mod.Mod
	PerStatsUsed         map[string][]mod.Mod
	EnemyPerStatsUsed    map[string][]mod.Mod
	TagTypesUsed         map[string][]mod.Mod
	ModsUsed             map[string][]mod.Mod
}

var (
	usedFireOrIgnite = regexp.MustCompile(`[fi][ig][rn][ei]t?e?`)
	usedColdOrFreeze = regexp.MustCompile(`[cf][or][le][de]z?e?`)
)

// envMinion is env.minion, the main skill's minion; calculator2 keeps it on the skill.
func envMinion(env *Environment) *Minion {
	if env.Player.MainSkill == nil {
		return nil
	}
	return env.Player.MainSkill.Minion
}

// BuildOutputUsed is the tail of calcs.buildOutput's "MAIN" mode that records which skills, conditions,
// multipliers, stats, tag types and mods the build reads, for the config tab's option visibility.
//
//pob2:block:src/Modules/Calcs.lua:589-733
func BuildOutputUsed(env *Environment) *UsedTables {
	u := &UsedTables{
		SkillsUsed:           map[string]bool{},
		ConditionsUsed:       map[string][]mod.Mod{},
		EnemyConditionsUsed:  map[string][]mod.Mod{},
		MinionConditionsUsed: map[string][]mod.Mod{},
		MultipliersUsed:      map[string][]mod.Mod{},
		EnemyMultipliersUsed: map[string][]mod.Mod{},
		PerStatsUsed:         map[string][]mod.Mod{},
		EnemyPerStatsUsed:    map[string][]mod.Mod{},
		TagTypesUsed:         map[string][]mod.Mod{},
		ModsUsed:             map[string][]mod.Mod{},
	}
	for _, activeSkill := range env.Player.ActiveSkillList {
		for _, skillEffect := range activeSkill.EffectList {
			u.SkillsUsed[skillEffect.GrantedEffect.Name] = true
		}
		if activeSkill.Minion != nil {
			for _, minionSkill := range activeSkill.Minion.ActiveSkillList {
				u.SkillsUsed[minionSkill.ActiveEffect.GrantedEffect.ID] = true
			}
		}
	}

	addTo := func(out map[string][]mod.Mod, v string, m mod.Mod) {
		// Do not count Base mods as mods being actually used as they are only used as descriptors for mods
		if m.GetSource() == "Base" {
			return
		}
		out[v] = append(out[v], m)
	}
	addVarTag := func(out map[string][]mod.Mod, vars []string, m mod.Mod) {
		for _, v := range vars {
			addTo(out, v, m)
		}
	}
	// addStatTag reads tag.statList only when the tag has a varList, which a PerStat tag never has, so
	// a statList tag records nothing.
	addStatTag := func(out map[string][]mod.Mod, statList []string, m mod.Mod) {
		if len(statList) == 1 {
			addTo(out, statList[0], m)
		}
	}
	thresholdVars := func(t *mod.MultiplierThresholdTag) []string {
		if len(t.VariableList) > 0 {
			return t.VariableList
		}
		return []string{t.Variable}
	}
	minion := envMinion(env)
	addModTags := func(isPlayer bool, m mod.Mod) {
		addTo(u.ModsUsed, m.Name(), m)

		// Imply enemy conditionals based on damage type
		// Needed to preemptively show config options for elemental ailments
		name := strings.ToLower(m.Name())
		if usedFireOrIgnite.MatchString(name) {
			addVarTag(u.EnemyConditionsUsed, []string{"Ignited", "Burning"}, m)
		}
		if usedColdOrFreeze.MatchString(name) {
			addTo(u.EnemyConditionsUsed, "Frozen", m)
		}

		for _, tag := range m.Tags() {
			addTo(u.TagTypesUsed, string(tag.Type()), m)
			switch t := tag.(type) {
			case *mod.IgnoreCondTag:
				return
			case *mod.ConditionTag:
				if isPlayer {
					addVarTag(u.ConditionsUsed, t.VarList, m)
				} else {
					addVarTag(u.MinionConditionsUsed, t.VarList, m)
				}
			case *mod.ActorConditionTag:
				if len(t.VariableList) != 1 {
					continue
				}
				if t.Actor != nil && *t.Actor == "enemy" {
					addTo(u.EnemyConditionsUsed, t.VariableList[0], m)
				} else {
					addTo(u.ConditionsUsed, t.VariableList[0], m)
				}
			case *mod.MultiplierTag:
				if t.TagActor == "" {
					if isPlayer {
						addVarTag(u.MultipliersUsed, t.VariableList, m)
					}
				} else if t.TagActor == "enemy" {
					addVarTag(u.EnemyMultipliersUsed, t.VariableList, m)
				}
			case *mod.MultiplierThresholdTag:
				if t.TagActor == "" {
					if isPlayer {
						addVarTag(u.MultipliersUsed, thresholdVars(t), m)
					}
				} else if t.TagActor == "enemy" {
					addVarTag(u.EnemyMultipliersUsed, thresholdVars(t), m)
				}
			case *mod.PerStatTag:
				if t.TagActor == "" {
					if isPlayer {
						addStatTag(u.PerStatsUsed, t.StatList, m)
					}
				} else if t.TagActor == "enemy" {
					addStatTag(u.EnemyPerStatsUsed, t.StatList, m)
				}
			case *mod.StatThresholdTag:
				if isPlayer && t.Stat != "" {
					addTo(u.PerStatsUsed, t.Stat, m)
				}
			}
		}
	}
	addModDB := func(isPlayer bool, db *moddb.ModDB) {
		for _, modList := range db.Mods {
			for _, m := range modList {
				addModTags(isPlayer, m)
			}
		}
	}
	addModDB(true, env.Player.ModDB)
	if minion != nil && minion.ModDB != nil {
		addModDB(false, minion.ModDB)
	}
	for _, activeSkill := range env.Player.ActiveSkillList {
		if activeSkill.BaseSkillModList != nil {
			for _, m := range activeSkill.BaseSkillModList.Mods() {
				addModTags(true, m)
			}
		}
		if activeSkill.SkillModList != nil {
			for _, m := range activeSkill.SkillModList.Mods() {
				addTo(u.ModsUsed, m.Name(), m)
				for _, tag := range m.Tags() {
					addTo(u.TagTypesUsed, string(tag.Type()), m)
				}
			}
		}
		if activeSkill.Minion != nil {
			for _, minionSkill := range activeSkill.Minion.ActiveSkillList {
				if minionSkill.BaseSkillModList != nil {
					for _, m := range minionSkill.BaseSkillModList.Mods() {
						addModTags(false, m)
					}
				}
			}
		}
	}
	for _, modList := range env.EnemyDB.Mods {
		for _, m := range modList {
		tags:
			for _, tag := range m.Tags() {
				switch t := tag.(type) {
				case *mod.IgnoreCondTag:
					break tags
				case *mod.ConditionTag:
					addVarTag(u.EnemyConditionsUsed, t.VarList, m)
				case *mod.ActorConditionTag:
					if len(t.VariableList) != 1 {
						continue
					}
					if t.Actor != nil && (*t.Actor == "enemy" || *t.Actor == "player") {
						addTo(u.ConditionsUsed, t.VariableList[0], m)
					} else {
						addTo(u.EnemyConditionsUsed, t.VariableList[0], m)
					}
				case *mod.MultiplierTag:
					if t.TagActor == "" {
						addVarTag(u.EnemyMultipliersUsed, t.VariableList, m)
					} else if t.TagActor == "enemy" || t.TagActor == "player" {
						addVarTag(u.MultipliersUsed, t.VariableList, m)
					}
				case *mod.MultiplierThresholdTag:
					if t.TagActor == "" {
						addVarTag(u.EnemyMultipliersUsed, thresholdVars(t), m)
					} else if t.TagActor == "enemy" || t.TagActor == "player" {
						addVarTag(u.MultipliersUsed, thresholdVars(t), m)
					}
				}
			}
		}
	}
	return u
}

// ConfigRelevance holds, per predicate kind of ConfigOptions.lua, the names a query asks about; the
// answer keeps the ones the calculated build satisfies. SkillTypes are SkillType numbers as strings.
type ConfigRelevance struct {
	Conds              []string
	MinionConds        []string
	EnemyConds         []string
	Mults              []string
	EnemyMults         []string
	Stats              []string
	EnemyStats         []string
	TagTypes           []string
	Flags              []string
	PlayerFlags        []string
	MinionFlags        []string
	Mods               []string
	Skills             []string
	TransfiguredSkills []string
	SkillTypes         []string
	SkillFlags         []string
	SkillData          []string
	GemFamilies        []string
}

// ConfigRelevance is the shownFuncs ConfigTab builds for each option, one name at a time; the
// implyCond and ifOption arms read the config inputs and stay with the caller.
//
//pob2:block:src/Classes/ConfigTab.lua:411-702
func (env *Environment) ConfigRelevance(u *UsedTables, query ConfigRelevance) ConfigRelevance {
	keep := func(names []string, pass func(string) bool) []string {
		out := make([]string, 0)
		for _, name := range names {
			if pass(name) {
				out = append(out, name)
			}
		}
		return out
	}
	in := func(m map[string][]mod.Mod) func(string) bool {
		return func(name string) bool { _, ok := m[name]; return ok }
	}
	minion := envMinion(env)
	skillFlagged := func(skill *ActiveSkill, name string) bool {
		if skill == nil {
			return false
		}
		if skill.ActiveEffect != nil && skill.ActiveEffect.StatSet != nil && skill.ActiveEffect.StatSet.SkillFlags[name] {
			return true
		}
		return skill.SkillModList != nil && skill.SkillModList.Flag(nil, name)
	}
	skillTypeNames := map[int]string{}
	for name, id := range poe2.SkillType {
		skillTypeNames[id] = name
	}
	return ConfigRelevance{
		Conds:       keep(query.Conds, in(u.ConditionsUsed)),
		MinionConds: keep(query.MinionConds, in(u.MinionConditionsUsed)),
		EnemyConds:  keep(query.EnemyConds, in(u.EnemyConditionsUsed)),
		Mults:       keep(query.Mults, in(u.MultipliersUsed)),
		EnemyMults:  keep(query.EnemyMults, in(u.EnemyMultipliersUsed)),
		Stats: keep(query.Stats, func(name string) bool {
			return in(u.PerStatsUsed)(name) || in(u.EnemyMultipliersUsed)(name)
		}),
		EnemyStats: keep(query.EnemyStats, in(u.EnemyPerStatsUsed)),
		TagTypes:   keep(query.TagTypes, in(u.TagTypesUsed)),
		// Check both the skill mods for flags and flags that are set via calcPerform
		Flags: keep(query.Flags, func(name string) bool {
			if skillFlagged(env.Player.MainSkill, name) {
				return true
			}
			return minion != nil && skillFlagged(minion.MainSkill, name)
		}),
		PlayerFlags: keep(query.PlayerFlags, func(name string) bool {
			return minion == nil && skillFlagged(env.Player.MainSkill, name)
		}),
		MinionFlags: keep(query.MinionFlags, func(name string) bool {
			return minion != nil && minion.ModDB != nil && minion.ModDB.Flag(nil, name)
		}),
		Mods:   keep(query.Mods, in(u.ModsUsed)),
		Skills: keep(query.Skills, func(name string) bool { return u.SkillsUsed[name] }),
		TransfiguredSkills: keep(query.TransfiguredSkills, func(name string) bool {
			if getGameIDFromGemName(name) == "" {
				return false
			}
			for skill := range u.SkillsUsed {
				if isGemIDSame(skill, name) {
					return true
				}
			}
			return false
		}),
		// only checking flags of skill in main env. rework may be required
		SkillTypes: keep(query.SkillTypes, func(id string) bool {
			n, err := strconv.Atoi(id)
			if err != nil {
				return false
			}
			for _, activeSkill := range env.Player.ActiveSkillList {
				if activeSkill.SkillTypes[skillTypeNames[n]] {
					return true
				}
			}
			return false
		}),
		SkillFlags: keep(query.SkillFlags, func(name string) bool {
			for _, activeSkill := range env.Player.ActiveSkillList {
				if activeSkill.ActiveEffect != nil && activeSkill.ActiveEffect.StatSet != nil && activeSkill.ActiveEffect.StatSet.SkillFlags[name] {
					return true
				}
			}
			return false
		}),
		SkillData: keep(query.SkillData, func(name string) bool {
			for _, activeSkill := range env.Player.ActiveSkillList {
				if v, ok := activeSkill.SkillData[name]; ok && v != nil && v != false {
					return true
				}
			}
			return false
		}),
		GemFamilies: keep(query.GemFamilies, func(family string) bool {
			for _, activeSkill := range env.Player.ActiveSkillList {
				for _, support := range activeSkill.SupportList {
					if support.GemData != nil && support.GemData.Data != nil && support.GemData.Data.GemFamily == family {
						return true
					}
				}
			}
			return false
		}),
	}
}

// SocketGroupView is one entry of the sidebar's main socket group list: its displayLabel and the
// labels of its displaySkillList.
type SocketGroupView struct {
	Label  string
	Skills []string
}

//pob2:src/Modules/CalcActiveSkill.lua:265-276
func getActiveSkillDisplayName(activeSkill *ActiveSkill) string {
	skillName := activeSkill.ActiveEffect.GrantedEffect.Name
	skillMinion := activeSkill.Minion
	if skillMinion != nil && skillMinion.MinionData != nil {
		if strings.HasPrefix(skillName, "Companion:") {
			return "Companion: " + skillMinion.MinionData.Name
		} else if strings.HasPrefix(skillName, "Spectre:") {
			return "Spectre: " + skillMinion.MinionData.Name
		}
	}
	return skillName
}

// SocketGroupViews is RefreshSkillSelectControls' socket group and main skill lists. The display label
// and skill list are the ones CalcSetup sets in "MAIN" mode; displaySkillList is the group's skills
// from the player's activeSkillList, in creation order. An explode source's "From" label is not ported.
//
//pob2:block:src/Modules/CalcSetup.lua:2406-2431
//pob2:block:src/Modules/Build.lua:2062-2087
func SocketGroupViews(env *Environment) []SocketGroupView {
	out := make([]SocketGroupView, 0, len(env.SocketGroupList))
	for _, group := range env.SocketGroupList {
		var label string
		// Create display label for the socket group if the user didn't specify one
		if strings.TrimSpace(group.Label) != "" {
			label = group.Label
		} else {
			for _, gemInstance := range group.GemList {
				grantedEffect := gemInstance.GrantedEffect
				var gemName string
				if gemInstance.GemData != nil {
					grantedEffect = gemInstance.GemData.GrantedEffect
					gemName = gemInstance.GemData.Data.Name
				}
				if grantedEffect != nil && !grantedEffect.Support && gemInstance.Enabled {
					name := gemName
					if strings.HasPrefix(grantedEffect.Name, "Companion:") || strings.HasPrefix(grantedEffect.Name, "Spectre:") {
						name = gemInstance.NameSpec
					} else if name == "" {
						name = grantedEffect.Name
					}
					if label != "" {
						label += ", "
					}
					label += name
				}
			}
			if label == "" {
				label = "<No active skills>"
			}
		}
		view := SocketGroupView{Label: label, Skills: []string{}}
		if group.UsingSkillSet == env.WeaponSet {
			for _, activeSkill := range env.Player.ActiveSkillList {
				if activeSkill.SocketGroup == group {
					view.Skills = append(view.Skills, getActiveSkillDisplayName(activeSkill))
				}
			}
		}
		out = append(out, view)
	}
	return out
}

// savedSocketGroup is the build file's entry for socketGroupList[index]: activeSkillSet's list, by
// reference.
//
//pob2:block:src/Classes/SkillsTab.lua:393-398
func savedSocketGroup(build *pob2.PathOfBuilding2, index int) *pob2.Skill {
	skills := &build.Skills
	list := &skills.Skills
	if len(skills.SkillSets) > 0 {
		list = &skills.SkillSets[0].Skills
		for i := range skills.SkillSets {
			if skills.SkillSets[i].ID == skills.ActiveSkillSet {
				list = &skills.SkillSets[i].Skills
			}
		}
	}
	n := 0
	for i := range *list {
		s := &(*list)[i]
		if s.Removed != nil && *s.Removed {
			continue
		}
		n++
		if n == index {
			return s
		}
	}
	return nil
}

// SetMainSocketGroup is the sidebar's main socket group dropdown.
//
//pob2:block:src/Modules/Build.lua:364-368
func SetMainSocketGroup(build *pob2.PathOfBuilding2, index int) {
	build.Build.MainSocketGroup = index
}

// SetMainActiveSkill is the sidebar's main skill dropdown: the main socket group's mainActiveSkill.
//
//pob2:block:src/Modules/Build.lua:376-381
func SetMainActiveSkill(build *pob2.PathOfBuilding2, index int) {
	if group := savedSocketGroup(build, max(build.Build.MainSocketGroup, 1)); group != nil {
		group.MainActiveSkill = pob2.Nil[int]{V: index}
	}
}

// activeConfigInputs is the active config set's saved input list; a file without config sets keeps
// the pre-ConfigSet layout Load reads into set 1.
//
//pob2:block:src/Classes/ConfigTab.lua:927-941
func activeConfigInputs(build *pob2.PathOfBuilding2) *[]pob2.Input {
	id := activeConfigSetID(build)
	for i := range build.Config.ConfigSets {
		if build.Config.ConfigSets[i].ID == id {
			return &build.Config.ConfigSets[i].Inputs
		}
	}
	return &build.Config.Inputs
}

// SetConfigInput is an option control's callback: input[var] = value. A value equal to the option's
// default state is dropped, as Save writes only inputs that differ from it; the caller passes that
// verdict as isDefault, since the defaults live in ConfigOptions.lua.
//
//pob2:block:src/Classes/ConfigTab.lua:308-331
//pob2:block:src/Classes/ConfigTab.lua:1009-1021
func SetConfigInput(build *pob2.PathOfBuilding2, name string, value ConfigValue, isDefault bool) {
	inputs := activeConfigInputs(build)
	idx := -1
	for i := range *inputs {
		if (*inputs)[i].Name == name {
			idx = i
			break
		}
	}
	if isDefault || !value.IsNumber && !value.IsString && !value.IsBoolean {
		if idx >= 0 {
			*inputs = append((*inputs)[:idx], (*inputs)[idx+1:]...)
		}
		return
	}
	in := pob2.Input{Name: name}
	switch {
	case value.IsNumber:
		v := value.Number
		in.Number = &v
	case value.IsBoolean:
		v := value.Boolean
		in.Boolean = &v
	default:
		v := value.String
		in.String = &v
	}
	if idx >= 0 {
		in.Unparsed = (*inputs)[idx].Unparsed
		(*inputs)[idx] = in
		return
	}
	*inputs = append(*inputs, in)
}

// ConfigInputs is the active config set's input table as configInput reads it.
func ConfigInputs(build *pob2.PathOfBuilding2) map[string]ConfigValue {
	return configInput(build)
}
