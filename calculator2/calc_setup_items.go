package calculator2

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob2"
)

//pob2:block:src/Classes/ItemsTab.lua:34-40
var characterRuneSlotList = [][2]string{
	{"Helmet Rune #1", "helmet"},
	{"Body Armour Rune #1", "body armour"},
	{"Body Armour Rune #2", "body armour"},
	{"Gloves Rune #1", "gloves"},
	{"Boots Rune #1", "boots"},
}

//pob2:block:src/Classes/ItemsTab.lua:32
var baseSlots = []string{"Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring 1", "Ring 2", "Ring 3", "Belt", "Charm 1", "Charm 2", "Charm 3", "Flask 1", "Flask 2", "Arm 1", "Arm 2", "Leg 1", "Leg 2"}

var (
	slotNumTrailingRE = regexp.MustCompile(`\d+$`)
	slotNumAnyRE      = regexp.MustCompile(`\d+`)
	slotTypeIDRE      = regexp.MustCompile(`^([A-Za-z ]+) (\d+)$`)
	jewelSocketSuffix = regexp.MustCompile(` Jewel Socket \d`)
	handedTypeRE      = regexp.MustCompile(`.+Handed`)
)

// ItemSlot is one ItemSlotControl: an equipment slot, a socket on an equipped item, or a tree socket.
type ItemSlot struct {
	SlotName string
	// SlotNum is tonumber(slotName:match("%d+$") or slotName:match("%d+")); 0 is nil.
	SlotNum int
	// WeaponSet is 1 or 2 on the weapon slots and their sockets; 0 is nil.
	WeaponSet  int
	NodeID     int
	ParentSlot *ItemSlot
	SelItemID  int
	Active     bool
}

type itemSetSlot struct {
	selItemID int
	active    bool
}

// ItemsTab is build.itemsTab after ItemsTab:Load: the items, the slots in orderedSlots order, and the
// active item set's selections.
type ItemsTab struct {
	Items         map[int]*Item
	ItemOrderList []int
	Slots         map[string]*ItemSlot
	OrderedSlots  []*ItemSlot
	Sockets       map[int]*ItemSlot

	UseSecondWeaponSet bool
	// RuneSlots is each character rune slot's selected rune; a slot holding "None" is absent.
	RuneSlots map[string]*poe2.RuneMod

	tree *tree2.PassiveTree
	spec *tree2.PassiveSpec
}

func slotNumOf(slotName string) int {
	s := slotNumTrailingRE.FindString(slotName)
	if s == "" {
		s = slotNumAnyRE.FindString(slotName)
	}
	n, _ := strconv.Atoi(s)
	return n
}

// NewItemsTab is ItemsTab:ItemsTab's slot construction followed by ItemsTab:Load of the build's
// <Items>, each item text parsed with modSource (nil is WithParser(ParseMod)).
//
//pob2:block:src/Classes/ItemsTab.lua:201-345
func NewItemsTab(build *pob2.PathOfBuilding2, tree *tree2.PassiveTree, modSource ModSource) *ItemsTab {
	if modSource == nil {
		modSource = WithParser(ParseMod)
	}
	tab := &ItemsTab{
		Items:   map[int]*Item{},
		Slots:   map[string]*ItemSlot{},
		Sockets: map[int]*ItemSlot{},
		tree:    tree,
	}
	addSlot := func(slot *ItemSlot) {
		slot.SlotNum = slotNumOf(slot.SlotName)
		tab.Slots[slot.SlotName] = slot
		tab.OrderedSlots = append(tab.OrderedSlots, slot)
	}
	addJewelSockets := func(parent *ItemSlot) {
		for i := 1; i <= 6; i++ {
			addSlot(&ItemSlot{SlotName: parent.SlotName + " Jewel Socket " + strconv.Itoa(i), ParentSlot: parent, WeaponSet: parent.WeaponSet})
		}
	}
	for _, slotName := range baseSlots {
		slot := &ItemSlot{SlotName: slotName}
		addSlot(slot)
		var swapSlot *ItemSlot
		if strings.Contains(slotName, "Weapon") {
			slot.WeaponSet = 1
			swapSlot = &ItemSlot{SlotName: slotName + " Swap", WeaponSet: 2}
			addSlot(swapSlot)
		}
		switch slotName {
		case "Weapon 1", "Weapon 2", "Helmet", "Gloves", "Body Armour", "Boots", "Belt", "Ring 1", "Ring 2", "Ring 3":
			addJewelSockets(slot)
			if swapSlot != nil {
				addJewelSockets(swapSlot)
			}
		}
	}
	if tree != nil {
		var socketOrder []int
		for id, node := range tree.Nodes {
			if node.Type == "Socket" || node.ContainJewelSocket {
				socketOrder = append(socketOrder, id)
			}
		}
		sort.Ints(socketOrder)
		for _, id := range socketOrder {
			slot := &ItemSlot{SlotName: "Jewel " + strconv.Itoa(id), NodeID: id}
			tab.Sockets[id] = slot
			addSlot(slot)
		}
	}
	tab.load(build, modSource)
	return tab
}

