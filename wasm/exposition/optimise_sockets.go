package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func OptimiseSocketsExports(r bind.Registry) {
	r.Func(OptimiseSocketsUseful, bind.AsPromise())
}

func OptimiseSocketsUseful(build *pob.PathOfBuilding, slotName string) bool {
	return calculator.OptimiseSocketsUseful(build, slotName)
}
