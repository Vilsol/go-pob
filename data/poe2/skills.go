package poe2

import "sort"

// SkillOverlay is the part of one data.skills[ID] entry that PoB-PoE2 writes by hand (the directives and
// Lua of src/Export/Skills/*.txt) rather than exporting from the game files: levels, stats, skill types,
// cast time and the other per-level numbers come from go-pob-data. Hidden is PoB's hidden flag, set for an
// effect no gem grants and by #noGem. AddSkillTypes is #addSkillTypes, already folded into PoB's skillTypes;
// DisplayName is the name a #skill directive gives. PreDamageFunc says the entry carries a Lua function,
// which the calc ports by hand. NoTemplate marks an effect PoB writes straight into its skill files, with
// no export template: its stat sets carry no ID, and nothing of it comes from the game files, so the game
// half of its data.skills entry (Name through HasQualityStats, and each set's Label through Levels) is
// written here instead; it is unset on every other effect.
//
//pob2:src/Modules/Data.lua:926-928
type SkillOverlay struct {
	ID                 string
	Hidden             bool
	FromItem           bool
	FromTree           bool
	HideFromSideBar    bool
	GrantsThornsDamage bool
	Legacy             bool
	MinionHasItemSet   bool
	PreDamageFunc      bool
	NoTemplate         bool
	MinionList         []string
	AddFlags           []string
	MinionUses         []string
	AddSkillTypes      []string
	DisplayName        string
	StatSets           []StatSetOverlay
	Name               string
	BaseTypeName       string
	Description        string
	SkillTypes         []string
	Levels             []LevelOverlay
	Color              int
	CastTime           float64
	HasBaseTypeName    bool
	HasCastTime        bool
	CannotBeSupported  bool
	HasQualityStats    bool
}

// LevelOverlay is a NoTemplate effect's levels[Level], its named numbers.
type LevelOverlay struct {
	Extra map[string]float64
	Level int
}

// ConstantStatOverlay is one { stat, value } of a NoTemplate set's constantStats.
type ConstantStatOverlay struct {
	Stat  string
	Value float64
}

// StatSetOverlay is the hand-written part of one statSets[i]: ID, the GrantedEffectStatSets id its #set
// directive exported it from (the join key to the game rows), baseFlags (nil on a support's set, which
// PoB leaves without one; HasBaseFlags tells that apart from an empty set), baseMods as ModCache-style
// mod JSON, and the set's own statMap sorted by stat. A NoTemplate effect's set also carries its game
// half; Levels are its level numbers, each an empty levels[n].
type StatSetOverlay struct {
	ID           string
	BaseFlags    []string
	HasBaseFlags bool
	BaseMods     []string
	StatMap      []StatMapEntry

	Label                             string
	StatDescriptionScope              string
	ConstantStats                     []ConstantStatOverlay
	Stats                             []string
	Levels                            []int
	BaseEffectiveness                 float64
	IncrementalEffectiveness          float64
	DamageIncrementalEffectiveness    float64
	HasLabel                          bool
	HasStatDescriptionScope           bool
	HasConstantStats                  bool
	HasStats                          bool
	HasBaseEffectiveness              bool
	HasIncrementalEffectiveness       bool
	HasDamageIncrementalEffectiveness bool
}

// StatMapEntry is statMap[Stat]: its mods and groups in PoB's order, its value, mult, div and base, and
// its skillFlag. PoB reads each scalar with `or`, so presence is kept beside the number: `value = 0`
// still replaces the stat's value.
type StatMapEntry struct {
	Stat                               string
	Value, Mult, Div, Base             float64
	HasValue, HasMult, HasDiv, HasBase bool
	SkillFlag                          string
	Items                              []StatMapItem
}

// StatMapItem is one element of a statMap entry: a mod (Mod, ModCache-style JSON without a source), or a
// group of mods sharing the group's own scalars.
type StatMapItem struct {
	Mod                                string
	Group                              []string
	Value, Mult, Div, Base             float64
	HasValue, HasMult, HasDiv, HasBase bool
}

// SkillOverlayByID is the hand-written part of data.skills[id].
//
//pob2:block:src/Modules/Data.lua:929-933
func SkillOverlayByID(id string) (*SkillOverlay, bool) {
	i := sort.Search(len(SkillOverlays), func(i int) bool { return SkillOverlays[i].ID >= id })
	if i < len(SkillOverlays) && SkillOverlays[i].ID == id {
		return &SkillOverlays[i], true
	}
	return nil, false
}

// StatMapLookup is the entry for stat in a sorted statMap, without the global fallback.
func StatMapLookup(entries []StatMapEntry, stat string) (*StatMapEntry, bool) {
	i := sort.Search(len(entries), func(i int) bool { return entries[i].Stat >= stat })
	if i < len(entries) && entries[i].Stat == stat {
		return &entries[i], true
	}
	return nil, false
}

// SkillStatMapEntry is data.skillStatMap[stat], the global map a stat set's statMap falls back to.
//
//pob2:block:src/Modules/Data.lua:915-915
func SkillStatMapEntry(stat string) (*StatMapEntry, bool) {
	return StatMapLookup(SkillStatMap, stat)
}

// SkillTypeName is SkillTypeName[id], Global.lua's reverse lookup.
//
//pob2:block:src/Data/Global.lua:629-632
func SkillTypeName(id int) (string, bool) {
	for name, v := range SkillType {
		if v == id {
			return name, true
		}
	}
	return "", false
}