//pob2:src/Classes/ItemsTab.lua:1220-1347
func (tab *ItemsTab) load(build *pob2.PathOfBuilding2, modSource ModSource) {
	items := build.Items
	for _, it := range items.Items {
		ranges := make([]ItemModRange, 0, len(it.ModRanges))
		for _, r := range it.ModRanges {
			ranges = append(ranges, ItemModRange{ID: r.ID, Range: r.Range})
		}
		item := LoadItem(it.ID, it.Raw, ItemVariants{
			Variant: it.Variant, VariantAlt: it.VariantAlt, VariantAlt2: it.VariantAlt2,
			VariantAlt3: it.VariantAlt3, VariantAlt4: it.VariantAlt4, VariantAlt5: it.VariantAlt5,
		}, ranges, ItemModSource(modSource))
		if item.Base != nil {
			tab.Items[item.ID] = item
			tab.ItemOrderList = append(tab.ItemOrderList, item.ID)
		}
	}
	for _, s := range items.Slots {
		if slot, ok := tab.Slots[s.Name]; ok {
			slot.SelItemID = s.ItemID
			if strings.Contains(s.Name, "Flask") || strings.Contains(s.Name, "Charm") {
				slot.Active = s.Active != nil && *s.Active
			}
		}
	}
	itemSets := map[int]map[string]*itemSetSlot{}
	useSecond := map[int]bool{}
	var itemSetOrderList []int
	for _, set := range items.ItemSets {
		itemSet := tab.createItemSet()
		itemSets[set.ID] = itemSet
		useSecond[set.ID] = !set.UseSecondWeaponSet.IsNil && set.UseSecondWeaponSet.V
		for _, s := range set.Slots {
			if ss, ok := itemSet[s.Name]; ok {
				ss.selItemID = s.ItemID
				ss.active = s.Active != nil && *s.Active
			}
		}
		itemSetOrderList = append(itemSetOrderList, set.ID)
	}
	if len(itemSetOrderList) == 0 {
		// SetActiveItemSet's prevSet is this new set, so the legacy <Slot> selections carry into it.
		itemSets[1] = tab.createItemSet()
		for name, ss := range itemSets[1] {
			ss.selItemID, ss.active = tab.Slots[name].SelItemID, tab.Slots[name].Active
		}
		useSecond[1] = !items.UseSecondWeaponSet.IsNil && items.UseSecondWeaponSet.V
		itemSetOrderList = append(itemSetOrderList, 1)
	}
	activeID := items.ActiveItemSet
	if activeID == 0 {
		activeID = 1
	}

	//pob2:block:src/Classes/ItemsTab.lua:1656-1694
	if _, ok := itemSets[activeID]; !ok {
		activeID = itemSetOrderList[0]
	}
	cur := itemSets[activeID]
	tab.UseSecondWeaponSet = useSecond[activeID]
	for name, slot := range tab.Slots {
		if slot.NodeID == 0 {
			slot.SelItemID = cur[name].selItemID
			slot.Active = cur[name].active
		}
	}
	tab.RuneSlots = map[string]*poe2.RuneMod{}
	for _, set := range items.ItemSets {
		if set.ID != activeID {
			continue
		}
		for _, rs := range set.RuneSlots {
			for _, slot := range characterRuneSlotList {
				if slot[0] == rs.SlotName {
					if r := characterRune(rs.RuneName, slot[1]); r != nil {
						tab.RuneSlots[rs.SlotName] = r
					}
				}
			}
		}
	}
	tab.populateSlots()
}

// characterRune is runeSlot:SelByValue(runeName, "name") over the slot's list: the runes that fit
// Martial Artist slots and are not socket-bound. A name not in the list leaves "None" selected.
//
//pob2:block:src/Classes/ItemsTab.lua:191-199
func characterRune(name, slotType string) *poe2.RuneMod {
	for i, r := range poe2.RuneModsByName(name) {
		if r.CanSocketInChakraSlots && !r.IsSocketBound && (r.Slot == slotType || r.Slot == "armour") {
			return &poe2.RuneModsByName(name)[i]
		}
	}
	return nil
}

//pob2:src/Classes/ItemsTab.lua:1599-1617
func (tab *ItemsTab) createItemSet() map[string]*itemSetSlot {
	set := map[string]*itemSetSlot{}
	for name, slot := range tab.Slots {
		if slot.NodeID == 0 {
			set[name] = &itemSetSlot{}
		}
	}
	return set
}

// SetActiveSpec is TreeTab:SetActiveSpec's socket half: each tree socket takes the spec's jewel, then
// the slots are repopulated against the spec.
//
//pob2:block:src/Classes/TreeTab.lua:550-569
func (tab *ItemsTab) SetActiveSpec(spec *tree2.PassiveSpec) {
	tab.spec = spec
	for _, slot := range tab.OrderedSlots {
		if slot.NodeID != 0 {
			slot.SelItemID = spec.Jewels[slot.NodeID]
		}
	}
	if len(tab.ItemOrderList) > 0 {
		tab.populateSlots()
	}
}

