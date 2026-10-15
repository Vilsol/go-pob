package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func TreeCompareJewelExports(r bind.Registry) {
	r.Func(CompareSpecJewelTooltip)
}

//pob:block:src/Classes/PassiveTreeView.lua:1779-1786
func CompareSpecJewelTooltip(build *pob.PathOfBuilding, specID int, nodeID int, stats calculator.ItemTooltipStats) (calculator.ItemTooltip, bool) {
	if build == nil || specID < 1 || specID > len(build.Tree.Specs) {
		return calculator.ItemTooltip{}, false
	}
	for _, s := range build.Tree.Specs[specID-1].Sockets {
		if s.NodeID != nodeID || s.ItemID == 0 {
			continue
		}
		for _, it := range build.Items.Items {
			if it.ID == s.ItemID {
				ctx := calculator.ItemTooltipContext{ShowFlavourText: showFlavourText, Stats: stats, CharacterLevel: build.Build.Level, ClassName: string(build.Build.ClassName)}
				return calculator.AddItemTooltip(calculator.ParseItem(it.Text), ctx), true
			}
		}
	}
	return calculator.ItemTooltip{}, false
}
