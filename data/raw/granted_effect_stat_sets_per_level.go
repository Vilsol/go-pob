package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type GrantedEffectStatSetsPerLevel struct {
	raw2.GrantedEffectStatSetsPerLevel
}

var GrantedEffectStatSetsPerLevels []*GrantedEffectStatSetsPerLevel

var grantedEffectStatSetsPerLevelsByIDMap map[int]map[int]*GrantedEffectStatSetsPerLevel

func InitializeGrantedEffectStatSetsPerLevels(version string) error {
	return InitHelper(version, "GrantedEffectStatSetsPerLevel", &GrantedEffectStatSetsPerLevels, func(count int64) {
		grantedEffectStatSetsPerLevelsByIDMap = make(map[int]map[int]*GrantedEffectStatSetsPerLevel, count)
	}, func(obj *GrantedEffectStatSetsPerLevel) {
		for _, effect := range obj.GrantedEffects {
			if _, ok := grantedEffectStatSetsPerLevelsByIDMap[effect]; !ok {
				grantedEffectStatSetsPerLevelsByIDMap[effect] = make(map[int]*GrantedEffectStatSetsPerLevel)
			}

			grantedEffectStatSetsPerLevelsByIDMap[effect][obj.GemLevel] = obj
		}
	})
}

func (g *GrantedEffectStatSetsPerLevel) GetFloatStats() []*Stat {
	if g.FloatStats == nil {
		return nil
	}

	out := make([]*Stat, len(g.FloatStats))
	for i, stat := range g.FloatStats {
		out[i] = Stats[stat]
	}
	return out
}

func (g *GrantedEffectStatSetsPerLevel) GetAdditionalStats() []*Stat {
	if g.AdditionalStats == nil {
		return nil
	}

	out := make([]*Stat, len(g.AdditionalStats))
	for i, stat := range g.AdditionalStats {
		out[i] = Stats[stat]
	}
	return out
}

func (g *GrantedEffectStatSetsPerLevel) GetAdditionalBooleanStats() []*Stat {
	if g.AdditionalBooleanStats == nil {
		return nil
	}

	out := make([]*Stat, len(g.AdditionalBooleanStats))
	for i, stat := range g.AdditionalBooleanStats {
		out[i] = Stats[stat]
	}
	return out
}
