package calculator

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

func setItems(build *pob.PathOfBuilding) (pob.ItemSet, map[int]string) {
	byID := make(map[int]string, len(build.Items.Items))
	for _, it := range build.Items.Items {
		byID[it.ID] = it.Text
	}
	if len(build.Items.ItemSets) == 0 {
		return pob.ItemSet{}, byID
	}
	return build.Items.ItemSets[activeItemSetIndex(build.Items)], byID
}

func setSlotItemID(set pob.ItemSet, slotName string) int {
	for _, s := range set.Slots {
		if s.Name == slotName {
			return s.ItemID
		}
	}
	return 0
}

func setSlotItem(set pob.ItemSet, byID map[int]string, slotName string) *Item {
	if text, ok := byID[setSlotItemID(set, slotName)]; ok {
		return parseItemGlobal(text)
	}
	return nil
}

var slotTypeIDRe = regexp.MustCompile(`^([A-Za-z ]+) (\d+)$`)

//pob:src/Classes/ItemsTab.lua:2560-2606
func IsItemValidForSlot(build *pob.PathOfBuilding, item *Item, slotName string) bool {
	return itemValidForSlot(item, slotName, slotTypeOf(slotName), func(weapon1Slot string) *Item {
		return activeSetSlotItem(build, weapon1Slot)
	})
}

func activeSetSlotItem(build *pob.PathOfBuilding, slotName string) *Item {
	id := 0
	if len(build.Items.ItemSets) > 0 {
		id = setSlotItemID(build.Items.ItemSets[activeItemSetIndex(build.Items)], slotName)
	}
	for k := len(build.Items.Items) - 1; k >= 0; k-- {
		if build.Items.Items[k].ID == id {
			return parseItemGlobal(build.Items.Items[k].Text)
		}
	}
	return nil
}

func slotTypeOf(slotName string) string {
	if m := slotTypeIDRe.FindStringSubmatch(slotName); m != nil {
		return m[1]
	}
	return slotName
}

func itemValidForSlot(item *Item, slotName string, slotType string, weapon1 func(weapon1Slot string) *Item) bool {
	if item == nil || item.Base == nil {
		return false
	}
	switch {
	case slotType == "Jewel":
		return false
	case item.Type == slotType:
		return true
	case item.Type == "Tincture" && slotType == "Flask":
		return true
	case item.Type == "Jewel" && item.Base.SubType == "Abyss" && strings.Contains(slotName, "Abyssal Socket"):
		return true
	case slotName == "Weapon 1" || slotName == "Weapon 1 Swap" || slotName == "Weapon":
		return item.Base.Weapon != nil
	case slotName == "Weapon 2" || slotName == "Weapon 2 Swap":
		weapon1Slot := "Weapon 1"
		if slotName == "Weapon 2 Swap" {
			weapon1Slot = "Weapon 1 Swap"
		}
		weapon1Type := "None"
		if w1 := weapon1(weapon1Slot); w1 != nil && w1.Base != nil {
			weapon1Type = w1.Base.Type
		}
		info := data.WeaponTypes[data.ItemClassName(item.Type)]
		switch weapon1Type {
		case "None":
			return item.Type == "Shield" || (info != nil && info.OneHand)
		case "Bow":
			return item.Type == "Quiver"
		}
		if w1Info := data.WeaponTypes[data.ItemClassName(weapon1Type)]; w1Info != nil && w1Info.OneHand {
			return item.Type == "Shield" || (info != nil && info.OneHand &&
				((weapon1Type == "Wand" && item.Type == "Wand") || (weapon1Type != "Wand" && item.Type != "Wand")))
		}
	}
	return false
}

func useSecondWeaponSet(set pob.ItemSet) bool {
	return set.UseSecondWeaponSet != nil && *set.UseSecondWeaponSet
}

