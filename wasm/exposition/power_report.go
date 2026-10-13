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
	r.Type(calculator.PowerShardRows{}, bind.Plain())
	r.Type(PowerBuilder{})

	r.Func(PowerStats)
	r.Func(NodePowerStats)
	r.Func(NewPowerBuilder)
	r.Func(NewPowerBuilderShard)
	r.Func(MergePowerReports)
	r.Func(MergePowerAscendancy)
}

func PowerStats() []calculator.PowerStat {
	return calculator.PowerStats()
}

func NodePowerStats() []calculator.PowerStat {
	return calculator.NodePowerStats()
}

type PowerBuilder struct {
	b     *calculator.PowerBuilder
	owned []int64
}

func NewPowerBuilder(m *MiscCalculator, stat string, maxDepth int) *PowerBuilder {
	for _, s := range calculator.NodePowerStats() {
		if s.Stat == stat {
			return &PowerBuilder{b: m.calc.NewPowerBuilder(s, maxDepth)}
		}
	}
	return nil
}

func NewPowerBuilderShard(m *MiscCalculator, stat string, maxDepth int, index int, count int) *PowerBuilder {
	p := NewPowerBuilder(m, stat, maxDepth)
	if p != nil {
		p.owned = p.b.Shard(index, count)
	}
	return p
}

func MergePowerReports(shards []calculator.PowerShardRows) []calculator.PowerReportEntry {
	return calculator.MergePowerReports(shards)
}

func MergePowerAscendancy(shards []calculator.PowerShardRows) []calculator.PowerReportEntry {
	return calculator.MergePowerAscendancy(shards)
}

func (p *PowerBuilder) Shard() calculator.PowerShardRows {
	return calculator.PowerShardRows{Rows: p.b.Report(), Owned: p.owned, Ascendancy: p.b.AscendancyPower()}
}

func (p *PowerBuilder) Tasks() int {
	return p.b.Tasks()
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
