package exposition

import "github.com/Vilsol/go-pob/calculator2"

// CompareOutputs is one calculation's output table and its minion's.
type CompareOutputs struct {
	Output map[string]float64
	Minion map[string]float64
}

// ItemCompareBlock is one stat difference block of an item tooltip: its header, the slot it compares in ("" for a
// flask or charm toggle) and that calculation's outputs.
type ItemCompareBlock struct {
	Header  string
	Slot    string
	Compare CompareOutputs
}

// ItemCompareView is what Build:AddStatComparesToTooltip compares: the calculator's base pass, the main skill's
// flags and the minion's, and the item's blocks.
type ItemCompareView struct {
	Base             CompareOutputs
	SkillFlags       map[string]bool
	MinionSkillFlags map[string]bool
	// Minion is the main skill's minion, whose stats are compared first.
	Minion bool
	Blocks []ItemCompareBlock
}

// miscCalc is the calculator over the last calculation; a worker holds one build, so one is kept.
var miscCalc struct {
	env  *calculator2.Environment
	misc *calculator2.MiscCalculator
}

//pob2:block:src/Classes/CalcsTab.lua:524-526
func (c *Calculator) miscCalculator() (*calculator2.MiscCalculator, error) {
	if c.env == nil || c.items == nil {
		return nil, notCalculatedError{}
	}
	if miscCalc.env != c.env {
		miscCalc.env, miscCalc.misc = c.env, calculator2.NewMiscCalculator(c.env)
	}
	return miscCalc.misc, nil
}

func compareOutputs(env *calculator2.Environment) CompareOutputs {
	out := CompareOutputs{Output: withFlags(env.Player.Output, env.Player.OutputFlags)}
	if main := env.Player.MainSkill; main != nil && main.Minion != nil {
		out.Minion = withFlags(main.Minion.Output, main.Minion.OutputFlags)
	}
	return out
}

// itemCompare is the item's tooltip stat differences over the last calculation.
func (c *Calculator) itemCompare(item *calculator2.Item) (*ItemCompareView, error) {
	misc, err := c.miscCalculator()
	if err != nil {
		return nil, err
	}
	view := &ItemCompareView{Base: compareOutputs(misc.Base), SkillFlags: statSetFlags(c.env.Player.MainSkill), Blocks: []ItemCompareBlock{}}
	if main := c.env.Player.MainSkill; main != nil && main.Minion != nil {
		view.Minion = true
		view.MinionSkillFlags = statSetFlags(main.Minion.MainSkill)
	}
	for _, b := range c.items.ItemCompares(item, misc, c.additionalRingSlot()) {
		view.Blocks = append(view.Blocks, ItemCompareBlock{Header: b.Header, Slot: b.Slot, Compare: compareOutputs(b.Env)})
	}
	return view, nil
}

// DisplayItemCompare is the editor's item's stat differences.
func (c *Calculator) DisplayItemCompare() (*ItemCompareView, error) {
	tab, err := c.itemsTab()
	if err != nil {
		return nil, err
	}
	if tab.DisplayItem == nil {
		return nil, itemsError("no item is open in the editor")
	}
	return c.itemCompare(tab.DisplayItem)
}

// BuildItemCompare is the stat differences of the build's item.
func (c *Calculator) BuildItemCompare(id int) (*ItemCompareView, error) {
	tab, err := c.itemsTab()
	if err != nil {
		return nil, err
	}
	item := tab.Items[id]
	if item == nil {
		return nil, itemsError("no such item")
	}
	return c.itemCompare(item)
}

// DBItemCompare is the stat differences of a database row's item.
func (c *Calculator) DBItemCompare(kind, name string) (*ItemCompareView, error) {
	item := calculator2.ItemDB(calculator2.ItemDBKind(kind))[name]
	if item == nil {
		return nil, itemsError("no such item")
	}
	return c.itemCompare(item)
}

// CraftItemCompare is the stat differences of the item the Craft Item popup would make.
func (c *Calculator) CraftItemCompare(rarity int, title, base string) (*ItemCompareView, error) {
	item, ok := calculator2.CraftItem(rarity, title, base, nil)
	if !ok {
		return nil, itemsError("no such base")
	}
	return c.itemCompare(item)
}
