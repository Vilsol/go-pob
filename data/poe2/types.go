// Package poe2 holds Path of Building (PoE2)'s game data tables, generated from the pinned PoB-PoE2
// checkout by `mise run data-regen --game poe2`. The PoE1 engine never imports it.
package poe2

// ItemBase is one data.itemBases entry. A numeric field PoB leaves nil reads 0 here; the emitter
// refuses a present 0 on every field PoB tests for presence, so 0 always means absent on those.
//
//pob2:src/Modules/Data.lua:1108-1113
type ItemBase struct {
	Name        string
	Type        string
	SubType     string
	Hidden      bool
	SocketLimit float64
	Quality     float64
	Spirit      float64
	CharmLimit  float64
	// HasCharmLimit is a present charmLimit, which belts carry as 0.
	HasCharmLimit                  bool
	GrantedSkillsHaveNoReservation bool
	Implicit                       string
	ImplicitModTypes               [][]string
	Tags                           []string
	VariantList                    []string
	Req                            BaseReq
	Armour                         *BaseArmour
	Weapon                         *BaseWeapon
	Flask                          *BaseFlask
	Charm                          *BaseCharm
}

type BaseReq struct {
	Level, Str, Dex, Int float64
}

type BaseArmour struct {
	Armour, Evasion, EnergyShield, Ward, MovementPenalty, BlockChance float64
}

type BaseWeapon struct {
	PhysicalMin, PhysicalMax   float64
	FireMin, FireMax           float64
	ColdMin, ColdMax           float64
	LightningMin, LightningMax float64
	ChaosMin, ChaosMax         float64
	CritChanceBase             float64
	AttackRateBase             float64
	Range                      float64
	ReloadTimeBase             float64
}

type BaseFlask struct {
	Life, Mana, Duration, ChargesMax, ChargesUsed float64
}

type BaseCharm struct {
	Duration, ChargesMax, ChargesUsed float64
	Buff                              []string
}

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
