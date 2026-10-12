package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func Skills30Exports(r bind.Registry) {
	r.Type(calculator.GemDisplay{}, bind.Plain())
	r.Type(calculator.GemDisplayRow{}, bind.Plain())
	r.Func(GemTooltip)
	r.Func(GemQualityTooltip)
	r.Func(MiscGemDisplays)
	r.Func(MiscSocketGroupTooltip)
	r.Type(calculator.GemSortRow{}, bind.Plain())
	r.Func(MiscGemSortCache)
	r.Func(CalcCustomModGroupStats, bind.AsPromise())
	r.Type(calculator.ModSectionData{}, bind.Plain())
	r.Type(calculator.ModSectionRow{}, bind.Plain())
	r.Type(calculator.TreeSpecRespec{}, bind.Plain())
	r.Func(TreeSpecRespecCost)
	r.Func(CurrentGameVersionLabel)
	r.Func(ItemSourceTooltip, bind.AsPromise())
}

func GemTooltip(build *pob.PathOfBuilding, gem pob.Gem, display calculator.GemDisplay, hasDisplay bool, stats calculator.ItemTooltipStats) []calculator.TooltipLine {
	ctx := calculator.ItemTooltipContext{ShowFlavourText: showFlavourText, Stats: stats}
	if build != nil {
		ctx.CharacterLevel = build.Build.Level
	}
	var d *calculator.GemDisplay
	if hasDisplay {
		d = &display
	}
	t := calculator.ItemTooltip{}
	if !calculator.AddGemTooltip(&t, ctx, gem, d, false) {
		return nil
	}
	return t.Lines
}

func GemQualityTooltip(gem pob.Gem) []calculator.TooltipLine {
	return calculator.GemQualityTooltip(gem)
}

func MiscGemDisplays(m *MiscCalculator) []calculator.GemDisplayRow {
	if m == nil || m.calc == nil {
		return nil
	}
	return m.calc.Base().GemDisplays()
}

func MiscSocketGroupTooltip(m *MiscCalculator, group int) []calculator.TooltipLine {
	if m == nil || m.calc == nil {
		return nil
	}
	return m.calc.Base().SocketGroupTooltip(group)
}

func MiscGemSortCache(m *MiscCalculator, group int, gems []pob.Gem) []calculator.GemSortRow {
	if m == nil || m.calc == nil {
		return nil
	}
	return m.calc.Base().GemSortCache(group, gems)
}

func CalcCustomModGroupStats(m *MiscCalculator, i int) StatOutputs {
	return statOutputs(m.calc.CalcCustomModGroup(i))
}

func TreeSpecRespecCost(build *pob.PathOfBuilding, index int) calculator.TreeSpecRespec {
	if build == nil {
		return calculator.TreeSpecRespec{}
	}
	return calculator.TreeSpecRespecCost(build, index)
}

func CurrentGameVersionLabel() string {
	return calculator.CurrentGameVersionLabel()
}

func ItemSourceTooltip(build *pob.PathOfBuilding, name string, stats calculator.ItemTooltipStats) (calculator.ItemTooltip, bool) {
	if build == nil {
		return calculator.ItemTooltip{}, false
	}
	item := calculator.ItemByName(build, name)
	if item == nil {
		return calculator.ItemTooltip{}, false
	}
	t := calculator.AddItemTooltip(item, calculator.ItemTooltipContext{CharacterLevel: build.Build.Level, ClassName: string(build.Build.ClassName), ShowFlavourText: showFlavourText, Stats: stats})
	appendItemModDebug(&t, item)
	return t, true
}
