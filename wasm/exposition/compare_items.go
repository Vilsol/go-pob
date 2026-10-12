package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
)

//crystalline:exports
func CompareItemsExports(r bind.Registry) {
	r.Type(calculator.CompareJewelSlot{}, bind.Plain())
	r.Type(calculator.CompareSkillBreakdownRow{}, bind.Plain())

	r.Func(calculator.CompareJewelSlots, bind.AsPromise())
	r.Func(calculator.CompareItemExpanded, bind.AsPromise())
	r.Func(calculator.NewCompareJewelPowerBuilder)
}
