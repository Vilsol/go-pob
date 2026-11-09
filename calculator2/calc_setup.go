package calculator2

import (
	"maps"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob2"
)

// newModList is new("ModList"):ModList() carrying PoB-PoE2's kernel rules, as every list here evaluates PoE2 mods.
func newModList() *moddb.ModList {
	l := moddb.NewModList()
	l.Rules = kernelRules
	return l
}

func newMod(name string, modType mod.Type, value float64, source mod.Source, tags ...mod.Tag) mod.Mod {
	m := mod.NewFloat(name, modType, value).Source(source)
	if len(tags) > 0 {
		m.Tag(tags...)
	}
	return m
}

func newFlagMod(name string, source mod.Source, tags ...mod.Tag) mod.Mod {
	m := mod.NewFlag(name, true).Source(source)
	if len(tags) > 0 {
		m.Tag(tags...)
	}
	return m
}

// characterConstant is data.characterConstants[name]; a name PoB-PoE2 does not define would make PoB
// add a nil-valued mod, so it is a data defect.
func characterConstant(name string) float64 {
	v, ok := poe2.CharacterConstants[name]
	if !ok {
		panic("data.characterConstants has no " + name)
	}
	return v
}

func gameConstant(name string) float64 {
	v, ok := poe2.GameConstants[name]
	if !ok {
		panic("data.gameConstants has no " + name)
	}
	return v
}

func miscConstant(name string) float64 {
	v, ok := poe2.Misc[name]
	if !ok {
		panic("data.misc has no " + name)
	}
	return v
}

