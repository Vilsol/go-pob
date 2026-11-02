package calculator2

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/pob2"
)

// The Items tab's edits over the build's items tab. An edit leaves the build to be calculated again; Save writes the
// tab into the build, as ItemsTab:Save does.

// migrateAugments is main.migrateAugments at its default.
const migrateAugments = true

var (
	itemClassRarityRe  = regexp.MustCompile(`^Item Class: .*\nRarity: `)
	rarityLinePrefixRe = regexp.MustCompile(`Rarity: \w+\n`)
)

type itemsEditError string

func (e itemsEditError) Error() string { return string(e) }

const (
	errNoItem        = itemsEditError("no such item")
	errNoSlot        = itemsEditError("no such slot")
	errNoItemSet     = itemsEditError("no such item set")
	errNoDisplayItem = itemsEditError("no item is open in the editor")
	errNoBase        = itemsEditError("the text is not an item PoB-PoE2 knows")
	errInvalidSlot   = itemsEditError("the item does not fit that slot")
	errNoRune        = itemsEditError("the rune does not fit that socket")
	errLastItemSet   = itemsEditError("the last item set cannot be deleted")
)

// activeItemSet is itemsTab.activeItemSet.
func (tab *ItemsTab) activeItemSet() *ItemSet { return tab.ItemSets[tab.ActiveItemSetID] }

// SetSelItemID is ItemSlotControl:SetSelItemId: a tree socket's jewel goes into the spec, any other slot's item into
// the active item set.
//
//pob2:src/Classes/ItemSlotControl.lua:103-115
func (tab *ItemsTab) SetSelItemID(slot *ItemSlot, selItemID int) {
	if slot.NodeID != 0 {
		if tab.spec != nil {
			tab.spec.Jewels[slot.NodeID] = selItemID
			tab.saveSpecSockets()
		}
	} else {
		tab.activeItemSet().Slots[slot.SlotName].selItemID = selItemID
	}
	slot.SelItemID = selItemID
}

// saveSpecSockets writes the spec's jewels into the active <Spec>, as PassiveSpec:Save does.
func (tab *ItemsTab) saveSpecSockets() {
	specs := tab.build.Tree.Specs
	idx := tab.build.Tree.ActiveSpec
	if idx < 1 || idx > len(specs) {
		idx = 1
	}
	if len(specs) == 0 {
		return
	}
	s := &specs[idx-1]
	if s.Sockets == nil {
		s.Sockets = &pob2.Sockets{}
	}
	s.Sockets.Sockets = s.Sockets.Sockets[:0]
	for _, sock := range tab.spec.Save().Sockets {
		node, _ := strconv.Atoi(sock.NodeID)
		item, _ := strconv.Atoi(sock.ItemID)
		s.Sockets.Sockets = append(s.Sockets.Sockets, pob2.Socket{NodeID: node, ItemID: item})
	}
}

// slotShown is slot:IsShown(); additionalRingSlot is the main calculation's AdditionalRingSlot flag.
//
//pob2:block:src/Classes/ItemsTab.lua:223-249
func (tab *ItemsTab) slotShown(slot *ItemSlot, additionalRingSlot bool) bool {
	if tab.slotInactive(slot) {
		return false
	}
	if slot.ParentSlot != nil {
		if slot.ParentSlot.WeaponSet != 0 {
			return (slot.WeaponSet == 2) == tab.activeItemSet().UseSecondWeaponSet
		}
		return tab.slotShown(slot.ParentSlot, additionalRingSlot)
	}
	switch {
	case slot.WeaponSet == 1:
		return !tab.activeItemSet().UseSecondWeaponSet
	case slot.WeaponSet == 2:
		return tab.activeItemSet().UseSecondWeaponSet
	case slot.SlotName == "Ring 3":
		return additionalRingSlot
	}
	return true
}

// SlotShown is slot:IsShown() over the main calculation's AdditionalRingSlot flag.
func (tab *ItemsTab) SlotShown(slot *ItemSlot, additionalRingSlot bool) bool {
	return tab.slotShown(slot, additionalRingSlot)
}

