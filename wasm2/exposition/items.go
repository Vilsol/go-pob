package exposition

import (
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator2"
)

// The Items tab's edits over the last calculation's items tab. Each edit of the build writes the tab into it, as
// ItemsTab:Save does, and leaves it to be calculated again; the item editor's edits change only the editor's item.

// ItemSlotView is one shown slot of the items tab: what it holds and the items its dropdown lists.
type ItemSlotView struct {
	Name  string
	Label string
	// Parent is the item slot a jewel socket belongs to.
	Parent string
	// NodeID is a tree jewel socket's node.
	NodeID int
	ItemID int
	// Choices are the items that fit, in the item list's order.
	Choices []int
	// Activatable is a flask or charm slot, with its checkbox.
	Activatable bool
	Active      bool
}

type ItemSetView struct {
	ID    int
	Title string
}

// RuneSlotView is one character rune slot: its rune and the runes it lists.
type RuneSlotView struct {
	Name    string
	Label   string
	Rune    string
	Choices []string
}

type RuneChoiceView struct {
	Name  string
	Lines []string
}

// DisplayItemView is the item open in the editor.
type DisplayItemView struct {
	// ID is the build's item it replaces on save; 0 is a new item.
	ID   int
	Item ItemView
	// Text is the editor's text, without its Rarity line.
	Text   string
	Rarity string
	// Fits is the shown slots it fits.
	Fits []string
	// CanHaveSockets shows the socket count; Runes is each socket's rune, chosen from RuneChoices.
	CanHaveSockets bool
	SocketCount    int
	Runes          []string
	RuneChoices    []RuneChoiceView
}

// ItemsPageView is the items tab: its item sets, the shown slots, the build's items and the editor.
type ItemsPageView struct {
	ActiveItemSet      int
	Sets               []ItemSetView
	UseSecondWeaponSet bool
	Slots              []ItemSlotView
	Items              []ItemView
	RuneSlots          []RuneSlotView
	Display            *DisplayItemView
}

// itemsTab is the last calculation's items tab.
func (c *Calculator) itemsTab() (*calculator2.ItemsTab, error) {
	if c.env == nil || c.items == nil {
		return nil, notCalculatedError{}
	}
	return c.items, nil
}

// additionalRingSlot is the Ring 3 slot's shown: the main calculation's AdditionalRingSlot flag.
func (c *Calculator) additionalRingSlot() bool {
	return c.env.ModDB.Flag(nil, "AdditionalRingSlot")
}

// itemsEdit runs an edit of the build's items and writes the tab into the build.
func (c *Calculator) itemsEdit(edit func(tab *calculator2.ItemsTab) error) error {
	tab, err := c.itemsTab()
	if err != nil {
		return err
	}
	if err := edit(tab); err != nil {
		return err
	}
	tab.Save(c.build)
	return nil
}

// displayEdit runs an edit of the editor's item.
func (c *Calculator) displayEdit(edit func(tab *calculator2.ItemsTab) error) error {
	tab, err := c.itemsTab()
	if err != nil {
		return err
	}
	return edit(tab)
}

