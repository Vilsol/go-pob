package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
)

func TreeCompare30Exports(r bind.Registry) {
	r.Type(calculator.CompareModRow{}, bind.Plain())
}