// populateSlots is PopulateSlots' selection half: a slot whose item is missing or invalid is emptied.
//
//pob2:src/Classes/ItemSlotControl.lua:117-134
func (tab *ItemsTab) populateSlots() {
	for _, slot := range tab.OrderedSlots {
		item := tab.Items[slot.SelItemID]
		if item == nil || !tab.IsItemValidForSlot(item, slot.SlotName, nil) {
			slot.SelItemID = 0
			if slot.NodeID != 0 && tab.spec != nil {
				//pob2:block:src/Classes/ItemSlotControl.lua:104-106
				tab.spec.Jewels[slot.NodeID] = 0
			}
		}
	}
}

// weaponFlagState is CalcSetup's weaponFlagState; nil is IsItemValidForSlot's no-calc default, where
// all three are assumed.
type weaponFlagState struct {
	giantsBlood, instrumentsOfPower, lordOfTheWilds bool
}

func (tab *ItemsTab) specNode(id int) (*tree2.Node, *tree2.SpecNode) {
	if tab.spec != nil {
		if n, ok := tab.spec.Tree.Nodes[id]; ok {
			return n, tab.spec.Nodes[id]
		}
		if sn, ok := tab.spec.Nodes[id]; ok {
			return sn.Node, sn
		}
		return nil, nil
	}
	if tab.tree != nil {
		return tab.tree.Nodes[id], nil
	}
	return nil, nil
}

// IsItemValidForSlot reads the active item set's selections through the slots, which hold them.
//
//pob2:src/Classes/ItemsTab.lua:2640-2724
func (tab *ItemsTab) IsItemValidForSlot(item *Item, slotName string, flagState *weaponFlagState) bool {
	slotType, slotID := slotName, ""
	if m := slotTypeIDRE.FindStringSubmatch(slotName); m != nil {
		slotType, slotID = m[1], m[2]
	}
	if slotType == "Jewel" {
		id, _ := strconv.Atoi(slotID)
		node, _ := tab.specNode(id)
		switch {
		case node == nil || item.Type != "Jewel":
			return false
		case node.Sinister && (item.Rarity == "UNIQUE" || item.Rarity == "RELIC"):
			return false
		case node.ContainJewelSocket:
			return !(item.Rarity == "UNIQUE" || item.Rarity == "RELIC" || (item.Base != nil && item.Base.SubType != ""))
		case isCharmSocket(tab, node) || item.Base.SubType == "Charm":
			return isCharmSocket(tab, node) && item.Base.SubType == "Charm"
		default:
			// No PoE2 node carries expansionJewel and no item clusterJewel, so every other socket fits.
			return true
		}
	}
	switch {
	case item.Type == "Flask" && slotType == "Flask":
		if strings.Contains(item.BaseName, "Life Flask") && strings.Contains(slotName, "Flask 1") {
			return true
		} else if strings.Contains(item.BaseName, "Mana Flask") && strings.Contains(slotName, "Flask 2") {
			return true
		}
	case item.Base.SubType == "Transcendent Arm" && slotType == "Arm":
		return true
	case item.Base.SubType == "Transcendent Leg" && slotType == "Leg":
		return true
	case item.Type == slotType:
		return true
	case item.Type == "Jewel" && strings.Contains(slotName, "Jewel Socket") && item.Rarity != "UNIQUE":
		parentSlotName := jewelSocketSuffix.ReplaceAllString(slotName, "")
		var slotItem *Item
		if parent, ok := tab.Slots[parentSlotName]; ok && parent.NodeID == 0 {
			slotItem = tab.Items[parent.SelItemID]
		}
		if slotItem != nil && (slotItem.CanSocketJewelBase == nil || slotItem.CanSocketJewelBase[item.BaseName]) {
			return true
		}
	case slotName == "Weapon 1" || slotName == "Weapon 1 Swap" || slotName == "Weapon":
		return hasTag(item.Base, "onehand") || hasTag(item.Base, "twohand")
	case slotName == "Weapon 2" || slotName == "Weapon 2 Swap":
		w1Name := "Weapon 1 Swap"
		if slotName == "Weapon 2" {
			w1Name = "Weapon 1"
		}
		var weapon1Base *poe2.ItemBase
		if w1 := tab.Items[tab.Slots[w1Name].SelItemID]; w1 != nil {
			weapon1Base = w1.Base
		}
		giantsBlood, instrumentsOfPower, lordOfTheWilds := true, true, true
		if flagState != nil {
			giantsBlood, instrumentsOfPower, lordOfTheWilds = flagState.giantsBlood, flagState.instrumentsOfPower, flagState.lordOfTheWilds
		}
		switch {
		case weapon1Base != nil && weapon1Base.Type == "Bow":
			return item.Type == "Quiver"
		case weapon1Base != nil && weapon1Base.Type == "Talisman" && lordOfTheWilds:
			return item.Type == "Sceptre" && item.Rarity != "UNIQUE" && item.Rarity != "RELIC"
		case weapon1Base != nil && weapon1Base.Type == "Staff" && instrumentsOfPower:
			return item.Type == "Focus"
		case weapon1Base == nil || hasTag(weapon1Base, "onehand") || (giantsBlood && (hasTag(weapon1Base, "axe") || hasTag(weapon1Base, "mace") || hasTag(weapon1Base, "sword"))):
			w1Type := ""
			if weapon1Base != nil {
				w1Type = weapon1Base.Type
			}
			return item.Type == "Shield" || item.Type == "Focus" || item.Type == "Sceptre" ||
				(hasTag(item.Base, "one_hand_weapon") && w1Type != "Wand" && w1Type != "Sceptre" && item.Type != "Spear") ||
				(giantsBlood && (hasTag(item.Base, "axe") || hasTag(item.Base, "mace") || hasTag(item.Base, "sword")))
		}
	}
	return false
}

