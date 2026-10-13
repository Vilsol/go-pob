package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
)

//crystalline:exports
func TreeConqueredNodesExports(r bind.Registry) {
	r.Type(calculator.SpecNodeRewrite{}, bind.Plain())

	r.Func(TreeConqueredNodes)
}

func TreeConqueredNodes(m *MiscCalculator) []calculator.SpecNodeRewrite {
	return calculator.SpecNodeRewrites(m.calc.Base())
}