// slotInactive is slot.inactive: a jewel socket past its item's jewel socket count, or a tree socket off the
// allocated nodes.
//
//pob2:block:src/Classes/ItemSlotControl.lua:136-144
//pob2:block:src/Classes/ItemsTab.lua:1740-1751
func (tab *ItemsTab) slotInactive(slot *ItemSlot) bool {
	if slot.NodeID != 0 {
		return tab.spec == nil || tab.spec.AllocNodes[slot.NodeID] == nil
	}
	if slot.ParentSlot != nil {
		parent := tab.Items[slot.ParentSlot.SelItemID]
		return slot.ParentSlot.SelItemID <= 0 || parent == nil || slot.SlotNum > parent.JewelSocketCount
	}
	return false
}

// PopulateSlots is ItemsTab:PopulateSlots: every slot drops an item that is gone or no longer fits.
//
//pob2:src/Classes/ItemsTab.lua:1733-1737
//pob2:src/Classes/ItemSlotControl.lua:117-148
func (tab *ItemsTab) PopulateSlots() {
	for _, slot := range tab.OrderedSlots {
		item := tab.Items[slot.SelItemID]
		if item == nil || !tab.IsItemValidForSlot(item, slot.SlotName, nil) {
			tab.SetSelItemID(slot, 0)
		}
	}
}

// AddItem adds the item to the build's item list; a new item takes the first unused id and, unless noAutoEquip, goes
// into the first empty shown slot it fits.
//
//pob2:src/Classes/ItemsTab.lua:1768-1806
func (tab *ItemsTab) AddItem(item *Item, noAutoEquip, additionalRingSlot bool) {
	if item.ID <= 0 {
		item.ID = 1
		for tab.Items[item.ID] != nil {
			item.ID++
		}
		tab.ItemOrderList = append(tab.ItemOrderList, item.ID)
		if !noAutoEquip {
			for _, slot := range tab.OrderedSlots {
				if slot.NodeID == 0 && slot.SelItemID == 0 && tab.slotShown(slot, additionalRingSlot) && tab.IsItemValidForSlot(item, slot.SlotName, nil) {
					tab.SetSelItemID(slot, item.ID)
					break
				}
			}
		}
	}
	tab.Items[item.ID] = item
	item.BuildModList()
	// No PoE2 item is a cluster jewel, so a replaced item never rebuilds the cluster jewel graphs.
}

// AddDisplayItem adds the editor's item to the build and closes the editor.
//
//pob2:src/Classes/ItemsTab.lua:1809-1817
func (tab *ItemsTab) AddDisplayItem(noAutoEquip, additionalRingSlot bool) error {
	if tab.DisplayItem == nil {
		return errNoDisplayItem
	}
	tab.AddItem(tab.DisplayItem, noAutoEquip, additionalRingSlot)
	tab.DisplayItem = nil
	tab.PopulateSlots()
	return nil
}

// DeleteItem drops the item from every slot, item set and passive tree; a tree socket that held it is deallocated.
//
//pob2:src/Classes/ItemsTab.lua:1857-1904
func (tab *ItemsTab) DeleteItem(id int) error {
	if tab.Items[id] == nil {
		return errNoItem
	}
	for _, slot := range tab.OrderedSlots {
		if slot.SelItemID == id {
			tab.SetSelItemID(slot, 0)
		}
		if slot.NodeID == 0 {
			for _, set := range tab.ItemSets {
				if ss := set.Slots[slot.SlotName]; ss != nil && ss.selItemID == id {
					ss.selItemID = 0
				}
			}
		}
	}
	for index, itemID := range tab.ItemOrderList {
		if itemID == id {
			tab.ItemOrderList = append(tab.ItemOrderList[:index:index], tab.ItemOrderList[index+1:]...)
			break
		}
	}
	// Every spec of the tree tab: the jewel leaves its socket and the socket node is deallocated. The loop also
	// clears depends' alloc flags by list index, which the saved spec does not keep.
	for si := range tab.build.Tree.Specs {
		spec := &tab.build.Tree.Specs[si]
		if spec.Sockets == nil {
			continue
		}
		for k := range spec.Sockets.Sockets {
			sock := &spec.Sockets.Sockets[k]
			if sock.ItemID != id {
				continue
			}
			sock.ItemID = 0
			if spec.Nodes != nil {
				var kept []string
				for _, n := range strings.Split(*spec.Nodes, ",") {
					if n != strconv.Itoa(sock.NodeID) {
						kept = append(kept, n)
					}
				}
				nodes := strings.Join(kept, ",")
				spec.Nodes = &nodes
			}
		}
	}
	if tab.spec != nil {
		for nodeID, itemID := range tab.spec.Jewels {
			if itemID == id {
				tab.spec.Jewels[nodeID] = 0
			}
		}
	}
	delete(tab.Items, id)
	tab.PopulateSlots()
	return nil
}

