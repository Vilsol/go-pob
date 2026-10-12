package exposition

import (
	"time"

	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
)

//crystalline:exports
func PowerReportExports(r bind.Registry) {
	r.Type(calculator.PowerStat{}, bind.Plain())
	r.Type(calculator.PowerReportEntry{}, bind.Plain())
	r.Type(PowerBuilder{})

	r.Func(PowerStats)
	r.Func(NodePowerStats)
	r.Func(NewPowerBuilder)
}

func PowerStats() []calculator.PowerStat {
	return calculator.PowerStats()
}

func NodePowerStats() []calculator.PowerStat {
	return calculator.NodePowerStats()
}

type PowerBuilder struct {
	b *calculator.PowerBuilder
}

func NewPowerBuilder(m *MiscCalculator, stat string, maxDepth int) *PowerBuilder {
	for _, s := range calculator.NodePowerStats() {
		if s.Stat == stat {
			return &PowerBuilder{b: m.calc.NewPowerBuilder(s, maxDepth)}
		}
	}
	return nil
}

func (p *PowerBuilder) Step(budgetMs int) float64 {
	return p.b.Step(time.Duration(budgetMs) * time.Millisecond)
}

func (p *PowerBuilder) Current() int64 {
	return p.b.Current()
}

func (p *PowerBuilder) Report() []calculator.PowerReportEntry {
	return p.b.Report()
}
