package calculator

import (
	"slices"
	"strconv"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

type TreeClusterNode struct {
	ID         int64
	Type       string
	Name       string
	Icon       string
	Stats      []string
	Group      int64
	Orbit      int64
	OrbitIndex int64
	X          float64
	Y          float64
	Linked     []int64
}

//pob:block:src/Classes/PassiveSpec.lua:2225-2268
func TreeClusterNodes(build *pob.PathOfBuilding) []TreeClusterNode {
	nodes, layout := buildClusterJewelLayout(&Environment{Build: build}, data.LatestTreeVersion, true)
	tree := specTree()
	out := make([]TreeClusterNode, 0, len(layout))
	for key, l := range layout {
		id, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			continue
		}
		node, ok := nodes[key]
		if l.mastery != nil {
			node, ok = *l.mastery, true
		}
		if !ok {
			continue
		}
		x, y, _ := tree.OrbitPosition(l.group, l.orbit, l.orbitIndex)
		n := TreeClusterNode{
			ID: id, Type: l.kind, Stats: node.Stats, Group: l.group,
			Orbit: int64(l.orbit), OrbitIndex: int64(l.orbitIndex), X: x, Y: y,
		}
		if node.Name != nil {
			n.Name = *node.Name
		}
		if node.Icon != nil {
			n.Icon = *node.Icon
		}
		for _, other := range node.Out {
			if v, err := strconv.ParseInt(other, 10, 64); err == nil {
				n.Linked = append(n.Linked, v)
			}
		}
		out = append(out, n)
	}
	slices.SortFunc(out, func(a, b TreeClusterNode) int { return int(a.ID - b.ID) })
	return out
}

//pob:block:src/Classes/PassiveTreeView.lua:1102-1107
func TreeSpecClusterNodes(build *pob.PathOfBuilding, specID int) []TreeClusterNode {
	if specID < 1 || specID > len(build.Tree.Specs) {
		return nil
	}
	other := *build
	other.Tree.ActiveSpec = specID
	return TreeClusterNodes(&other)
}