// SetSlotItem is choosing an item in a slot's dropdown, which lists the items that fit it; 0 empties the slot.
//
//pob2:block:src/Classes/ItemSlotControl.lua:23-30
func (tab *ItemsTab) SetSlotItem(slotName string, id int) error {
	slot := tab.Slots[slotName]
	if slot == nil {
		return errNoSlot
	}
	if id != 0 {
		item := tab.Items[id]
		if item == nil {
			return errNoItem
		}
		if !tab.IsItemValidForSlot(item, slotName, nil) {
			return errInvalidSlot
		}
	}
	if id != slot.SelItemID {
		tab.SetSelItemID(slot, id)
		tab.PopulateSlots()
	}
	return nil
}

// SetSlotActive is a flask or charm slot's activate checkbox.
//
//pob2:block:src/Classes/ItemSlotControl.lua:58-81
func (tab *ItemsTab) SetSlotActive(slotName string, state bool) error {
	slot := tab.Slots[slotName]
	if slot == nil || !(strings.Contains(slotName, "Flask") || strings.Contains(slotName, "Charm")) {
		return errNoSlot
	}
	slot.Active = state
	tab.activeItemSet().Slots[slotName].active = state
	return nil
}

// SetUseSecondWeaponSet is the Weapon Set I and II buttons.
//
//pob2:block:src/Classes/ItemsTab.lua:347-368
func (tab *ItemsTab) SetUseSecondWeaponSet(second bool) {
	set := tab.activeItemSet()
	set.UseSecondWeaponSet, set.useSecondWeaponSetNil = second, false
	tab.UseSecondWeaponSet = second
}

// SetCharacterRune is a character rune slot's dropdown: a rune that fits Martial Artist slots, or "None".
//
//pob2:block:src/Classes/ItemsTab.lua:267-273
func (tab *ItemsTab) SetCharacterRune(slotName, runeName string) error {
	for _, slot := range characterRuneSlotList {
		if slot[0] != slotName {
			continue
		}
		if runeName != "None" && characterRune(runeName, slot[1]) == nil {
			return errNoRune
		}
		tab.activeItemSet().RuneNames[slotName] = runeName
		tab.updateRuneSlots()
		return nil
	}
	return errNoSlot
}

// CharacterRunes is the runes each character rune slot lists, "None" first.
//
//pob2:block:src/Classes/ItemsTab.lua:190-199
func CharacterRunes(slotName string) []string {
	for _, slot := range characterRuneSlotList {
		if slot[0] != slotName {
			continue
		}
		out := []string{"None"}
		seen := map[string]bool{}
		for _, r := range sortedRuneModLines() {
			if r.CanSocketInChakraSlots && !r.IsSocketBound && (r.Slot == slot[1] || r.Slot == "armour") && !seen[r.Rune] {
				seen[r.Rune] = true
				out = append(out, r.Rune)
			}
		}
		return out
	}
	return nil
}

// CharacterRuneSlots is characterRuneSlotList's slot names, in order.
func CharacterRuneSlots() []string {
	out := make([]string, len(characterRuneSlotList))
	for i, slot := range characterRuneSlotList {
		out[i] = slot[0]
	}
	return out
}

// RuneNameOf is the active item set's runeName for the character rune slot.
func (tab *ItemsTab) RuneNameOf(slotName string) string {
	return tab.activeItemSet().RuneNames[slotName]
}

// SetActiveItemSet makes the item set active: the slots take its selections, and the rune slots its runes.
//
//pob2:src/Classes/ItemsTab.lua:1656-1698
func (tab *ItemsTab) SetActiveItemSet(itemSetID int) {
	if len(tab.ItemSetOrderList) == 0 {
		tab.NewItemSet(1, "Default")
	}
	if tab.ItemSets[itemSetID] == nil {
		itemSetID = tab.ItemSetOrderList[0]
	}
	// The slots and rune slots already write through to the previous set, so it needs no update.
	tab.ActiveItemSetID = itemSetID
	curSet := tab.ItemSets[itemSetID]
	for name, slot := range tab.Slots {
		if slot.NodeID == 0 {
			slot.SelItemID = curSet.Slots[name].selItemID
			slot.Active = curSet.Slots[name].active
		}
	}
	tab.UseSecondWeaponSet = curSet.UseSecondWeaponSet
	tab.updateRuneSlots()
	tab.PopulateSlots()
}

