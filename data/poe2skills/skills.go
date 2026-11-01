// Package poe2skills builds PoB-PoE2's data.skills: the game's skill tables from go-pob-data, read the
// way src/Export/Scripts/skills.lua reads them, joined with the hand-written overlay in data/poe2, then
// processed as src/Modules/Data.lua loads them.
package poe2skills

import (
	"context"
	"fmt"
	"sort"

	"github.com/Vilsol/go-pob-data/loader"
	game "github.com/Vilsol/go-pob-data/poe2"

	"github.com/Vilsol/go-pob/data/poe2"
)

// Mod is one mod table as PoB holds it, in the ModCache-style JSON shape (name, type, value, flags,
// keywordFlags, source, tags); a nil value is set when the stat is merged.
type Mod = map[string]any

// GrantedEffect is data.skills[ID]. Nil slices and maps are fields PoB leaves nil; Has* flags keep a
// field that can be zero apart from an absent one.
type GrantedEffect struct {
	Overlay           *poe2.SkillOverlay
	Levels            map[int]*Level
	SkillTypes        map[string]bool
	MinionSkillTypes  map[string]bool
	WeaponTypes       map[string]bool
	ID                string
	Name              string
	BaseTypeName      string
	Description       string
	Icon              string
	ModSource         string
	RequireSkillTypes []string
	AddSkillTypes     []string
	ExcludeSkillTypes []string
	GemFamily         []string
	FlavourText       []string
	QualityStats      []QualityStat
	AltQualityStats   []QualityStat
	StatSets          []*StatSet
	Color             int
	SkillTotemID      int
	CastTime          float64
	HasBaseTypeName   bool
	HasIcon           bool
	HasCastTime       bool
	HasSkillTotemID   bool
	Hidden            bool
	Support           bool
	IsLineage         bool
	IsTrigger         bool
	SupportGemsOnly   bool
	IgnoreMinionTypes bool
	CannotBeSupported bool
	HasGlobalEffect   bool
}

// QualityStat is one { stat, value, { statSetIndex... } } row of qualityStats.
type QualityStat struct {
	Stat     string
	StatSets []float64
	Value    float64
}

// Level is levels[n]: the extra numbers skills.lua writes beside it by their PoB names, and its cost.
type Level struct {
	Extra map[string]float64
	Cost  map[string]float64
	Level int
}

// StatSet is statSets[i].
type StatSet struct {
	effect                         *GrantedEffect
	Overlay                        *poe2.StatSetOverlay
	BaseFlags                      map[string]bool
	Levels                         map[int]*StatSetLevel
	statMap                        map[string]*StatMap
	Label                          string
	StatDescriptionScope           string
	BaseMods                       []Mod
	ConstantStats                  []ConstantStat
	Stats                          []string
	NotMinionStat                  []string
	BaseEffectiveness              float64
	IncrementalEffectiveness       float64
	DamageIncrementalEffectiveness float64
	HasLabel                       bool
	HasStatDescriptionScope        bool
	HasStats                       bool
	HasBaseEffectiveness           bool
	HasIncrementalEffectiveness    bool
	HasDamageIncrementalEffect     bool
}

// ConstantStat is one { stat, value } of constantStats.
type ConstantStat struct {
	Stat  string
	Value float64
}

// StatSetLevel is statSets[i].levels[n]: the stat values in stats order, the extra numbers by name, and
// statInterpolation.
type StatSetLevel struct {
	Extra             map[string]float64
	Values            []float64
	StatInterpolation []float64
	ActorLevel        float64
	Level             int
	HasActorLevel     bool
}

// StatMap is statMap[stat] with its mods processed. PoB reads each scalar with `or`, so presence is kept
// beside the number.
type StatMap struct {
	Stat                               string
	SkillFlag                          string
	Items                              []StatMapItem
	Value, Mult, Div, Base             float64
	HasValue, HasMult, HasDiv, HasBase bool
}

// StatMapItem is a mod, or a group of mods sharing the group's own scalars. GroupSource is the source
// PoB's metatable writes onto a group it copies from the global map.
type StatMapItem struct {
	Mod                                Mod
	GroupSource                        string
	Group                              []Mod
	Value, Mult, Div, Base             float64
	HasValue, HasMult, HasDiv, HasBase bool
}

// Skills is data.skills.
type Skills struct {
	byID map[string]*GrantedEffect
	gems map[string]*Gem
	ids  []string
}

// Effect is data.skills[id].
func (s *Skills) Effect(id string) (*GrantedEffect, bool) {
	ge, ok := s.byID[id]
	return ge, ok
}

// IDs are every granted effect id, sorted.
func (s *Skills) IDs() []string {
	return s.ids
}

// Load reads version's skill tables and the flavour text the lineage supports quote, then builds
// data.skills.
func Load(ctx context.Context, version string, cache loader.AssetCache) (*Skills, error) {
	tables, err := game.LoadSkillTables(ctx, version, cache)
	if err != nil {
		return nil, fmt.Errorf("skill tables: %w", err)
	}
	rows, err := loader.LoadRaw[*game.FlavourText](ctx, version, game.FlavourTextTable, nil, cache)
	if err != nil {
		return nil, fmt.Errorf("flavour text: %w", err)
	}
	flavourText := make([]game.FlavourText, len(rows))
	for i, row := range rows {
		flavourText[i] = *row
	}
	return Build(tables, flavourText), nil
}

// Build is data.skills from the game's skill tables and the overlay.
func Build(tables *game.SkillTables, flavourText []game.FlavourText) *Skills {
	s := &Skills{byID: make(map[string]*GrantedEffect, len(poe2.SkillOverlays))}
	e := newExporter(tables, flavourText)
	for i := range poe2.SkillOverlays {
		o := &poe2.SkillOverlays[i]
		ge := e.exportSkill(o)
		s.byID[o.ID] = ge
		s.ids = append(s.ids, o.ID)
	}
	sort.Strings(s.ids)
	for _, id := range s.ids {
		loadGrantedEffect(s.byID[id], id)
	}
	s.setupGems()
	return s
}

// LevelFor is the level row a gem level reads, levels[level], nil when the effect has none.
func (ge *GrantedEffect) LevelFor(level int) *Level {
	return ge.Levels[level]
}

// LevelFor is the stat set's levels[level], nil when it has none.
func (set *StatSet) LevelFor(level int) *StatSetLevel {
	return set.Levels[level]
}
