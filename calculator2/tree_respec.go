package calculator2

import "github.com/Vilsol/go-pob/data/tree2"

// TreeSpecRespec is the tree list tooltip's cost of switching from the active spec to another: the gold, and the
// passive and ascendancy nodes refunded.
type TreeSpecRespec struct {
	Gold       float64
	Passives   int
	Ascendancy int
}

// TreeSpecRespecCost is the respec cost of switching from active to spec at characterLevel; zero when their classes
// differ or nothing is refunded.
//
//pob2:block:src/Classes/TreeTab.lua:85-102
func TreeSpecRespecCost(active, spec *tree2.PassiveSpec, characterLevel int) TreeSpecRespec {
	var out TreeSpecRespec
	if spec.CurClassID != active.CurClassID {
		return out
	}
	for nodeID, node := range active.AllocNodes {
		// Assumption: Nodes >= 65536 are small cluster passives.
		if node.Type != "ClassStart" && node.Type != "AscendClassStart" &&
			(active.Tree.ClusterNodeMap[node.Dn] == nil || node.IsKeystone || node.IsJewelSocket) && nodeID < 65536 &&
			spec.AllocNodes[nodeID] == nil {
			if node.AscendancyName != "" {
				out.Ascendancy++
			} else {
				out.Passives++
			}
		}
	}
	if out.Passives > 0 || out.Ascendancy > 0 {
		if goldCost, ok := goldRespecPrice(characterLevel); ok {
			out.Gold = float64(float64(out.Passives)*goldCost) + float64(float64(out.Ascendancy)*goldCost*5)
		}
	}
	return out
}
