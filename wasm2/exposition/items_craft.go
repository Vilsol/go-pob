package exposition

import "github.com/Vilsol/go-pob/calculator2"

// CraftBaseGroup is one entry of the Craft Item popup's Type list and the bases its Base list shows.
type CraftBaseGroup struct {
	Type  string
	Bases []string
}

// CraftBases is the Craft Item popup's Type list, with each type's bases in its Base list order.
//
//pob2:block:src/Classes/ItemsTab.lua:2838-2846
func CraftBases() []CraftBaseGroup {
	lists, types := calculator2.ItemBaseLists(), calculator2.ItemBaseTypeList()
	out := make([]CraftBaseGroup, 0, len(types))
	for _, typ := range types {
		g := CraftBaseGroup{Type: typ, Bases: []string{}}
		for _, e := range lists[typ] {
			g.Bases = append(g.Bases, e.Name)
		}
		out = append(out, g)
	}
	return out
}

// CraftItemPreview is the Base list's tooltip: the item Create would make. rarity is the 1-based rarity choice.
//
//pob2:block:src/Classes/ItemsTab.lua:2847-2852
func CraftItemPreview(rarity int, title, base string) (ItemView, error) {
	item, ok := calculator2.CraftItem(rarity, title, base, nil)
	if !ok {
		return ItemView{}, itemsError("no such base")
	}
	return itemTooltip(item), nil
}

// CraftDisplayItem is the Craft Item popup's Create: the item opens in the editor.
func (c *Calculator) CraftDisplayItem(rarity int, title, base string) error {
	return c.displayEdit(func(tab *calculator2.ItemsTab) error { return tab.CraftDisplayItem(rarity, title, base) })
}
