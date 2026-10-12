package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

func TreeTooltipExports(r bind.Registry) {
	r.Type(calculator.TreeCompareSpec{}, bind.Plain())
	r.Type(calculator.TreeCompareMastery{}, bind.Plain())

	r.Func(NodeSkillTooltips)
	r.Func(GoldRespecPrice)
	r.Func(TreeSpecCompare)
	r.Func(LeapHeldNodes, bind.AsPromise())
}

func NodeSkillTooltips(build *pob.PathOfBuilding, nodeID int64, stats calculator.ItemTooltipStats) [][]calculator.TooltipLine {
	return calculator.NodeSkillTooltips(build, nodeID, calculator.ItemTooltipContext{ShowFlavourText: true, Stats: stats})
}

func GoldRespecPrice(build *pob.PathOfBuilding) int {
	return calculator.GoldRespecPrice(build)
}

func TreeSpecCompare(build *pob.PathOfBuilding, specID int) calculator.TreeCompareSpec {
	return calculator.TreeSpecCompare(build, specID)
}

func (m *MiscCalculator) SplitPersonalityPath() []int64 {
	return m.calc.SplitPersonalityPath()
}

func (m *MiscCalculator) SocketDistanceToClassStart(nodeID int) int {
	return m.calc.SocketDistanceToClassStart(nodeID)
}

func LeapHeldNodes(build *pob.PathOfBuilding) []int64 {
	return calculator.LeapHeldNodes(build)
}
