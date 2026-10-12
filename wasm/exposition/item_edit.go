package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func ItemEditExports(r bind.Registry) {
	r.Type(calculator.DisplayItemInfo{}, bind.Plain())
	r.Type(calculator.DisplayModLine{}, bind.Plain())
	r.Type(calculator.ItemEdit{}, bind.Plain())
	r.Type(calculator.CraftBaseGroup{}, bind.Plain())
	r.Type(calculator.EnchantChoices{}, bind.Plain())
	r.Type(calculator.EnchantSkill{}, bind.Plain())
	r.Type(calculator.EnchantSource{}, bind.Plain())
	r.Type(calculator.AnointNotable{}, bind.Plain())
	r.Type(AnointCompare{}, bind.Plain())
	r.Type(calculator.CorruptChoices{}, bind.Plain())
	r.Type(calculator.CorruptMod{}, bind.Plain())
	r.Type(calculator.CorruptRoll{}, bind.Plain())
	r.Type(calculator.ImplicitChoices{}, bind.Plain())
	r.Type(calculator.ImplicitSource{}, bind.Plain())
	r.Type(calculator.ImplicitGroup{}, bind.Plain())
	r.Type(calculator.ImplicitMod{}, bind.Plain())
	r.Type(calculator.CustomModChoices{}, bind.Plain())
	r.Type(calculator.CustomMod{}, bind.Plain())
	r.Type(calculator.AffixControls{}, bind.Plain())
	r.Type(calculator.AffixControl{}, bind.Plain())
	r.Type(calculator.AffixSeries{}, bind.Plain())
	r.Type(calculator.ClusterInfo{}, bind.Plain())
	r.Type(calculator.ClusterSkillChoice{}, bind.Plain())
	r.Type(calculator.CrucibleChoices{}, bind.Plain())
	r.Type(calculator.CrucibleChoice{}, bind.Plain())

	r.Func(DisplayItem, bind.AsPromise())
	r.Func(EditItem, bind.AsPromise())
	r.Func(CraftBaseGroups, bind.AsPromise())
	r.Func(CraftItemText, bind.AsPromise())
	r.Func(ItemEnchantChoices, bind.AsPromise())
	r.Func(AnointNotables, bind.AsPromise())
	r.Func(ItemCorruptChoices, bind.AsPromise())
	r.Func(CorruptedRangeLine, bind.AsPromise())
	r.Func(CorruptItemText, bind.AsPromise())
	r.Func(ItemImplicitChoices, bind.AsPromise())
	r.Func(ImplicitText, bind.AsPromise())
	r.Func(ItemCustomModChoices, bind.AsPromise())
	r.Func(CustomModText, bind.AsPromise())
	r.Func(ItemAffixControls, bind.AsPromise())
	r.Func(AffixEditText, bind.AsPromise())
	r.Func(AffixRollTooltip, bind.AsPromise())
	r.Func(ItemCrucibleChoices, bind.AsPromise())
	r.Func(CrucibleText, bind.AsPromise())
}

func DisplayItem(text string) (calculator.DisplayItemInfo, bool) {
	return calculator.DisplayItem(text)
}

func EditItem(text string, edit calculator.ItemEdit) (string, error) {
	return calculator.EditItem(text, edit)
}

func CraftBaseGroups() []calculator.CraftBaseGroup {
	return calculator.CraftBaseGroups()
}

func CraftItemText(rarity int, title string, baseName string) (string, bool) {
	return calculator.CraftItemText(rarity, title, baseName)
}

func ItemEnchantChoices(build *pob.PathOfBuilding, text string) (calculator.EnchantChoices, bool) {
	return calculator.ItemEnchantChoices(build, text)
}

func AnointNotables() []calculator.AnointNotable {
	return calculator.AnointNotables()
}

func (m *MiscCalculator) EnchantSortValues(text string, slot int, lines []string, stat string) []float64 {
	return m.calc.EnchantSortValues(text, slot, lines, stat)
}

func (m *MiscCalculator) AnointPowers(text string, slot int, ids []int64, stat string) []float64 {
	return m.calc.AnointPowers(text, slot, ids, stat)
}

type AnointCompare struct {
	Note   string
	Header string
	Base   StatOutputs
	New    StatOutputs
}

func (m *MiscCalculator) CompareAnoint(text string, slot int, id int64) (AnointCompare, bool) {
	c, ok := m.calc.CompareAnoint(text, slot, id, "")
	if !ok {
		return AnointCompare{}, false
	}
	if c.Note != "" {
		return AnointCompare{Note: c.Note}, true
	}
	return AnointCompare{Header: c.Header, Base: statOutputs(c.Base), New: statOutputs(c.New)}, true
}

func ItemCorruptChoices(text string) (calculator.CorruptChoices, bool) {
	return calculator.ItemCorruptChoices(text)
}

func CorruptedRangeLine(text string, index int, value float64) string {
	return calculator.CorruptedRangeLine(text, index, value)
}

func CorruptItemText(text string, source string, ids []string, downside []string, rolls bool, rollIndex []int, rollValue []float64) (string, bool) {
	var r map[int]float64
	if rolls {
		r = map[int]float64{}
		for k, i := range rollIndex {
			if k < len(rollValue) {
				r[i] = rollValue[k]
			}
		}
	}
	return calculator.CorruptItemText(text, source, ids, downside, r)
}

func (m *MiscCalculator) CorruptSortValues(text string, source string, modType string, ids []string, stat string) []float64 {
	return m.calc.CorruptSortValues(text, source, modType, ids, stat)
}

func ItemImplicitChoices(text string) (calculator.ImplicitChoices, bool) {
	return calculator.ItemImplicitChoices(text)
}

func ImplicitText(text string, sourceID string, modID string, custom string) (string, bool) {
	return calculator.ImplicitText(text, sourceID, modID, custom)
}

func (m *MiscCalculator) ImplicitSortValues(text string, sourceID string, ids []string, stat string) []float64 {
	return m.calc.ImplicitSortValues(text, sourceID, ids, stat)
}

func ItemCustomModChoices(text string) (calculator.CustomModChoices, bool) {
	return calculator.ItemCustomModChoices(text)
}

func CustomModText(text string, sourceID string, modID string, custom string) (string, bool) {
	return calculator.CustomModText(text, sourceID, modID, custom)
}

func (m *MiscCalculator) CustomModSortValues(text string, sourceID string, ids []string, stat string) []float64 {
	return m.calc.CustomModSortValues(text, sourceID, ids, stat)
}

func ItemAffixControls(text string) (calculator.AffixControls, bool) {
	return calculator.ItemAffixControls(text)
}

func AffixEditText(text string, slot int, series int, slider float64) (string, bool) {
	return calculator.AffixEditText(text, slot, series, slider)
}

func AffixRollTooltip(text string, slot int, slider float64) []string {
	return calculator.AffixRollTooltip(text, slot, slider)
}

func (m *MiscCalculator) AffixPowers(text string, slot int, stat string) []float64 {
	return m.calc.AffixPowers(text, slot, stat)
}

func ItemCrucibleChoices(text string) (calculator.CrucibleChoices, bool) {
	return calculator.ItemCrucibleChoices(text)
}

func CrucibleText(text string, ids []string) (string, bool) {
	return calculator.CrucibleText(text, ids)
}
