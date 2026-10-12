package exposition

import (
	"github.com/Vilsol/crystalline/bind"
)

//crystalline:exports
func TreeMasteryExports(r bind.Registry) {
	r.Func(CalcMasteryEffect, bind.AsPromise())
	r.Func(CalcTreeSpec, bind.AsPromise())
}

func CalcMasteryEffect(m *MiscCalculator, nodeID int, effect int, addNodes []int64) StatOutputs {
	return statOutputs(m.calc.CalcMasteryEffect(int64(nodeID), int64(effect), addNodes))
}

func CalcTreeSpec(m *MiscCalculator, specID int) StatOutputs {
	return statOutputs(m.calc.CalcSpec(specID))
}
