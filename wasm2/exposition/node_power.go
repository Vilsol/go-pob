package exposition

import (
	"slices"

	"github.com/Vilsol/go-pob/calculator2"
)

// PowerStatView is a heat map stat choice: TreeTab's powerStatList, data.powerStatList without the item-only entries.
// Stat is "" for the combined Offence/Defence entry.
type PowerStatView struct {
	Stat  string
	Label string
}

// PowerProgress is how far the power builder has got: Done of Tasks nodes, Total being the nodes within the depth.
type PowerProgress struct {
	Done     int
	Tasks    int
	Total    int
	Finished bool
}

// NodePowerView is every node's power as flat lists by node, for the heat map: Distance always; Single and Path where
// HasSingle / HasPath; Offence and Defence for the combined power. Max is calcsTab.powerMax as [singleStat, offence,
// offencePerPoint, defence, defencePerPoint].
type NodePowerView struct {
	IDs       []int
	Distance  []int
	Single    []float64
	Path      []float64
	Offence   []float64
	Defence   []float64
	HasSingle []bool
	HasPath   []bool
	Max       []float64
}

// powerRun is the power builder over the last calculation; a worker holds one build, so one is kept.
var powerRun struct {
	env     *calculator2.Environment
	builder *calculator2.PowerBuilder
}

type powerError string

func (e powerError) Error() string { return string(e) }

//pob2:block:src/Classes/TreeTab.lua:260-265
func PowerStats() []PowerStatView {
	out := []PowerStatView{}
	for _, s := range calculator2.PowerStatList {
		if !s.IgnoreForNodes {
			out = append(out, PowerStatView{Stat: s.Stat, Label: s.Label})
		}
	}
	return out
}

// StartNodePower starts the power builder for stat ("" for Offence/Defence) out to maxDepth (0 for all) over the last
// calculation, dropping a run in progress.
//
//pob2:block:src/Classes/TreeTab.lua:924-930
func (c *Calculator) StartNodePower(stat string, maxDepth int) (*PowerProgress, error) {
	misc, err := c.miscCalculator()
	if err != nil {
		return nil, err
	}
	var powerStat *calculator2.PowerStat
	for i := range calculator2.PowerStatList {
		s := &calculator2.PowerStatList[i]
		if !s.IgnoreForNodes && s.Stat == stat {
			powerStat = s
			break
		}
	}
	if powerStat == nil {
		return nil, powerError("no such power stat: " + stat)
	}
	powerRun.env, powerRun.builder = c.env, calculator2.NewPowerBuilder(misc, powerStat, maxDepth)
	return c.nodePowerProgress()
}

func (c *Calculator) nodePowerBuilder() (*calculator2.PowerBuilder, error) {
	if powerRun.builder == nil || powerRun.env != c.env {
		return nil, powerError("the power builder is not running over the last calculation")
	}
	return powerRun.builder, nil
}

func (c *Calculator) nodePowerProgress() (*PowerProgress, error) {
	b, err := c.nodePowerBuilder()
	if err != nil {
		return nil, err
	}
	return &PowerProgress{Done: b.Done(), Tasks: b.Tasks(), Total: b.Total, Finished: b.Done() >= b.Tasks()}, nil
}

// StepNodePower calculates up to n more nodes.
func (c *Calculator) StepNodePower(n int) (*PowerProgress, error) {
	b, err := c.nodePowerBuilder()
	if err != nil {
		return nil, err
	}
	b.Step(n)
	return c.nodePowerProgress()
}

// NodePower is the builder's node powers so far.
func (c *Calculator) NodePower() (*NodePowerView, error) {
	b, err := c.nodePowerBuilder()
	if err != nil {
		return nil, err
	}
	n := len(b.Power)
	view := &NodePowerView{
		IDs: make([]int, 0, n), Distance: make([]int, 0, n), Single: make([]float64, 0, n), Path: make([]float64, 0, n),
		Offence: make([]float64, 0, n), Defence: make([]float64, 0, n), HasSingle: make([]bool, 0, n), HasPath: make([]bool, 0, n),
		Max: []float64{b.Max.SingleStat, b.Max.Offence, b.Max.OffencePerPoint, b.Max.Defence, b.Max.DefencePerPoint},
	}
	ids := make([]int, 0, n)
	for id := range b.Power {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		p := b.Power[id]
		view.IDs = append(view.IDs, id)
		view.Distance = append(view.Distance, p.Distance)
		view.Single = append(view.Single, p.SingleStat)
		view.Path = append(view.Path, p.PathPower)
		view.Offence = append(view.Offence, p.Offence)
		view.Defence = append(view.Defence, p.Defence)
		view.HasSingle = append(view.HasSingle, p.HasSingle)
		view.HasPath = append(view.HasPath, p.HasPath)
	}
	return view, nil
}

// NodePowerReport is the power report's rows, unsorted and before the stat's display scale and format; empty for the
// combined power.
func (c *Calculator) NodePowerReport() ([]calculator2.PowerReportRow, error) {
	b, err := c.nodePowerBuilder()
	if err != nil {
		return nil, err
	}
	rows := b.PowerReportList()
	if rows == nil {
		rows = []calculator2.PowerReportRow{}
	}
	return rows, nil
}