// createItemSetWithID is ItemsTab:CreateItemSet; id 0 is the first unused id.
//
//pob2:src/Classes/ItemsTab.lua:1599-1617
func (tab *ItemsTab) createItemSetWithID(itemSetID int, title string) *ItemSet {
	if itemSetID == 0 {
		itemSetID = 1
		for tab.ItemSets[itemSetID] != nil {
			itemSetID++
		}
	}
	set := &ItemSet{ID: itemSetID, Title: title, Slots: tab.createItemSet(), RuneNames: newRuneNames(), SocketPbURLs: map[int]string{}, useSecondWeaponSetNil: true}
	tab.ItemSets[itemSetID] = set
	return set
}

//pob2:src/Classes/ItemsTab.lua:1619-1624
func (tab *ItemsTab) NewItemSet(itemSetID int, title string) *ItemSet {
	set := tab.createItemSetWithID(itemSetID, title)
	tab.ItemSetOrderList = append(tab.ItemSetOrderList, set.ID)
	return set
}

// luaLen is #t for a table keyed by item set id: LuaJIT's unbound search for a border.
func luaLen(has func(int) bool) int {
	if !has(1) {
		return 0
	}
	i, j := 1, 2
	for has(j) {
		i = j
		j *= 2
	}
	for j-i > 1 {
		m := (i + j) / 2
		if has(m) {
			i = m
		} else {
			j = m
		}
	}
	return i
}

//pob2:src/Classes/ItemsTab.lua:1626-1634
func (tab *ItemsTab) CopyItemSet(sourceItemSetID int, title string) (*ItemSet, error) {
	src := tab.ItemSets[sourceItemSetID]
	if src == nil {
		return nil, errNoItemSet
	}
	set := &ItemSet{Title: title, UseSecondWeaponSet: src.UseSecondWeaponSet, useSecondWeaponSetNil: src.useSecondWeaponSetNil, Slots: map[string]*itemSetSlot{}, RuneNames: map[string]string{}, SocketPbURLs: map[int]string{}}
	for name, ss := range src.Slots {
		c := *ss
		set.Slots[name] = &c
	}
	for name, r := range src.RuneNames {
		set.RuneNames[name] = r
	}
	for id, url := range src.SocketPbURLs {
		set.SocketPbURLs[id] = url
	}
	set.ID = luaLen(func(id int) bool { return tab.ItemSets[id] != nil }) + 1
	tab.ItemSetOrderList = append(tab.ItemSetOrderList, set.ID)
	tab.ItemSets[set.ID] = set
	return set, nil
}

//pob2:src/Classes/ItemsTab.lua:1637-1646
func (tab *ItemsTab) RenameItemSet(itemSetID int, title string) error {
	set := tab.ItemSets[itemSetID]
	if set == nil {
		return errNoItemSet
	}
	set.Title = title
	return nil
}

// NewItemSetService is ItemSetService:NewItemSet: a new set, made active.
//
//pob2:src/Classes/ItemSetService.lua:17-23
func (tab *ItemsTab) NewItemSetService(title string) int {
	set := tab.NewItemSet(0, title)
	tab.SetActiveItemSet(set.ID)
	return set.ID
}

//pob2:src/Classes/ItemSetService.lua:25-31
func (tab *ItemsTab) CopyItemSetService(itemSetID int, title string) (int, error) {
	set, err := tab.CopyItemSet(itemSetID, title)
	if err != nil {
		return 0, err
	}
	tab.SetActiveItemSet(set.ID)
	return set.ID, nil
}

// DeleteItemSet is ItemSetService:DeleteItemSet: the last set stays, and deleting the active set activates the one
// before it.
//
//pob2:src/Classes/ItemSetService.lua:40-50
//pob2:src/Classes/ItemsTab.lua:1649-1653
func (tab *ItemsTab) DeleteItemSet(itemSetID int) error {
	if tab.ItemSets[itemSetID] == nil {
		return errNoItemSet
	}
	if len(tab.ItemSetOrderList) <= 1 {
		return errLastItemSet
	}
	orderListIndex := 0
	for i, id := range tab.ItemSetOrderList {
		if id == itemSetID {
			orderListIndex = i + 1
		}
	}
	tab.ItemSetOrderList = append(tab.ItemSetOrderList[:orderListIndex-1:orderListIndex-1], tab.ItemSetOrderList[orderListIndex:]...)
	delete(tab.ItemSets, itemSetID)
	if itemSetID == tab.ActiveItemSetID {
		tab.SetActiveItemSet(tab.ItemSetOrderList[max(1, orderListIndex-1)-1])
	}
	return nil
}