// ItemsPage is the items tab over the last calculation.
//
//pob2:block:src/Classes/ItemsTab.lua:201-345
//pob2:block:src/Classes/ItemSlotControl.lua:117-148
func (c *Calculator) ItemsPage() (*ItemsPageView, error) {
	tab, err := c.itemsTab()
	if err != nil {
		return nil, err
	}
	out := &ItemsPageView{ActiveItemSet: tab.ActiveItemSetID, UseSecondWeaponSet: tab.UseSecondWeaponSet, Sets: []ItemSetView{},
		Slots: []ItemSlotView{}, Items: []ItemView{}, RuneSlots: []RuneSlotView{}}
	for _, id := range tab.ItemSetOrderList {
		out.Sets = append(out.Sets, ItemSetView{ID: id, Title: tab.ItemSets[id].Title})
	}
	ring := c.additionalRingSlot()
	socketIndex := 0
	for _, slot := range tab.OrderedSlots {
		if !tab.SlotShown(slot, ring) {
			continue
		}
		v := ItemSlotView{Name: slot.SlotName, Label: strings.TrimSuffix(slot.SlotName, " Swap"), NodeID: slot.NodeID, ItemID: slot.SelItemID,
			Choices: []int{}, Activatable: strings.Contains(slot.SlotName, "Flask") || strings.Contains(slot.SlotName, "Charm"), Active: slot.Active}
		switch {
		case slot.NodeID != 0:
			socketIndex++
			v.Label = "Socket #" + strconv.Itoa(socketIndex)
		case slot.ParentSlot != nil:
			v.Parent = slot.ParentSlot.SlotName
			v.Label = "Jewel #" + strconv.Itoa(slot.SlotNum)
		}
		for _, id := range tab.ItemOrderList {
			if tab.IsItemValidForSlot(tab.Items[id], slot.SlotName, nil) {
				v.Choices = append(v.Choices, id)
			}
		}
		out.Slots = append(out.Slots, v)
	}
	for _, id := range tab.ItemOrderList {
		out.Items = append(out.Items, itemTooltip(tab.Items[id]))
	}
	//pob2:block:src/Classes/ItemsTab.lua:302-309
	if c.env.ModDB.Flag(nil, "SocketRunesOnCharacter") {
		labels := map[string]string{"Helmet Rune #1": "Helmet Rune 1", "Body Armour Rune #1": "Body Rune 1", "Body Armour Rune #2": "Body Rune 2",
			"Gloves Rune #1": "Gloves Rune 1", "Boots Rune #1": "Boots Rune 1"}
		for _, name := range calculator2.CharacterRuneSlots() {
			out.RuneSlots = append(out.RuneSlots, RuneSlotView{Name: name, Label: labels[name], Rune: tab.RuneNameOf(name), Choices: calculator2.CharacterRunes(name)})
		}
	}
	if item := tab.DisplayItem; item != nil {
		d := &DisplayItemView{Item: itemTooltip(item), Text: tab.DisplayItemRawText(), Rarity: item.Rarity,
			CanHaveSockets: calculator2.CanHaveAugmentSockets(item), SocketCount: item.ItemSocketCount, Runes: []string{}, RuneChoices: []RuneChoiceView{}}
		if tab.Items[item.ID] != nil {
			d.ID = item.ID
		}
		d.Fits = []string{}
		for _, slot := range tab.OrderedSlots {
			if tab.SlotShown(slot, ring) && tab.IsItemValidForSlot(item, slot.SlotName, nil) {
				d.Fits = append(d.Fits, slot.SlotName)
			}
		}
		for i := 0; i < item.ItemSocketCount; i++ {
			name := "None"
			if i < len(item.Runes) {
				name = item.Runes[i]
			}
			d.Runes = append(d.Runes, name)
		}
		if item.ItemSocketCount > 0 {
			for _, r := range calculator2.GetValidRunesForItem(item) {
				d.RuneChoices = append(d.RuneChoices, RuneChoiceView{Name: r.Name, Lines: r.Lines})
			}
		}
		out.Display = d
	}
	return out, nil
}

// PasteItem opens the item text in the editor, as Ctrl+V on the items tab does.
//
//pob2:block:src/Classes/ItemsTab.lua:1483-1487
func (c *Calculator) PasteItem(raw string) error {
	return c.displayEdit(func(tab *calculator2.ItemsTab) error { return tab.CreateDisplayItemFromRaw(raw, true) })
}

// EditItem opens the build's item in the editor.
func (c *Calculator) EditItem(id int) error {
	return c.displayEdit(func(tab *calculator2.ItemsTab) error { return tab.EditItem(id) })
}

// EditDisplayItemText replaces the editor's item with the text; rarity is for a text with no Rarity line.
func (c *Calculator) EditDisplayItemText(text, rarity string) error {
	return c.displayEdit(func(tab *calculator2.ItemsTab) error {
		return tab.EditDisplayItemText(text, rarity, false, c.additionalRingSlot())
	})
}

// SetDisplayItemRune sets the editor's item's socket (1-based) to the rune.
func (c *Calculator) SetDisplayItemRune(socket int, runeName string) error {
	return c.displayEdit(func(tab *calculator2.ItemsTab) error { return tab.SetDisplayItemRune(socket, runeName) })
}