//pob2:block:src/Classes/PassiveTree.lua:370-372
func isCharmSocket(tab *ItemsTab, node *tree2.Node) bool {
	tree := tab.tree
	if tab.spec != nil {
		tree = tab.spec.Tree
	}
	if tree == nil {
		return false
	}
	_, socket := tree.Sockets[node.ID]
	return socket && node.Name == "Charm Socket"
}

// SpecItem is the spec's view of build.itemsTab.items[itemID]: jewel radius and jewelData.
func (tab *ItemsTab) SpecItem(itemID int) *tree2.Item {
	item := tab.Items[itemID]
	if item == nil {
		return nil
	}
	out := &tree2.Item{JewelRadiusIndex: item.JewelRadiusIndex}
	if item.JewelData != nil {
		jd := &tree2.JewelData{FromNothingKeystones: map[string]bool{}}
		jd.IntuitiveLeapLike, _ = item.JewelData["intuitiveLeapLike"].(bool)
		jd.FromNothingKeystone, _ = item.JewelData["fromNothingKeystone"].(string)
		jd.AlternateClassStart, _ = item.JewelData["alternateClassStart"].(string)
		jd.LimitDisabled, _ = item.JewelData["limitDisabled"].(bool)
		if fn, ok := item.JewelData["fromNothingKeystones"].(map[string]any); ok {
			for k, v := range fn {
				if b, ok := v.(bool); ok && b {
					jd.FromNothingKeystones[k] = true
				}
			}
		}
		out.JewelData = jd
	}
	return out
}

func multAdd(db *moddb.ModDB, name string, delta float64) {
	db.Multipliers[name] += delta
}

