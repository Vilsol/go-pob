package calculator

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/utils"
)

var tinctureBaseIDRe = regexp.MustCompile(`Metadata/Items/Tinctures/Tincture[0-9]`)

type ItemBaseReq struct {
	Level int
	Str   int
	Dex   int
	Int   int
}

type ItemBaseArmour struct {
	ArmourMin       int
	ArmourMax       int
	EvasionMin      int
	EvasionMax      int
	EnergyShieldMin int
	EnergyShieldMax int
	WardMin         int
	WardMax         int
	MovementPenalty int
	BlockChance     int
}

type ItemBaseWeapon struct {
	PhysicalMin    int
	PhysicalMax    int
	CritChanceBase float64
	AttackRateBase float64
	Range          int
}

type ItemBaseFlask struct {
	Life        int
	Mana        int
	Duration    float64
	ChargesMax  int
	ChargesUsed int
	Buff        []string
}

type ItemBaseTincture struct{}

type ItemBase struct {
	Name        string
	Type        string
	SubType     string
	SocketLimit int
	Req         ItemBaseReq
	Armour      *ItemBaseArmour
	Weapon      *ItemBaseWeapon
	Flask       *ItemBaseFlask
	Tincture    *ItemBaseTincture
}

//pob:block:src/Export/Scripts/bases.lua:119-127
func (b *ItemBase) IsWeaponType() bool {
	return weaponTypes[b.Type]
}

var itemClassTypes = map[string]string{
	"Abyss Jewels":              "Jewel",
	"Amulets":                   "Amulet",
	"Belts":                     "Belt",
	"Body Armours":              "Body Armour",
	"Boots":                     "Boots",
	"Bows":                      "Bow",
	"Claws":                     "Claw",
	"Daggers":                   "Dagger",
	"Rune Daggers":              "Dagger",
	"Gloves":                    "Gloves",
	"Helmets":                   "Helmet",
	"Jewels":                    "Jewel",
	"Life Flasks":               "Flask",
	"Mana Flasks":               "Flask",
	"Hybrid Flasks":             "Flask",
	"Utility Flasks":            "Flask",
	"One Hand Axes":             "One Handed Axe",
	"One Hand Maces":            "One Handed Mace",
	"One Hand Swords":           "One Handed Sword",
	"Thrusting One Hand Swords": "One Handed Sword",
	"Two Hand Axes":             "Two Handed Axe",
	"Two Hand Maces":            "Two Handed Mace",
	"Two Hand Swords":           "Two Handed Sword",
	"Quivers":                   "Quiver",
	"Rings":                     "Ring",
	"Sceptres":                  "Sceptre",
	"Shields":                   "Shield",
	"Staves":                    "Staff",
	"Warstaves":                 "Staff",
	"Wands":                     "Wand",
	"Fishing Rods":              "Fishing Rod",
	"Tinctures":                 "Tincture",
}

var itemClassSubTypes = map[string]string{
	"Life Flasks":    "Life",
	"Mana Flasks":    "Mana",
	"Hybrid Flasks":  "Hybrid",
	"Utility Flasks": "Utility",
	"Abyss Jewels":   "Abyss",
	"Rune Daggers":   "Rune Dagger",
	"Warstaves":      "Warstaff",

	"Thrusting One Hand Swords": "Thrusting",
}

//pob:block:src/Data/Bases/flask.lua:5-399
var flaskBuffs = map[string][]string{
	"Diamond Flask":     {"100% increased Global Critical Strike Chance"},
	"Ruby Flask":        {"+5% to maximum Fire Resistance", "+40% to Fire Resistance"},
	"Sapphire Flask":    {"+5% to maximum Cold Resistance", "+40% to Cold Resistance"},
	"Topaz Flask":       {"+5% to maximum Lightning Resistance", "+40% to Lightning Resistance"},
	"Granite Flask":     {"+1500 to Armour"},
	"Quicksilver Flask": {"40% increased Movement Speed"},
	"Amethyst Flask":    {"+35% to Chaos Resistance"},
	"Quartz Flask":      {"+10% chance to Suppress Spell Damage", "Phasing"},
	"Jade Flask":        {"+1500 to Evasion Rating"},
	"Basalt Flask":      {"20% more Armour"},
	"Aquamarine Flask":  {"60% reduced Effect of Freeze on you"},
	"Stibnite Flask":    {"20% more Evasion Rating"},
	"Sulphur Flask":     {"40% increased Damage"},
	"Silver Flask":      {"Onslaught"},
	"Bismuth Flask":     {"+35% to all Elemental Resistances"},
	"Gold Flask":        {"30% increased Rarity of Items found"},
	"Corundum Flask": {"30% increased Stun Threshold",
		"Cannot be Stunned if you've been Stunned during Effect"},
	"Iron Flask": {"+200 to Ward"},
}

