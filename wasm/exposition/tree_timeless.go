package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func TreeTimelessExports(r bind.Registry) {
	r.Type(calculator.TimelessJewelType{}, bind.Plain())
	r.Type(calculator.TimelessSocket{}, bind.Plain())
	r.Type(calculator.TimelessMod{}, bind.Plain())
	r.Type(calculator.TimelessSearchQuery{}, bind.Plain())
	r.Type(calculator.TimelessDesiredNode{}, bind.Plain())
	r.Type(calculator.TimelessNodeMatch{}, bind.Plain())
	r.Type(calculator.TimelessSearchResult{}, bind.Plain())
	r.Type(calculator.TimelessTradeQuery{}, bind.Plain())
	r.Type(calculator.TimelessTrade{}, bind.Plain())
	r.Type(pob.TimelessData{}, bind.Plain())
	r.Type(TimelessSearch{})

	r.Func(TimelessJewelTypes)
	r.Func(TimelessConquerors)
	r.Func(TimelessDevotionVariants)
	r.Func(TimelessTradeTypes)
	r.Func(TimelessRealms)
	r.Func(TimelessSockets)
	r.Func(TimelessMods)
	r.Func(TimelessProtectableNodes)
	r.Func(GetTimelessData)
	r.Func(SetTimelessData)
	r.Func(NewTimelessSearch)
	r.Func(TimelessFallbackModes)
}

func TimelessJewelTypes() []calculator.TimelessJewelType {
	return calculator.TimelessJewelTypes
}

func TimelessConquerors(jewelType int) []string {
	return calculator.TimelessConquerorTypes[jewelType]
}

func TimelessDevotionVariants() []string {
	return calculator.TimelessDevotionVariants
}

func TimelessTradeTypes() []string {
	return calculator.TimelessTradeTypes
}

func TimelessRealms() []string {
	return calculator.TimelessRealms
}

func TimelessSockets(build *pob.PathOfBuilding) []calculator.TimelessSocket {
	return calculator.TimelessSockets(build)
}

func TimelessMods(jewelType int) []calculator.TimelessMod {
	return calculator.TimelessMods(jewelType)
}

func TimelessProtectableNodes(m *MiscCalculator, socketID int64) []calculator.TimelessMod {
	return m.calc.TimelessProtectableNodes(socketID)
}

func GetTimelessData(build *pob.PathOfBuilding) pob.TimelessData {
	return build.GetTimelessData()
}

func SetTimelessData(build *pob.PathOfBuilding, d pob.TimelessData) {
	build.SetTimelessData(d)
}

type TimelessSearch struct {
	s       *calculator.TimelessSearcher
	q       calculator.TimelessSearchQuery
	results calculator.TimelessSearchResults
	rolls   map[int]*[2]int
}

func NewTimelessSearch(m *MiscCalculator, q calculator.TimelessSearchQuery) *TimelessSearch {
	s := m.calc.NewTimelessSearcher(q)
	if s == nil {
		return nil
	}
	return &TimelessSearch{s: s, q: q}
}

func (t *TimelessSearch) Step() float64 {
	done := t.s.Step()
	if done >= 1 {
		t.results = t.s.Results()
	}
	return done
}

func (t *TimelessSearch) Count() int {
	return len(t.results.Results)
}

func (t *TimelessSearch) Desired() []calculator.TimelessDesiredNode {
	return t.results.Desired
}

func (t *TimelessSearch) Rows(start, count int) []calculator.TimelessSearchResult {
	if start < 0 || start >= len(t.results.Results) {
		return nil
	}
	return t.results.Results[start:min(len(t.results.Results), start+count)]
}

func (t *TimelessSearch) TradeURL(q calculator.TimelessTradeQuery) calculator.TimelessTrade {
	q.Seeds = make([]int, len(t.results.Results))
	for i, r := range t.results.Results {
		q.Seeds[i] = r.Seed
	}
	q.JewelType = t.q.JewelType
	q.ConquerorType = t.q.ConquerorType
	return calculator.TimelessTradeURL(q)
}

func (t *TimelessSearch) Item(row int, devotion1, devotion2 int, socketLabel string) string {
	if row < 0 || row >= len(t.results.Results) {
		return ""
	}
	r := t.results.Results[row]
	if r.SocketLabel != "" {
		socketLabel = r.SocketLabel
	}
	if t.rolls == nil {
		t.rolls = map[int]*[2]int{}
	}
	if t.rolls[row] == nil {
		t.rolls[row] = &[2]int{}
	}
	return calculator.TimelessResultItem(t.q.JewelType, t.q.ConquerorType, devotion1, devotion2, r.Seed, r.Total, socketLabel, t.rolls[row])
}

func (t *TimelessSearch) AddItem(build *pob.PathOfBuilding, row int, devotion1, devotion2 int, socketLabel string) int {
	text := t.Item(row, devotion1, devotion2, socketLabel)
	if text == "" {
		return 0
	}
	id := build.AddItem(text)
	calculator.PopulateSlots(build)
	return id
}

func (t *TimelessSearch) Socket(build *pob.PathOfBuilding, socketID int, itemID int) bool {
	return calculator.TimelessSocketJewel(build, socketID, itemID)
}

func TimelessFallbackModes() []calculator.PowerStat {
	return calculator.TimelessFallbackModes()
}

func (m *MiscCalculator) TimelessFallbackWeights(jewelType int, stat string) (string, bool) {
	return m.calc.TimelessFallbackWeights(jewelType, stat)
}
