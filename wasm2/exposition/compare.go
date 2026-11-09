package exposition

import (
	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/data/tree2"
)

const errNoCompare = itemsError("no compared build")

// CopyCompareSpec adds the compared build's active spec to this build's tree list, without its jewels, whose items
// belong to the compared build; andUse makes it the active spec.
//
//pob2:src/Classes/CompareTab.lua:1386-1423
func (c *Calculator) CopyCompareSpec(from *Calculator, andUse bool) error {
	if from == nil {
		return errNoCompare
	}
	sourceSpec, err := from.specAt(from.activeSpecIndex())
	if err != nil {
		return err
	}
	t, err := loadTree()
	if err != nil {
		return err
	}
	newSpec := tree2.NewPassiveSpec(t)
	newSpec.RestoreUndoState(sourceSpec.CreateUndoState())
	title := specTitle(&from.build.Tree.Specs[from.activeSpecIndex()-1]) + " (Compared)"
	c.build.Tree.Specs = append(c.build.Tree.Specs, newSpecElement(newSpec, title, nil))
	if andUse {
		c.SetActiveTreeSpec(len(c.build.Tree.Specs))
	}
	return nil
}

// CopyCompareItem adds the item in the compared build's slot to this build's item list, without equipping it unless
// andUse, which puts it in the same slot; false when that slot holds no item.
//
//pob2:src/Classes/CompareTab.lua:1494-1514
func (c *Calculator) CopyCompareItem(from *Calculator, slotName string, andUse bool) (bool, error) {
	if from == nil {
		return false, errNoCompare
	}
	cTab, err := from.itemsTab()
	if err != nil {
		return false, err
	}
	cSlot := cTab.Slots[slotName]
	if cSlot == nil || cTab.Items[cSlot.SelItemID] == nil || cTab.Items[cSlot.SelItemID].Raw == "" {
		return false, nil
	}
	raw := cTab.Items[cSlot.SelItemID].Raw
	return true, c.itemsEdit(func(pItemsTab *calculator2.ItemsTab) error {
		newItem := calculator2.ParseItem(raw, calculator2.ItemModSource(calculator2.WithParser(calculator2.ParseMod)))
		newItem.NormaliseQuality()
		pItemsTab.AddItem(newItem, true, c.additionalRingSlot())
		if andUse {
			if pSlot := pItemsTab.Slots[slotName]; pSlot != nil {
				pItemsTab.SetSelItemID(pSlot, newItem.ID)
			}
		}
		pItemsTab.PopulateSlots()
		return nil
	})
}
