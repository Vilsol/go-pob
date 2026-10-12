package pob

import "strconv"

func (b *PathOfBuilding) SetSetSlotItem(setID int, slotName string, itemID int) bool {
	for i := range b.Items.ItemSets {
		if id, err := strconv.Atoi(b.Items.ItemSets[i].ID); err == nil && id == setID {
			b.Items.ItemSets[i].slot(slotName).ItemID = itemID
			return true
		}
	}
	return false
}

func (b *PathOfBuilding) SetSlotItemID(setID int, slotName string) int {
	for _, set := range b.Items.ItemSets {
		if id, err := strconv.Atoi(set.ID); err == nil && id == setID {
			for _, s := range set.Slots {
				if s.Name == slotName {
					return s.ItemID
				}
			}
		}
	}
	return 0
}