// MoveItemSet is dragging an item set in the manage list: the set at from (1-based) moves to to.
//
//pob2:block:src/Classes/ListControl.lua:440-448
func (tab *ItemsTab) MoveItemSet(from, to int) error {
	list := tab.ItemSetOrderList
	if from < 1 || from > len(list) || to < 1 || to > len(list) {
		return errNoItemSet
	}
	id := list[from-1]
	list = append(list[:from-1:from-1], list[from:]...)
	list = append(list[:to-1:to-1], append([]int{id}, list[to-1:]...)...)
	tab.ItemSetOrderList = list
	return nil
}

// RuneChoice is one entry of GetValidRunesForItem: a rune or soul core, its lines over every slot entry that fits.
type RuneChoice struct {
	Name          string
	Lines         []string
	Req           float64
	Limit         float64
	IsSocketBound bool
}

// sortedRuneModLines is runeModLines without "None", in its sort order.
//
//pob2:block:src/Classes/ItemsTab.lua:2248-2283
func sortedRuneModLines() []poe2.RuneMod {
	type entry struct {
		rune         poe2.RuneMod
		order, group float64
	}
	entries := make([]entry, len(poe2.RuneMods))
	for i, r := range poe2.RuneMods {
		order := 0.0
		if len(r.StatOrder) > 0 {
			order = r.StatOrder[0]
		} else if len(r.BondedStatOrder) > 0 {
			order = r.BondedStatOrder[0]
		}
		entries[i] = entry{rune: r, order: order, group: float64(len(r.Lines) + len(r.Bonded))}
	}
	sort.SliceStable(entries, func(a, b int) bool {
		x, y := entries[a], entries[b]
		switch {
		case x.order == y.order:
			return x.rune.LevelReq < y.rune.LevelReq
		case x.group == y.group:
			return x.order < y.order
		default:
			return x.group < y.group
		}
	})
	out := make([]poe2.RuneMod, len(entries))
	for i, e := range entries {
		out[i] = e.rune
	}
	return out
}

// GetValidRunesForItem is the runes and soul cores that fit the item's sockets, "None" first.
//
//pob2:src/Classes/ItemsTab.lua:2285-2323
func GetValidRunesForItem(item *Item) []*RuneChoice {
	runes := []*RuneChoice{{Name: "None", Lines: []string{"None"}, Req: 1}}
	added := map[string]*RuneChoice{}
	socketedItemType := ""
	if item.BaseModList != nil {
		if flagValue(item.BaseModList, "SocketedSoulCoresOnly") {
			socketedItemType = "SoulCore"
		} else if flagValue(item.BaseModList, "SocketedRunesOnly") {
			socketedItemType = "Rune"
		}
	}
	baseType, specificType := item.GetSocketedAugmentTypes()
	uniqueItem := item.Rarity == "UNIQUE" || item.Rarity == "RELIC"
	jewellery := item.Type == "Ring" || item.Type == "Amulet" || item.Type == "Belt"
	for _, r := range sortedRuneModLines() {
		if (r.Slot == baseType || r.Slot == specificType || (r.Type == "SoulCore" && item.SocketedSoulCoreTypes[r.Slot])) &&
			(socketedItemType == "" || r.Type == socketedItemType) && (!uniqueItem || r.CanSocketInUniqueItems) && (!jewellery || r.CanSocketInJewellery) {
			choice := added[r.Rune]
			if choice == nil {
				choice = &RuneChoice{Name: r.Rune, Req: r.LevelReq, Limit: r.Limit, IsSocketBound: r.IsSocketBound}
				runes = append(runes, choice)
				added[r.Rune] = choice
			}
			choice.Lines = append(choice.Lines, r.Lines...)
			for _, line := range r.Bonded {
				choice.Lines = append(choice.Lines, runeBondedPrefix+line)
			}
		}
	}
	return runes
}

