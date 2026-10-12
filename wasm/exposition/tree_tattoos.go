package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func TreeTattooExports(r bind.Registry) {
	r.Type(calculator.TattooCountLine{}, bind.Plain())

	r.Func(ResetTreeNodes, bind.AsPromise())
	r.Func(RemoveAllTreeTattoos, bind.AsPromise())
	r.Func(GetTattooCountLines, bind.AsPromise())
}

func ResetTreeNodes(build *pob.PathOfBuilding) {
	calculator.ResetTree(build)
}

func RemoveAllTreeTattoos(build *pob.PathOfBuilding) {
	calculator.RemoveAllTattoos(build)
}

func GetTattooCountLines(build *pob.PathOfBuilding) []calculator.TattooCountLine {
	return calculator.TattooCountLines(build)
}
