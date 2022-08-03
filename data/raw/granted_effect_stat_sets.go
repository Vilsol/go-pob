package raw

type GrantedEffectStatSet struct {
	Key                      int     `json:"_key"`
	ID                       string  `json:"Id"`
	ImplicitStats            []int   `json:"ImplicitStats"`
	ConstantStats            []int   `json:"ConstantStats"`
	ConstantStatsValues      []int   `json:"ConstantStatsValues"`
	BaseEffectiveness        float64 `json:"BaseEffectiveness"`
	IncrementalEffectiveness float64 `json:"IncrementalEffectiveness"`
}

var GrantedEffectStatSets []*GrantedEffectStatSet
var GrantedEffectStatSetsMap map[int]*GrantedEffectStatSet

func InitializeGrantedEffectStatSets(version string) error {
	if err := InitHelper(version, "GrantedEffectStatSets", &GrantedEffectStatSets); err != nil {
		return err
	}

	GrantedEffectStatSetsMap = make(map[int]*GrantedEffectStatSet)
	for _, i := range GrantedEffectStatSets {
		GrantedEffectStatSetsMap[i.Key] = i
	}

	return nil
}

func (g *GrantedEffectStatSet) GetImplicitStats() []*Stat {
	if g.ImplicitStats == nil {
		return nil
	}

	out := make([]*Stat, len(g.ImplicitStats))
	for i, stat := range g.ImplicitStats {
		out[i] = StatsMap[stat]
	}
	return out
}

func (g *GrantedEffectStatSet) GetConstantStats() []*Stat {
	if g.ConstantStats == nil {
		return nil
	}

	out := make([]*Stat, len(g.ConstantStats))
	for i, stat := range g.ConstantStats {
		out[i] = StatsMap[stat]
	}
	return out
}