var socketLimits = map[string]int{
	"Body Armour": 6,
	"Helmet":      4,
	"Gloves":      4,
	"Boots":       4,
	"Shield":      3,

	"Claw":                       3,
	"Dagger":                     3,
	"Rune Dagger":                3,
	"Wand":                       3,
	"One Handed Sword":           3,
	"Thrusting One Handed Sword": 3,
	"One Handed Axe":             3,
	"One Handed Mace":            3,
	"Sceptre":                    3,

	"Bow":              6,
	"Staff":            6,
	"Warstaff":         6,
	"Two Handed Sword": 6,
	"Two Handed Axe":   6,
	"Two Handed Mace":  6,
	"Fishing Rod":      4,
}

var weaponTypes = map[string]bool{
	"Claw": true, "Dagger": true, "Rune Dagger": true, "Wand": true,
	"One Handed Sword": true, "Thrusting One Handed Sword": true,
	"One Handed Axe": true, "One Handed Mace": true, "Sceptre": true,
	"Bow": true, "Staff": true, "Warstaff": true,
	"Two Handed Sword": true, "Two Handed Axe": true, "Two Handed Mace": true,
	"Fishing Rod": true,
}

const (
	twoTonedBoots = "Two-Toned Boots"
	energyBlade   = "Energy Blade"

	conditionAffectedByEnergyBlade = "AffectedByEnergyBlade"
)

//pob:block:src/Export/Scripts/bases.lua:116-118
func exportDisplayName(name, itemType string) string {
	name = strings.TrimSpace(strings.ReplaceAll(name, "ö", "o"))
	if name == energyBlade {
		if itemType == "One Handed Sword" {
			return energyBlade + " One Handed"
		}
		return energyBlade + " Two Handed"
	}
	return name
}

var itemBaseAliases = map[string]string{
	"Metadata/Items/Rings/Ring12": "Two-Stone Ring (Fire/Lightning)",
	"Metadata/Items/Rings/Ring13": "Two-Stone Ring (Cold/Lightning)",
	"Metadata/Items/Rings/Ring14": "Two-Stone Ring (Fire/Cold)",
}

//pob:block:src/Data/Bases/boots.lua:547,src/Data/Bases/boots.lua:660,src/Data/Bases/boots.lua:795
func twoTonedBootsVariant(req ItemBaseReq) string {
	switch {
	case req.Str > 0 && req.Int > 0:
		return twoTonedBoots + " (Armour/Energy Shield)"
	case req.Str > 0 && req.Dex > 0:
		return twoTonedBoots + " (Armour/Evasion)"
	case req.Dex > 0 && req.Int > 0:
		return twoTonedBoots + " (Evasion/Energy Shield)"
	}
	return twoTonedBoots
}

var (
	itemBasesOnce sync.Once
	itemBases     map[string]*ItemBase
)

//pob:block:src/Modules/Data.lua:1212-1215
func ItemBases() map[string]*ItemBase {
	itemBasesOnce.Do(buildItemBases)
	return itemBases
}

func ItemBaseByName(name string) *ItemBase {
	return ItemBases()[name]
}