//pob2:src/Classes/ItemsTab.lua:2325-2335
func isSocketBoundRune(runeName string, validRunes []*RuneChoice) bool {
	if runeName == "" || runeName == "None" {
		return false
	}
	for _, r := range validRunes {
		if r.Name == runeName {
			return r.IsSocketBound
		}
	}
	return false
}

// updateRuneControls is UpdateRuneControls' selection half: a socket whose rune does not fit takes "None".
//
//pob2:src/Classes/ItemsTab.lua:2338-2366
func updateRuneControls(item *Item) {
	runes := GetValidRunesForItem(item)
	runesUpdated := false
	for i := 0; i < item.ItemSocketCount && i < len(item.Runes); i++ {
		found := false
		for _, r := range runes {
			if r.Name == item.Runes[i] {
				found = true
			}
		}
		if !found {
			item.Runes[i] = "None"
			runesUpdated = true
		}
	}
	if runesUpdated {
		item.UpdateRunes()
	}
}

//pob2:block:src/Classes/ItemsTab.lua:1906-1912
func isAnointable(item *Item) bool { return item.CanBeAnointed || item.Base.Type == "Amulet" }
func isAugmentable(item *Item) bool {
	return len(item.Sockets) > 0 || item.Base.SocketLimit > 0
}

// copyAnointsAndAugments gives a new item the anoint and runes of the item it would replace.
//
//pob2:src/Classes/ItemsTab.lua:1914-1995
func (tab *ItemsTab) copyAnointsAndAugments(newItem *Item, copyAugments, overwrite bool) {
	isWeapon := hasTag(newItem.Base, "onehand") || hasTag(newItem.Base, "twohand")
	isShield := hasTag(newItem.Base, "shield") || hasTag(newItem.Base, "focus")
	newItemType := newItem.Base.Type
	if isWeapon {
		newItemType = "Weapon 1"
	} else if isShield {
		newItemType = "Weapon 2"
	}
	if newItem.Name == "Tabula Rasa, Garment" {
		return
	}
	setSlot := tab.activeItemSet().Slots[newItemType]
	if setSlot == nil {
		return
	}
	currentItem := tab.Items[setSlot.selItemID]
	if currentItem == nil {
		return
	}
	modifiableItem := !(newItem.Corrupted || newItem.Mirrored || newItem.Sanctified)
	if isAnointable(newItem) && (len(newItem.EnchantModLines) == 0 || overwrite) && setSlot.selItemID > 0 && modifiableItem {
		newItem.EnchantModLines = currentItem.EnchantModLines
	}
	hasEmptySockets := true
	for _, r := range newItem.Runes {
		if r != "None" {
			hasEmptySockets = false
			break
		}
	}
	shouldChangeAugments := copyAugments && isAugmentable(newItem) && (len(newItem.Runes) == 0 || hasEmptySockets || overwrite)
	var currentRunes []string
	for _, r := range currentItem.Runes {
		if r != "None" {
			currentRunes = append(currentRunes, r)
		}
	}
	if shouldChangeAugments && isAugmentable(newItem) && modifiableItem && len(newItem.Sockets) < len(currentItem.Sockets) {
		maxSockets := min(len(currentItem.Sockets), int(newItem.Base.SocketLimit))
		neededSockets := max(0, maxSockets-len(newItem.Sockets))
		for range neededSockets {
			newItem.Runes = append(newItem.Runes, "None")
			newItem.Sockets = append(newItem.Sockets, ItemSocket{Group: len(newItem.Sockets) + 1})
		}
		newItem.ItemSocketCount = len(newItem.Sockets)
		newItem.UpdateRunes()
	}
	validRunes := GetValidRunesForItem(newItem)
	skipped := 0
	if shouldChangeAugments {
		for i := range newItem.Sockets {
			r := ""
			if i < len(newItem.Runes) {
				r = newItem.Runes[i]
			}
			if isSocketBoundRune(r, validRunes) {
				skipped++
				continue
			}
			for len(newItem.Runes) <= i {
				newItem.Runes = append(newItem.Runes, "")
			}
			newItem.Runes[i] = "None"
			if i-skipped < len(currentRunes) {
				for _, v := range validRunes {
					if v.Name == currentRunes[i-skipped] {
						newItem.Runes[i] = currentRunes[i-skipped]
						break
					}
				}
			}
		}
		newItem.UpdateRunes()
	}
	newItem.BuildAndParseRaw()
}

