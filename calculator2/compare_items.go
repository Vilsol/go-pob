package calculator2

import (
	"slices"

	"github.com/Vilsol/go-pob/data/tree2"
)

// CompareItemSlot is one row of the Compare page's Items view: the slot's item in each build. Missing is a Ring 3 the
// primary build has no slot for; the warnings are a jewel in a socket its build's tree leaves unallocated.
type CompareItemSlot struct {
	Slot           string
	Label          string
	NodeID         int
	PItem, CItem   *Item
	Missing        bool
	PWarn, CWarn   bool
	PNodeAllocated bool
}

// CompareItemSlots is the Items view's base slot rows, then its jewel rows.
//
//pob2:block:src/Classes/CompareTab.lua:3670-3675
//pob2:block:src/Classes/CompareTab.lua:3856-3862
//pob2:block:src/Classes/CompareTab.lua:3894-3907
func CompareItemSlots(primary, compare ComparePowerSide) ([]CompareItemSlot, []CompareItemSlot) {
	baseSlots := []string{"Weapon 1", "Weapon 2", "Weapon 1 Swap", "Weapon 2 Swap", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring 1", "Ring 2", "Belt", "Flask 1", "Flask 2", "Charm 1", "Charm 2", "Charm 3"}
	if shouldShowRing3(primary.Env, compare.Env) {
		baseSlots = slices.Insert(baseSlots, 11, "Ring 3")
	}
	primaryHasRing3 := primary.Env.ModDB.Flag(nil, "AdditionalRingSlot")
	slots := make([]CompareItemSlot, 0, len(baseSlots))
	for _, slotName := range baseSlots {
		slots = append(slots, CompareItemSlot{
			Slot: slotName, Label: slotName, PItem: selItem(primary.Items, slotName), CItem: selItem(compare.Items, slotName),
			Missing: slotName == "Ring 3" && !primaryHasRing3,
		})
	}
	jewelSlots := CompareJewelSlots(primary, compare)
	jewels := make([]CompareItemSlot, 0, len(jewelSlots))
	for _, j := range jewelSlots {
		jewels = append(jewels, CompareItemSlot{
			Slot: j.SlotName, Label: j.Label, NodeID: j.NodeID, PItem: j.PItem, CItem: j.CItem,
			PWarn: j.PItem != nil && !j.PNodeAllocated, CWarn: j.CItem != nil && !j.CNodeAllocated, PNodeAllocated: j.PNodeAllocated,
		})
	}
	return slots, jewels
}

// CompareEquipItem is the stat comparison the Equip button shows: the compared build's item in slotName of the
// primary build, its socket allocated for the calculation when the tree leaves it unallocated.
//
//pob2:block:src/Classes/CompareTab.lua:3936-3966
func CompareEquipItem(m *MiscCalculator, primary ComparePowerSide, slotName string, item *Item) ItemCompare {
	newItem := compareItemCopy(item.Raw)
	pSlot := primary.Items.Slots[slotName]
	var sel *Item
	override := calcOverride{RepSlotName: slotName, RepItem: newItem}
	slotLabel := slotName
	if pSlot != nil {
		sel = primary.Items.Items[pSlot.SelItemID]
		slotLabel = primary.Items.slotLabel(pSlot)
		if pSlot.NodeID != 0 {
			pSpec := primary.Env.Spec
			if l, ok := socketLabel(primary.Items, pSpec, pSlot.NodeID); ok {
				slotLabel = l
			}
			if pSpec.AllocNodes[pSlot.NodeID] == nil {
				if node := pSpec.Nodes[pSlot.NodeID]; node != nil {
					override.AddNodes = map[*tree2.SpecNode]bool{node: true}
				}
			}
		}
	}
	output := m.calc(override, true)
	header := "^7Equipping this item in " + slotLabel + " will give you:"
	if sel != nil {
		header = "^7Equipping this item in " + slotLabel + " will give you:\n(replacing " + rarityColor(sel.Rarity) + sel.Name + "^7)"
	}
	return ItemCompare{Header: header, Slot: slotName, Env: output}
}

// rarityColor is colorCodes[rarity] or "^7".
func rarityColor(rarity string) string {
	if c, ok := rarityColors[rarity]; ok {
		return c
	}
	return "^7"
}

// ArmourDataValue is item:GetArmourDataValue.
func (i *Item) ArmourDataValue(name string, level int) float64 {
	return itemArmourDataValue(i, name, level)
}
