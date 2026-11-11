package calculator2

import (
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2skills"
	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob2"
)

// Environment is CalcSetup's env: the build's actors, databases and allocated tree.
//
//pob2:block:src/Modules/CalcSetup.lua:857-914
type Environment struct {
	Build *pob2.PathOfBuilding2
	Mode  string
	// socketGroupsSynced is set once the Skills tab's edits have written SocketGroupList into Build.
	socketGroupsSynced bool

	Spec    *tree2.PassiveSpec
	ClassID int

	ModDB     *moddb.ModDB
	EnemyDB   *moddb.ModDB
	ItemModDB *moddb.ModDB

	EnemyLevel int
	Player     *Actor
	Enemy      *Actor
	// Minion is env.minion: the main skill's minion, nil when it has none.
	Minion *Minion

	// ConfigInput is build.configTab.input for the active config set.
	ConfigInput map[string]ConfigValue

	ModeBuffs     bool
	ModeCombat    bool
	ModeEffective bool

	MainSocketGroup int
	WeaponSet       int

	AllocNodes      map[int]*tree2.SpecNode
	GrantedPassives map[int]bool
	KeystonesAdded  map[string]bool

	// nodeModLists holds each tree node's ProcessStats modList, keyed by node id.
	nodeModLists map[int]*moddb.ModList

	// radiusJewelList, extraRadiusNodeList and radiusJewelData (PoB's GlobalCache radius cache) are
	// the radius jewel state CalcSetup keeps.
	radiusJewelList     []*radiusJewel
	extraRadiusNodeList map[int]*tree2.SpecNode
	radiusJewelData     map[int]*radiusJewelCacheEntry
	modSource           ModSource
	items               *ItemsTab

	// Flasks and Charms are env.flasks / env.charms: the active ones.
	Flasks        map[*Item]bool
	Charms        map[*Item]bool
	ItemSlotIndex map[*Item]int

	TalismanModList *moddb.ModList
	TheIronMass     *moddb.ModList
	WeaponModList1  *moddb.ModList

	// data is env.data's skill half: data.skills and the gem lookups.
	data *skillData
	// buffs is env.buffs, env.debuffs, env.minionBuffs and env.curseSlots as perform leaves them.
	buffs *buffState
	// partyExports is buffExports, nil while the party export is off.
	partyExports *partyBuffExports
	// SocketGroupList is build.skillsTab.socketGroupList as initEnv leaves it.
	SocketGroupList []*SocketGroup

	grantedSkillsNodes []*grantedSkill
	grantedSkillsItems []*grantedSkill
	explodeSources     []any
	// fromItemEffects stands in for the fromItem flag CalcSetup writes onto a data.skills entry an
	// ExtraSupport names.
	fromItemEffects map[*poe2skills.GrantedEffect]bool

	requirementsTableItems []requirementSource
	requirementsTableGems  []requirementSource
	requirementsTable      []requirementSource

	VirtuousMoteSkillCount map[string]float64
	// AuxSkillList is env.auxSkillList: the active skills with buffs.
	AuxSkillList          []*ActiveSkill
	useAltGemQualityStats bool
	// sourceGemPropertyInfo is env.sourceGemPropertyInfo.
	sourceGemPropertyInfo map[*GemInstance][]supportedGemProperty
	// weaponSetRestart is set where calcs.initEnv restarted for the main group's other weapon set.
	weaponSetRestart bool

	// override is calcs.initEnv's override.
	override calcOverride
	// WeaponSetEnvs is env.weaponSetEnvs: the other weapon set's context, keyed by its set.
	WeaponSetEnvs map[int]*Environment
	// WeaponSetValidationSkillList is env.weaponSetValidationSkillList: the active skill list before
	// the other set's groups are swapped in.
	WeaponSetValidationSkillList []*ActiveSkill
	// partyTab is build.partyTab, parsed from the buffers it applies.
	partyTab *partyTab
	// configPlaceholder is the config set's placeholder table after BuildModList's options ran.
	configPlaceholder map[string]float64
}

