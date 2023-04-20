package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type GrantedEffectsPerLevel struct {
	raw2.GrantedEffectsPerLevel
}

var GrantedEffectsPerLevels []*GrantedEffectsPerLevel

var grantedEffectsPerLevelsByIDMap map[int]map[int]*GrantedEffectsPerLevel

func InitializeGrantedEffectsPerLevels(version string) error {
	return InitHelper(version, "GrantedEffectsPerLevel", &GrantedEffectsPerLevels, func(count int64) {
		grantedEffectsPerLevelsByIDMap = make(map[int]map[int]*GrantedEffectsPerLevel, count)
	}, func(obj *GrantedEffectsPerLevel) {
		if _, ok := grantedEffectsPerLevelsByIDMap[obj.GrantedEffect]; !ok {
			grantedEffectsPerLevelsByIDMap[obj.GrantedEffect] = make(map[int]*GrantedEffectsPerLevel)
		}

		grantedEffectsPerLevelsByIDMap[obj.GrantedEffect][obj.Level] = obj
	})
}

func (l *GrantedEffectsPerLevel) GetCostTypes() []*CostType {
	if l.CostTypes == nil {
		return nil
	}

	out := make([]*CostType, len(l.CostTypes))
	for i, costType := range l.CostTypes {
		out[i] = CostTypes[costType]
	}
	return out
}
