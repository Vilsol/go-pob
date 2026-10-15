package calculator

import (
	"slices"

	"github.com/Vilsol/go-pob/pob"
)

func (m *MiscCalculator) over(change func(b *pob.PathOfBuilding)) *MiscCalculator {
	b := *m.build
	b.Tree.Specs = slices.Clone(m.build.Tree.Specs)
	change(&b)
	return &MiscCalculator{build: &b, cache: m.cache, base: m.base}
}

//pob:block:src/Classes/TreeTab.lua:67-72
func (m *MiscCalculator) CalcSpec(specID int) *Environment {
	return m.over(func(b *pob.PathOfBuilding) { b.SetActiveSpec(specID) }).Calc(&CalcOverride{})
}
