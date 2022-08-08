package raw

type GrantedEffectQualityStat struct {
	GrantedEffectsKey   int   `json:"GrantedEffectsKey"`
	SetID               int   `json:"SetId"`
	StatsKeys           []int `json:"StatsKeys"`
	StatsValuesPermille []int `json:"StatsValuesPermille"`
	Weight              int   `json:"Weight"`
	Key                 int   `json:"_key"`
}

var GrantedEffectQualityStats []*GrantedEffectQualityStat

var grantedEffectQualityStatsByIDMap map[int]map[int]*GrantedEffectQualityStat

func InitializeGrantedEffectQualityStats(version string) error {
	if err := InitHelper(version, "GrantedEffectQualityStats", &GrantedEffectQualityStats); err != nil {
		return err
	}

	grantedEffectQualityStatsByIDMap = make(map[int]map[int]*GrantedEffectQualityStat)
	for _, i := range GrantedEffectQualityStats {
		if _, ok := grantedEffectQualityStatsByIDMap[i.GrantedEffectsKey]; !ok {
			grantedEffectQualityStatsByIDMap[i.GrantedEffectsKey] = make(map[int]*GrantedEffectQualityStat)
		}

		grantedEffectQualityStatsByIDMap[i.GrantedEffectsKey][i.SetID] = i
	}

	return nil
}

func (s *GrantedEffectQualityStat) GetStats() []*Stat {
	if s.StatsKeys == nil {
		return nil
	}

	out := make([]*Stat, len(s.StatsKeys))
	for i, key := range s.StatsKeys {
		out[i] = Stats[key]
	}
	return out
}
