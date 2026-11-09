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

func (c *Calculator) comparePowerSide() (calculator2.ComparePowerSide, error) {
	if c.env == nil || c.items == nil {
		return calculator2.ComparePowerSide{}, notCalculatedError{}
	}
	c.syncSkills()
	return calculator2.ComparePowerSide{Build: c.build, Env: c.env, Items: c.items}, nil
}

// NewComparePowerBuilder is the compare power report of what from has and this build lacks, over the last
// calculation of each, for every stat in stats.
//
//pob2:block:src/Classes/CompareTab.lua:3129-3157
func (c *Calculator) NewComparePowerBuilder(from *Calculator, stats []string, categories calculator2.ComparePowerCategories) (*calculator2.ComparePowerBuilder, error) {
	if from == nil {
		return nil, errNoCompare
	}
	primary, err := c.comparePowerSide()
	if err != nil {
		return nil, err
	}
	compare, err := from.comparePowerSide()
	if err != nil {
		return nil, err
	}
	misc, err := c.miscCalculator()
	if err != nil {
		return nil, err
	}
	t, err := loadTree()
	if err != nil {
		return nil, err
	}
	return calculator2.NewComparePowerBuilder(primary, compare, misc, t, stats, categories)
}

// CompareItemSlotView is one row of the Compare page's Items view; A and B are nil for an empty slot.
type CompareItemSlotView struct {
	Slot         string
	Label        string
	NodeID       int
	A, B         *ItemView
	Missing      bool
	AWarn, BWarn bool
}

// CompareItemsView is the Items view's base slot rows and jewel rows.
type CompareItemsView struct {
	Slots  []CompareItemSlotView
	Jewels []CompareItemSlotView
}

func compareItemView(item *calculator2.Item) *ItemView {
	if item == nil {
		return nil
	}
	v := itemTooltip(item)
	return &v
}

func compareItemSlotViews(rows []calculator2.CompareItemSlot) []CompareItemSlotView {
	out := make([]CompareItemSlotView, len(rows))
	for i, r := range rows {
		out[i] = CompareItemSlotView{
			Slot: r.Slot, Label: r.Label, NodeID: r.NodeID, A: compareItemView(r.PItem), B: compareItemView(r.CItem),
			Missing: r.Missing, AWarn: r.PWarn, BWarn: r.CWarn,
		}
	}
	return out
}

// CompareItems is the Items view's rows for this build against from.
func (c *Calculator) CompareItems(from *Calculator) (*CompareItemsView, error) {
	if from == nil {
		return nil, errNoCompare
	}
	primary, err := c.comparePowerSide()
	if err != nil {
		return nil, err
	}
	compare, err := from.comparePowerSide()
	if err != nil {
		return nil, err
	}
	slots, jewels := calculator2.CompareItemSlots(primary, compare)
	return &CompareItemsView{Slots: compareItemSlotViews(slots), Jewels: compareItemSlotViews(jewels)}, nil
}

// CompareEquipItem is the Equip button's stat comparison of from's item in slotName equipped in this build.
func (c *Calculator) CompareEquipItem(from *Calculator, slotName string) (*ItemCompareView, error) {
	if from == nil {
		return nil, errNoCompare
	}
	primary, err := c.comparePowerSide()
	if err != nil {
		return nil, err
	}
	compare, err := from.comparePowerSide()
	if err != nil {
		return nil, err
	}
	slots, jewels := calculator2.CompareItemSlots(primary, compare)
	var item *calculator2.Item
	for _, r := range append(slots, jewels...) {
		if r.Slot == slotName {
			item = r.CItem
		}
	}
	if item == nil {
		return nil, itemsError("the compared build has no item in " + slotName)
	}
	misc, err := c.miscCalculator()
	if err != nil {
		return nil, err
	}
	b := calculator2.CompareEquipItem(misc, primary, slotName, item)
	view := &ItemCompareView{Base: compareOutputs(misc.Base), SkillFlags: statSetFlags(c.env.Player.MainSkill)}
	if main := c.env.Player.MainSkill; main != nil && main.Minion != nil {
		view.Minion = true
		view.MinionSkillFlags = statSetFlags(main.Minion.MainSkill)
	}
	view.Blocks = []ItemCompareBlock{{Header: b.Header, Slot: b.Slot, Compare: compareOutputs(b.Env)}}
	return view, nil
}

// CompareModsEqual is, for each sectionData, whether its mod rows match between this build's last calculation and
// from's.
func (c *Calculator) CompareModsEqual(from *Calculator, sds []calculator2.ModSectionData) ([]bool, error) {
	if from == nil {
		return nil, errNoCompare
	}
	if c.env == nil || from.env == nil {
		return nil, notCalculatedError{}
	}
	out := make([]bool, len(sds))
	for i, sd := range sds {
		out[i] = calculator2.CompareModRowsMatch(c.env, from.env, sd)
	}
	return out, nil
}

// CalcsCompareModRows is the Calcs view tooltip's mod rows for sd over this build's last calculation.
func (c *Calculator) CalcsCompareModRows(sd calculator2.ModSectionData) ([]calculator2.CompareModRow, error) {
	if c.env == nil {
		return nil, notCalculatedError{}
	}
	return c.env.CompareModRows(sd), nil
}