//pob:block:src/Classes/ItemsTab.lua:156-201,src/Classes/ItemSlotControl.lua:109-121
func slotShown(build *pob.PathOfBuilding, s itemSlot, additionalRingSlot bool) bool {
	set, byID := setItems(build)
	activeWeaponSet := 1
	if useSecondWeaponSet(set) {
		activeWeaponSet = 2
	}
	if s.weaponSet != 0 && s.weaponSet != activeWeaponSet {
		return false
	}
	if s.parentSlot != "" {
		parent := setSlotItem(set, byID, s.parentSlot)
		return parent != nil && parent.AbyssalSocketCount >= float64(s.slotNum)
	}
	switch s.name {
	case "Graft 1", "Graft 2":
		return strings.Contains(string(data.LatestTreeVersion), "3_27")
	case "Ring 3":
		return additionalRingSlot
	}
	return true
}

func slotLabel(s itemSlot) string {
	if s.parentSlot != "" {
		return "Abyssal #" + strconv.Itoa(s.slotNum)
	}
	return strings.TrimSuffix(s.name, " Swap")
}

type ItemSlotCompare struct {
	SlotName  string
	Label     string
	Replacing *Item
	Env       *Environment
}

func (m *MiscCalculator) additionalRingSlot() bool {
	return m.base.InitialNodeModDB != nil && m.base.InitialNodeModDB.Flag(nil, "AdditionalRingSlot")
}

//pob:block:src/Classes/ItemsTab.lua:4910-5018
func (m *MiscCalculator) CompareItemSlots(item *Item) []ItemSlotCompare {
	if item == nil || item.Base == nil || item.Base.Flask != nil || item.Base.Tincture != nil {
		return nil
	}
	set, byID := setItems(m.build)
	ring := m.additionalRingSlot()

	type entry struct {
		ItemSlotCompare
		selItemID    int
		isSameUnique bool
	}
	isUnique := item.Rarity == RarityUnique || item.Rarity == RarityRelic
	var slots []entry
	currentSameUniqueCount := 0
	for _, s := range orderedItemSlots {
		if !IsItemValidForSlot(m.build, item, s.name) || !slotShown(m.build, s, ring) {
			continue
		}
		selItemID := setSlotItemID(set, s.name)
		selItem := setSlotItem(set, byID, s.name)
		e := entry{
			ItemSlotCompare: ItemSlotCompare{SlotName: s.name, Label: slotLabel(s), Replacing: selItem},
			selItemID:       selItemID,
		}
		e.Env = m.Calc(&CalcOverride{RepSlotName: s.name, RepItem: item})
		e.isSameUnique = isUnique && selItem != nil && item.Name == selItem.Name
		if isUnique && e.isSameUnique && item.Limit != nil {
			currentSameUniqueCount++
		}
		slots = append(slots, e)
	}
	for n, socket := range m.activeTreeSockets() {
		if !IsJewelValidForSocket(item, socket.id) {
			continue
		}
		e := entry{
			ItemSlotCompare: ItemSlotCompare{SlotName: "Jewel " + strconv.Itoa(socket.id), Label: "Socket #" + strconv.Itoa(n+1), Replacing: socket.item},
			selItemID:       socket.itemID,
		}
		o := &CalcOverride{SocketNodeID: socket.id, SocketItem: item}
		if socket.item != nil && sameItemText(socket.item.Raw, item.Raw) {
			o.SocketItem = nil
		}
		e.Env = m.Calc(o)
		e.isSameUnique = isUnique && socket.item != nil && item.Name == socket.item.Name
		if isUnique && e.isSameUnique && item.Limit != nil {
			currentSameUniqueCount++
		}
		slots = append(slots, e)
	}

	out := make([]ItemSlotCompare, 0, len(slots))
	if item.Limit != nil && currentSameUniqueCount == *item.Limit {
		for _, e := range slots {
			if e.isSameUnique {
				out = append(out, e.ItemSlotCompare)
			}
		}
		return out
	}

	similar := func(compareItem *Item, sameUnique bool) float64 {
		if compareItem == nil {
			return 0
		}
		sameBaseType := !isUnique && compareItem.Rarity != RarityUnique && compareItem.Rarity != RarityRelic &&
			compareItem.Base != nil && item.Base.Type == compareItem.Base.Type && item.Base.SubType == compareItem.Base.SubType
		if sameBaseType || sameUnique {
			return 1
		}
		return 0
	}
	params := func(e entry) []any {
		empty := 0.0
		if e.selItemID == 0 {
			empty = 1
		}
		out := e.Env.Player.Output
		return []any{empty, similar(e.Replacing, e.isSameUnique), out["FullDPS"], out["CombinedDPS"], out["TotalEHP"], e.Label, e.SlotName}
	}
	slices.SortStableFunc(slots, func(a, b entry) int {
		ap, bp := params(a), params(b)
		for i := range ap {
			switch av := ap[i].(type) {
			case float64:
				if bv := bp[i].(float64); av > bv {
					return -1
				} else if av < bv {
					return 1
				}
			case string:
				if bv := bp[i].(string); av > bv {
					return -1
				} else if av < bv {
					return 1
				}
			}
		}
		return 0
	})
	for _, e := range slots {
		out = append(out, e.ItemSlotCompare)
	}
	return out
}

