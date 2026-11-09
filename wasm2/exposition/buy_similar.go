package exposition

import "github.com/Vilsol/go-pob/calculator2"

const errNoItem = itemsError("no item in that slot")

// BuySimilarView is the Buy Similar popup for the item this build equips in a slot, as the Compare tab opens it for
// the compared build's item: whether it searches by unique name, the name it shows, the Category label, the base the
// base box names and the defence rows.
type BuySimilarView struct {
	Name     string
	BaseName string
	Category string
	Defences []calculator2.BuySimilarDefence
	Listed   []string
	Unique   bool
}

func (c *Calculator) buySimilarItem(slotName string) (*calculator2.Item, error) {
	tab, err := c.itemsTab()
	if err != nil {
		return nil, err
	}
	slot := tab.Slots[slotName]
	if slot == nil || tab.Items[slot.SelItemID] == nil {
		return nil, errNoItem
	}
	return tab.Items[slot.SelItemID], nil
}

// BuySimilar is the popup for the item in slotName.
//
//pob2:block:src/Classes/CompareBuySimilar.lua:284-329
//pob2:block:src/Classes/CompareBuySimilar.lua:421-434
func (c *Calculator) BuySimilar(slotName string) (*BuySimilarView, error) {
	item, err := c.buySimilarItem(slotName)
	if err != nil {
		return nil, err
	}
	listed := make([]string, len(calculator2.BuySimilarListed))
	for i, l := range calculator2.BuySimilarListed {
		listed[i] = l.Label
	}
	return &BuySimilarView{
		Unique:   item.Rarity == "UNIQUE" || item.Rarity == "RELIC",
		Name:     item.Name,
		BaseName: item.BaseName,
		Category: calculator2.TradeCategoryLabel(item),
		Defences: calculator2.BuySimilarDefences(item),
		Listed:   listed,
	}, nil
}

// BuySimilarURL is the Open URL button's trade search for the item in slotName with the popup's selections.
func (c *Calculator) BuySimilarURL(slotName string, o calculator2.BuySimilarOptions) (string, error) {
	item, err := c.buySimilarItem(slotName)
	if err != nil {
		return "", err
	}
	return calculator2.BuySimilarURL(item, slotName, o), nil
}
