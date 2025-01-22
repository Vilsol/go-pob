package exposition

import (
	"github.com/Vilsol/go-pob/calculator"
)

type ElementWrapper struct {
	Elements map[string]calculator.ColProps
}

func SetCalcTabElements(elements *ElementWrapper) {
	calculator.SetColProps(elements.Elements)
}
