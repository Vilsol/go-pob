package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

func MasteryExports(r bind.Registry) {
	r.Type(calculator.MasteryOption{}, bind.Plain())

	r.Func(GetMasteryOptions, bind.AsPromise())
	r.Func(SelectMasteryEffect, bind.AsPromise())
	r.Func(GetMasterySelections)
}

func GetMasteryOptions(build *pob.PathOfBuilding, nodeID int64) []calculator.MasteryOption {
	return calculator.MasteryOptions(build, nodeID)
}

func SelectMasteryEffect(build *pob.PathOfBuilding, nodeID int64, effectID int64, path []int64) error {
	return calculator.SelectMasteryEffect(build, nodeID, effectID, path)
}

func GetMasterySelections(build *pob.PathOfBuilding) map[int64]int64 {
	out := map[int64]int64{}
	if i := build.ActiveSpecIndex(); i >= 0 {
		for _, s := range pob.SpecMasterySelections(build.Tree.Specs[i]) {
			out[s.Node] = s.Effect
		}
	}
	return out
}
