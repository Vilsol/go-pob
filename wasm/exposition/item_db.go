package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

func ItemDBExports(r bind.Registry) {
	r.Type(ItemDBInfo{}, bind.Plain())
	r.Type(ItemDBRow{}, bind.Plain())
	r.Type(ItemDBSortStat{}, bind.Plain())
	r.Type(calculator.ItemDBFilter{}, bind.Plain())

	r.Func(ItemDBLists, bind.AsPromise())
	r.Func(ItemDBTooltip, bind.AsPromise())
	r.Func(ItemDBText, bind.AsPromise())
	r.Func(ItemDBRaw, bind.AsPromise())
	r.Func(AddDBItem, bind.AsPromise())
}

type ItemDBSortStat struct {
	Stat  string
	Label string
}

type ItemDBInfo struct {
	Slots        []string
	Types        []string
	Leagues      []string
	Requirements []string
	Obtainable   []string
	SearchModes  []string
	SortStats    []ItemDBSortStat
}

func ItemDBLists(kind string) ItemDBInfo {
	leagues, types := calculator.ItemDBLists(calculator.ItemDBKind(kind))
	out := ItemDBInfo{Slots: calculator.ItemDBSlots, Types: types, SearchModes: calculator.ItemDBSearchModes}
	if calculator.ItemDBKind(kind) == calculator.ItemDBUnique {
		out.Leagues, out.Requirements, out.Obtainable = leagues, calculator.ItemDBRequirements, calculator.ItemDBObtainable
		for _, s := range calculator.ItemDBSortStats() {
			out.SortStats = append(out.SortStats, ItemDBSortStat{Stat: s.Stat, Label: s.Label})
		}
	}
	return out
}

type ItemDBRow struct {
	Name        string
	Title       string
	BaseName    string
	Rarity      string
	Type        string
	PrimarySlot string
	Level       int
	League      string
	Source      string
	Power       float64
}

func (m *MiscCalculator) ItemDBRows(kind string, f calculator.ItemDBFilter, level int, str, dex, intel float64, stat string) []ItemDBRow {
	list := calculator.FilterItemDB(calculator.ItemDBKind(kind), f, calculator.ItemDBContext{CharacterLevel: level, Str: str, Dex: dex, Int: intel})
	var powers []float64
	if s, ok := calculator.PowerStatByName(stat); ok && stat != "" {
		powers = m.calc.SortItemDB(list, s)
	}
	out := make([]ItemDBRow, len(list))
	for k, item := range list {
		out[k] = ItemDBRow{Name: item.Name, Title: item.Title, BaseName: item.BaseName, Rarity: string(item.Rarity), Type: item.Type,
			PrimarySlot: item.PrimarySlot(), Level: int(item.Requirements["level"]), League: item.League, Source: item.Source}
		if powers != nil {
			out[k].Power = powers[k]
		}
	}
	return out
}

func ItemDBTooltip(build *pob.PathOfBuilding, kind string, name string, stats calculator.ItemTooltipStats) (calculator.ItemTooltip, bool) {
	item := calculator.ItemDBItem(calculator.ItemDBKind(kind), name)
	if item == nil {
		return calculator.ItemTooltip{}, false
	}
	ctx := calculator.ItemTooltipContext{ShowFlavourText: showFlavourText, Stats: stats, DBMode: true}
	if build != nil {
		ctx.CharacterLevel = build.Build.Level
		ctx.ClassName = string(build.Build.ClassName)
	}
	t := calculator.AddItemTooltip(item, ctx)
	appendItemModDebug(&t, item)
	t.Children = calculator.GrantedSkillTooltips(t, ctx)
	return t, true
}

func ItemDBText(kind string, name string) (string, bool) {
	return calculator.ItemDBText(calculator.ItemDBKind(kind), name)
}

func ItemDBRaw(kind string, name string) (string, bool) {
	item := calculator.ItemDBItem(calculator.ItemDBKind(kind), name)
	if item == nil {
		return "", false
	}
	return item.Raw, true
}

func AddDBItem(build *pob.PathOfBuilding, kind string, name string, equip bool, second bool) (int, bool) {
	return calculator.AddDBItem(build, calculator.ItemDBKind(kind), name, equip, second)
}
