package calculator2

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/Vilsol/go-pob-data/loader"
	game "github.com/Vilsol/go-pob-data/poe2"
	"github.com/tinylib/msgp/msgp"

	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/poe2skills"
	"github.com/Vilsol/go-pob/storage"
)

// skillDataVersion is the game version whose skill tables data.skills is built from.
const skillDataVersion = "0.5"

// skillData is data.skills and the gem lookups Data.lua builds beside it.
type skillData struct {
	skills       *poe2skills.Skills
	gemsByGameID map[string]map[string]*poe2skills.Gem
	gemForSkill  map[*poe2skills.GrantedEffect]string
	// characterMeleeSkills is data.characterMeleeSkills[mainHand][offHand]; a nil gem is PoB's false.
	characterMeleeSkills map[string]map[string][]*poe2skills.Gem
}

// gameDataCache is the asset cache the game data loads through.
func gameDataCache() loader.AssetCache {
	return assetcache.AssetLoaderWrapper{Storage: storage.Get()}
}

// gameDataContext reads the .min assets of the tables go-pob-data's field list names.
func gameDataContext() context.Context {
	return loader.WithVariants(context.Background(), game.MinVariant)
}

// loadRows is loader.LoadRaw's rows as values.
func loadRows[T any, PT interface {
	*T
	msgp.Decodable
}](ctx context.Context, name string, cache loader.AssetCache) ([]T, error) {
	rows, err := loader.LoadRaw[PT](ctx, skillDataVersion, name, nil, cache)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	out := make([]T, len(rows))
	for i, row := range rows {
		out[i] = *row
	}
	return out, nil
}

var loadSkillData = sync.OnceValues(func() (*skillData, error) {
	ctx := gameDataContext()
	cache := gameDataCache()
	tables, err := game.LoadSkillTables(ctx, skillDataVersion, cache)
	if err != nil {
		return nil, fmt.Errorf("skill tables: %w", err)
	}
	flavourText, err := loadRows[game.FlavourText](ctx, game.FlavourTextTable, cache)
	if err != nil {
		return nil, fmt.Errorf("flavour text: %w", err)
	}
	melee, err := loadRows[game.CharacterMeleeSkill](ctx, game.CharacterMeleeSkillsTable, cache)
	if err != nil {
		return nil, fmt.Errorf("character melee skills: %w", err)
	}
	d := newSkillData(poe2skills.Build(tables, flavourText))
	d.characterMeleeSkills = d.resolveCharacterMeleeSkills(tables.CharacterMeleeSkills(melee))
	return d, nil
})

// PoB takes variants[next(variants)], an arbitrary variant when a game id has several; the first
// variant id in sort order is taken here.
//
//pob2:block:src/Modules/Data.lua:1072-1081
func (d *skillData) resolveCharacterMeleeSkills(byGameID map[string]map[string][]string) map[string]map[string][]*poe2skills.Gem {
	out := make(map[string]map[string][]*poe2skills.Gem, len(byGameID))
	for mainHand, offHandSkills := range byGameID {
		out[mainHand] = make(map[string][]*poe2skills.Gem, len(offHandSkills))
		for offHand, gameIDs := range offHandSkills {
			gems := make([]*poe2skills.Gem, len(gameIDs))
			for index, gameID := range gameIDs {
				variants := d.gemsByGameID[gameID]
				variantIDs := make([]string, 0, len(variants))
				for id := range variants {
					variantIDs = append(variantIDs, id)
				}
				sort.Strings(variantIDs)
				if len(variantIDs) > 0 {
					gems[index] = variants[variantIDs[0]]
				}
			}
			out[mainHand][offHand] = gems
		}
	}
	return out
}

// PoB fills gemForSkill in pairs() order, so of two gems granting one effect an arbitrary one wins;
// ids are walked sorted here and the last wins.
//
//pob2:block:src/Modules/Data.lua:980-986
func newSkillData(skills *poe2skills.Skills) *skillData {
	d := &skillData{
		skills:       skills,
		gemsByGameID: map[string]map[string]*poe2skills.Gem{},
		gemForSkill:  map[*poe2skills.GrantedEffect]string{},
	}
	ids := make([]string, len(poe2.Gems))
	for i := range poe2.Gems {
		ids[i] = poe2.Gems[i].ID
	}
	sort.Strings(ids)
	for _, id := range ids {
		gem, ok := skills.Gem(id)
		if !ok {
			continue
		}
		if gem.GrantedEffect != nil {
			d.gemForSkill[gem.GrantedEffect] = id
		}
		if d.gemsByGameID[gem.Data.GameID] == nil {
			d.gemsByGameID[gem.Data.GameID] = map[string]*poe2skills.Gem{}
		}
		d.gemsByGameID[gem.Data.GameID][gem.Data.VariantID] = gem
	}
	return d
}

func (d *skillData) effect(id string) *poe2skills.GrantedEffect {
	ge, _ := d.skills.Effect(id)
	return ge
}

func (d *skillData) gem(id string) *poe2skills.Gem {
	g, _ := d.skills.Gem(id)
	return g
}

// gemForBaseName is data.gems[data.gemForBaseName[name]].
func (d *skillData) gemForBaseName(name string) *poe2skills.Gem {
	g := gemByBaseName(name)
	if g == nil {
		return nil
	}
	return d.gem(g.ID)
}

var gemTagSets sync.Map

// gemTags is gemData.tags as a set.
func gemTags(gem *poe2.Gem) map[string]bool {
	if v, ok := gemTagSets.Load(gem); ok {
		return v.(map[string]bool)
	}
	tags := make(map[string]bool, len(gem.Tags))
	for _, t := range gem.Tags {
		tags[t] = true
	}
	v, _ := gemTagSets.LoadOrStore(gem, tags)
	return v.(map[string]bool)
}

// levelCount is #grantedEffect.levels: the border of the level table from 1.
func levelCount(ge *poe2skills.GrantedEffect) int {
	n := 0
	for ge.Levels[n+1] != nil {
		n++
	}
	return n
}

// levelRequirement is levels[level].levelRequirement; false when the row or the field is absent.
func levelRequirement(ge *poe2skills.GrantedEffect, level int) (float64, bool) {
	row := ge.Levels[level]
	if row == nil {
		return 0, false
	}
	v, ok := row.Extra["levelRequirement"]
	return v, ok
}

// lowestLevel is the lowest numeric key of grantedEffect.levels, 0 when there is none.
func lowestLevel(ge *poe2skills.GrantedEffect) int {
	lowest := 0
	for level := range ge.Levels {
		if lowest == 0 || level < lowest {
			lowest = level
		}
	}
	return lowest
}