// calcOverride is CalcOverride's weaponSet, mainSocketGroup and skipWeaponSetContexts; 0 is nil. With RepSlotName
// set, RepItem (nil empties it) stands in that slot's item; ToggleFlask and ToggleCharm flip one item's activation.
type calcOverride struct {
	WeaponSet             int
	MainSocketGroup       int
	SkipWeaponSetContexts bool
	RepSlotName           string
	RepItem               *Item
	ToggleFlask           *Item
	ToggleCharm           *Item
	// AddNodes and RemoveNodes are addNodes and removeNodes keyed by node; RemoveNodeIDs is removeNodes keyed by node
	// id, as a granted passive is.
	AddNodes      map[*tree2.SpecNode]bool
	RemoveNodes   map[*tree2.SpecNode]bool
	RemoveNodeIDs map[int]bool
}

// SocketGroup is one build.skillsTab.socketGroupList entry.
//
//pob2:block:src/Classes/SkillsTab.lua:293-306
type SocketGroup struct {
	SourceItem *Item
	SourceNode *tree2.SpecNode
	GemList    []*GemInstance
	// DisplayGemList is the group's gems plus the supports cross-linked into it.
	DisplayGemList []*GemInstance

	Label  string
	Source string
	Slot   string

	GroupCount           float64
	MainActiveSkill      int
	MainActiveSkillCalcs int
	UsingSkillSet        int

	Enabled          bool
	IncludeInFullDPS bool
	Set1             bool
	Set2             bool
	NoSupports       bool
	HasGroupCount    bool

	// savedIndex is the group's 1-based place among the active skill set's saved groups; 0 for a group the
	// calculation generated.
	savedIndex int
}

// GemInstance is one gemList entry of a socket group.
//
//pob2:block:src/Classes/SkillsTab.lua:308-386
type GemInstance struct {
	GemData       *poe2skills.Gem
	GrantedEffect *poe2skills.GrantedEffect
	// DisplayEffect and SupportEffect are the effects the main pass built from this gem.
	DisplayEffect *GemEffect
	SupportEffect *GemEffect

	// StatSet and StatSetCalcs are the selected stat set index per granted effect id.
	StatSet      map[string]int
	StatSetCalcs map[string]int
	// ExplodeSource is gemInstance.explodeSource: the *Item or *tree2.SpecNode an EnemyExplode gem is for.
	ExplodeSource any

	NameSpec  string
	GemID     string
	SkillID   string
	QualityID string
	// SkillMinion and SkillMinionCalcs are the minion the gem summons, by data.minions key; "" is nil.
	SkillMinion      string
	SkillMinionCalcs string
	// SkillMinionItemSet and SkillMinionItemSetCalcs are the item set id the gem's minion uses; 0 is nil.
	SkillMinionItemSet      int
	SkillMinionItemSetCalcs int
	// SkillMinionSkill and SkillMinionSkillCalcs are the minion's main skill, 1-based; nil is unset.
	SkillMinionSkill      *int
	SkillMinionSkillCalcs *int
	// SkillMinionSkillStatSetIndexLookup (and its Calcs half) is each minion skill's stat set, by granted effect id and
	// minion skill index.
	SkillMinionSkillStatSetIndexLookup      map[string]map[int]int
	SkillMinionSkillStatSetIndexLookupCalcs map[string]map[int]int
	// SkillStageCount and SkillMineCount (and their Calcs halves) are the saved stage and mine counts; nil is unset.
	SkillStageCount      *int
	SkillStageCountCalcs *int
	SkillMineCount       *int
	SkillMineCountCalcs  *int

	Level         int
	Quality       float64
	Count         float64
	CorruptLevel  int
	SourceLevel   int
	TriggerChance float64

	ReqLevel float64
	ReqStr   float64
	ReqDex   float64
	ReqInt   float64

	Enabled        bool
	EnableGlobal1  bool
	EnableGlobal2  bool
	Corrupted      bool
	FromItem       bool
	FromTree       bool
	NoSupports     bool
	NoReservation  bool
	Triggered      bool
	HasSourceLevel bool
}

func (g *GemInstance) enableGlobal(index int) bool {
	switch index {
	case 1:
		return g.EnableGlobal1
	case 2:
		return g.EnableGlobal2
	}
	return false
}