// mergeItems is initEnv's item build and merge, with the jewel sockets items grant. Left out: override paths (repSlotName, toggleFlask),
// weaponFlagState (read only by those overrides and by ValidateWeaponSlots, which a first calc skips),
// granted skills and explode sources, radius jewel functions, the requirements
// table, the Energy Blade swap, and item warnings.
//
//pob2:block:src/Modules/CalcSetup.lua:1164-1754
func mergeItems(env *Environment, nodesModsList *moddb.ModList, canUseBonded bool) {
	tab := env.items
	if tab == nil {
		return
	}
	modDB := env.ModDB

	_, ignoreJewelLimits := env.ConfigInput["ignoreJewelLimits"]
	ignoreJewelLimits = ignoreJewelLimits && env.ConfigInput["ignoreJewelLimits"].Boolean

	//pob2:block:src/Modules/CalcSetup.lua:1183-1191
	grantedNodes := collectGrantedPassiveNodesFromItems(env.Spec, tab, env.AllocNodes, ignoreJewelLimits, nodesModsList, env.WeaponSet)
	if env.Mode == "MAIN" {
		// ItemsTab:UpdateSockets only relabels the socket controls.
		env.Spec.SetGrantedPassiveNodes(grantedNodes)
	}
	for _, id := range sortedNodeIDs(grantedNodes) {
		//pob2:block:src/Modules/CalcSetup.lua:19-23
		node := grantedNodes[id]
		if n, ok := env.Spec.Nodes[id]; ok {
			node = n
		}
		env.AllocNodes[id] = node
		env.GrantedPassives[id] = true
	}

	augmentCounts := map[string]int{}
	//pob2:block:src/Modules/CalcSetup.lua:1195-1208
	if modDB.Flag(nil, "SocketRunesOnCharacter") || nodesModsList.Flag(nil, "SocketRunesOnCharacter") {
		for _, slot := range characterRuneSlotList {
			r := tab.RuneSlots[slot[0]]
			if r == nil {
				continue
			}
			augmentCounts[r.Rune]++
			//pob2:block:src/Classes/ItemsTab.lua:2264-2270
			for _, line := range r.Lines {
				mods, _ := env.modSource(line)
				for _, m := range mods {
					env.ItemModDB.AddMod(setSource(m.Clone(), "Rune:"+r.Rune))
				}
			}
		}
	}
	items := map[string]*Item{}
	jewelLimits := map[string]float64{}

	//pob2:block:src/Modules/CalcSetup.lua:1214-1348
	for _, slot := range tab.OrderedSlots {
		slotName := slot.SlotName
		if slotName == "Ring 3" && !nodesModsList.Flag(nil, "AdditionalRingSlot") {
			continue
		}
		item := tab.Items[slot.SelItemID]
		if slot.WeaponSet != 0 && slot.WeaponSet != env.WeaponSet {
			continue
		}
		if slot.WeaponSet == 2 && env.WeaponSet == 2 {
			slotName = strings.ReplaceAll(slotName, " Swap", "")
		}
		if slot.NodeID != 0 {
			switch {
			case env.AllocNodes[slot.NodeID] == nil:
				continue
			case item != nil && !tab.IsItemValidForSlot(item, slot.SlotName, nil):
				continue
			case item != nil:
				if item.JewelData != nil {
					delete(item.JewelData, "limitDisabled")
				}
				if item.Type == "Jewel" && strings.Contains(item.Name, "The Adorned, Diamond") {
					if v, ok := item.JewelData["corruptedMagicJewelIncEffect"].(float64); ok {
						env.ModDB.Multipliers["CorruptedMagicJewelEffect"] = v / 100
					}
					if v, ok := item.JewelData["corruptedRareJewelIncEffect"].(float64); ok {
						env.ModDB.Multipliers["CorruptedRareJewelEffect"] = v / 100
					}
				}
				if item.Limit != nil && !ignoreJewelLimits {
					limitKey := item.Title
					if item.Base.SubType == "Timeless" {
						limitKey = "Historic"
					}
					if n, ok := jewelLimits[limitKey]; ok && n >= *item.Limit {
						if item.JewelData != nil {
							item.JewelData["limitDisabled"] = true
						}
						continue
					}
					jewelLimits[limitKey]++
				}
			}
		}
		items[slotName] = item
	}

	//pob2:block:src/Modules/CalcSetup.lua:1358-1410
	if v := env.ConfigInput["ignoreItemDisablers"]; !v.Boolean {
		itemDisabled := map[string]string{}
		itemDisablers := map[string]string{}
		if modDB.Flag(nil, "CanNotUseHelm") {
			itemDisabled["Helmet"] = ""
		}
		for _, slot := range tab.OrderedSlots {
			slotName := slot.SlotName
			if items[slotName] == nil {
				continue
			}
			for _, m := range items[slotName].GetActiveModListForSlotNum(slot.SlotNum, canUseBonded) {
				for _, tag := range luaTags(m) {
					t, ok := tag.(*mod.DisablesItemTag)
					if !ok {
						continue
					}
					if t.TagExcludeItemType != "" && items[t.TagSlotName] != nil && items[t.TagSlotName].Type == t.TagExcludeItemType {
						break
					}
					itemDisablers[slotName] = t.TagSlotName
					itemDisabled[t.TagSlotName] = slotName
					break
				}
			}
		}
		visited := map[string]bool{}
		trueDisabled := map[string]bool{}
		disablerSlots := make([]string, 0, len(itemDisablers))
		for s := range itemDisablers {
			disablerSlots = append(disablerSlots, s)
		}
		sort.Strings(disablerSlots)
		for _, slot := range disablerSlots {
			if visited[slot] {
				continue
			}
			// { slot = true } keys the string "slot", not the slot; ported as written.
			curChain := map[string]bool{"slot": true}
			for {
				next, ok := itemDisabled[slot]
				if !ok {
					break
				}
				slot = next
				if curChain[slot] {
					break
				}
				curChain[slot] = true
			}
			for {
				visited[slot] = true
				next, ok := itemDisablers[slot]
				if !ok {
					break
				}
				slot = next
				visited[slot] = true
				trueDisabled[slot] = true
				next, ok = itemDisablers[slot]
				if !ok {
					break
				}
				slot = next
				if visited[slot] {
					break
				}
			}
		}
		for slot := range trueDisabled {
			delete(items, slot)
		}
	}

	//pob2:block:src/Modules/CalcSetup.lua:1412-1717
	env.ItemSlotIndex = map[*Item]int{}
	for _, slot := range tab.OrderedSlots {
		slotName := slot.SlotName
		item := items[slotName]
		var node *tree2.SpecNode
		if slot.NodeID != 0 {
			node = env.Spec.Nodes[slot.NodeID]
		}
		if item != nil {
			slotIndex := 0
			if s := slotNumAnyRE.FindString(slotName); s != "" {
				slotIndex, _ = strconv.Atoi(s)
			}
			if !(env.Flasks[item] || env.Charms[item]) {
				env.ItemSlotIndex[item] = slotIndex
			} else if slot.Active {
				env.ItemSlotIndex[item] = min(env.ItemSlotIndex[item], slotIndex)
			}
		}
		if item != nil && item.Type == "Flask" {
			if slot.Active {
				env.Flasks[item] = true
			}
			if item.Base.SubType == "Life" {
				highestLifeRecovery := env.ItemModDB.Multipliers["LifeFlaskRecovery"]
				if item.FlaskData["lifeTotal"] > highestLifeRecovery {
					env.ItemModDB.Multipliers["LifeFlaskRecovery"] = item.FlaskData["lifeTotal"]
				}
			}
			env.ItemModDB.Multipliers[item.Base.SubType+strings.ReplaceAll(slotName, " ", "")+"MaxCharges"] = item.FlaskData["chargesMax"]
			item = nil
		} else if item != nil && item.Type == "Charm" {
			if slot.Active {
				env.Charms[item] = true
			}
			item = nil
		}
		scale := 1.0
		if item != nil && item.Type == "Jewel" && slot.ParentSlot != nil {
			parentItem := env.Player.ItemList[slot.ParentSlot.SlotName]
			parentSlotHidden := slot.ParentSlot.SlotName == "Ring 3" && !(modDB.Flag(nil, "AdditionalRingSlot") || nodesModsList.Flag(nil, "AdditionalRingSlot"))
			if parentSlotHidden || parentItem == nil || parentItem.JewelSocketCount < slot.SlotNum {
				item = nil
			} else {
				scale = parentItem.SocketedJewelEffectModifier
			}
		}
		// jewelIncEffectFromClassStart needs node.distanceToClassStart, which the spec does not compute.

		addSourceSlotNum := false
		if slot.NodeID != 0 && item != nil && item.Type == "Jewel" && node != nil && node.ContainJewelSocket {
			addSourceSlotNum = true
			inc := env.nodeModList(node).Sum(mod.TypeIncrease, nil, "SocketedJewelEffect")
			scale += inc / 100
		}

		if item == nil {
			continue
		}
		env.Player.ItemList[slotName] = item
		srcList := item.GetActiveModListForSlotNum(slot.SlotNum, canUseBonded)
		corruptedJewelEffect := 0.0
		if slot.NodeID != 0 {
			corruptedJewelEffect = getCorruptedJewelEffect(env, item, node)
		}

		if nodesModsList.Flag(nil, "CannotGainSpiritFromEquipment") {
			kept := make([]mod.Mod, 0, len(srcList))
			for _, m := range srcList {
				if m.Name() == "Spirit" && m.Type() == mod.TypeBase {
					continue
				}
				kept = append(kept, m)
			}
			srcList = kept
		}
		// sourceSlotNum has no field on a Go mod; nothing in the defence slice reads it.
		_ = addSourceSlotNum

		//pob2:block:src/Modules/CalcSetup.lua:1505-1530
		socketed := 0
		for i := 0; i < item.ItemSocketCount; i++ {
			if i >= len(item.Runes) {
				break
			}
			runeName := item.Runes[i]
			if runeName == "" || runeName == "None" {
				continue
			}
			augmentCounts[runeName]++
			socketed++
			augType := ""
			if rd := poe2.RuneModsByName(runeName); len(rd) > 0 {
				augType = rd[0].Type
			}
			if augType == "Idol" {
				multAdd(env.ItemModDB, "IdolsInEquipment", 1)
			} else if augType != "" {
				multAdd(env.ItemModDB, "NonIdolAugmentsInEquipment", 1)
			}
		}
		env.ItemModDB.Multipliers["RunesSocketedIn"+slotName] = float64(socketed)

		if item.Type == "Jewel" && item.Base.SubType == "Abyss" {
			//pob2:block:src/Modules/CalcSetup.lua:1532-1548
			cond := "Have" + strings.ReplaceAll(item.BaseName, " ", "")
			if !env.ItemModDB.Conditions[cond] {
				env.ItemModDB.Conditions[cond] = true
				multAdd(env.ItemModDB, "AbyssJewelType", 1)
			}
			if slot.ParentSlot != nil {
				env.ItemModDB.Conditions[cond+"In"+slot.ParentSlot.SlotName] = true
			}
			multAdd(env.ItemModDB, "AbyssJewel", 1)
			switch item.Rarity {
			case "NORMAL":
				multAdd(env.ItemModDB, "NormalAbyssJewels", 1)
			case "MAGIC":
				multAdd(env.ItemModDB, "MagicAbyssJewels", 1)
			case "RARE":
				multAdd(env.ItemModDB, "RareAbyssJewels", 1)
			case "UNIQUE", "RELIC":
				multAdd(env.ItemModDB, "UniqueAbyssJewels", 1)
			}
			multAdd(env.ItemModDB, strings.ReplaceAll(item.BaseName, " ", ""), 1)
		}

		talisman := env.AllocNodes[39935]
		switch {
		case item.Type == "Amulet" && talisman != nil && talisman.Dn == "Necromantic Talisman":
			env.TalismanModList = newModList()
			for _, m := range srcList {
				env.TalismanModList.ScaleAddMod(m, scale, false)
			}
		case slotName == "Weapon 1" && item.Name == "The Iron Mass, Gladius":
			env.TheIronMass = newModList()
			for _, m := range srcList {
				if !hasSocketedInTag(m) {
					env.TheIronMass.ScaleAddMod(m, scale, false)
				}
				env.ItemModDB.ScaleAddMod(m, scale, false)
			}
		case slotName == "Weapon 1" && len(item.GrantedSkill) > 0 && item.GrantedSkill[0].SkillID == "UniqueAnimateWeapon":
			env.WeaponModList1 = newModList()
			for _, m := range srcList {
				if !hasSocketedInTag(m) {
					env.WeaponModList1.ScaleAddMod(m, scale, false)
				} else {
					env.ItemModDB.ScaleAddMod(m, scale, false)
				}
			}
		case item.Type == "Focus" && calcLibMod(nodesModsList, nil, "EffectOfBonusesFromFocus") != 1:
			scale = calcLibMod(nodesModsList, nil, "EffectOfBonusesFromFocus") - 1
			combined := mergeModList(nil, srcList, false)
			scaledList := newModList()
			for _, m := range combined {
				scaledList.ScaleAddMod(m, scale, false)
			}
			combined = mergeModList(combined, scaledList.Mods(), true)
			for _, m := range combined {
				env.ItemModDB.AddMod(m)
			}
		case strings.Contains(item.Name, "Kalandra's Touch"):
			kalandrasTouch(env, items, slot, slotName, item, srcList, scale)
		case corruptedJewelEffect != 0:
			scale += corruptedJewelEffect
			for _, m := range mergeModList(nil, srcList, false) {
				env.ItemModDB.ScaleAddMod(m, scale, false)
			}
		default:
			for _, m := range srcList {
				env.ItemModDB.ScaleAddMod(m, scale, false)
			}
		}

		//pob2:block:src/Modules/CalcSetup.lua:1685-1715
		if item.ClassRestriction != "" {
			// Lua stores the class name string as the condition value; any non-nil value reads as true.
			env.ItemModDB.Conditions[strings.ReplaceAll(item.Title, " ", "")] = true
		}
		if item.CharmLimit != nil {
			env.ModDB.AddMod(newMod("CharmLimit", mod.TypeBase, *item.CharmLimit, mod.Source(item.Title)))
		}
		if item.SpiritValue != nil && !nodesModsList.Flag(nil, "CannotGainSpiritFromEquipment") {
			env.ModDB.AddMod(newMod("Spirit", mod.TypeBase, *item.SpiritValue, mod.Source(item.Title)))
		}
		if item.Type != "Jewel" && item.Type != "Flask" && item.Type != "Charm" {
			key := "NormalItem"
			switch item.Rarity {
			case "UNIQUE", "RELIC":
				key = "UniqueItem"
			case "RARE":
				key = "RareItem"
			case "MAGIC":
				key = "MagicItem"
			}
			multAdd(env.ItemModDB, key, 1)
			env.ItemModDB.Conditions[key+"In"+slotName] = true
			if item.Corrupted {
				multAdd(env.ItemModDB, "CorruptedItem", 1)
			} else {
				multAdd(env.ItemModDB, "NonCorruptedItem", 1)
			}
			multAdd(env.ItemModDB, handedTypeRE.ReplaceAllString(strings.ReplaceAll(item.Type, " ", ""), "")+"Item", 1)
		}
	}
}

