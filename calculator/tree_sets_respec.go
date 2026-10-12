package calculator

import (
	"strconv"

	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

type TreeSpecRespec struct {
	Gold       int
	Passives   int
	Ascendancy int
}

//pob:block:src/Classes/TreeTab.lua:69-97
func TreeSpecRespecCost(build *pob.PathOfBuilding, index int) TreeSpecRespec {
	active := build.ActiveSpecIndex()
	if index < 0 || index >= len(build.Tree.Specs) || active < 0 || index == active {
		return TreeSpecRespec{}
	}
	cur, other := build.Tree.Specs[active], build.Tree.Specs[index]
	if cur.ClassID != other.ClassID {
		return TreeSpecRespec{}
	}
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	clusters := clusterNodeMap(data.LatestTreeVersion)
	target := map[int64]bool{}
	for _, id := range pob.SpecNodes(other) {
		target[id] = true
	}
	out := TreeSpecRespec{}
	seen := map[int64]bool{}
	for _, id := range pob.SpecNodes(cur) {
		node, ok := tree.Nodes[strconv.FormatInt(id, 10)]
		if seen[id] || !ok {
			continue
		}
		seen[id] = true
		if isClassStartNode(node) || isAscendClassStartNode(node) || id >= 65536 || target[id] {
			continue
		}
		if node.Name != nil {
			if _, cluster := clusters[*node.Name]; cluster && !(node.IsKeystone != nil && *node.IsKeystone) && !(node.IsJewelSocket != nil && *node.IsJewelSocket) {
				continue
			}
		}
		if node.AscendancyName != nil && *node.AscendancyName != "" {
			out.Ascendancy++
		} else {
			out.Passives++
		}
	}
	if out.Passives == 0 && out.Ascendancy == 0 {
		return TreeSpecRespec{}
	}
	prices := raw2.GoldRespecPrices()
	if level := build.Build.Level; level >= 1 && level <= len(prices) {
		cost := prices[level-1]
		out.Gold = out.Passives*cost + out.Ascendancy*cost*5
	}
	return out
}