func (c *Calculator) SetDisplayItemSocketCount(count int) error {
	return c.displayEdit(func(tab *calculator2.ItemsTab) error { return tab.SetDisplayItemSocketCount(count) })
}

// CloseDisplayItem is the editor's Cancel.
//
//pob2:block:src/Classes/ItemsTab.lua:456-458
func (c *Calculator) CloseDisplayItem() error {
	return c.displayEdit(func(tab *calculator2.ItemsTab) error { tab.SetDisplayItem(nil); return nil })
}

// AddDisplayItem adds the editor's item to the build, or saves it over the item it was opened from; slot, unless "",
// equips it there, as dropping it on the slot does.
//
//pob2:block:src/Classes/ItemSlotControl.lua:154-166
func (c *Calculator) AddDisplayItem(slot string) (int, error) {
	id := 0
	err := c.itemsEdit(func(tab *calculator2.ItemsTab) error {
		item := tab.DisplayItem
		if err := tab.AddDisplayItem(slot != "", c.additionalRingSlot()); err != nil {
			return err
		}
		id = item.ID
		if slot != "" {
			return tab.SetSlotItem(slot, id)
		}
		return nil
	})
	return id, err
}

// ItemText is the item's text as the item list copies it.
//
//pob2:block:src/Classes/ItemListControl.lua:332-335
func (c *Calculator) ItemText(id int) (string, error) {
	tab, err := c.itemsTab()
	if err != nil {
		return "", err
	}
	item := tab.Items[id]
	if item == nil {
		return "", itemsError("no such item")
	}
	return item.BuildRaw(), nil
}

type itemsError string

func (e itemsError) Error() string { return string(e) }

func (c *Calculator) DeleteItem(id int) error {
	return c.itemsEdit(func(tab *calculator2.ItemsTab) error { return tab.DeleteItem(id) })
}

// SetSlotItem equips the item in the slot; 0 empties it.
func (c *Calculator) SetSlotItem(slot string, id int) error {
	return c.itemsEdit(func(tab *calculator2.ItemsTab) error { return tab.SetSlotItem(slot, id) })
}

func (c *Calculator) SetSlotActive(slot string, state bool) error {
	return c.itemsEdit(func(tab *calculator2.ItemsTab) error { return tab.SetSlotActive(slot, state) })
}

func (c *Calculator) SetUseSecondWeaponSet(second bool) error {
	return c.itemsEdit(func(tab *calculator2.ItemsTab) error { tab.SetUseSecondWeaponSet(second); return nil })
}

func (c *Calculator) SetCharacterRune(slot, runeName string) error {
	return c.itemsEdit(func(tab *calculator2.ItemsTab) error { return tab.SetCharacterRune(slot, runeName) })
}

func (c *Calculator) SetActiveItemSet(id int) error {
	return c.itemsEdit(func(tab *calculator2.ItemsTab) error { tab.SetActiveItemSet(id); return nil })
}

// NewItemSet adds an empty item set and makes it active.
func (c *Calculator) NewItemSet(title string) error {
	return c.itemsEdit(func(tab *calculator2.ItemsTab) error { tab.NewItemSetService(title); return nil })
}

// CopyItemSet copies the item set under the title and makes the copy active.
func (c *Calculator) CopyItemSet(id int, title string) error {
	return c.itemsEdit(func(tab *calculator2.ItemsTab) error { _, err := tab.CopyItemSetService(id, title); return err })
}

func (c *Calculator) RenameItemSet(id int, title string) error {
	return c.itemsEdit(func(tab *calculator2.ItemsTab) error { return tab.RenameItemSet(id, title) })
}

func (c *Calculator) DeleteItemSet(id int) error {
	return c.itemsEdit(func(tab *calculator2.ItemsTab) error { return tab.DeleteItemSet(id) })
}

// MoveItemSet moves the item set at from (1-based) to to.
func (c *Calculator) MoveItemSet(from, to int) error {
	return c.itemsEdit(func(tab *calculator2.ItemsTab) error { return tab.MoveItemSet(from, to) })
}