//pob2:block:src/Modules/CalcSetup.lua:1637-1673
func kalandrasTouch(env *Environment, items map[string]*Item, slot *ItemSlot, slotName string, item *Item, srcList []mod.Mod, scale float64) {
	if item.Corrupted {
		multAdd(env.ItemModDB, "CorruptedItem", -1)
	} else {
		multAdd(env.ItemModDB, "NonCorruptedItem", 1)
	}
	var otherRing *Item
	switch slotName {
	case "Ring 1":
		otherRing = items["Ring 2"]
	case "Ring 2":
		otherRing = items["Ring 1"]
	}
	if otherRing != nil && !strings.Contains(otherRing.Name, "Kalandra's Touch") {
		list := otherRing.ModList
		if list == nil && slot.SlotNum >= 1 && slot.SlotNum <= len(otherRing.SlotModList) {
			list = otherRing.SlotModList[slot.SlotNum-1]
		}
		for _, m := range list {
			if hasSocketedInTag(m) {
				continue
			}
			env.ItemModDB.ScaleAddMod(setSource(m.Clone(), item.modSourceName), scale, false)
		}
		if item.Corrupted {
			multAdd(env.ItemModDB, "CorruptedItem", 1)
		} else {
			multAdd(env.ItemModDB, "NonCorruptedItem", -1)
		}
	}
	for _, m := range srcList {
		if m.Name() == "ExtraSkill" {
			env.ItemModDB.ScaleAddMod(m, scale, false)
		}
	}
}

