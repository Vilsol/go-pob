// Package poe2 holds Path of Building (PoE2)'s game data tables, generated from the pinned PoB-PoE2
// checkout by `mise run data-regen --game poe2`. The PoE1 engine never imports it.
package poe2

import "github.com/Vilsol/go-pob-data/poe2/pobdata"

// ItemBase is one data.itemBases entry, published by go-pob-data. A numeric field PoB leaves nil reads 0
// here; the emitter refuses a present 0 on every field PoB tests for presence, so 0 always means absent on
// those.
//
//pob2:src/Modules/Data.lua:1108-1113
type ItemBase = pobdata.ItemBase

type (
	BaseReq    = pobdata.BaseReq
	BaseArmour = pobdata.BaseArmour
	BaseWeapon = pobdata.BaseWeapon
	BaseFlask  = pobdata.BaseFlask
	BaseCharm  = pobdata.BaseCharm
)

// RuneMod is one slot entry of a data.itemMods.Runes rune or soul core; its tradeHashes are not
// carried.
//
//pob2:src/Modules/Data.lua:699-699
type RuneMod struct {
	Rune                           string
	Slot                           string
	Type                           string
	Lines                          []string
	StatOrder                      []float64
	Bonded                         []string
	BondedStatOrder                []float64
	Limit                          float64
	LimitID                        string
	LocalMod                       bool
	LevelReq                       float64
	CanSocketInUniqueItems         bool
	CanSocketInJewellery           bool
	CanSocketInCorruptedSanctified bool
	CanSocketInChakraSlots         bool
	IsSocketBound                  bool
}

// Gem is one data.gems entry, with its granted effect's name and support flag.
//
//pob2:src/Modules/Data.lua:968-1051
type Gem struct {
	ID                         string
	Name                       string
	GameID                     string
	VariantID                  string
	BaseTypeName               string
	GrantedEffectID            string
	GrantedEffectName          string
	Support                    bool
	AdditionalGrantedEffectIDs []string
	AdditionalStatSets         []string
	GrantedEffectDisplayOrder  []float64
	SecondaryEffectName        string
	GemType                    string
	GemFamily                  string
	TagString                  string
	Tags                       []string
	WeaponRequirements         string
	ReqStr, ReqDex, ReqInt     float64
	Tier                       float64
	NaturalMaxLevel            float64
}

// LegionNode is one tree.legion.nodes entry: a node a timeless jewel can put in place of a tree node.
//
//pob2:src/Classes/PassiveTree.lua:438-452
type LegionNode struct {
	ID  string
	Dn  string
	Sd  []string
	Ks  bool
	Not bool
	M   bool
}

// ItemMod is one mod of a data.itemMods affix table (src/Data/Mod*.lua), published by go-pob-data.
//
//pob2:src/Modules/Data.lua:692-702
type ItemMod = pobdata.ItemMod

// ExclusiveMod is one data.itemMods.Exclusive entry's lines and stat orders, published by go-pob-data.
//
//pob2:block:src/Classes/Item.lua:1631-1638
type ExclusiveMod = pobdata.ExclusiveMod
