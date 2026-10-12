package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func LoadoutExports(r bind.Registry) {
	r.Type(calculator.LoadoutList{}, bind.Plain())

	r.Func(GetLoadouts, bind.AsPromise())
	r.Func(SelectLoadout, bind.AsPromise())
	r.Func(NewLoadout, bind.AsPromise())
}

func GetLoadouts(build *pob.PathOfBuilding) calculator.LoadoutList {
	return calculator.Loadouts(build)
}

func SelectLoadout(build *pob.PathOfBuilding, name string) bool {
	return calculator.SelectLoadout(build, name)
}

func NewLoadout(build *pob.PathOfBuilding, name string) {
	calculator.NewLoadout(build, name)
}
