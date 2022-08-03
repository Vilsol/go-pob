package raw

type GrantedEffectStatSetsPerLevel struct {
	AdditionalBooleanStats []int     `json:"AdditionalFlags"`
	AdditionalStats        []int     `json:"AdditionalStats"`
	AdditionalStatsValues  []int     `json:"AdditionalStatsValues"`
	AttackCritChance       int       `json:"SpellCritChance"`
	BaseMultiplier         int       `json:"BaseMultiplier"`
	BaseResolvedValues     []int     `json:"BaseResolvedValues"`
	DamageEffectiveness    int       `json:"DamageEffectiveness"`
	FloatStats             []int     `json:"FloatStats"`
	FloatStatsValues       []float64 `json:"FloatStatsValues"`
	GemLevel               int       `json:"GemLevel"`
	GrantedEffects         []int     `json:"GrantedEffects"`
	InterpolationBases     []int     `json:"InterpolationBases"`
	PlayerLevelReq         int       `json:"PlayerLevelReq"`
	OffhandCritChance      int       `json:"AttackCritChance"`
	StatInterpolations     []int     `json:"StatInterpolations"`
	StatSet                int       `json:"StatSet"`
	Key                    int       `json:"_key"`
}

var GrantedEffectStatSetsPerLevels []*GrantedEffectStatSetsPerLevel
var GrantedEffectStatSetsPerLevelsMap map[int]*GrantedEffectStatSetsPerLevel

var grantedEffectStatSetsPerLevelsByIDMap map[int]map[int]*GrantedEffectStatSetsPerLevel

func InitializeGrantedEffectStatSetsPerLevels(version string) error {
	if err := InitHelper(version, "GrantedEffectStatSetsPerLevel", &GrantedEffectStatSetsPerLevels); err != nil {
		return err
	}

	GrantedEffectStatSetsPerLevelsMap = make(map[int]*GrantedEffectStatSetsPerLevel)
	for _, i := range GrantedEffectStatSetsPerLevels {
		GrantedEffectStatSetsPerLevelsMap[i.Key] = i
	}

	grantedEffectStatSetsPerLevelsByIDMap = make(map[int]map[int]*GrantedEffectStatSetsPerLevel)
	for _, i := range GrantedEffectStatSetsPerLevels {
		for _, effect := range i.GrantedEffects {
			if _, ok := grantedEffectStatSetsPerLevelsByIDMap[effect]; !ok {
				grantedEffectStatSetsPerLevelsByIDMap[effect] = make(map[int]*GrantedEffectStatSetsPerLevel)
			}

			grantedEffectStatSetsPerLevelsByIDMap[effect][i.GemLevel] = i
		}
	}

	return nil
}

func (g *GrantedEffectStatSetsPerLevel) GetFloatStats() []*Stat {
	if g.FloatStats == nil {
		return nil
	}

	out := make([]*Stat, len(g.FloatStats))
	for i, stat := range g.FloatStats {
		out[i] = StatsMap[stat]
	}
	return out
}

func (g *GrantedEffectStatSetsPerLevel) GetAdditionalStats() []*Stat {
	if g.AdditionalStats == nil {
		return nil
	}

	out := make([]*Stat, len(g.AdditionalStats))
	for i, stat := range g.AdditionalStats {
		out[i] = StatsMap[stat]
	}
	return out
}

func (g *GrantedEffectStatSetsPerLevel) GetAdditionalBooleanStats() []*Stat {
	if g.AdditionalBooleanStats == nil {
		return nil
	}

	out := make([]*Stat, len(g.AdditionalBooleanStats))
	for i, stat := range g.AdditionalBooleanStats {
		out[i] = StatsMap[stat]
	}
	return out
}