//pob:block:src/Classes/ItemsTab.lua:4866-4872
func (m *MiscCalculator) FlaskActive(item *Item) bool {
	active := m.base.Flasks
	if item.Base != nil && item.Base.Tincture != nil {
		active = m.base.Tinctures
	}
	for _, f := range active {
		if f.Raw == item.Raw {
			return true
		}
	}
	return false
}

//pob:block:src/Classes/ItemsTab.lua:1652-1656,src/Classes/ItemSlotControl.lua:105-122
func PopulateSlots(build *pob.PathOfBuilding) {
	for _, s := range orderedItemSlots {
		set, byID := setItems(build)
		id := setSlotItemID(set, s.name)
		if id == 0 {
			continue
		}
		text, ok := byID[id]
		if !ok || !IsItemValidForSlot(build, parseItemGlobal(text), s.name) {
			build.SetSlotItem(s.name, 0)
			continue
		}
		if s.parentSlot != "" {
			parent := setSlotItem(set, byID, s.parentSlot)
			if parent == nil || parent.AbyssalSocketCount < float64(s.slotNum) {
				build.SetSlotItem(s.name, 0)
			}
		}
	}
}

//pob:block:src/Classes/ItemSlotControl.lua:22-29
func EquipItem(build *pob.PathOfBuilding, text, slotName string) int {
	id := build.AddItem(text)
	build.SetSlotItem(slotName, id)
	PopulateSlots(build)
	return id
}

//pob:block:src/Classes/ItemSlotControl.lua:93-104
func SlotItemIDs(build *pob.PathOfBuilding, slotName string) []int {
	var ids []int
	for _, it := range build.Items.Items {
		if IsItemValidForSlot(build, parseItemGlobal(it.Text), slotName) {
			ids = append(ids, it.ID)
		}
	}
	slices.Sort(ids)
	return ids
}

func SlotItemIDsFor(build *pob.PathOfBuilding, names []string) [][]int {
	items := make([]*Item, len(build.Items.Items))
	for k, it := range build.Items.Items {
		items[k] = parseItemGlobal(it.Text)
	}
	set, byID := setItems(build)
	weapon1 := map[string]*Item{}
	weapon1Of := func(slot string) *Item {
		w, ok := weapon1[slot]
		if !ok {
			w = setSlotItem(set, byID, slot)
			weapon1[slot] = w
		}
		return w
	}
	out := make([][]int, len(names))
	for n, name := range names {
		slotType := slotTypeOf(name)
		var ids []int
		for k, it := range build.Items.Items {
			if itemValidForSlot(items[k], name, slotType, weapon1Of) {
				ids = append(ids, it.ID)
			}
		}
		slices.Sort(ids)
		out[n] = ids
	}
	return out
}