// CreateDisplayItemFromRaw opens the item text in the editor; normalise sets its quality as a new item's.
//
//pob2:src/Classes/ItemsTab.lua:1998-2008
func (tab *ItemsTab) CreateDisplayItemFromRaw(itemRaw string, normalise bool) error {
	newItem := ParseItem(itemRaw, tab.modSource)
	if newItem.Base == nil {
		return errNoBase
	}
	tab.copyAnointsAndAugments(newItem, migrateAugments, false)
	if normalise {
		newItem.NormaliseQuality()
		newItem.BuildModList()
	}
	tab.SetDisplayItem(newItem)
	return nil
}

// SetDisplayItem opens the item in the editor; nil closes it.
//
//pob2:block:src/Classes/ItemsTab.lua:2096-2160
func (tab *ItemsTab) SetDisplayItem(item *Item) {
	tab.DisplayItem = item
	if item != nil {
		updateRuneControls(item)
	}
}

// EditItem opens a copy of the build's item in the editor, as double-clicking it in the item list does.
//
//pob2:block:src/Classes/ItemListControl.lua:321-328
func (tab *ItemsTab) EditItem(id int) error {
	item := tab.Items[id]
	if item == nil {
		return errNoItem
	}
	newItem := ParseItem(item.BuildRaw(), tab.modSource)
	newItem.ID = item.ID
	tab.SetDisplayItem(newItem)
	return nil
}

// EditDisplayItemText is the item text editor's Save: the text replaces the editor's item, keeping its id; rarity is
// the editor's rarity for a text with no Rarity line.
//
//pob2:src/Classes/ItemsTab.lua:2871-2899
func (tab *ItemsTab) EditDisplayItemText(text, rarity string, alsoAddItem, additionalRingSlot bool) error {
	raw := text
	if !(itemClassRarityRe.MatchString(text) || strings.HasPrefix(text, "Rarity: ")) {
		raw = "Rarity: " + rarity + "\n" + text
	}
	id := 0
	if tab.DisplayItem != nil {
		id = tab.DisplayItem.ID
	}
	if err := tab.CreateDisplayItemFromRaw(raw, tab.DisplayItem == nil); err != nil {
		return err
	}
	tab.DisplayItem.ID = id
	if alsoAddItem {
		return tab.AddDisplayItem(false, additionalRingSlot)
	}
	return nil
}

// DisplayItemRawText is the editor's text for the item, without its Rarity line.
//
//pob2:block:src/Classes/ItemsTab.lua:2884
func (tab *ItemsTab) DisplayItemRawText() string {
	if tab.DisplayItem == nil {
		return ""
	}
	return rarityLinePrefixRe.ReplaceAllString(tab.DisplayItem.BuildRaw(), "")
}

// SetDisplayItemRune is the editor's Rune #i dropdown (1-based).
//
//pob2:block:src/Classes/ItemsTab.lua:792-800
func (tab *ItemsTab) SetDisplayItemRune(i int, runeName string) error {
	item := tab.DisplayItem
	if item == nil {
		return errNoDisplayItem
	}
	if i < 1 || i > item.ItemSocketCount || i > 6 {
		return errNoSlot
	}
	found := false
	for _, r := range GetValidRunesForItem(item) {
		found = found || r.Name == runeName
	}
	if !found {
		return errNoRune
	}
	for len(item.Runes) < i {
		item.Runes = append(item.Runes, "None")
	}
	item.Runes[i-1] = runeName
	item.UpdateRunes()
	item.BuildAndParseRaw()
	return nil
}

// SetDisplayItemSocketCount is the editor's socket count; more than 6 is 6.
//
//pob2:block:src/Classes/ItemsTab.lua:613-624
func (tab *ItemsTab) SetDisplayItemSocketCount(count int) error {
	item := tab.DisplayItem
	if item == nil {
		return errNoDisplayItem
	}
	count = min(max(count, 0), 6)
	item.ItemSocketCount = count
	item.UpdateRunes()
	item.BuildAndParseRaw()
	updateRuneControls(item)
	return nil
}

// CanHaveAugmentSockets is whether the editor shows the item's socket count.
//
//pob2:block:src/Classes/ItemsTab.lua:77-83
func CanHaveAugmentSockets(item *Item) bool {
	return item != nil && item.Base != nil && (item.Base.SocketLimit > 0 || item.SocketedAugmentTypeOverride != "" || item.ItemSocketCount > 0)
}

