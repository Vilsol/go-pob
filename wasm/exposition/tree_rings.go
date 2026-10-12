package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

func TreeRingExports(r bind.Registry) {
	r.Type(calculator.TreeJewelRing{}, bind.Plain())
	r.Type(calculator.TreeSocketRadius{}, bind.Plain())
	r.Type(calculator.TreeJewelOverlay{}, bind.Plain())
	r.Type(calculator.TreeClusterNode{}, bind.Plain())

	r.Func(GetTreeJewelRings, bind.AsPromise())
	r.Func(GetTreeSocketRadii, bind.AsPromise())
	r.Func(GetTreeJewelOverlays, bind.AsPromise())
	r.Func(GetTreeClusterNodes, bind.AsPromise())
	r.Func(GetTreeSpecClusterNodes, bind.AsPromise())
}

func GetTreeJewelRings(build *pob.PathOfBuilding, compared []int64) []calculator.TreeJewelRing {
	return calculator.TreeJewelRings(build, compared)
}

func GetTreeSocketRadii(build *pob.PathOfBuilding, nodeID int) []calculator.TreeSocketRadius {
	return calculator.TreeSocketRadii(build, int64(nodeID))
}

func GetTreeJewelOverlays(build *pob.PathOfBuilding) []calculator.TreeJewelOverlay {
	return calculator.TreeJewelOverlays(build)
}

func GetTreeClusterNodes(build *pob.PathOfBuilding) []calculator.TreeClusterNode {
	return calculator.TreeClusterNodes(build)
}

func GetTreeSpecClusterNodes(build *pob.PathOfBuilding, specID int) []calculator.TreeClusterNode {
	return calculator.TreeSpecClusterNodes(build, specID)
}