//pob:block:src/Classes/ItemSlotControl.lua:22-29
func EquipItemID(build *pob.PathOfBuilding, slotName string, itemID int) bool {
	set, _ := setItems(build)
	if itemID == setSlotItemID(set, slotName) || (itemID != 0 && !slices.Contains(SlotItemIDs(build, slotName), itemID)) {
		return false
	}
	build.SetSlotItem(slotName, itemID)
	PopulateSlots(build)
	return true
}

//pob:src/Classes/ItemsTab.lua:1686-1710
func (m *MiscCalculator) AddItem(text string) (int, string) {
	item := ParseItem(text)
	ring := m.additionalRingSlot()
	set, _ := setItems(m.build)
	id := m.build.AddItem(text)
	slotName := ""
	for _, s := range orderedItemSlots {
		if setSlotItemID(set, s.name) == 0 && slotShown(m.build, s, ring) && IsItemValidForSlot(m.build, item, s.name) {
			m.build.SetSlotItem(s.name, id)
			slotName = s.name
			break
		}
	}
	//pob:block:src/Classes/ItemsTab.lua:1812
	AddForbiddenJewelCounterpart(m.build, item)
	PopulateSlots(m.build)
	return id, slotName
}

//pob:block:src/Classes/ItemsTab.lua:2729-2732,src/Classes/ItemsTab.lua:1712-1714,src/Classes/ItemsTab.lua:1804-1814
func (m *MiscCalculator) ReplaceItem(id int, text string) bool {
	oldText, ok := buildItemText(m.build, id)
	if !ok || !m.build.SetItemText(id, text) {
		return false
	}
	//pob:block:src/Classes/ItemsTab.lua:1806,src/Classes/ItemsTab.lua:1811-1812
	item := ParseItem(text)
	UpdateForbiddenJewelCounterpart(m.build, id, parseItemGlobal(oldText), item)
	AddForbiddenJewelCounterpart(m.build, item)
	PopulateSlots(m.build)
	return true
}

//pob:block:src/Classes/ItemsTab.lua:37-40
var forbiddenJewelCounterpart = map[string]string{
	"Forbidden Flame": "Forbidden Flesh",
	"Forbidden Flesh": "Forbidden Flame",
}

func selectedVariantName(item *Item) string {
	if item == nil || item.Variant == nil || *item.Variant < 1 || *item.Variant > len(item.VariantList) {
		return ""
	}
	return item.VariantList[*item.Variant-1]
}

//pob:src/Classes/ItemsTab.lua:1728-1771
func AddForbiddenJewelCounterpart(build *pob.PathOfBuilding, item *Item) int {
	otherTitle := forbiddenJewelCounterpart[item.Title]
	variantName := selectedVariantName(item)
	if otherTitle == "" || variantName == "" {
		return 0
	}
	for _, it := range build.Items.Items {
		other := parseItemGlobal(it.Text)
		if other.Title == otherTitle && selectedVariantName(other) == variantName {
			return 0
		}
	}
	uniques := ItemDB(ItemDBUnique)
	k := slices.IndexFunc(uniques, func(u *Item) bool { return u.Title == otherTitle })
	if k < 0 {
		return 0
	}
	newItem := ParseItem(uniques[k].Raw)
	index := slices.Index(newItem.VariantList, variantName)
	if index < 0 {
		return 0
	}
	newItem.Variant = new(index + 1)
	newItem.NormaliseQuality()
	return build.AddItem(buildRawText(newItem))
}