// Save writes the items tab into the build: every item's text as BuildRaw writes it, then the item sets.
//
//pob2:src/Classes/ItemsTab.lua:1349-1439
func (tab *ItemsTab) Save(build *pob2.PathOfBuilding2) {
	out := &build.Items
	active := tab.activeItemSet()
	out.ActiveItemSet = tab.ActiveItemSetID
	out.UseSecondWeaponSet = pob2.Nil[bool]{V: active.UseSecondWeaponSet, IsNil: active.useSecondWeaponSetNil}
	show := tab.ShowStatDifferences
	out.ShowStatDifferences = &show
	out.Items = out.Items[:0:0]
	out.Slots = nil
	for _, id := range tab.ItemOrderList {
		item := tab.Items[id]
		child := pob2.Item{ID: id, Variant: item.Variant}
		if item.HasAltVariant {
			child.VariantAlt = item.VariantAlt
		}
		if item.HasAltVariant2 {
			child.VariantAlt2 = item.VariantAlt2
		}
		if item.HasAltVariant3 {
			child.VariantAlt3 = item.VariantAlt3
		}
		if item.HasAltVariant4 {
			child.VariantAlt4 = item.VariantAlt4
		}
		if item.HasAltVariant5 {
			child.VariantAlt5 = item.VariantAlt5
		}
		item.BuildAndParseRaw()
		child.Raw = item.Raw
		rangeID := len(item.BuffModLines) + 1
		for _, list := range [][]*ModLine{item.EnchantModLines, item.RuneModLines, item.ImplicitModLines, item.ExplicitModLines} {
			for _, modLine := range list {
				if modLine.Range != nil {
					child.ModRanges = append(child.ModRanges, pob2.ModRange{ID: rangeID, Range: *modLine.Range})
				}
				rangeID++
			}
		}
		out.Items = append(out.Items, child)
	}
	out.ItemSets = out.ItemSets[:0:0]
	for _, itemSetID := range tab.ItemSetOrderList {
		itemSet := tab.ItemSets[itemSetID]
		title := itemSet.Title
		child := pob2.ItemSet{ID: itemSetID, Title: &title, UseSecondWeaponSet: pob2.Nil[bool]{V: itemSet.UseSecondWeaponSet, IsNil: itemSet.useSecondWeaponSetNil}}
		for _, slot := range tab.OrderedSlots {
			ss := itemSet.Slots[slot.SlotName]
			if slot.ParentSlot != nil && ss.selItemID == 0 {
				continue
			}
			if slot.NodeID == 0 {
				s := pob2.Slot{Name: slot.SlotName, ItemID: ss.selItemID, Note: ss.note}
				url := ss.pbURL
				s.ItemPbURL = &url
				if ss.active {
					t := true
					s.Active = &t
				}
				child.Slots = append(child.Slots, s)
			} else if tab.spec != nil && tab.spec.AllocNodes[slot.NodeID] != nil {
				url := itemSet.SocketPbURLs[slot.NodeID]
				child.SocketIDURLs = append(child.SocketIDURLs, pob2.SocketIDURL{Name: slot.SlotName, NodeID: slot.NodeID, ItemPbURL: &url})
			}
		}
		for _, slot := range characterRuneSlotList {
			runeName := itemSet.RuneNames[slot[0]]
			if runeName == "" {
				runeName = "None"
			}
			child.RuneSlots = append(child.RuneSlots, pob2.RuneSlot{SlotName: slot[0], RuneName: runeName})
		}
		out.ItemSets = append(out.ItemSets, child)
	}
	// Load keeps only the weights of stats in data.powerStatList, which the trade query is not ported to read; the
	// loaded ones go back with a weight above 0.
	weights := &pob2.TradeSearchWeights{}
	if tab.tradeSearchWeights != nil {
		for _, st := range tab.tradeSearchWeights.Stats {
			if st.WeightMult == nil {
				continue
			}
			if w, ok := luaToNumber(*st.WeightMult); ok && w > 0 {
				st.Unparsed = pob2.Unparsed{}
				f := strconv.FormatFloat(w, 'f', 2, 64)
				st.WeightMult = &f
				weights.Stats = append(weights.Stats, st)
			}
		}
	}
	out.TradeSearchWeights = weights
}
