package calculator2

import (
	"sort"
	"strings"

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

	Spec    *tree2.PassiveSpec
	ClassID int

	ModDB     *moddb.ModDB
	EnemyDB   *moddb.ModDB
	ItemModDB *moddb.ModDB

	EnemyLevel int
	Player     *Actor
	Enemy      *Actor

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
}

// Actor is env.player / env.enemy.
type Actor struct {
	ModDB  *moddb.ModDB
	Level  int
	Output map[string]float64

	Player *Actor
	Enemy  *Actor
	Parent *Actor

	// ItemList is actor.itemList: the equipped item per slot name.
	ItemList map[string]*Item
	// WeaponData1Type and WeaponData2Type are actor.weaponData1/2.type; "" is nil.
	WeaponData1Type string
	WeaponData2Type string

	appliedEnemyModifiers map[mod.Mod]bool
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