var ItemBaseNames = sync.OnceValue(func() []string {
	bases := ItemBases()
	names := make([]string, 0, len(bases))
	for n := range bases {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
})

var itemBaseNameIndex = sync.OnceValues(func() (map[uint16][]string, []string) {
	byBigram := make(map[uint16][]string)
	var short []string
	for _, n := range ItemBaseNames() {
		switch {
		case n == "":
		case len(n) < 2:
			short = append(short, n)
		default:
			k := uint16(n[0])<<8 | uint16(n[1])
			byBigram[k] = append(byBigram[k], n)
		}
	}
	return byBigram, short
})

func hasEmptyItemBaseName() bool {
	_, ok := ItemBases()[""]
	return ok
}

func findItemBaseInName(name string) (string, int) {
	byBigram, short := itemBaseNameIndex()
	best, bestS := "", -1
	consider := func(cand string, s int) {
		if len(cand) > len(best) || (len(cand) == len(best) && cand < best) {
			best, bestS = cand, s
		}
	}
	for s := 0; s <= len(name); s++ {
		for _, cand := range short {
			if strings.HasPrefix(name[s:], cand) {
				consider(cand, s)
			}
		}
		if s+1 < len(name) {
			for _, cand := range byBigram[uint16(name[s])<<8|uint16(name[s+1])] {
				if strings.HasPrefix(name[s:], cand) {
					consider(cand, s)
				}
			}
		}
	}
	return best, bestS
}

//pob:src/Export/Scripts/bases.lua:35-209
//pob:src/Export/Scripts/bases.lua:276-293
func baseReqLevel(b *poe.BaseItemType, hasWeaponOrArmour bool, hasFlask bool, modByKey map[int]*poe.Mod) int {
	level := 1

	if hasWeaponOrArmour && b.DropLevel > 4 {
		level = b.DropLevel
	}

	if hasFlask && b.DropLevel > 2 {
		level = b.DropLevel
	}

	for _, key := range b.ImplicitModsKeys {
		m := modByKey[key]
		if m == nil {
			continue
		}

		if implicit := int(math.Floor(float64(m.Level) * 0.8)); implicit > level {
			level = implicit
		}
	}

	if level <= 1 {
		return 0
	}

	return level
}

func buildItemBases() {
	classByKey := make(map[int]*poe.ItemClass, len(poe.ItemClasses))
	for _, c := range poe.ItemClasses {
		classByKey[c.Key] = c
	}

	reqByBaseID := make(map[string]*poe.ComponentAttributeRequirement, len(poe.ComponentAttributeRequirements))
	for _, r := range poe.ComponentAttributeRequirements {
		reqByBaseID[r.BaseItemTypesKey] = r
	}

	armourByBaseKey := make(map[int]*poe.ArmourType, len(poe.ArmourTypes))
	for _, a := range poe.ArmourTypes {
		armourByBaseKey[a.BaseItemTypesKey] = a
	}

	flaskByBaseKey := make(map[int]*poe.Flask, len(poe.Flasks))
	for _, f := range poe.Flasks {
		flaskByBaseKey[f.BaseItemTypesKey] = f
	}

	weaponByBaseKey := make(map[int]*poe.WeaponType, len(poe.WeaponTypes))
	for _, wt := range poe.WeaponTypes {
		weaponByBaseKey[wt.BaseItemTypesKey] = wt
	}

	chargesByBaseID := make(map[string]*poe.ComponentCharge, len(poe.ComponentCharges))
	for _, c := range poe.ComponentCharges {
		chargesByBaseID[c.BaseItemTypesKey] = c
	}

	shieldByBaseKey := make(map[int]*poe.ShieldType, len(poe.ShieldTypes))
	for _, s := range poe.ShieldTypes {
		shieldByBaseKey[s.BaseItemTypesKey] = s
	}

	mods := raw.Mods()
	modByKey := make(map[int]*poe.Mod, len(mods))
	for _, m := range mods {
		modByKey[m.Key] = m
	}

	itemBases = make(map[string]*ItemBase, len(poe.BaseItemTypes))
	for _, b := range poe.BaseItemTypes {
		if strings.Contains(b.ID, "Royale") {
			continue
		}

		class := classByKey[b.ItemClassesKey]
		if class == nil {
			continue
		}

		itemType, ok := itemClassTypes[class.Name]
		if !ok {
			continue
		}

		if itemType == "Tincture" && !tinctureBaseIDRe.MatchString(b.ID) {
			continue
		}

		armour := armourByBaseKey[b.Key]
		weapon := weaponByBaseKey[b.Key]
		flask := flaskByBaseKey[b.Key]

		base := &ItemBase{
			Name:        exportDisplayName(b.Name, itemType),
			Type:        itemType,
			SubType:     itemClassSubTypes[class.Name],
			SocketLimit: socketLimits[itemType],
			Req:         ItemBaseReq{Level: baseReqLevel(b, weapon != nil || armour != nil, flask != nil, modByKey)},
		}

		if r := reqByBaseID[b.ID]; r != nil {
			base.Req.Str = r.ReqStr
			base.Req.Dex = r.ReqDex
			base.Req.Int = r.ReqInt
		}

		if b.Name == twoTonedBoots {
			base.Name = twoTonedBootsVariant(base.Req)
		}

		if a := armour; a != nil {
			base.Armour = &ItemBaseArmour{
				ArmourMin: a.ArmourMin, ArmourMax: a.ArmourMax,
				EvasionMin: a.EvasionMin, EvasionMax: a.EvasionMax,
				EnergyShieldMin: a.EnergyShieldMin, EnergyShieldMax: a.EnergyShieldMax,
				WardMin: a.WardMin, WardMax: a.WardMax,
				MovementPenalty: -a.IncreasedMovementSpeed,
			}
			if s := shieldByBaseKey[b.Key]; s != nil {
				base.Armour.BlockChance = s.Block
			}
		}

		if wt := weapon; wt != nil {
			base.Weapon = &ItemBaseWeapon{
				PhysicalMin:    wt.DamageMin,
				PhysicalMax:    wt.DamageMax,
				CritChanceBase: float64(wt.Critical) / 100,
				AttackRateBase: utils.RoundTo(1000/float64(wt.Speed), 2),
				Range:          wt.RangeMax,
			}
		}

		if f := flask; f != nil {
			flask := &ItemBaseFlask{
				Life:     f.LifePerUse,
				Mana:     f.ManaPerUse,
				Duration: float64(f.RecoveryTime) / 10,
				Buff:     flaskBuffs[b.Name],
			}
			if c := chargesByBaseID[b.ID]; c != nil {
				flask.ChargesMax = c.MaxCharges
				flask.ChargesUsed = c.PerCharge
			}
			base.Flask = flask
		}

		if itemType == "Tincture" {
			base.Tincture = &ItemBaseTincture{}
		}

		itemBases[base.Name] = base

		if alias, ok := itemBaseAliases[b.ID]; ok {
			aliased := *base
			aliased.Name = alias
			itemBases[alias] = &aliased
		}
	}
}
