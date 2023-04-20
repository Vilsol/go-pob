package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type GrantedEffectStatSet struct {
	raw2.GrantedEffectStatSet
}

var GrantedEffectStatSets []*GrantedEffectStatSet

func InitializeGrantedEffectStatSets(version string) error {
	return InitHelper(version, "GrantedEffectStatSets", &GrantedEffectStatSets, nil)
}

func (g *GrantedEffectStatSet) GetImplicitStats() []*Stat {
	if g.ImplicitStats == nil {
		return nil
	}

	out := make([]*Stat, len(g.ImplicitStats))
	for i, stat := range g.ImplicitStats {
		out[i] = Stats[stat]
	}
	return out
}

func (g *GrantedEffectStatSet) GetConstantStats() []*Stat {
	if g.ConstantStats == nil {
		return nil
	}

	out := make([]*Stat, len(g.ConstantStats))
	for i, stat := range g.ConstantStats {
		out[i] = Stats[stat]
	}
	return out
}
