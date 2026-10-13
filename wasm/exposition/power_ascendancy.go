package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
)

//crystalline:exports
func PowerAscendancyExports(r bind.Registry) {
	r.Func(PowerAscendancy)
}

func PowerAscendancy(p *PowerBuilder) []calculator.PowerReportEntry {
	return p.b.AscendancyPower()
}
