package exposition

import "github.com/Vilsol/go-pob/calculator2"

// ItemDBInfo is the item database list's dropdowns (ItemDBControl.lua).
type ItemDBInfo struct {
	Slots        []string
	Types        []string
	Leagues      []string
	Requirements []string
	Obtainable   []string
	SearchModes  []string
}

func ItemDBLists(kind string) ItemDBInfo {
	leagues, types := calculator2.ItemDBLists(calculator2.ItemDBKind(kind))
	out := ItemDBInfo{Slots: calculator2.ItemDBSlots, Types: types, SearchModes: calculator2.ItemDBSearchModes}
	if calculator2.ItemDBKind(kind) == calculator2.ItemDBUnique {
		out.Leagues, out.Requirements, out.Obtainable = leagues, calculator2.ItemDBRequirements, calculator2.ItemDBObtainable
	}
	return out
}

// ItemDBRow is one row of the item database list.
type ItemDBRow struct {
	Name        string
	Title       string
	BaseName    string
	Rarity      string
	Type        string
	PrimarySlot string
	Level       float64
	League      string
	Source      string
}

// ItemDBRows is the list for the filters; the requirement filter reads the last calculation's level and attributes.
func (c *Calculator) ItemDBRows(kind string, f calculator2.ItemDBFilter) ([]ItemDBRow, error) {
	if c.env == nil {
		return nil, notCalculatedError{}
	}
	output := c.env.Player.Output
	ctx := calculator2.ItemDBContext{CharacterLevel: c.env.Player.Level, Str: output["Str"], Dex: output["Dex"], Int: output["Int"]}
	out := []ItemDBRow{}
	for _, item := range calculator2.FilterItemDB(calculator2.ItemDBKind(kind), f, ctx) {
		row := ItemDBRow{Name: item.Name, Title: item.Title, BaseName: item.BaseName, Rarity: item.Rarity, Type: item.Type,
			PrimarySlot: item.GetPrimarySlot(), League: item.League, Source: item.Source}
		if item.Requirements.Level != nil {
			row.Level = *item.Requirements.Level
		}
		out = append(out, row)
	}
	return out, nil
}

// ItemDBTooltip is the row's tooltip.
//
//pob2:block:src/Classes/ItemDBControl.lua:333-341
func ItemDBTooltip(kind, name string) (ItemView, error) {
	item := calculator2.ItemDB(calculator2.ItemDBKind(kind))[name]
	if item == nil {
		return ItemView{}, itemsError("no such item")
	}
	return itemTooltip(item), nil
}

// ItemDBRaw is the row's text, as Ctrl+C copies it.
//
//pob2:block:src/Classes/ItemDBControl.lua:384-386
func ItemDBRaw(kind, name string) (string, error) {
	item := calculator2.ItemDB(calculator2.ItemDBKind(kind))[name]
	if item == nil {
		return "", itemsError("no such item")
	}
	return item.Raw, nil
}

// OpenDBItem opens the row's item in the editor, as a double click does.
func (c *Calculator) OpenDBItem(kind, name string) error {
	return c.displayEdit(func(tab *calculator2.ItemsTab) error { return tab.OpenDBItem(calculator2.ItemDBKind(kind), name) })
}

// AddDBItem adds the row's item to the build; equip equips it as Ctrl+click does, in the slot's second with second.
func (c *Calculator) AddDBItem(kind, name string, equip, second bool) (int, error) {
	id := 0
	err := c.itemsEdit(func(tab *calculator2.ItemsTab) error {
		item, err := tab.AddDBItem(calculator2.ItemDBKind(kind), name, equip, second, c.additionalRingSlot())
		if item != nil {
			id = item.ID
		}
		return err
	})
	return id, err
}
