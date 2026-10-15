package calculator

import "github.com/Vilsol/go-pob/pob"

//pob:block:src/Classes/PassiveMasteryControl.lua:50-56,src/Classes/PassiveTreeView.lua:1695-1718
func (m *MiscCalculator) CalcMasteryEffect(nodeID int64, effect int64, addNodes []int64) *Environment {
	return m.over(func(b *pob.PathOfBuilding) { b.SetMasteryEffect(nodeID, effect) }).Calc(&CalcOverride{AddNodes: addNodes})
}