//pob:src/Classes/ItemsTab.lua:1774-1801
func UpdateForbiddenJewelCounterpart(build *pob.PathOfBuilding, id int, oldItem, item *Item) {
	otherTitle := forbiddenJewelCounterpart[item.Title]
	if otherTitle == "" || oldItem == nil || oldItem.Title != item.Title {
		return
	}
	oldName, newName := selectedVariantName(oldItem), selectedVariantName(item)
	if oldName == "" || newName == "" || oldName == newName {
		return
	}
	for _, it := range build.Items.Items {
		other := parseItemGlobal(it.Text)
		if it.ID == id || other.Title != otherTitle || selectedVariantName(other) != oldName {
			continue
		}
		if index := slices.Index(other.VariantList, newName); index >= 0 {
			moved := ParseItem(it.Text)
			moved.Variant = new(index + 1)
			build.SetItemText(it.ID, buildRawText(moved))
		}
	}
}

var (
	zeroModLineStartRe = regexp.MustCompile(`^\+?0%? `)
	zeroModLineRe      = regexp.MustCompile(` \+?0%? `)
	zeroToNonZeroRe    = regexp.MustCompile(`0 to [1-9]`)
)

type ModLineKind string

const (
	ModLineMagic       ModLineKind = "MAGIC"
	ModLineDisabled    ModLineKind = "DISABLED"
	ModLineUnsupported ModLineKind = "UNSUPPORTED"
	ModLineFractured   ModLineKind = "FRACTURED"
	ModLineCrafted     ModLineKind = "CRAFTED"
	ModLineMutated     ModLineKind = "MUTATED"
	ModLineScourge     ModLineKind = "SCOURGE"
	ModLineCustom      ModLineKind = "CUSTOM"
	ModLineVestigial   ModLineKind = "VESTIGIAL"
	ModLineCrucible    ModLineKind = "CRUCIBLE"
)

func FormatModLine(ml ModLine) (string, ModLineKind, bool) {
	return formatModLine(ml, false)
}

//pob:src/Modules/ItemTools.lua:361-382
func formatModLine(ml ModLine, dbMode bool) (string, ModLineKind, bool) {
	line := ml.Line
	if !dbMode && (ml.Range != nil || ml.CorruptedRange != nil) {
		rangeVal := 0.5
		if ml.Range != nil {
			rangeVal = *ml.Range
		}
		scalar := ml.ValueScalar
		if scalar == 0 {
			scalar = 1
		}
		line = data.ApplyRange(line, rangeVal, &scalar, ml.CorruptedRange)
	}
	if zeroModLineStartRe.MatchString(line) || (zeroModLineRe.MatchString(line) && !zeroToNonZeroRe.MatchString(line)) ||
		strings.Contains(line, " 0-0 ") || strings.Contains(line, " 0 to 0 ") {
		return "", "", false
	}
	if ml.Disabled {
		return line, ModLineDisabled, true
	}
	if ml.Extra != "" {
		return line, ModLineUnsupported, true
	}
	switch {
	case ml.Fractured:
		return line, ModLineFractured, true
	case ml.Crafted:
		return line, ModLineCrafted, true
	case ml.Mutated:
		return line, ModLineMutated, true
	case ml.Scourge:
		return line, ModLineScourge, true
	case ml.Custom:
		return line, ModLineCustom, true
	case ml.Crucible:
		return line, ModLineCrucible, true
	case ml.Vestigial:
		return line, ModLineVestigial, true
	}
	return line, ModLineMagic, true
}

func (i *Item) ItemModLineVisible(ml ModLine) bool {
	return i.checkModLineVariant(ml)
}

func OrderedItemSlotNames() []string {
	names := make([]string, 0, len(orderedItemSlots))
	for _, s := range orderedItemSlots {
		names = append(names, s.name)
	}
	return names
}

//pob:block:src/Classes/ItemsTab.lua:178-186
func (m *MiscCalculator) ShownConditionalSlots() []string {
	ring := m.additionalRingSlot()
	var out []string
	for _, s := range orderedItemSlots {
		switch s.name {
		case "Ring 3", "Graft 1", "Graft 2":
			if slotShown(m.build, s, ring) {
				out = append(out, s.name)
			}
		}
	}
	return out
}