func hasSocketedInTag(m mod.Mod) bool {
	for _, tag := range luaTags(m) {
		if tag != nil && tag.Type() == mod.TypeSocketedIn {
			return true
		}
	}
	return false
}

// mergeModList is ModList:MergeMod applied to each of src in turn onto list.
//
//pob2:src/Classes/ModList.lua:75-88
func mergeModList(list []mod.Mod, src []mod.Mod, skipNonAdditive bool) []mod.Mod {
	for _, m := range src {
		merged := false
		if m.Type() == mod.TypeBase || m.Type() == mod.TypeIncrease || m.Type() == mod.TypeMore {
			for i, have := range list {
				if moddb.CompareModParams(have, m) {
					c := have.Clone()
					if v := c.Value(); v != nil && v.Type() == mod.ModValueMultiTypeFloat && m.Value() != nil {
						v.SetFloat(v.Float() + m.Value().Float())
					}
					list[i] = c
					merged = true
					break
				}
			}
		}
		if !merged && !skipNonAdditive {
			list = append(list, m)
		}
	}
	return list
}

//pob2:src/Modules/CalcSetup.lua:120-126
func getCorruptedJewelEffect(env *Environment, item *Item, node *tree2.SpecNode) float64 {
	if item == nil || item.Type != "Jewel" || !item.Corrupted || node == nil || node.ContainJewelSocket || node.Sinister || item.Base.SubType == "Charm" {
		return 0
	}
	rarity := ""
	if item.Rarity != "" {
		rarity = item.Rarity[:1] + strings.ToLower(item.Rarity[1:])
	}
	return env.ModDB.Multipliers["Corrupted"+rarity+"JewelEffect"]
}

