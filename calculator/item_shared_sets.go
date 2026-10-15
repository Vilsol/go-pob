package calculator

import (
	"slices"
	"strconv"

	"github.com/Vilsol/go-pob/pob"
)

type SharedItemSet struct {
	Title string            `json:"title"`
	Slots map[string]string `json:"slots"`
}

//pob:block:src/Classes/SharedItemSetListControl.lua:80-97
func ShareItemSet(build *pob.PathOfBuilding, setID int) (SharedItemSet, bool) {
	for _, set := range build.Items.ItemSets {
		if id, err := strconv.Atoi(set.ID); err != nil || id != setID {
			continue
		}
		out := SharedItemSet{Title: "Default", Slots: map[string]string{}}
		if set.Title != nil {
			out.Title = *set.Title
		}
		for _, s := range orderedItemSlots {
			if text, ok := buildItemText(build, setSlotItemID(set, s.name)); ok {
				out.Slots[s.name] = buildRawText(ParseItem(text))
			}
		}
		return out, true
	}
	return SharedItemSet{}, false
}

//pob:block:src/Classes/ItemSetListControl.lua:97-110
func ImportSharedItemSet(build *pob.PathOfBuilding, shared SharedItemSet) int {
	setID := build.NewItemSet(shared.Title)
	for _, s := range orderedItemSlots {
		text, ok := shared.Slots[s.name]
		if !ok {
			continue
		}
		item := ParseItem(text)
		item.NormaliseQuality()
		build.SetSetSlotItem(setID, s.name, build.AddItem(buildRawText(item)))
	}
	PopulateSlots(build)
	return setID
}

//pob:src/Classes/SharedItemSetListControl.lua:58-70
func SharedItemSetTooltip(shared SharedItemSet) []string {
	var lines []string
	for _, s := range orderedItemSlots {
		if text, ok := shared.Slots[s.name]; ok {
			item := ParseItem(text)
			lines = append(lines, "^7"+slotLabel(s)+": "+rarityColorCodes[item.Rarity]+item.Name)
		}
	}
	return lines
}

//pob:block:src/Classes/SharedItemListControl.lua:52-61
func ShareItem(build *pob.PathOfBuilding, id int, kind ItemDBKind, name string) (string, bool) {
	text, inBuild, ok := dropText(build, id, kind, name)
	if !ok {
		return "", false
	}
	item := ParseItem(text)
	if !inBuild {
		item.NormaliseQuality()
	}
	return buildRawText(item), true
}

//pob:block:src/Classes/ItemListControl.lua:283-293,src/Classes/ItemSlotControl.lua:125-142
func AddSharedItem(build *pob.PathOfBuilding, text string, slotName string) (int, bool) {
	item := ParseItem(text)
	if slotName != "" && !IsItemValidForSlot(build, item, slotName) {
		return 0, false
	}
	item.NormaliseQuality()
	id := build.AddItem(buildRawText(item))
	if slotName != "" {
		build.SetSlotItem(slotName, id)
	}
	//pob:block:src/Classes/ItemListControl.lua:288,src/Classes/ItemSlotControl.lua:137
	AddForbiddenJewelCounterpart(build, item)
	PopulateSlots(build)
	return id, true
}

//pob:block:src/Classes/ItemsTab.lua:1906-1945
func CopyAnointsAndEldritchImplicits(build *pob.PathOfBuilding, newItem *Item, copyEldritchImplicits, overwrite bool, sourceSlotName string) {
	newItemType := sourceSlotName
	if newItemType == "" {
		if newItem.Base.Weapon != nil {
			newItemType = "Weapon 1"
		} else {
			newItemType = newItem.Base.Type
		}
	}
	set, byID := setItems(build)
	selItemID := setSlotItemID(set, newItemType)
	currentItem := setSlotItem(set, byID, newItemType)
	if currentItem == nil {
		return
	}
	if isAnointable(newItem) && (len(newItem.EnchantModLines) == 0 || overwrite) && selItemID > 0 {
		if len(currentItem.EnchantModLines) == 1 {
			newItem.EnchantModLines = currentItem.EnchantModLines
		}
	}
	inf := newItem.Influence
	if inf.Shaper || inf.Elder || inf.Adjudicator || inf.Basilisk || inf.Crusader || inf.Eyrie {
		return
	}

	modifiableItem := !(newItem.Corrupted || newItem.Mirrored)
	if copyEldritchImplicits && slices.Contains([]string{"Helmet", "Body Armour", "Gloves", "Boots"}, newItem.Base.Type) &&
		slices.Contains([]ItemRarity{"NORMAL", "MAGIC", "RARE"}, newItem.Rarity) &&
		(len(newItem.ImplicitModLines) == 0 || overwrite) && modifiableItem && (currentItem.Influence.Cleansing || currentItem.Influence.Tangle) {
		newItem.ImplicitModLines = currentItem.ImplicitModLines
		newItem.Influence.Tangle = currentItem.Influence.Tangle
		newItem.Influence.Cleansing = currentItem.Influence.Cleansing
	}

	if (newItem.Base.Weapon != nil || newItem.Base.Type == "Body Armour") && (len(newItem.EnchantModLines) == 0 || overwrite) && selItemID > 0 &&
		modifiableItem {
		newItem.EnchantModLines = currentItem.EnchantModLines
	}
}

//pob:block:src/Classes/ItemsTab.lua:1948-1958
func NormaliseDisplayItem(build *pob.PathOfBuilding, text string, copyEldritchImplicits bool) string {
	newItem := ParseItem(text)
	if newItem.Base == nil {
		return text
	}
	CopyAnointsAndEldritchImplicits(build, newItem, copyEldritchImplicits, false, "")
	newItem.NormaliseQuality()
	return buildRawText(newItem)
}