// GemEffect is an active or support effect: activeEffect, a supportList entry, or an effectList entry.
type GemEffect struct {
	GrantedEffect *poe2skills.GrantedEffect
	GemData       *poe2skills.Gem
	SrcInstance   *GemInstance
	// StatSet is activeEffect.statSet (or statSetCalcs): the chosen index, the set and its skill flags.
	StatSet *StatSetChoice
	// Values holds the gem properties applyGemMods adds by key (effect[value.key]).
	Values map[string]float64
	// IsSupporting is the set of gem instances this support supports; nil when it tracks none.
	IsSupporting map[*GemInstance]bool

	QualityID string

	Level            int
	Quality          float64
	CorruptLevel     int
	ActiveSkillLevel int

	Corrupted  bool
	Superseded bool
	Enabled    bool

	// ActorLevel is activeEffect.actorLevel, set for a minion's skill.
	ActorLevel    float64
	HasActorLevel bool
	// GrantedEffectLevel is activeEffect.grantedEffectLevel: levels[level] with the stat set's level row over it.
	GrantedEffectLevel *grantedEffectLevel
	// GemPropertyInfo is effect.gemPropertyInfo: the GemProperty mods applyGemMods applied.
	GemPropertyInfo []gemPropertyInfo
}

// StatSetChoice is activeEffect.statSet: { index, statSet, skillFlags }.
type StatSetChoice struct {
	StatSet    *poe2skills.StatSet
	SkillFlags map[string]bool
	Index      int
}

// ActiveSkill is the table calcs.createActiveSkill returns.
//
//pob2:block:src/Modules/CalcActiveSkill.lua:146-154
type ActiveSkill struct {
	ActiveEffect *GemEffect
	SupportList  []*GemEffect
	EffectList   []*GemEffect
	Actor        *Actor
	SummonSkill  *ActiveSkill
	SocketGroup  *SocketGroup
	SkillData    map[string]any

	SkillTypes       map[string]bool
	MinionSkillTypes map[string]bool
	// SkillFlags is activeEffect.statSet.skillFlags until buildActiveSkillModList sets its own.
	SkillFlags map[string]bool

	SlotName string

	// AilmentCfg (activeSkill[ailment:lower().."Cfg"] and its "OH" off-hand twin, by that key), DecayCfg,
	// DotCfg and DotTypeCfg (activeSkill["dot"..damageType.."Cfg"], by damage type) are the cfgs
	// calcs.offence keeps for damage over time.
	AilmentCfg map[string]*moddb.ListCfg
	DecayCfg   *moddb.ListCfg
	DotCfg     *moddb.ListCfg
	DotTypeCfg map[string]*moddb.ListCfg

	// The fields calcs.buildActiveSkillModList sets.
	SkillModList      *moddb.ModList
	BaseSkillModList  *moddb.ModList
	SkillCfg          *moddb.ListCfg
	Weapon1Cfg        *moddb.ListCfg
	Weapon2Cfg        *moddb.ListCfg
	TriggeredBy       *GemEffect
	ExtraSkillModList []mod.Mod
	BuffList          []*skillBuff
	MinionList        []string
	// Minion is activeSkill.minion: the minion the skill creates, nil when it creates none.
	Minion           *Minion
	SkillPartName    string
	DisableReason    string
	Weapon1Flags     mod.MFlag
	Weapon2Flags     mod.MFlag
	SkillPart        int
	SkillTotemID     int
	ActiveMineCount  float64
	ActiveStageCount float64
}

// grantedSkill is one env.grantedSkills entry: a skill a tree node or an item grants.
type grantedSkill struct {
	SourceItem    *Item
	SourceNode    *tree2.SpecNode
	SkillID       string
	NameSpec      string
	Source        string
	SlotName      string
	Level         int
	TriggerChance float64
	NoSupports    bool
	NoReservation bool
	Triggered     bool
}

