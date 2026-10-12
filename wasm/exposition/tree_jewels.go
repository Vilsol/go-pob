package exposition

import (
	"fmt"
	"slices"

	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

func TreeJewelExports(r bind.Registry) {
	r.Type(TreeJewel{}, bind.Plain())

	r.Func(TreeSocketChoices, bind.AsPromise())
	r.Func(TreeSocketedJewels, bind.AsPromise())
	r.Func(SocketTreeJewel, bind.AsPromise())
}

type TreeJewel struct {
	NodeID int
	ItemID int
	Name   string
	Rarity string
}

func treeJewel(build *pob.PathOfBuilding, nodeID int, itemID int) (TreeJewel, bool) {
	for _, it := range build.Items.Items {
		if it.ID == itemID {
			item := calculator.PeekItem(it.Text)
			return TreeJewel{NodeID: nodeID, ItemID: itemID, Name: item.Name, Rarity: string(item.Rarity)}, true
		}
	}
	return TreeJewel{}, false
}

func TreeSocketChoices(build *pob.PathOfBuilding, nodeID int) []TreeJewel {
	var out []TreeJewel
	for _, id := range calculator.TreeSocketItemIDs(build, nodeID) {
		if j, ok := treeJewel(build, 0, id); ok {
			out = append(out, j)
		}
	}
	return out
}

func TreeSocketedJewels(build *pob.PathOfBuilding) []TreeJewel {
	var out []TreeJewel
	if i := build.ActiveSpecIndex(); i >= 0 {
		for _, s := range build.Tree.Specs[i].Sockets {
			if j, ok := treeJewel(build, s.NodeID, s.ItemID); ok {
				out = append(out, j)
			}
		}
	}
	slices.SortFunc(out, func(a, b TreeJewel) int { return a.NodeID - b.NodeID })
	return out
}

func SocketTreeJewel(build *pob.PathOfBuilding, nodeID int, itemID int) (bool, error) {
	if itemID != 0 && !slices.Contains(calculator.TreeSocketItemIDs(build, nodeID), itemID) {
		return false, fmt.Errorf("item %d does not fit socket %d", itemID, nodeID)
	}
	return calculator.SocketTreeJewel(build, nodeID, itemID), nil
}
