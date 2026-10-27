package calculator2

import (
	"context"
	"sort"
	"sync"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/poe2skills"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// skillDataVersion is the game version whose skill tables data.skills is built from.
const skillDataVersion = "0.5"

// skillData is data.skills and the gem lookups Data.lua builds beside it.
type skillData struct {
	skills       *poe2skills.Skills
	gemsByGameID map[string]map[string]*poe2skills.Gem
	gemForSkill  map[*poe2skills.GrantedEffect]string
}

var loadSkillData = sync.OnceValues(func() (*skillData, error) {
	skills, err := poe2skills.Load(context.Background(), skillDataVersion, raw.AssetLoaderWrapper{Storage: storage.Get()})
	if err != nil {
		return nil, err
	}
	return newSkillData(skills), nil
})

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
