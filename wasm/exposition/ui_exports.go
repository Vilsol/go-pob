package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
)

//crystalline:exports
func Uigo2Exports(r bind.Registry) {
	r.Func(calculator.SetDefaultItemAffixQuality, bind.AsPromise())
}