//pob2:src/Modules/CalcSetup.lua:26-114
func initModDB(env *Environment, modDB *moddb.ModDB) {
	maxRes := characterConstant("base_maximum_all_resistances_%")
	modDB.AddMod(newMod("FireResistMax", mod.TypeBase, maxRes, "Base"))
	modDB.AddMod(newMod("ColdResistMax", mod.TypeBase, maxRes, "Base"))
	modDB.AddMod(newMod("LightningResistMax", mod.TypeBase, maxRes, "Base"))
	modDB.AddMod(newMod("ChaosResistMax", mod.TypeBase, maxRes, "Base"))
	modDB.AddMod(newMod("TotemFireResistMax", mod.TypeBase, 75, "Base"))
	modDB.AddMod(newMod("TotemColdResistMax", mod.TypeBase, 75, "Base"))
	modDB.AddMod(newMod("TotemLightningResistMax", mod.TypeBase, 75, "Base"))
	modDB.AddMod(newMod("TotemChaosResistMax", mod.TypeBase, 75, "Base"))
	modDB.AddMod(newMod("BaseBlockChanceMax", mod.TypeBase, characterConstant("object_inherent_base_maximum_block_%_from_ot"), "Base"))
	modDB.AddMod(newMod("BaseSpellBlockChanceMax", mod.TypeBase, characterConstant("object_inherent_base_maximum_block_%_from_ot"), "Base"))
	modDB.AddMod(newMod("SpellDodgeChanceMax", mod.TypeBase, 75, "Base"))
	modDB.AddMod(newMod("ChargeDuration", mod.TypeBase, 15, "Base"))
	modDB.AddMod(newMod("PowerChargesMax", mod.TypeBase, characterConstant("max_power_charges"), "Base"))
	modDB.AddMod(newMod("FrenzyChargesMax", mod.TypeBase, characterConstant("max_frenzy_charges"), "Base"))
	modDB.AddMod(newMod("EnduranceChargesMax", mod.TypeBase, characterConstant("max_endurance_charges"), "Base"))
	modDB.AddMod(newMod("MaximumRage", mod.TypeBase, gameConstant("BaseMaximumRage"), "Base"))
	modDB.AddMod(newMod("SiphoningChargesMax", mod.TypeBase, 0, "Base"))
	modDB.AddMod(newMod("ChallengerChargesMax", mod.TypeBase, 0, "Base"))
	modDB.AddMod(newMod("BlitzChargesMax", mod.TypeBase, 0, "Base"))
	modDB.AddMod(newMod("InspirationChargesMax", mod.TypeBase, 5, "Base"))
	modDB.AddMod(newMod("CrabBarriersMax", mod.TypeBase, 0, "Base"))
	modDB.AddMod(newMod("BrutalChargesMax", mod.TypeBase, 0, "Base"))
	modDB.AddMod(newMod("AbsorptionChargesMax", mod.TypeBase, 0, "Base"))
	modDB.AddMod(newMod("AfflictionChargesMax", mod.TypeBase, 0, "Base"))
	modDB.AddMod(newMod("BloodChargesMax", mod.TypeBase, 5, "Base"))
	modDB.AddMod(newMod("MaxLifeLeechRate", mod.TypeBase, 20, "Base"))
	modDB.AddMod(newMod("MaxManaLeechRate", mod.TypeBase, 20, "Base"))
	modDB.AddMod(newMod("ImpaleStacksMax", mod.TypeBase, 5, "Base"))
	modDB.AddMod(newMod("IgniteStacksMax", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("BleedStacksMax", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("PoisonStacksMax", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("MaxEnergyShieldLeechRate", mod.TypeBase, 10, "Base"))
	modDB.AddMod(newMod("MaxLifeLeechInstance", mod.TypeBase, 10, "Base"))
	modDB.AddMod(newMod("MaxManaLeechInstance", mod.TypeBase, 10, "Base"))
	modDB.AddMod(newMod("MaxEnergyShieldLeechInstance", mod.TypeBase, 10, "Base"))
	modDB.AddMod(newMod("TrapThrowingTime", mod.TypeBase, 0.6, "Base"))
	modDB.AddMod(newMod("MineLayingTime", mod.TypeBase, 0.3, "Base"))
	modDB.AddMod(newMod("WarcryCastTime", mod.TypeBase, 0.8, "Base"))
	modDB.AddMod(newMod("TotemPlacementTime", mod.TypeBase, 0.6, "Base"))
	modDB.AddMod(newMod("BallistaPlacementTime", mod.TypeBase, 0.35, "Base"))
	modDB.AddMod(newMod("ShockStacksMax", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("ChillStacksMax", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("ScorchStacksMax", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("MovementSpeed", mod.TypeIncrease, -30, "Base", mod.Condition("Maimed")))
	modDB.AddMod(newMod("Evasion", mod.TypeIncrease, -15, "Base", mod.Condition("Maimed")))
	modDB.AddMod(newMod("MovementSpeed", mod.TypeIncrease, -30, "Base", mod.Condition("Hindered")))
	modDB.AddMod(newMod("AilmentThreshold", mod.TypeBase, 50, "Base", mod.PercentStat("Life", 1)))
	modDB.AddMod(newMod("PoiseThreshold", mod.TypeBase, 50, "Base", mod.PercentStat("Life", 1)))
	modDB.AddMod(newMod("DamageTaken", mod.TypeIncrease, 10, "Base", mod.Condition("Intimidated")))
	modDB.AddMod(newMod("DamageTaken", mod.TypeIncrease, 10, "Base", mod.Condition("Intimidated").Neg(true), mod.Condition("Party:Intimidated")))
	modDB.AddMod(newMod("Damage", mod.TypeIncrease, -10, "Base", mod.Condition("Intimidated")))
	modDB.AddMod(newMod("Damage", mod.TypeIncrease, -10, "Base", mod.Condition("Intimidated").Neg(true), mod.Condition("Party:Intimidated")))
	modDB.AddMod(newMod("DamageTaken", mod.TypeIncrease, 10, "Base", mod.Condition("Unnerved")).Flag(modFlagSpell))
	modDB.AddMod(newMod("DamageTaken", mod.TypeIncrease, 10, "Base", mod.Condition("Unnerved").Neg(true), mod.Condition("Party:Unnerved")).Flag(modFlagSpell))
	modDB.AddMod(newMod("Damage", mod.TypeMore, -10, "Base", mod.Condition("Debilitated")))
	modDB.AddMod(newMod("DamageGainAsRandom", mod.TypeBase, 2, "Faerie Fire", mod.Multiplier("FaerieFireStack").Limit(10).Actor("enemy")))
	modDB.AddMod(newFlagMod("Condition:Burning", "Base", mod.IgnoreCond(), mod.Condition("Ignited")))
	modDB.AddMod(newFlagMod("Condition:Poisoned", "Base", mod.IgnoreCond(), mod.MultiplierThreshold("PoisonStack").Threshold(1)))
	modDB.AddMod(newMod("ShockBase", mod.TypeBase, gameConstant("BaseShockMagnitude"), "Base", mod.ActorCondition("enemy", "Shocked")))
	modDB.AddMod(newMod("ChillBase", mod.TypeBase, 30, "Base", mod.ActorCondition("enemy", "Chilled")))
	for _, flag := range []string{"Blind:Blinded", "Chill:Chilled", "Freeze:Frozen", "Fortify:Fortify", "Fortified:Fortified",
		"Fanaticism:Fanaticism", "Onslaught:Onslaught", "UnholyMight:UnholyMight", "ChaoticMight:ChaoticMight",
		"Adrenaline:Adrenaline", "LesserMassiveShrine:LesserMassiveShrine", "LesserBrutalShrine:LesserBrutalShrine",
		"DiamondShrine:DiamondShrine", "MassiveShrine:MassiveShrine", "AlchemistsGenius:AlchemistsGenius", "LuckyHits:LuckyHits"} {
		name, cond, _ := strings.Cut(flag, ":")
		modDB.AddMod(newFlagMod(name, "Base", mod.Condition(cond)))
	}
	modDB.AddMod(newMod("HeavyStunBuildup", mod.TypeMore, 50, "Base", mod.Condition("Dazed")))
	// PoB passes no source here.
	modDB.AddMod(mod.NewFlag("ColdCannotHeavyStun", true))
	modDB.AddMod(newFlagMod("Convergence", "Base", mod.Condition("Convergence")))
	modDB.AddMod(newMod("PhysicalDamageReduction", mod.TypeBase, -15, "Base", mod.Condition("Crushed")))
	modDB.AddMod(newMod("CritChanceCap", mod.TypeBase, 100, "Base"))
	modDB.AddMod(newMod("PresenceRadius", mod.TypeBase, characterConstant("base_presence_radius"), "Base"))
	modDB.AddMod(newMod("SurroundedRadius", mod.TypeBase, miscConstant("SurroundedRadiusBase"), "Base"))
	modDB.AddMod(newMod("SurroundedMinimum", mod.TypeBase, gameConstant("BaseRequiredEnemiesToBeConsideredSurrounded"), "Base"))
	modDB.Conditions["Buffed"] = env.ModeBuffs
	modDB.Conditions["Combat"] = env.ModeCombat
	modDB.Conditions["Effective"] = env.ModeEffective
}

// PoB-PoE2's Data/Global.lua ModFlag values; they differ from PoE1's from Thorns on.
//
//pob2:block:src/Data/Global.lua:3-38
const (
	modFlagAttack = mod.MFlag(0x0000000000000001)
	modFlagSpell  = mod.MFlag(0x0000000000000002)
	modFlagDot    = mod.MFlag(0x0000000000000008)
	modFlagCast   = mod.MFlag(0x0000000000000010)
	modFlagMelee  = mod.MFlag(0x0000000000000100)
)

//pob2:block:src/Data/Global.lua:296
const keywordFlagBleed = mod.KeywordFlag(0x00400000)

var weaponSetConditionRE = regexp.MustCompile(`^WeaponSet`)

// processStats is PassiveTree:ProcessStats for one node's stat lines: the node's modList.
//
//pob2:src/Classes/PassiveTree.lua:467-544
func processStats(source ModSource, nodeID int, nodeType, dn string, sd []string) *moddb.ModList {
	modList := newModList()
	var lines []string
	for _, line := range sd {
		for _, l := range strings.Split(line, "\n") {
			if l != "" {
				lines = append(lines, l)
			}
		}
	}
	type nodeMod struct {
		list     []mod.Mod
		extra    string
		combined bool
		set      bool
	}
	mods := make([]nodeMod, len(lines))
	for i := 0; i < len(lines); {
		line := lines[i]
		list, extra := source(line)
		if list == nil || extra != "" {
			for endI := i + 1; endI < len(lines); endI++ {
				comb := line
				for ci := i + 1; ci <= endI; ci++ {
					comb += " " + lines[ci]
				}
				list, extra = source(comb)
				if list != nil && extra == "" {
					for ci := i + 1; ci <= endI; ci++ {
						mods[ci] = nodeMod{list: []mod.Mod{}, combined: true, set: true}
					}
					break
				}
			}
		}
		mods[i] = nodeMod{list: list, extra: extra, set: true}
		i++
		for i < len(lines) && mods[i].set {
			i++
		}
	}
	nodeSource := mod.Source("Tree:" + strconv.Itoa(nodeID))
	for _, m := range mods {
		if m.list != nil && m.extra == "" {
			for _, mo := range m.list {
				modList.AddMod(mod.SetSource(mo, nodeSource))
			}
		}
	}
	return modList
}

//pob2:block:src/Classes/PassiveTree.lua:541-543
func keystoneMod(nodeID int, dn string) mod.Mod {
	return mod.NewList("Keystone", dn).Source(mod.Source("Tree" + strconv.Itoa(nodeID)))
}

func (env *Environment) nodeModList(node *tree2.SpecNode) *moddb.ModList {
	if l, ok := env.nodeModLists[node.ID]; ok {
		return l
	}
	l := processStats(env.modSource, node.ID, node.Type, node.Dn, node.Sd)
	env.nodeModLists[node.ID] = l
	return l
}

//pob2:src/Modules/CalcSetup.lua:200-412
func buildModListForNode(env *Environment, node *tree2.SpecNode, incSmallPassiveSkill float64, includeKeystoneMods bool) *moddb.ModList {
	localSmallIncEffect := 0.0
	localNotableIncEffect := 0.0
	modList := newModList()
	if node.Type == "Keystone" {
		if includeKeystoneMods {
			modList.AddList(env.nodeModList(node))
		}
		modList.AddMod(keystoneMod(node.ID, node.Dn))
	} else {
		modList.AddList(env.nodeModList(node))
	}

	radiusFirstPass(env, node, modList)

	var hasNoEffect, hasAllocNoEffect, hasScale, hasOtherEffect bool
	for _, m := range modList.Mods() {
		switch m.Name() {
		case "PassiveSkillHasNoEffect":
			hasNoEffect = true
		case "AllocatedPassiveSkillHasNoEffect":
			hasAllocNoEffect = true
		case "PassiveSkillEffect":
			hasScale = true
		case "PassiveSkillHasOtherEffect":
			hasOtherEffect = true
		}
	}

	_, allocated := env.AllocNodes[node.ID]
	if (hasNoEffect && modList.Flag(nil, "PassiveSkillHasNoEffect")) || (allocated && (hasAllocNoEffect && modList.Flag(nil, "AllocatedPassiveSkillHasNoEffect"))) {
		modList = newModList()
		hasScale = false
		hasOtherEffect = false
	}

	if hasScale {
		scale := calcLibMod(modList, nil, "PassiveSkillEffect")
		if scale != 1 {
			scaledList := newModList()
			scaledList.ScaleAddList(modList, scale)
			modList = scaledList
		}
	}

	if radiusSecondPass(env, node, modList) {
		hasOtherEffect = false
		for _, m := range modList.Mods() {
			if m.Name() == "PassiveSkillHasOtherEffect" {
				hasOtherEffect = true
			}
		}
	}

	if hasOtherEffect && modList.Flag(nil, "PassiveSkillHasOtherEffect") {
		newMods := modList.List(nil, "NodeModifier")
		for i, v := range newMods {
			if i == 0 {
				modList = newModList()
			}
			switch nm := v.(type) {
			case nodeModifier:
				modList.AddMod(nm.Mod)
			case LuaTable:
				if m, ok := nm["mod"].(mod.Mod); ok {
					modList.AddMod(m)
				}
			}
		}
	}

	//pob2:block:src/Modules/CalcSetup.lua:337-341
	if modList.Flag(nil, "CanExplode") {
		env.explodeSources = append(env.explodeSources, node)
	}

	out := newModList()
	for _, m := range modList.Mods() {
		m = m.Clone()
		added := false
		tags := m.Tags()
		kept := make([]mod.Tag, 0, len(tags)+1)
		for _, t := range tags {
			if c, ok := t.(*mod.ConditionTag); ok && len(c.VarList) > 0 && weaponSetConditionRE.MatchString(c.VarList[0]) {
				added = true
				if node.AllocMode != 0 {
					kept = append(kept, mod.Condition("WeaponSet"+strconv.Itoa(node.AllocMode)).Neg(c.Negative))
				}
				continue
			}
			kept = append(kept, t)
		}
		if !added && node.AllocMode != 0 {
			kept = append(kept, mod.Condition("WeaponSet"+strconv.Itoa(node.AllocMode)))
		}
		m.ClearTags()
		if len(kept) > 0 {
			m.Tag(kept...)
		}

		if m.Name() == "JewelSmallPassiveSkillEffect" || m.Name() == "JewelNotablePassiveSkillEffect" {
			hasWSCondition := false
			value := 0.0
			if v := m.Value(); v != nil {
				value = v.Float()
			}
			effect := 0.0
			for _, t := range m.Tags() {
				if c, ok := t.(*mod.ConditionTag); ok && len(c.VarList) > 0 && weaponSetConditionRE.MatchString(c.VarList[0]) {
					if n, err := strconv.Atoi(c.VarList[0][len(c.VarList[0])-1:]); err == nil && n == env.WeaponSet {
						effect = value
					}
					hasWSCondition = true
				}
			}
			if !hasWSCondition {
				effect = value
			}
			if m.Name() == "JewelSmallPassiveSkillEffect" {
				localSmallIncEffect = effect
			} else {
				localNotableIncEffect = effect
			}
		}
		out.AddMod(m)
	}
	modList = out

	if (incSmallPassiveSkill+localSmallIncEffect) > 0 && node.Type == "Normal" && !node.IsAttribute && node.AscendancyName == "" {
		scale := 1 + (incSmallPassiveSkill+localSmallIncEffect)/100
		scaledList := newModList()
		scaledList.ScaleAddList(modList, scale)
		modList = scaledList
	}

	if localNotableIncEffect > 0 && node.Type == "Notable" && !node.IsAttribute && node.AscendancyName == "" {
		scale := 1 + localNotableIncEffect/100
		scaledList := newModList()
		scaledList.ScaleAddList(modList, scale)
		modList = scaledList
	}

	return modList
}

func sortedNodeIDs(nodes map[int]*tree2.SpecNode) []int {
	ids := make([]int, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// PoB walks the node list with pairs(); ids are walked sorted here.
//
//pob2:src/Modules/CalcSetup.lua:415-464
func buildModListForNodeList(env *Environment, nodeList map[int]*tree2.SpecNode, includeKeystoneMods bool) *moddb.ModList {
	initRadiusJewels(env)

	inc := 0.0
	ids := sortedNodeIDs(nodeList)
	for _, id := range ids {
		inc += env.nodeModList(nodeList[id]).Sum(mod.TypeIncrease, nil, "SmallPassiveSkillEffect")
	}

	modList := newModList()
	for _, id := range ids {
		modList.AddList(buildModListForNode(env, nodeList[id], inc, includeKeystoneMods))
	}

	finishRadiusJewels(env, modList, inc)
	return modList
}

// buildConfigModList is ConfigTab:BuildModList for the options ported so far: the quest rewards
// (ConfigOptions.lua's check and list options built from data.questRewards), the options in
// configVarList and the custom modifier blocks.
//
//pob2:src/Classes/ConfigTab.lua:1169-1241
func buildConfigModList(env *Environment, modList, enemyModList *moddb.ModList) {
	addParsed := func(source mod.Source, line string) {
		mods, extra := env.modSource(line)
		if mods != nil && extra == "" {
			for _, m := range mods {
				modList.AddMod(mod.SetSource(m, source))
			}
		}
	}

	//pob2:block:src/Modules/ConfigOptions.lua:40-108
	questLine := func(source mod.Source, str string) {
		for _, line := range strings.FieldsFunc(str, func(r rune) bool { return r == '\r' || r == '\n' }) {
			addParsed(source, strings.TrimRight(strings.TrimLeft(stripEscapes(line), luaSpaceOrQuery), luaSpaceOrQuery))
		}
	}
	for _, quest := range poe2.QuestRewards {
		if quest.NoConfig {
			continue
		}
		key := "quest" + quest.Description + quest.Area + quest.Info
		source := mod.Source("Quest:" + quest.Description + ": " + quest.Area)
		input, set := env.ConfigInput[key]
		if quest.Options == nil {
			// type = "check", defaultState = true: applied while input[var] is truthy.
			if !set || !input.IsBoolean || input.Boolean {
				questLine(source, quest.Stat)
			}
			continue
		}
		// type = "list", defaultIndex = 1 ("None").
		if set && input.IsString && input.String != "None" {
			questLine(source, input.String)
		}
	}
	applyConfigVars(env, modList, enemyModList)

	for _, block := range customModifierBlocks(env.Build) {
		if block.Enabled != nil && !*block.Enabled || block.Text == "" {
			continue
		}
		title := "Default"
		if block.Title != nil {
			title = *block.Title
		}
		for _, line := range strings.Split(block.Text, "\n") {
			addParsed(mod.Source("Custom:"+title), strings.Trim(stripEscapes(line), luaSpace))
		}
	}
}

const (
	luaSpace        = " \t\n\r\f\v"
	luaSpaceOrQuery = luaSpace + "?"
)

var (
	colourEscapeRE = regexp.MustCompile(`\^[0-9]`)
	hexEscapeRE    = regexp.MustCompile(`\^x[0-9A-Fa-f]{6}`)
)

//pob2:src/_SimpleGraphic.def.lua:330-333
func stripEscapes(text string) string {
	return hexEscapeRE.ReplaceAllString(colourEscapeRE.ReplaceAllString(text, ""), "")
}

// allocatedCounts is PassiveSpec's allocatedNotableCount / SmithBodyArmour / Mastery counts.
type allocatedCounts struct {
	notable, smithBodyArmour, mastery, masteryType int
	masteryTypes                                   map[string]int
}

//pob2:block:src/Classes/PassiveSpec.lua:1793-1838
func countAllocated(spec *tree2.PassiveSpec) allocatedCounts {
	c := allocatedCounts{masteryTypes: map[string]int{}}
	for _, id := range sortedNodeIDs(spec.Nodes) {
		node := spec.Nodes[id]
		switch {
		case node.Type == "Mastery" && spec.MasterySelections[id] != 0:
			if _, ok := spec.AllocNodes[id]; ok {
				c.mastery++
				if c.masteryTypes[node.Name] == 0 {
					c.masteryType++
				}
				c.masteryTypes[node.Name]++
			}
		case node.Type == "Notable" && node.Alloc:
			c.notable++
			if node.ApplyToArmour {
				c.smithBodyArmour++
			}
		}
	}
	return c
}

// overrideAllocNodes is env.allocNodes: the spec's allocated nodes with the override's addNodes and without its
// removeNodes, with the allocated notable and mastery counts following them.
//
//pob2:block:src/Modules/CalcSetup.lua:1081-1139
func overrideAllocNodes(env *Environment, counts *allocatedCounts) map[int]*tree2.SpecNode {
	o := env.override
	var nodes map[int]*tree2.SpecNode
	if o.AddNodes != nil || o.RemoveNodes != nil {
		nodes = map[int]*tree2.SpecNode{}
		for node := range o.AddNodes {
			nodes[node.ID] = node
			switch node.Type {
			case "Mastery":
				counts.mastery++
				prevCount, ok := counts.masteryTypes[node.Name]
				counts.masteryTypes[node.Name] = prevCount + 1
				if !ok || prevCount == 0 {
					counts.masteryType++
				}
			case "Notable":
				counts.notable++
				if node.ApplyToArmour {
					counts.smithBodyArmour++
				}
			}
		}
		for _, node := range env.Spec.AllocNodes {
			if !o.RemoveNodes[node] {
				nodes[node.ID] = node
				continue
			}
			switch node.Type {
			case "Mastery":
				counts.mastery--
				counts.masteryTypes[node.Name]--
				if counts.masteryTypes[node.Name] == 0 {
					counts.masteryType--
				}
			case "Notable":
				counts.notable--
				if node.ApplyToArmour {
					counts.smithBodyArmour--
				}
			}
		}
	} else {
		nodes = make(map[int]*tree2.SpecNode, len(env.Spec.AllocNodes))
		for id, node := range env.Spec.AllocNodes {
			nodes[id] = node
		}
	}
	for id, node := range nodes {
		if node.IsGrantedPassive && node.IsFreeAllocate {
			delete(nodes, id)
		}
	}
	return nodes
}

// accelerate is calcs.initEnv's specEnv.accelerate: the blocks an env reused by wipeEnv keeps rather than rebuilds.
type accelerate struct {
	nodeAlloc, requirementsItems, requirementsGems, skills, everything bool
}

// specEnv is calcs.initEnv's specEnv: the base databases a previous call cached, and the env to reuse.
type specEnv struct {
	cachedPlayerDB, cachedEnemyDB, cachedMinionDB *moddb.ModDB
	env                                           *Environment
	accelerate                                    accelerate
}

// wipeModStore is wipeTable over a database's mods, conditions and multipliers, keeping the database.
func wipeModStore(db *moddb.ModDB) {
	db.Mods = map[string][]mod.Mod{}
	db.Conditions = map[string]bool{}
	db.Multipliers = map[string]float64{}
}

//pob2:src/Modules/CalcSetup.lua:466-541
func wipeEnv(env *Environment, accelerate accelerate) {
	// Always wipe the below as we will be pushing in the modifiers,
	// multipliers and conditions for player and enemy DBs via `parent`
	// extensions of those DBs later which allow us to do a table-pointer
	// link and save time on having to do a copyTable() function.
	wipeModStore(env.ModDB)
	wipeModStore(env.EnemyDB)
	if env.Minion != nil {
		wipeModStore(env.Minion.ModDB)
	}

	if accelerate.everything {
		return
	}

	// Passive tree node allocations
	// Also in a further pass tracks Legion influenced mods
	if !accelerate.nodeAlloc {
		env.AllocNodes = map[int]*tree2.SpecNode{}
		// Usually states: `Allocates <NAME>` (e.g., amulet anointment)
		env.GrantedPassives = map[int]bool{}
		env.grantedSkillsNodes = nil
	}

	if !accelerate.requirementsItems {
		// Item-related tables
		wipeModStore(env.ItemModDB)
		env.radiusJewelList = nil
		env.extraRadiusNodeList = nil
		env.Player.ItemList = map[string]*Item{}
		env.grantedSkillsItems = nil
		env.Flasks = map[*Item]bool{}
		env.Charms = map[*Item]bool{}

		// Special / Unique Items that have their own ModDB()
		env.TalismanModList = nil
		env.TheIronMass = nil
		env.WeaponModList1 = nil

		// Requirements from Items (Str, Dex, Int)
		env.requirementsTableItems = nil
	}

	// Requirements from Gems (Str, Dex, Int)
	if !accelerate.requirementsGems {
		env.requirementsTableGems = nil
	}

	if !accelerate.skills {
		// Player Active Skills generation
		env.Player.ActiveSkillList = nil
		env.sourceGemPropertyInfo = nil

		// Enhances Active Skills with skill ModFlags, KeywordFlags
		// and modifiers that affect skill scaling (e.g., global buffs/effects)
		env.AuxSkillList = nil
	}
}

//pob2:src/Modules/Common.lua:542-561
func specCopy(env *Environment) (*moddb.ModDB, *moddb.ModDB, *moddb.ModDB) {
	copyDB := func(src *moddb.ModDB) *moddb.ModDB {
		db := moddb.NewModDB()
		db.Rules = kernelRules
		db.AddDB(src)
		db.Conditions = maps.Clone(src.Conditions)
		db.Multipliers = maps.Clone(src.Multipliers)
		return db
	}
	modDB := copyDB(env.ModDB)
	enemyDB := moddb.NewModDB()
	enemyDB.Rules = kernelRules
	if env.EnemyDB != nil {
		enemyDB = copyDB(env.EnemyDB)
	}
	var minionDB *moddb.ModDB
	if env.Minion != nil {
		minionDB = copyDB(env.Minion.ModDB)
	}
	return modDB, enemyDB, minionDB
}

// initEnv is calcs.initEnv for a build's player and enemy databases, tree, config and skills.
func initEnv(env *Environment) {
	initEnvSpec(env, nil)
}

// initEnvSpec is calcs.initEnv with a specEnv: with spec.env set, env is that env, wiped and rebuilt but for the
// blocks spec.accelerate keeps. It returns the cached base databases, which only a call with a specEnv takes.
//
//pob2:src/Modules/CalcSetup.lua:842-2604
func initEnvSpec(env *Environment, spec *specEnv) (*moddb.ModDB, *moddb.ModDB, *moddb.ModDB) {
	// accelerator variables
	var cachedPlayerDB, cachedEnemyDB, cachedMinionDB *moddb.ModDB
	var accelerate accelerate
	if spec != nil {
		cachedPlayerDB, cachedEnemyDB, cachedMinionDB = spec.cachedPlayerDB, spec.cachedEnemyDB, spec.cachedMinionDB
		accelerate = spec.accelerate
	}

	var modDB, enemyDB *moddb.ModDB
	if spec == nil || spec.env == nil {
		modDB = moddb.NewModDB()
		modDB.Rules = kernelRules
		env.ModDB = modDB
		enemyDB = moddb.NewModDB()
		enemyDB.Rules = kernelRules
		env.EnemyDB = enemyDB
		env.ItemModDB = moddb.NewModDB()
		env.ItemModDB.Rules = kernelRules

		env.EnemyLevel = configEnemyLevel(env)

		//pob2:block:src/Modules/CalcSetup.lua:877-891
		env.Player = &Actor{ModDB: modDB, Level: env.Build.Build.Level, ItemList: map[string]*Item{}}
		modDB.Actor = env.Player
		env.Enemy = &Actor{ModDB: enemyDB, Level: env.EnemyLevel, ItemList: map[string]*Item{}}
		enemyDB.Actor = env.Enemy
		env.Player.Enemy = env.Enemy
		env.Enemy.Enemy = env.Player
		env.Enemy.Player = env.Player
		env.Player.Player = env.Player

		env.GrantedPassives = map[int]bool{}
		env.Flasks = map[*Item]bool{}
		env.Charms = map[*Item]bool{}
	} else {
		wipeEnv(env, accelerate)
		modDB = env.ModDB
		enemyDB = env.EnemyDB
	}

	//pob2:block:src/Modules/CalcSetup.lua:928-951
	env.ModeBuffs, env.ModeCombat, env.ModeEffective = true, true, true

	classStats := env.Spec.Tree.Classes[env.ClassID]

	if cachedPlayerDB == nil {
		initBaseModDBs(env, classStats)
		// Go must differ: only a caller with a specEnv takes the copy
		if spec != nil {
			cachedPlayerDB, cachedEnemyDB, cachedMinionDB = specCopy(env)
		}
	} else {
		//pob2:block:src/Modules/CalcSetup.lua:1040-1046
		modDB.Parent = cachedPlayerDB
		enemyDB.Parent = cachedEnemyDB
		if cachedMinionDB != nil && env.Minion != nil {
			env.Minion.ModDB.Parent = cachedMinionDB
		}
	}

	initEnvRest(env, accelerate)
	return cachedPlayerDB, cachedEnemyDB, cachedMinionDB
}

// initBaseModDBs is the player and enemy base, config and party modifiers calcs.initEnv adds, and caches, when no
// cached database is passed.
func initBaseModDBs(env *Environment, classStats *tree2.Class) {
	modDB, enemyDB := env.ModDB, env.EnemyDB
	//pob2:block:src/Modules/CalcSetup.lua:956-1024
	modDB.AddMod(newMod("Str", mod.TypeBase, float64(classStats.BaseStr), "Base"))
	modDB.AddMod(newMod("Dex", mod.TypeBase, float64(classStats.BaseDex), "Base"))
	modDB.AddMod(newMod("Int", mod.TypeBase, float64(classStats.BaseInt), "Base"))
	modDB.Multipliers["Level"] = math.Max(1, math.Min(100, float64(env.Build.Build.Level)))
	initModDB(env, modDB)
	modDB.AddMod(newMod("Life", mod.TypeBase, characterConstant("life_per_level"), "Base", mod.Multiplier("Level").Base(16)))
	modDB.AddMod(newMod("Mana", mod.TypeBase, characterConstant("mana_per_level"), "Base", mod.Multiplier("Level").Base(30)))
	modDB.AddMod(newMod("ManaRegen", mod.TypeBase, miscConstant("ManaRegenBase"), "Base", mod.PerStat(1, "Mana"), mod.Condition("NoInherentManaRegen").Neg(true)))
	modDB.AddMod(newMod("WardRegenPercent", mod.TypeBase, gameConstant("BaseWardRegenerationPercentPerMinute")/60, "Base"))
	modDB.AddMod(newMod("Spirit", mod.TypeBase, 0, "Base"))
	modDB.AddMod(newMod("Devotion", mod.TypeBase, 0, "Base"))
	modDB.AddMod(newMod("Tribute", mod.TypeBase, 0, "Base"))
	modDB.AddMod(newMod("Evasion", mod.TypeBase, characterConstant("base_evasion_rating"), "Base"))
	modDB.AddMod(newMod("Accuracy", mod.TypeBase, characterConstant("accuracy_rating_per_level"), "Base", mod.Multiplier("Level").Base(-characterConstant("accuracy_rating_per_level"))))
	modDB.AddMod(newMod("CritMultiplier", mod.TypeBase, characterConstant("base_critical_hit_damage_bonus"), "Base"))
	modDB.AddMod(newMod("DotMultiplier", mod.TypeBase, 50, "Base", mod.Condition("CriticalStrike")))
	resistancePenalty := -60.0
	if v, ok := env.ConfigInput["resistancePenalty"]; ok && v.IsNumber {
		resistancePenalty = v.Number
	}
	modDB.AddMod(newMod("FireResist", mod.TypeBase, resistancePenalty, "Base"))
	modDB.AddMod(newMod("ColdResist", mod.TypeBase, resistancePenalty, "Base"))
	modDB.AddMod(newMod("LightningResist", mod.TypeBase, resistancePenalty, "Base"))
	modDB.AddMod(newMod("ChaosResist", mod.TypeBase, 0, "Base"))
	modDB.AddMod(newMod("TotemFireResist", mod.TypeBase, 40, "Base"))
	modDB.AddMod(newMod("TotemColdResist", mod.TypeBase, 40, "Base"))
	modDB.AddMod(newMod("TotemLightningResist", mod.TypeBase, 40, "Base"))
	modDB.AddMod(newMod("TotemChaosResist", mod.TypeBase, 20, "Base"))
	modDB.AddMod(newMod("MaximumFortification", mod.TypeBase, characterConstant("base_max_fortification"), "Base"))
	modDB.AddMod(newMod("MaximumValour", mod.TypeBase, 50, "Base"))
	modDB.AddMod(newMod("SoulEaterMax", mod.TypeBase, 45, "Base"))
	modDB.AddMod(newMod("Multiplier:IntensityLimit", mod.TypeBase, 3, "Base"))
	modDB.AddMod(newMod("Multiplier:DemonFlameMaximum", mod.TypeBase, 10, "Base"))
	rampageLimit := characterConstant("max_rampage_stacks") / 20
	modDB.AddMod(newMod("Damage", mod.TypeIncrease, characterConstant("damage_+%_per_10_rampage_stacks"), "Base", mod.Multiplier("Rampage").Limit(rampageLimit).Div(20)))
	modDB.AddMod(newMod("MovementSpeed", mod.TypeIncrease, characterConstant("movement_velocity_+%_per_10_rampage_stacks"), "Base", mod.Multiplier("Rampage").Limit(rampageLimit).Div(20)))
	modDB.AddMod(newMod("Speed", mod.TypeIncrease, 5, "Base", mod.Multiplier("SoulEater")).Flag(modFlagAttack))
	modDB.AddMod(newMod("Speed", mod.TypeIncrease, 5, "Base", mod.Multiplier("SoulEater")).Flag(modFlagCast))
	modDB.AddMod(newMod("ActiveTrapLimit", mod.TypeBase, characterConstant("base_number_of_traps_allowed"), "Base"))
	modDB.AddMod(newMod("ActiveMineLimit", mod.TypeBase, characterConstant("base_number_of_remote_mines_allowed"), "Base"))
	modDB.AddMod(newMod("MineThrowCount", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("TrapThrowCount", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("ActiveBrandLimit", mod.TypeBase, 3, "Base"))
	modDB.AddMod(newMod("EnemyCurseLimit", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("EnemyMarkLimit", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("SocketedCursesHexLimitValue", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("ProjectileCount", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("PhysicalHeavyStunBuildup", mod.TypeMore, characterConstant("physical_hit_damage_stun_multiplier_+%_final_from_ot"), "Physical Damage"))
	modDB.AddMod(newMod("EnemyHeavyStunBuildup", mod.TypeMore, characterConstant("melee_hit_damage_stun_multiplier_+%_final_from_ot"), "Melee Damage").Flag(modFlagMelee))
	modDB.AddMod(newMod("AilmentMagnitude", mod.TypeMore, 100, "Base", mod.ActorCondition("enemy", "Moving", "BleedAggravated"), mod.Condition("NoExtraBleedDamageToMovingEnemy").Neg(true)).KeywordFlag(keywordFlagBleed))
	modDB.AddMod(newFlagMod("Condition:BloodStance", "Base", mod.Condition("SandStance").Neg(true)))
	modDB.AddMod(newFlagMod("Condition:PrideMinEffect", "Base", mod.Condition("PrideMaxEffect").Neg(true)))
	modDB.AddMod(newMod("MovementSpeed", mod.TypeIncrease, 50, "Base", mod.Condition("Sprinting")))
	modDB.AddMod(newFlagMod("Condition:CanSprint", "Base", mod.Condition("CannotSprint").Neg(true)))
	modDB.AddMod(newMod("PerBrutalTripleDamageChance", mod.TypeBase, characterConstant("chance_to_deal_triple_damage_%_per_brutal_charge"), "Base"))
	modDB.AddMod(newMod("PerAfflictionAilmentDamage", mod.TypeBase, 8, "Base"))
	modDB.AddMod(newMod("PerAfflictionNonDamageEffect", mod.TypeBase, 8, "Base"))
	modDB.AddMod(newMod("PerAbsorptionElementalEnergyShieldRecoup", mod.TypeBase, characterConstant("elemental_damage_taken_goes_to_energy_shield_over_4_seconds_%_per_absorption_charge"), "Base"))
	modDB.AddMod(newMod("CharmLimit", mod.TypeBase, 0, "Base"))
	modDB.AddMod(newMod("ManaDegenPercent", mod.TypeBase, 1, "Base", mod.Multiplier("EffectiveManaBurnStacks")))
	modDB.AddMod(newMod("LifeDegenPercent", mod.TypeBase, 1, "Base", mod.Multiplier("WeepingWoundsStacks")))
	modDB.AddMod(newMod("WeaponSwapSpeed", mod.TypeBase, characterConstant("base_weapon_swap_duration_ms"), "Base"))
	modDB.AddMod(newMod("Speed", mod.TypeIncrease, 2, "Base", mod.Multiplier("Tailwind").Limit(10)).Flag(modFlagAttack))
	modDB.AddMod(newMod("Speed", mod.TypeIncrease, 2, "Base", mod.Multiplier("Tailwind").Limit(10)).Flag(modFlagCast))
	modDB.AddMod(newMod("MovementSpeed", mod.TypeIncrease, 1, "Base", mod.Multiplier("Tailwind").Limit(10)))
	modDB.AddMod(newMod("DeflectEffect", mod.TypeIncrease, 1, "Base", mod.Multiplier("Tailwind").Limit(10)))
	modDB.AddMod(newMod("Evasion", mod.TypeIncrease, 10, "Base", mod.Multiplier("Tailwind").Limit(10)))
	modDB.AddMod(newMod("SkillSlots", mod.TypeBase, 9, "Base"))
	modDB.AddMod(newMod("MaxLineageCount", mod.TypeBase, 1, "Base"))
	modDB.AddMod(newMod("Multiplier:AncestralBoostEffect", mod.TypeBase, 1, "Base", mod.Condition("AncestrallyBoosted")))
	modDB.AddMod(newMod("AncestralBoostMoreDamage", mod.TypeBase, 30, "Base", mod.Multiplier("AncestralBoostEffect"), mod.SkillType("Slam"), mod.Condition("AncestrallyBoosted")))
	modDB.AddMod(newMod("AncestralBoostAreaOfEffect", mod.TypeIncrease, 25, "Base", mod.Multiplier("AncestralBoostEffect"), mod.SkillType("Slam"), mod.Condition("AncestrallyBoosted")))
	modDB.AddMod(newMod("AdditionalStrikeTarget", mod.TypeBase, 2, "Base", mod.Multiplier("AncestralBoostEffect"), mod.SkillType("Slam").Neg(true), mod.Condition("AncestrallyBoosted")))
	modDB.AddMod(newFlagMod("Condition:Empowered", "Base", mod.Condition("AncestrallyBoosted")))

	//pob2:block:src/Modules/CalcSetup.lua:1027-1037
	initModDB(env, enemyDB)
	enemyDB.AddMod(newMod("Accuracy", mod.TypeBase, poe2.MonsterAccuracyTable[env.EnemyLevel-1], "Base"))
	enemyDB.AddMod(newFlagMod("Condition:AgainstDamageOverTime", "Base", mod.ActorCondition("player", "Combat")).Flag(modFlagDot))
	configModList, configEnemyModList := newModList(), newModList()
	buildConfigModList(env, configModList, configEnemyModList)
	modDB.AddList(configModList)
	enemyDB.AddList(configEnemyModList)
	// Add mods from the party tab
	//pob2:block:src/Modules/CalcSetup.lua:1036-1037
	enemyDB.AddList(env.partyMembersTab().enemyModList)
}

// initEnvRest is calcs.initEnv from the weapon set on.
func initEnvRest(env *Environment, accelerate accelerate) {
	modDB := env.ModDB
	counts := countAllocated(env.Spec)

	//pob2:block:src/Modules/CalcSetup.lua:1060-1076
	if env.override.MainSocketGroup != 0 {
		// The other weapon set's context runs on the socket group list the main pass left.
		env.MainSocketGroup = min(max(len(env.SocketGroupList), 1), env.override.MainSocketGroup)
	} else {
		env.MainSocketGroup = min(max(len(activeSkillSet(env)), 1), max(env.Build.Build.MainSocketGroup, 1))
		env.Build.Build.MainSocketGroup = env.MainSocketGroup
	}
	if env.override.WeaponSet != 0 {
		env.WeaponSet = env.override.WeaponSet
	} else {
		env.WeaponSet = socketGroupWeaponSet(env)
	}
	env.WeaponSetEnvs = nil
	env.WeaponSetValidationSkillList = nil

	if !accelerate.nodeAlloc {
		env.AllocNodes = overrideAllocNodes(env, &counts)
	}

	nodesModsList := buildModListForNodeList(env, env.AllocNodes, true)
	canUseBonded := nodesModsList.Flag(nil, "CanUseBonded")
	//pob2:block:src/Modules/CalcSetup.lua:1142
	env.useAltGemQualityStats = nodesModsList.Flag(nil, "GemlingQuality")

	//pob2:block:src/Modules/CalcSetup.lua:1145-1162
	if counts.notable > 0 {
		modDB.AddMod(mod.NewFloat("Multiplier:AllocatedNotable", mod.TypeBase, float64(counts.notable)))
	}
	if counts.smithBodyArmour > 0 {
		modDB.AddMod(mod.NewFloat("Multiplier:AllocatedConnectedNotable", mod.TypeBase, float64(counts.smithBodyArmour)))
	}
	if counts.mastery > 0 {
		modDB.AddMod(mod.NewFloat("Multiplier:AllocatedMastery", mod.TypeBase, float64(counts.mastery)))
	}
	if counts.masteryType > 0 {
		modDB.AddMod(mod.NewFloat("Multiplier:AllocatedMasteryType", mod.TypeBase, float64(counts.masteryType)))
	}
	if counts.masteryTypes["Life Mastery"] > 0 {
		modDB.AddMod(mod.NewFloat("Multiplier:AllocatedLifeMastery", mod.TypeBase, float64(counts.masteryTypes["Life Mastery"])))
	}
	modDB.AddMod(newFlagMod("Condition:WeaponSet"+strconv.Itoa(env.WeaponSet), "Weapon Set"))

	if !accelerate.requirementsItems {
		mergeItems(env, nodesModsList, canUseBonded)
	}

	//pob2:block:src/Modules/CalcSetup.lua:1756-1757
	mergeDB(env.ModDB, env.ItemModDB)

	//pob2:block:src/Modules/CalcSetup.lua:1759-1768
	if !accelerate.nodeAlloc {
		for _, passive := range env.ModDB.List(nil, "GrantedPassive") {
			var gp tree2.GrantedPassive
			switch v := passive.(type) {
			case string:
				gp = tree2.GrantedPassive{Name: v}
			case LuaTable:
				typ, _ := v["type"].(string)
				count, _ := v["count"].(float64)
				gp = tree2.GrantedPassive{Table: true, Type: typ, Count: int(count)}
			default:
				continue
			}
			for _, node := range env.Spec.ResolveGrantedPassiveNodes(gp) {
				if node != nil && !env.override.RemoveNodeIDs[node.ID] {
					env.AllocNodes[node.ID] = node
					env.GrantedPassives[node.ID] = true
				}
			}
		}
	}

	//pob2:block:src/Modules/CalcSetup.lua:1788-1789
	env.ModDB.AddList(buildModListForNodeList(env, env.AllocNodes, false))

	addSlotBonusEffects(env, canUseBonded)

	if setupSkills(env, accelerate) {
		return
	}
	setupGemCounts(env)
}

// newWeaponSetEnv is the env calcs.initEnv(build, mode, override) starts from for another weapon set's
// context: the same build, items, tree, data and socket group list.
func newWeaponSetEnv(env *Environment, mode string, override calcOverride) *Environment {
	return &Environment{
		Build:           env.Build,
		Mode:            mode,
		ConfigInput:     env.ConfigInput,
		modSource:       env.modSource,
		items:           env.items,
		nodeModLists:    map[int]*moddb.ModList{},
		Spec:            env.Spec,
		data:            env.data,
		ClassID:         env.ClassID,
		SocketGroupList: env.SocketGroupList,
		fromItemEffects: env.fromItemEffects,
		override:        override,
		partyTab:        env.partyTab,
	}
}

//pob2:src/Modules/Common.lua:530-540
func mergeDB(srcDB, modDB *moddb.ModDB) {
	srcDB.AddDB(modDB)
	for k, v := range modDB.Conditions {
		srcDB.Conditions[k] = v
	}
	for k, v := range modDB.Multipliers {
		srcDB.Multipliers[k] = v
	}
}

// configEnemyLevel is configTab.enemyLevel after BuildModList: UpdateLevel once more after the
// enemyIsBoss option (default "Pinnacle") sets the enemyLevel placeholder.
//
//pob2:block:src/Classes/ConfigTab.lua:1157-1167
//pob2:block:src/Modules/ConfigOptions.lua:1994-1996
//pob2:block:src/Modules/ConfigOptions.lua:2077-2079
func configEnemyLevel(env *Environment) int {
	characterLevel := float64(env.Build.Build.Level)
	placeholder, hasPlaceholder := configPlaceholderEnemyLevel(env.Build)
	boss := "Pinnacle"
	if v, ok := env.ConfigInput["enemyIsBoss"]; ok && v.IsString {
		boss = enemyIsBossInput(v.String)
	}
	switch boss {
	case "None", "Boss":
		placeholder, hasPlaceholder = characterLevel, true
	case "Pinnacle", "Uber":
		placeholder, hasPlaceholder = 82, true
	}
	level := characterLevel
	if v, ok := env.ConfigInput["enemyLevel"]; ok && v.IsNumber && v.Number > 0 {
		level = v.Number
	} else if hasPlaceholder && placeholder > 0 {
		level = placeholder
	}
	return int(math.Min(miscConstant("MaxEnemyLevel"), level))
}

var capitaliseWordsRE = regexp.MustCompile(`[a-z][a-zA-Z0-9]*`)

//pob2:block:src/Classes/ConfigTab.lua:892-894
func enemyIsBossInput(s string) string {
	s = capitaliseWordsRE.ReplaceAllStringFunc(strings.ToLower(s), func(w string) string { return strings.ToUpper(w[:1]) + w[1:] })
	s = strings.ReplaceAll(s, "Uber Atziri", "Boss")
	s = strings.ReplaceAll(s, "Shaper", "Pinnacle")
	return strings.ReplaceAll(s, "Sirus", "Pinnacle")
}

//pob2:block:src/Classes/ConfigTab.lua:907-913
func configPlaceholderEnemyLevel(build *pob2.PathOfBuilding2) (float64, bool) {
	placeholders := build.Config.Placeholders
	for _, set := range build.Config.ConfigSets {
		if set.ID == activeConfigSetID(build) {
			placeholders = set.Placeholders
		}
	}
	for _, p := range placeholders {
		if p.Name == "enemyLevel" && p.Number != nil {
			return *p.Number, true
		}
	}
	return 0, false
}

//pob2:src/Classes/SkillsTab.lua:1398-1405
func socketGroupWeaponSet(env *Environment) int {
	skills := activeSkillSet(env)
	if env.MainSocketGroup > len(skills) {
		return useSecondWeaponSet(env)
	}
	group := skills[env.MainSocketGroup-1]
	if group.Set1 && group.Set2 {
		return useSecondWeaponSet(env)
	}
	if group.Set2 {
		return 2
	}
	return 1
}
