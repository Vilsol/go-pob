package calculator

import (
	"github.com/Vilsol/go-pob/pob"
)

var rarityColorCodes = map[ItemRarity]string{
	"NORMAL": "^xC8C8C8",
	"MAGIC":  "^x8888FF",
	"RARE":   "^xFFFF77",
	"UNIQUE": "^xAF6025",
	"RELIC":  "^x60C060",
}

//pob:src/Classes/ItemsTab.lua:3982-3991
func ItemSetTooltip(build *pob.PathOfBuilding, set pob.ItemSet) []string {
	if build == nil {
		return nil
	}
	items := make(map[int]string, len(build.Items.Items))
	for _, it := range build.Items.Items {
		items[it.ID] = it.Text
	}
	var lines []string
	for _, slot := range orderedItemSlots {
		text, ok := items[setSlotItemID(set, slot.name)]
		if !ok {
			continue
		}
		item := parseItemGlobal(text)
		if item == nil {
			continue
		}
		lines = append(lines, "^7"+slotLabel(slot)+": "+rarityColorCodes[item.Rarity]+item.Name)
	}
	return lines
}

func ActiveSetSlotItemName(build *pob.PathOfBuilding, slotName string) string {
	if build == nil {
		return ""
	}
	if item := activeSetSlotItem(build, slotName); item != nil {
		return item.Name
	}
	return ""
}
