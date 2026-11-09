package exposition

import (
	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/data/tree2"
)

// NodeCompareBlockView is one stat difference block of a node tooltip: its header, the node count its per-point
// values divide by (0 for none) and that calculation's outputs.
type NodeCompareBlockView struct {
	Header     string
	PathLength int
	Compare    CompareOutputs
}

// NodeCompareView is what Build:AddStatComparesToTooltip compares for a node: the calculator's base pass, the main
// skill's flags and the minion's, the node's blocks, and the line shown when no block has a difference.
type NodeCompareView struct {
	Base             CompareOutputs
	SkillFlags       map[string]bool
	MinionSkillFlags map[string]bool
	Minion           bool
	Blocks           []NodeCompareBlockView
	NoChange         string
}

// NodeTipView is the part of a node tooltip from the spec: the pathing lines, the distance a socket shows, and
// whether an item grants the node.
type NodeTipView struct {
	PathLines       []string
	DistanceToStart int
	Granted         bool
}

// TraceView is the trace path after a Shift hover, and whether the hovered node is highlighted.
type TraceView struct {
	Trace []int
	Hover bool
}

// specNodes is PassiveTreeView.tracePath: nil out of trace mode, else the nodes (none when a trace starts).
func (c *Calculator) specNodes(ids []int, tracing bool) ([]*tree2.SpecNode, error) {
	if !tracing {
		return nil, nil
	}
	spec, err := c.spec()
	if err != nil {
		return nil, err
	}
	out := make([]*tree2.SpecNode, 0, len(ids))
	for _, id := range ids {
		n, ok := spec.Nodes[id]
		if !ok {
			return nil, unknownNodeError(id)
		}
		out = append(out, n)
	}
	return out, nil
}

// NodeCompare is the node's tooltip stat differences over the last calculation; trace is the Shift-traced path, while
// tracing.
func (c *Calculator) NodeCompare(id int, trace []int, tracing bool) (*NodeCompareView, error) {
	misc, err := c.miscCalculator()
	if err != nil {
		return nil, err
	}
	node, ok := c.env.Spec.Nodes[id]
	if !ok {
		return nil, unknownNodeError(id)
	}
	tracePath, err := c.specNodes(trace, tracing)
	if err != nil {
		return nil, err
	}
	cmp := calculator2.NodeCompares(misc, node, tracePath)
	view := &NodeCompareView{Base: compareOutputs(misc.Base), SkillFlags: statSetFlags(c.env.Player.MainSkill),
		Blocks: []NodeCompareBlockView{}, NoChange: cmp.NoChange}
	if main := c.env.Player.MainSkill; main != nil && main.Minion != nil {
		view.Minion = true
		view.MinionSkillFlags = statSetFlags(main.Minion.MainSkill)
	}
	for _, b := range cmp.Blocks {
		view.Blocks = append(view.Blocks, NodeCompareBlockView{Header: b.Header, PathLength: b.PathLength, Compare: compareOutputs(b.Env)})
	}
	return view, nil
}

// NodeTip is the node's pathing lines and socket distance; trace is the Shift-traced path, while tracing.
func (c *Calculator) NodeTip(id int, trace []int, tracing bool) (*NodeTipView, error) {
	spec, err := c.spec()
	if err != nil {
		return nil, err
	}
	node, ok := spec.Nodes[id]
	if !ok {
		return nil, unknownNodeError(id)
	}
	tracePath, err := c.specNodes(trace, tracing)
	if err != nil {
		return nil, err
	}
	return &NodeTipView{
		PathLines:       calculator2.NodePathLines(spec, node, tracePath, c.build.Build.Level),
		DistanceToStart: node.DistanceToClassStart,
		Granted:         c.env.GrantedPassives[id],
	}, nil
}

// TraceNode is one hover while Shift is held: the trace path so far (empty to start one) and the hovered node.
func (c *Calculator) TraceNode(trace []int, id int) (*TraceView, error) {
	spec, err := c.spec()
	if err != nil {
		return nil, err
	}
	tracePath, err := c.specNodes(trace, true)
	if err != nil {
		return nil, err
	}
	next, hover := calculator2.TraceStep(tracePath, spec.Nodes[id])
	ids := []int{}
	for _, n := range next {
		ids = append(ids, n.ID)
	}
	return &TraceView{Trace: ids, Hover: hover != nil}, nil
}
