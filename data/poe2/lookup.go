package poe2

import "sort"

// BaseByName is data.itemBases[name].
//
//pob2:block:src/Classes/Item.lua:855-855
func BaseByName(name string) (*ItemBase, bool) {
	i := sort.Search(len(ItemBases), func(i int) bool { return ItemBases[i].Name >= name })
	if i < len(ItemBases) && ItemBases[i].Name == name {
		return &ItemBases[i], true
	}
	return nil, false
}

// RuneModsByName is data.itemMods.Runes[name]: its slot entries, sorted by slot.
//
//pob2:block:src/Classes/Item.lua:1723-1723
func RuneModsByName(name string) []RuneMod {
	lo := sort.Search(len(RuneMods), func(i int) bool { return RuneMods[i].Rune >= name })
	hi := lo
	for hi < len(RuneMods) && RuneMods[hi].Rune == name {
		hi++
	}
	return RuneMods[lo:hi:hi]
}

// RuneModForSlot is data.itemMods.Runes[name][slot].
//
//pob2:block:src/Classes/Item.lua:2187-2187
func RuneModForSlot(name, slot string) (*RuneMod, bool) {
	mods := RuneModsByName(name)
	i := sort.Search(len(mods), func(i int) bool { return mods[i].Slot >= slot })
	if i < len(mods) && mods[i].Slot == slot {
		return &mods[i], true
	}
	return nil, false
}

// GemByID is data.gems[id].
//
//pob2:block:src/Classes/ItemsTab.lua:4176-4176
func GemByID(id string) (*Gem, bool) {
	i := sort.Search(len(Gems), func(i int) bool { return Gems[i].ID >= id })
	if i < len(Gems) && Gems[i].ID == id {
		return &Gems[i], true
	}
	return nil, false
}

// ItemModsOf is data.itemMods[category], sorted by id; nil when PoB has no such table.
//
//pob2:block:src/Classes/Item.lua:1101-1103
func ItemModsOf(category string) []ItemMod {
	lo := sort.Search(len(ItemMods), func(i int) bool { return ItemMods[i].Category >= category })
	hi := lo
	for hi < len(ItemMods) && ItemMods[hi].Category == category {
		hi++
	}
	if lo == hi {
		return nil
	}
	return ItemMods[lo:hi:hi]
}

// ItemModByID is data.itemMods[category][id].
func ItemModByID(category, id string) (*ItemMod, bool) {
	mods := ItemModsOf(category)
	i := sort.Search(len(mods), func(i int) bool { return mods[i].ID >= id })
	if i < len(mods) && mods[i].ID == id {
		return &mods[i], true
	}
	return nil, false
}