// Actor is env.player / env.enemy.
type Actor struct {
	ModDB  *moddb.ModDB
	Level  int
	Output map[string]float64
	// SpecificTypeAvoidance is output.specificTypeAvoidance, a boolean the float output table does not hold.
	SpecificTypeAvoidance bool
	// OutputFlags is output's boolean entries, which Output does not hold.
	OutputFlags map[string]bool
	// OutputStrings is output's string entries; a key in it is absent from Output.
	OutputStrings map[string]string
	// OutputMainHand and OutputOffHand are output.MainHand/OffHand, nil unless the main skill attacks.
	OutputMainHand map[string]float64
	OutputOffHand  map[string]float64
	// SkillDPS is output.SkillDPS, the Full DPS rows; nil where it is.
	SkillDPS []SkillDPS
	// Breakdown is actor.breakdown, nil unless env.buildBreakdown.
	Breakdown *Breakdown
	// SpectreLifeList and CompanionLifeList are actor.spectreLifeList and actor.companionLifeList.
	SpectreLifeList   []minionLife
	CompanionLifeList []minionLife

	Player *Actor
	Enemy  *Actor
	Parent *Actor

	// ActiveSkillList is actor.activeSkillList; MainSkill is actor.mainSkill.
	ActiveSkillList []*ActiveSkill
	MainSkill       *ActiveSkill

	// ItemList is actor.itemList: the equipped item per slot name.
	ItemList map[string]*Item
	// WeaponData1Type and WeaponData2Type are actor.weaponData1/2.type; "" is nil.
	WeaponData1Type string
	WeaponData2Type string
	// WeaponData1AsThoughUsing and WeaponData2AsThoughUsing are actor.weaponData1/2.asThoughUsing.
	WeaponData1AsThoughUsing map[string]bool
	WeaponData2AsThoughUsing map[string]bool
	// WeaponData1FacebreakerItemDamage is actor.weaponData1.FacebreakerItemDamage, and
	// WeaponData1Facebreaker its Facebreaker<type>Min/Max entries.
	WeaponData1FacebreakerItemDamage bool
	WeaponData1Facebreaker           map[string]float64
	// WeaponData1 and WeaponData2 are actor.weaponData1/2's numbers, as weaponData gives them.
	WeaponData1 map[string]float64
	WeaponData2 map[string]float64
	// weaponDataSet marks a minion, whose weaponData1/2 createMinion sets directly.
	weaponDataSet bool
	// WeaponRange1 and WeaponRange2 are actor.weaponRange1/2.
	WeaponRange1 float64
	WeaponRange2 float64

	appliedEnemyModifiers map[mod.Mod]bool
	// partyMembers is actor.partyMembers, the party tab's actor.
	partyMembers *partyActor
}

// buildBreakdown is env.buildBreakdown.
//
//pob2:block:src/Modules/CalcSetup.lua:865
func (env *Environment) buildBreakdown() bool {
	return env.Mode == "MAIN" || env.Mode == "CALCS"
}

func (a *Actor) setOutputFlag(key string, v bool) {
	if a.OutputFlags == nil {
		a.OutputFlags = map[string]bool{}
	}
	a.OutputFlags[key] = v
}

func (a *Actor) setOutputString(key string, v string) {
	if a.OutputStrings == nil {
		a.OutputStrings = map[string]string{}
	}
	delete(a.Output, key)
	a.OutputStrings[key] = v
}

func (a *Actor) GetOutput(key string) (float64, bool) {
	if a == nil || a.Output == nil {
		return 0, false
	}
	v, ok := a.Output[key]
	return v, ok
}

//pob2:src/Classes/ModStore.lua:73-79
func (a *Actor) RelatedModDB(name string) moddb.ModStoreFuncs {
	var other *Actor
	switch name {
	case "player":
		switch {
		case a.Player != nil:
			other = a.Player
		case a.Parent != nil && a.Parent.Player != nil:
			other = a.Parent.Player
		case a.Enemy != nil:
			other = a.Enemy.Player
		}
	case "enemy":
		other = a.Enemy
	case "parent":
		other = a.Parent
	case "partyMembers":
		if a.partyMembers == nil {
			return nil
		}
		return a.partyMembers.modDB
	}
	if other == nil || other.ModDB == nil {
		return nil
	}
	return other.ModDB
}

// ConfigValue is one config input: a number, a string or a boolean, as the XML stored it.
type ConfigValue struct {
	Number    float64
	String    string
	Boolean   bool
	IsNumber  bool
	IsString  bool
	IsBoolean bool
}

// The ItemQueries methods give ModStore's ItemCondition tag the actor's itemList.
//
//pob2:block:src/Classes/ModStore.lua:800-856
var _ moddb.ItemQueries = (*Actor)(nil)

func (a *Actor) EquippedSlots() []string {
	slots := make([]string, 0, len(a.ItemList))
	for slot, item := range a.ItemList {
		if item != nil && item.Type != "Jewel" {
			slots = append(slots, slot)
		}
	}
	sort.Strings(slots)
	return slots
}

func (a *Actor) ItemInSlot(slot string) bool {
	return a.ItemList[slot] != nil
}

func (a *Actor) ItemModifierMatches(slot, searchCond string) bool {
	item := a.ItemList[slot]
	return item != nil && findModifierSubstring(item, searchCond, strings.ToLower(slot))
}

func (a *Actor) ItemRarityIs(slot, rarity string) bool {
	item := a.ItemList[slot]
	return item != nil && item.Rarity == rarity
}

