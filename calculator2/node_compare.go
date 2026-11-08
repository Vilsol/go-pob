package calculator2

import (
	"slices"
	"strconv"

	"github.com/Vilsol/go-pob/data/tree2"
)

// NodeCompareBlock is one stat difference block of a node tooltip: its header without colour codes, the calculation,
// and PathLength, the node count its per-point values divide by (0 for none).
type NodeCompareBlock struct {
	Header     string
	PathLength int
	Env        *Environment
}

// NodeCompare is a node tooltip's stat differences: its blocks, and the line shown in their place when no block has a
// difference.
type NodeCompare struct {
	Blocks   []NodeCompareBlock
	NoChange string
}

// NodeCompares is AddNodeTooltip's mod differences for node, tracePath being the Shift-traced path (nil for none).
//
//pob2:block:src/Classes/PassiveTreeView.lua:1967-2004
func NodeCompares(m *MiscCalculator, node *tree2.SpecNode, tracePath []*tree2.SpecNode) NodeCompare {
	var path []*tree2.SpecNode
	switch {
	case node.Alloc:
		path = node.Depends
	case tracePath != nil:
		path = tracePath
	default:
		path = node.Path
	}
	pathLength := len(path)
	pathNodes := make(map[*tree2.SpecNode]bool, pathLength)
	for _, n := range path {
		pathNodes[n] = true
	}
	var nodeOutput, pathOutput *Environment
	isGranted := m.main.GrantedPassives[node.ID]
	realloc := false
	switch {
	case node.Alloc:
		nodeOutput = m.CalcNodes(NodeOverride{RemoveNodes: map[*tree2.SpecNode]bool{node: true}})
		if pathLength > 1 {
			pathOutput = m.CalcNodes(NodeOverride{RemoveNodes: pathNodes})
		}
	case isGranted:
		nodeOutput = m.CalcNodes(NodeOverride{RemoveNodeIDs: map[int]bool{node.ID: true}})
	default:
		nodeOutput = m.CalcNodes(NodeOverride{AddNodes: map[*tree2.SpecNode]bool{node: true}})
		if pathLength > 1 {
			pathOutput = m.CalcNodes(NodeOverride{AddNodes: pathNodes})
		}
	}
	var header string
	switch {
	case realloc:
		header = "Reallocating this node will give you:"
	case node.Alloc:
		header = "Unallocating this node will give you:"
	case isGranted:
		header = "This node is granted by an item. Removing it will give you:"
	default:
		header = "Allocating this node will give you:"
	}
	out := NodeCompare{Blocks: []NodeCompareBlock{{Header: header, Env: nodeOutput}}}
	if pathLength > 1 && !isGranted && (len(node.IntuitiveLeapLikesAffecting) == 0 || node.Alloc) {
		header = "Allocating this node and all nodes leading to it will give you:"
		if node.Alloc {
			header = "Unallocating this node and all nodes depending on it will give you:"
		}
		out.Blocks = append(out.Blocks, NodeCompareBlock{Header: header, PathLength: pathLength, Env: pathOutput})
	}
	if isGranted {
		out.NoChange = "This node is granted by an item. Removing it will cause no changes"
	} else {
		verb := "allocating"
		if node.Alloc {
			verb = "unallocating"
		}
		// the Lua's " or the nodes leading to it" needs intuitiveLeapLikesAffecting (a table) == 0, which never holds
		out.NoChange = "No changes from " + verb + " this node."
	}
	return out
}

// NodePathLines is AddNodeTooltip's pathing distance and refund lines without colour codes, tracePath being the
// Shift-traced path (nil out of trace mode). The gold lines are left out: data.goldRespecPrices is not ported.
//
//pob2:block:src/Classes/PassiveTreeView.lua:2011-2025
//pob2:block:src/Classes/PassiveTreeView.lua:2030-2053
func NodePathLines(spec *tree2.PassiveSpec, node *tree2.SpecNode, tracePath []*tree2.SpecNode) []string {
	var lines []string
	if node.HasPath && len(node.Path) > 0 {
		if tracePath != nil && slices.Contains(tracePath, node) {
			lines = append(lines, strconv.Itoa(len(tracePath))+" nodes in trace path")
		} else {
			line := strconv.Itoa(node.PathDist) + " points to node"
			if len(node.IntuitiveLeapLikesAffecting) > 0 {
				line += " (Can be allocated without pathing to it)"
			}
			lines = append(lines, line)
			if len(node.Path) > 1 {
				lines = append(lines, "Tip: To reach this node by a different path, hold Shift, then trace the path and click this node")
			}
		}
	}
	if len(node.Depends) > 1 {
		dependCount := len(node.Depends)
		for _, depNode := range node.Depends {
			if depNode.UnlockConstraint == nil {
				continue
			}
			for _, reqNodeID := range depNode.UnlockConstraint.Nodes {
				reqNode := spec.Nodes[reqNodeID]
				if reqNode == node && !depNode.Alloc {
					dependCount--
					break
				}
				if reqNode != nil && !reqNode.Alloc {
					dependCount--
					break
				}
			}
		}
		if dependCount > 0 {
			lines = append(lines, strconv.Itoa(dependCount)+" points gained from unallocating these nodes")
		}
	}
	return lines
}

// TraceStep is one hover in Shift path tracing: the trace path after hovering node, and the node it highlights (nil
// for one it cannot reach from the path's end).
//
//pob2:block:src/Classes/PassiveTreeView.lua:321-350
func TraceStep(tracePath []*tree2.SpecNode, node *tree2.SpecNode) ([]*tree2.SpecNode, *tree2.SpecNode) {
	switch {
	case node == nil:
	case !node.HasPath:
		node = nil
	case len(tracePath) == 0:
		for _, pathNode := range node.Path {
			tracePath = slices.Insert(tracePath, 0, pathNode)
		}
	default:
		lastPathNode := tracePath[len(tracePath)-1]
		if node == lastPathNode {
			break
		}
		if !slices.Contains(node.Linked, lastPathNode) {
			node = nil
			break
		}
		if index := slices.Index(tracePath, node); index >= 0 {
			tracePath = slices.Delete(slices.Clone(tracePath), index, index+1)
		}
		tracePath = append(slices.Clip(tracePath), node)
	}
	return tracePath, node
}