//pob2:block:src/Modules/CalcSetup.lua:1803-1821
func addSlotBonusEffects(env *Environment, canUseBonded bool) {
	modDB := env.ModDB
	if w2 := env.Player.ItemList["Weapon 2"]; w2 != nil && w2.Type == "Quiver" {
		quiverEffectMod := env.ModDB.Sum(mod.TypeIncrease, nil, "EffectOfBonusesFromQuiver") / 100
		for _, m := range w2.GetActiveModListForSlotNum(2, canUseBonded) {
			modCopy := m.Clone().Source(mod.Source("Many Sources:^x88FFFF" + luaNum(quiverEffectMod*100) + "% Quiver Bonus Effect"))
			modDB.ScaleAddMod(modCopy, quiverEffectMod, false)
		}
	}
	if amulet := env.Player.ItemList["Amulet"]; amulet != nil && amulet.Type == "Amulet" {
		amuletEffectMod := env.ModDB.Sum(mod.TypeIncrease, nil, "EffectOfBonusesFromAmulet") / 100
		for _, m := range amulet.GetActiveModListForSlotNum(0, canUseBonded) {
			modCopy := m.Clone().Source(mod.Source("Many Sources:^x88FFFF" + luaNum(amuletEffectMod*100) + "% Amulet Bonus Effect"))
			modDB.ScaleAddMod(modCopy, amuletEffectMod, false)
		}
	}
}

func grantedPassiveOf(v any) (tree2.GrantedPassive, bool) {
	switch p := v.(type) {
	case string:
		return tree2.GrantedPassive{Name: p}, true
	case LuaTable:
		typ, _ := p["type"].(string)
		count, _ := p["count"].(float64)
		return tree2.GrantedPassive{Table: true, Type: typ, Count: int(count)}, true
	}
	return tree2.GrantedPassive{}, false
}

//pob2:src/Classes/PassiveSpec.lua:1196-1222
func getItemForGrantedPassiveSlot(spec *tree2.PassiveSpec, tab *ItemsTab, slot *ItemSlot, allocNodes map[int]*tree2.SpecNode, activeWeaponSet int, nodesModsList *moddb.ModList) *Item {
	if slot.NodeID != 0 {
		if allocNodes[slot.NodeID] == nil {
			return nil
		}
		itemID := spec.Jewels[slot.NodeID]
		if s := tab.Sockets[slot.NodeID]; itemID == 0 && s != nil {
			itemID = s.SelItemID
		}
		return tab.Items[itemID]
	}
	if slot.WeaponSet != 0 && slot.WeaponSet != activeWeaponSet {
		return nil
	}
	if slot.SlotName == "Ring 3" && !nodesModsList.Flag(nil, "AdditionalRingSlot") {
		return nil
	}
	return tab.Items[slot.SelItemID]
}

// collectGrantedPassiveNodesFromItems is PassiveSpec:CollectGrantedPassiveNodesFromItems: the jewel
// sockets the equipped items' GrantedPassive mods allocate, repeated while a granted socket's jewel
// grants another.
//
//pob2:src/Classes/PassiveSpec.lua:1256-1307
func collectGrantedPassiveNodesFromItems(spec *tree2.PassiveSpec, tab *ItemsTab, baseAllocNodes map[int]*tree2.SpecNode, ignoreJewelLimits bool, nodesModsList *moddb.ModList, activeWeaponSet int) map[int]*tree2.SpecNode {
	granted := map[int]*tree2.SpecNode{}
	allocNodes := map[int]*tree2.SpecNode{}
	for id, node := range baseAllocNodes {
		if !(node.IsGrantedPassive && node.IsFreeAllocate) {
			allocNodes[id] = node
		}
	}
	jewelLimits := map[string]float64{}
	changed := true
	for safety := 0; changed && safety < 8; safety++ {
		changed = false
		for _, slot := range tab.OrderedSlots {
			item := getItemForGrantedPassiveSlot(spec, tab, slot, allocNodes, activeWeaponSet, nodesModsList)
			if item == nil || item.ModList == nil {
				continue
			}
			if slot.NodeID != 0 && !tab.IsItemValidForSlot(item, slot.SlotName, nil) {
				continue
			}
			if slot.NodeID != 0 && item.Limit != nil && !ignoreJewelLimits {
				limitKey := item.Title
				if item.Base.SubType == "Timeless" {
					limitKey = "Historic"
				}
				if n, ok := jewelLimits[limitKey]; ok && n >= *item.Limit {
					continue
				}
				jewelLimits[limitKey]++
			}
			for _, m := range item.ModList {
				if m.Name() != "GrantedPassive" || m.Value() == nil {
					continue
				}
				passive, ok := grantedPassiveOf(m.Value().List())
				if !ok {
					continue
				}
				for _, node := range spec.ResolveGrantedPassiveNodes(passive) {
					if node == nil || node.Type != "Socket" || granted[node.ID] != nil {
						continue
					}
					specNode := node
					if n, ok := spec.Nodes[node.ID]; ok {
						specNode = n
					}
					granted[node.ID] = specNode
					allocNodes[node.ID] = specNode
					changed = true
				}
			}
		}
	}
	return granted
}