func (a *Actor) ItemCorruptedIs(slot string, corrupted bool) bool {
	item := a.ItemList[slot]
	return item != nil && item.Corrupted == corrupted
}

func (a *Actor) ItemNameIs(slot, name string) bool {
	item := a.ItemList[slot]
	return item != nil && item.Name == name
}

// findModifierSubstring is ItemClass:FindModifierSubstring; substring and the special lists are Lua patterns.
//
//pob2:src/Classes/Item.lua:280-322
func findModifierSubstring(item *Item, substring, itemSlotName string) bool {
	explicit := strings.Contains(substring, "explicit ")
	substring = strings.ReplaceAll(substring, "explicit ", "")
	modLines := append([]*ModLine{}, item.ExplicitModLines...)
	if !explicit {
		modLines = append(modLines, item.EnchantModLines...)
		modLines = append(modLines, item.ImplicitModLines...)
	}
	find := func(line, pattern string) bool {
		re, ok := luaPattern(pattern)
		return ok && re.MatchString(line)
	}
	for _, v := range modLines {
		if v.Disabled || !item.CheckModLineVariant(v) {
			continue
		}
		line := strings.ToLower(v.Line)
		if find(line, substring) && !find(line, substring+" modifier") {
			excluded := false
			for _, specialMod := range itemTagSpecialExclusionPattern[substring][itemSlotName] {
				if find(line, strings.ToLower(specialMod)) {
					excluded = true
					break
				}
			}
			if !excluded {
				return true
			}
		}
		for _, specialMod := range itemTagSpecial[substring][itemSlotName] {
			if find(line, strings.ToLower(specialMod)) {
				return true
			}
		}
	}
	return false
}

//pob2:block:src/Modules/Data.lua:734-757
var itemTagSpecial = map[string]map[string][]string{
	"life": {
		"body armour": {"Blood Magic", "Eternal Youth", "Ghost Reaver", "Mind Over Matter", "The Agnostic", "Vaal Pact", "Zealot's Oath", "^Cannot Leech$"},
	},
	"evasion": {
		"ring": {"chance to Evade", "Cannot Evade"},
	},
}

//pob2:block:src/Modules/Data.lua:758-849
var itemTagSpecialExclusionPattern = map[string]map[string][]string{
	"life": {
		"amulet":      {"lower Life on Hit", "your Spectres' Life", "when on Full Life", "when on Low Life", "^Allocates"},
		"body armour": {"Life as Physical Damage", "Life as Extra Maximum Energy Shield", "maximum Life as Fire Damage", "when on Full Life", "when on Low Life", "Gain Maximum Life instead of Maximum Energy Shield", "^Socketed Gems are Supported by Level", "^Allocates"},
		"boots":       {"Enemy's Life", "^Enemies Cannot Leech Life", "when on Full Life", "when on Low Life", "^Allocates"},
		"belt":        {"Life as Extra Maximum Energy Shield", "Life Recovery from Flasks", "Life Flasks gain", "when on Full Life", "when on Low Life", "^Allocates"},
		"gloves":      {"maximum Life as Physical Damage", "Traps Cost Life", "when on Full Life", "when on Low Life", "^Allocates"},
		"helmet":      {"Recouped as Life", "Life when you Suppress", "Leech when on Low Life", "while no Life is Reserved", "^Socketed Gems are Supported by Level", "when on Full Life", "when on Low Life", "^Allocates"},
		"ring 1":      {"Energy Shield instead of Life", "increased Damage while Leeching Life", "when on Full Life", "when on Low Life", "^Allocates"},
		"ring 2":      {"Energy Shield instead of Life", "increased Damage while Leeching Life", "when on Full Life", "when on Low Life", "^Allocates"},
		"weapon 1":    {"^Socketed Gems are Supported by Level", "maximum Life as Chaos Damage", "total Maximum Life and Energy Shield as Fire Damage", "Life as Physical Damage", "maximum Life as Fire Damage", "when on Full Life", "when on Low Life", "^Allocates"},
		"weapon 2":    {"maximum Life as Chaos Damage", "^Socketed Gems Cost and Reserve Life", "increased Damage while Leeching Life", "Life as Physical Damage", "maximum Life as Fire Damage", "when on Full Life", "when on Low Life", "^Allocates"},
	},
	"evasion": {
		"ring": {},
	},
}
