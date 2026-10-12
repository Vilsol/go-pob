package calculator

import (
	"slices"
	"strconv"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

//pob:block:src/Classes/ItemsTab.lua:2463-2483
func IsJewelValidForSocket(item *Item, nodeID int) bool {
	node, ok := data.TreeVersions[data.LatestTreeVersion].Tree().Nodes[strconv.Itoa(nodeID)]
	if !ok || item == nil || item.Base == nil || item.Type != "Jewel" {
		return false
	}
	charmSocket := node.Name != nil && *node.Name == "Charm Socket"
	cluster, isCluster := data.ClusterJewels.Jewels[item.BaseName]
	switch {
	case charmSocket || item.Base.SubType == "Charm":
		return charmSocket && item.Base.SubType == "Charm"
	case isCluster && node.ExpansionJewel == nil:
		return false
	case node.ExpansionJewel == nil || node.ExpansionJewel.Size == 2:
		return true
	default:
		return !isCluster || int64(cluster.SizeIndex) <= node.ExpansionJewel.Size
	}
}

//pob:block:src/Classes/ItemSlotControl.lua:85-93
func TreeSocketItemIDs(build *pob.PathOfBuilding, nodeID int) []int {
	var ids []int
	for _, it := range build.Items.Items {
		if IsJewelValidForSocket(parseItemGlobal(it.Text), nodeID) {
			ids = append(ids, it.ID)
		}
	}
	slices.Sort(ids)
	return ids
}

//pob:block:src/Classes/ItemSlotControl.lua:12-19,src/Classes/ItemSlotControl.lua:61-68
func SocketTreeJewel(build *pob.PathOfBuilding, nodeID int, itemID int) bool {
	if itemID == build.SocketedJewel(nodeID) || (itemID != 0 && !slices.Contains(TreeSocketItemIDs(build, nodeID), itemID)) {
		return false
	}
	build.SocketJewel(nodeID, itemID)
	return true
}
