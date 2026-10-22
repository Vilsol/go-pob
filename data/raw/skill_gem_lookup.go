package raw

import (
	"strings"
	"sync"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
)

func init() {
	builds.GemGrantedEffectID = func(gemID string) (string, bool) {
		row, ok := Gems[gemID]
		if !ok || row.GrantedEffectID == "" {
			return "", false
		}
		return row.GrantedEffectID, true
	}
}

var (
	gemLookupMutex   sync.Mutex
	skillGemByGameID map[string]*poe.SkillGem
)

func SkillGemForGrantedEffectID(effectID string) *poe.SkillGem {
	gameID, ok := GemGameIDForGrantedEffectID[effectID]
	if !ok {
		return nil
	}

	gemLookupMutex.Lock()
	defer gemLookupMutex.Unlock()
	if skillGemByGameID == nil {
		skillGemByGameID = make(map[string]*poe.SkillGem, len(poe.SkillGems))
		for _, gem := range poe.SkillGems {
			base := gem.GetBaseItemType()
			if base == nil {
				continue
			}
			if _, exists := skillGemByGameID[base.ID]; !exists {
				skillGemByGameID[base.ID] = gem
			}
		}
	}

	return skillGemByGameID[gameID]
}

func GrantedEffectsForGem(gem *poe.SkillGem) []*poe.GrantedEffect {
	if gem == nil {
		return nil
	}
	base := gem.GetBaseItemType()
	if base == nil {
		return nil
	}

	row, ok := Gems[base.ID]
	if !ok {
		return nil
	}

	out := make([]*poe.GrantedEffect, 0, 2)
	for _, id := range []string{row.GrantedEffectID, row.SecondaryGrantedEffectID} {
		if id == "" {
			continue
		}
		if effect := poe.GrantedEffectByID(id); effect != nil {
			out = append(out, effect)
		}
	}
	return out
}

var (
	gemGameIDByNameOnce sync.Once
	gemGameIDByName     map[string]string
)

func GemGameIDForName(name string) string {
	if name == "" {
		return ""
	}
	gemGameIDByNameOnce.Do(func() {
		gemGameIDByName = make(map[string]string, len(Gems))
		for gameID, row := range Gems {
			if row.Name == "" {
				continue
			}
			key := strings.ToLower(row.Name)
			if prev, exists := gemGameIDByName[key]; !exists || gameID < prev {
				gemGameIDByName[key] = gameID
			}
		}
	})
	return gemGameIDByName[strings.ToLower(name)]
}
