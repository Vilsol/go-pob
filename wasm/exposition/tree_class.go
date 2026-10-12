package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

func TreeClassExports(r bind.Registry) {
	r.Type(calculator.ClassChange{}, bind.Plain())

	r.Func(ChangeTreeClass, bind.AsPromise())
	r.Func(ChangeTreeAscendancy, bind.AsPromise())
	r.Func(ClickTreeAscendancyNode, bind.AsPromise())
	r.Func(MoveTreeSpec, bind.AsPromise())
}

func ChangeTreeClass(build *pob.PathOfBuilding, className string, decision int) calculator.ClassChange {
	return calculator.ChangeClass(build, pob.ClassName(className), calculator.ClassDecision(decision))
}

func ChangeTreeAscendancy(build *pob.PathOfBuilding, ascendancy string) {
	calculator.SelectAscendClass(build, pob.AscendancyName(ascendancy))
}

func ClickTreeAscendancyNode(build *pob.PathOfBuilding, nodeID int, decision int) calculator.ClassChange {
	return calculator.ClickAscendancyNode(build, int64(nodeID), calculator.ClassDecision(decision))
}

func MoveTreeSpec(build *pob.PathOfBuilding, from int, to int) {
	build.MoveSpec(from, to)
}
