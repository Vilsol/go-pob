package calculator

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator/calclib"
	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

//pob:block:src/Classes/ItemsTab.lua:144-152
func slotWeaponSet(name string) int {
	if strings.HasSuffix(name, " Swap") {
		return 2
	}
	if name == "Weapon 1" || name == "Weapon 2" {
		return 1
	}
	return 0
}

type itemSlot struct {
	name       string
	parentSlot string
	slotNum    int
	weaponSet  int
}

//pob:src/Classes/ItemsTab.lua:33
var itemSlotBaseNames = []string{
	"Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet",
	"Ring 1", "Ring 2", "Ring 3", "Belt", "Graft 1", "Graft 2",
	"Flask 1", "Flask 2", "Flask 3", "Flask 4", "Flask 5",
}

//pob:src/Classes/ItemsTab.lua:95-136
var orderedItemSlots = buildOrderedItemSlots()

//pob:src/Classes/ItemsTab.lua:95-136
func buildOrderedItemSlots() []itemSlot {
	const abyssalSocketsPerItem = 6

	slots := make([]itemSlot, 0, len(itemSlotBaseNames)+4*abyssalSocketsPerItem)
	for _, name := range itemSlotBaseNames {
		base := itemSlot{name: name, slotNum: slotNum(name)}
		if strings.Contains(name, "Weapon") {
			base.weaponSet = 1
		}
		slots = append(slots, base)

		if strings.Contains(name, "Weapon") {
			swap := itemSlot{name: name + " Swap", slotNum: slotNum(name), weaponSet: 2}
			slots = append(slots, swap)
			for i := 1; i <= abyssalSocketsPerItem; i++ {
				slots = append(slots, itemSlot{
					name:       swap.name + " Abyssal Socket " + strconv.Itoa(i),
					parentSlot: swap.name,
					slotNum:    i,
					weaponSet:  2,
				})
			}
		}

		switch name {
		case "Weapon 1", "Weapon 2", "Helmet", "Gloves", "Body Armour", "Boots", "Belt":
			for i := 1; i <= abyssalSocketsPerItem; i++ {
				slots = append(slots, itemSlot{
					name:       name + " Abyssal Socket " + strconv.Itoa(i),
					parentSlot: name,
					slotNum:    i,
					weaponSet:  base.weaponSet,
				})
			}
		}
	}
	return slots
}

//pob:src/Classes/ItemSlotControl.lua:31
func slotNum(name string) int {
	if m := slotTrailingNumRe.FindStringSubmatch(name); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	if m := slotFirstNumRe.FindStringSubmatch(name); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

var (
	slotTrailingNumRe = regexp.MustCompile(`(\d+)$`)
	slotFirstNumRe    = regexp.MustCompile(`(\d+)`)
)

func itemToItemData(item *Item) *ItemData {
	id := &ItemData{
		Type:                      item.Type,
		Name:                      item.Name,
		Rarity:                    string(item.Rarity),
		Corrupted:                 item.Corrupted,
		SocketColourAlwaysMatches: item.SocketColourAlwaysMatches,
		GrantedSkills:             item.GrantedSkills,
	}
	if len(item.Sockets) > 0 {
		id.SocketColors = make([]string, len(item.Sockets))
		for i, s := range item.Sockets {
			id.SocketColors[i] = s.Color
		}
	}

	appendLines := func(dst []string, lines []ModLine) []string {
		for _, ml := range lines {
			if !item.checkModLineVariant(ml) {
				continue
			}
			dst = append(dst, strings.ToLower(ml.Line))
		}
		return dst
	}
	id.ExplicitModLines = appendLines(nil, item.ExplicitModLines)
	id.OtherModLines = appendLines(nil, item.EnchantModLines)
	id.OtherModLines = appendLines(id.OtherModLines, item.ScourgeModLines)
	id.OtherModLines = appendLines(id.OtherModLines, item.ImplicitModLines)

	if item.ArmourData != nil {
		id.ArmourData = &ArmourData{
			Armour:       item.ArmourData["Armour"],
			Evasion:      item.ArmourData["Evasion"],
			EnergyShield: item.ArmourData["EnergyShield"],
			Ward:         item.ArmourData["Ward"],
			BlockChance:  item.ArmourData["BlockChance"],
		}
	}

	if item.WeaponData != nil {
		id.WeaponData = []interface{}{
			newWeaponSkillData(data.ItemClassName(item.Type), item.WeaponData[1]),
			newWeaponSkillData(data.ItemClassName(item.Type), item.WeaponData[2]),
		}
	}

	//pob:block:src/Classes/Item.lua:1423-1430
	if item.Base != nil && (item.Base.Weapon != nil || item.Type == "Ring") {
		id.SlotModList = map[int]*moddb.ModList{
			1: asModList(item.Mods),
			2: asModList(item.slotModList2),
		}
		if item.slotModList3 != nil {
			id.SlotModList[3] = asModList(item.slotModList3)
		}
	} else {
		id.ModList = asModList(item.Mods)
	}

	return id
}

func newWeaponSkillData(itemType data.ItemClassName, wd map[string]float64) *SkillData {
	if wd == nil {
		return nil
	}
	sd := &SkillData{Type: itemType}
	for k, v := range wd {
		utils.Set(sd, k, v)
	}
	return sd
}

//pob:src/Classes/ItemsTab.lua:1584-1590
func activeItemSetIndex(items pob.Items) int {
	for i, set := range items.ItemSets {
		if id, err := strconv.Atoi(set.ID); err == nil && id == items.ActiveItemSet {
			return i
		}
	}
	return 0
}

//pob:block:src/Modules/CalcSetup.lua:497-745
func mergeEquippedItems(env *Environment) {
	items := env.Build.Items
	if len(items.ItemSets) == 0 {
		return
	}

	set := items.ItemSets[activeItemSetIndex(items)]

	activeWeaponSet := 1
	if set.UseSecondWeaponSet != nil && *set.UseSecondWeaponSet {
		activeWeaponSet = 2
	}

	byID := itemTextByID(env)

	if env.Player.ItemList == nil {
		env.Player.ItemList = map[string]*ItemData{}
	}

	slotsByName := make(map[string]pob.Slot, len(set.Slots))
	for _, slot := range set.Slots {
		slotsByName[slot.Name] = slot
	}

	itemList := make(map[string]*Item, len(set.Slots))

	//pob:block:src/Modules/CalcSetup.lua:820-839
	for _, s := range orderedItemSlots {
		slotName, item, ok := resolveEquippedSlot(env, s, slotsByName, byID, activeWeaponSet, true)
		if !ok {
			continue
		}
		if item.Type == "Flask" {
			//pob:block:src/Modules/CalcSetup.lua:833-835
			if item.Base != nil && item.Base.SubType == "Life" &&
				item.FlaskData["lifeTotal"] > env.ItemModDB.Multipliers["LifeFlaskRecovery"] {
				env.ItemModDB.Multipliers["LifeFlaskRecovery"] = item.FlaskData["lifeTotal"]
			}
			continue
		}
		if item.Type == "Jewel" && item.Base != nil && item.Base.SubType == "Abyss" && s.parentSlot != "" {
			parentItem := itemList[s.parentSlot]
			if parentItem == nil || parentItem.AbyssalSocketCount < float64(s.slotNum) {
				continue
			}
		}
		itemList[slotName] = item
	}

	disabledSlots := disabledItemSlots(env, slotsByName, byID, activeWeaponSet)
	for slotName := range disabledSlots {
		delete(itemList, slotName)
	}

	for _, s := range orderedItemSlots {
		slotName, item, ok := resolveEquippedSlot(env, s, slotsByName, byID, activeWeaponSet, false)
		if !ok {
			continue
		}
		if disabledSlots[slotName] {
			continue
		}
		slot := slotsByName[s.name]

		if item.Type == "Flask" {
			//pob:block:src/Modules/CalcSetup.lua:927
			env.ItemModDB.Conditions["Have"+strings.Join(strings.Fields(item.BaseName), "")] = true
			if slot.Active {
				env.Flasks = append(env.Flasks, item)
			}
			if item.Base != nil && item.Base.SubType == "Life" {
				if item.FlaskData["chargesMax"] > env.ItemModDB.Multipliers["LifeFlaskCharges"] {
					env.ItemModDB.Multipliers["LifeFlaskCharges"] = item.FlaskData["chargesMax"]
				}
			}
			continue
		}

		//pob:block:src/Modules/CalcSetup.lua:943-947
		if item.Type == "Tincture" {
			if slot.Active {
				env.Tinctures = append(env.Tinctures, item)
			}
			continue
		}

		//pob:block:src/Modules/CalcSetup.lua:949-958
		scale := 1.0
		if item.Type == "Jewel" && item.Base != nil && item.Base.SubType == "Abyss" && s.parentSlot != "" {
			parentItem := itemList[s.parentSlot]
			if parentItem == nil || parentItem.AbyssalSocketCount < float64(s.slotNum) {
				continue
			}
			scale = parentItem.SocketedJewelEffectModifier
		}

		itemList[slotName] = item
		env.Player.ItemList[slotName] = itemToItemData(item)
		updateAbyssJewelConditions(env, item, s.parentSlot)

		if item.Requirements != nil {
			env.RequirementsTableItems = append(env.RequirementsTableItems, &RequirementsTable{
				Source:     "Item",
				SourceItem: item,
				Str:        int(item.Requirements["strMod"]),
				Dex:        int(item.Requirements["dexMod"]),
				Int:        int(item.Requirements["intMod"]),
			})
		}

		srcList := item.modListForSlot(slotNum(slotName))
		if withheld, kept, isAegis := splitNecromanticAegisMods(env, item, srcList); isAegis {
			env.AegisModList = moddb.NewModList()
			env.AegisModList.ScaleAddList(asModList(withheld), scale)
			env.ItemModDB.ScaleAddList(asModList(kept), scale)
		} else if replacement, isEnergyBlade := energyBladeReplacement(env, slotName, item); isEnergyBlade {
			if replacement != nil {
				itemList[slotName] = replacement
				env.Player.ItemList[slotName] = itemToItemData(replacement)
			} else {
				env.ItemModDB.ScaleAddList(asModList(srcList), scale)
			}
		} else if mirrored, isKalandra := kalandraMirroredMods(itemList, slotName, item); isKalandra {
			//pob:block:src/Modules/CalcSetup.lua:1087-1125
			env.ItemModDB.ScaleAddList(asModList(mirrored), scale)
			own := make([]mod.Mod, 0)
			for _, m := range srcList {
				if m.Name() == "ExtraSkill" {
					own = append(own, m)
				}
			}
			env.ItemModDB.ScaleAddList(asModList(own), scale)
			kalandraMirrorItemCounts(env, itemList, slotName, item)
		} else if withheld, isIronMass := splitIronMassMods(item, slotName, srcList); isIronMass {
			//pob:block:src/Modules/CalcSetup.lua:1051-1068
			env.TheIronMass = moddb.NewModList()
			env.TheIronMass.ScaleAddList(asModList(withheld), scale)
			env.ItemModDB.ScaleAddList(asModList(srcList), scale)
		} else if withheld, kept := splitDancingDervishMods(item, slotName, srcList); withheld != nil {
			if env.WeaponModList1 == nil {
				env.WeaponModList1 = moddb.NewModList()
			}
			env.WeaponModList1.ScaleAddList(asModList(withheld), scale)
			env.ItemModDB.ScaleAddList(asModList(kept), scale)
		} else if widowHailMod, ok := quiverBonusMultiplier(env, itemList, item); ok {
			//pob:block:src/Modules/CalcSetup.lua:1126-1134
			env.ModDB.AddMod(mod.NewFloat("WidowHailMultiplier", mod.TypeBase, widowHailMod).Source("Widowhail"))
			env.ItemModDB.ScaleAddList(mergedModList(srcList), scale*widowHailMod)
		} else if bonusScale, ok := slotBonusScale(env, item); ok {
			//pob:block:src/Modules/CalcSetup.lua:1142-1165
			combined := mergedModList(srcList)
			scaled := moddb.NewModList()
			scaled.ScaleAddList(combined, bonusScale)
			for _, m := range scaled.Mods() {
				mergeModSkippingNonAdditive(combined, m)
			}
			env.ItemModDB.AddList(combined)
		} else {
			env.ItemModDB.ScaleAddList(asModList(srcList), scale)
		}

		recordClassRestriction(env, item)
		updateItemCounts(env, item, slotName)
	}
	// Override empty socket calculation if set in config
	//pob:block:src/Modules/CalcSetup.lua:1261-1265
	if env.Build != nil {
		for _, colour := range []string{"Red", "Green", "Blue", "White"} {
			if v, ok := env.Build.GetNumberOptionOK("overrideEmpty" + colour + "Sockets"); ok {
				env.ItemModDB.Multipliers["Empty"+colour+"SocketsInAnySlot"] = v
			}
		}
	}

	mergeTreeJewels(env, byID)
}

const necromanticAegisNodeID = "45175"

//pob:block:src/Modules/CalcSetup.lua:998
func necromanticAegisAllocated(env *Environment) bool {
	node, allocated := env.AllocatedNodes[necromanticAegisNodeID]
	if !allocated {
		return false
	}
	if rewrite, ok := nodeRewrites(env)[necromanticAegisNodeID]; ok {
		if replacement := rewrite.effectiveReplacement(); replacement != nil {
			return replacement.DN == "Necromantic Aegis"
		}
	}
	return node.Name != nil && *node.Name == "Necromantic Aegis"
}

//pob:block:src/Modules/CalcSetup.lua:641-665
func energyBladeReplacement(env *Environment, slotName string, item *Item) (*Item, bool) {
	if slotName != "Weapon 1" && slotName != "Weapon 2" {
		return nil, false
	}
	if !env.ModDB.Conditions[conditionAffectedByEnergyBlade] {
		return nil, false
	}

	info := data.WeaponTypes[data.ItemClassName(item.Type)]
	if info == nil || data.ItemClassName(item.Type) == data.Bow {
		return nil, true
	}

	name := energyBlade + " Two Handed"
	if info.OneHand {
		name = energyBlade + " One Handed"
	}
	if ItemBaseByName(name) == nil {
		return nil, true
	}

	replacement := ParseItem("Rarity: NORMAL\n" + name + "\nQuality: 0\n")

	replacement.Sockets = item.Sockets
	replacement.AbyssalSocketCount = item.AbyssalSocketCount
	return replacement, true
}

//pob:block:src/Modules/CalcSetup.lua:998-1015
func splitNecromanticAegisMods(env *Environment, item *Item, srcList []mod.Mod) ([]mod.Mod, []mod.Mod, bool) {
	if item.Type != "Shield" {
		return nil, nil, false
	}
	if !necromanticAegisAllocated(env) {
		return nil, nil, false
	}

	withheld := make([]mod.Mod, 0, len(srcList))
	kept := make([]mod.Mod, 0)
	for _, m := range srcList {
		socketed := false
		for _, tag := range m.Tags() {
			if tag.Type() == mod.TypeSocketedIn {
				socketed = true
				break
			}
		}
		if socketed {
			kept = append(kept, m)
		} else {
			withheld = append(withheld, m)
		}
	}
	return withheld, kept, true
}

//pob:block:src/Modules/CalcSetup.lua:1069-1086
func kalandraMirroredMods(itemList map[string]*Item, slotName string, item *Item) ([]mod.Mod, bool) {
	if item == nil || !strings.Contains(item.Name, "Kalandra's Touch") {
		return nil, false
	}
	var otherSlot string
	switch slotName {
	case "Ring 1":
		otherSlot = "Ring 2"
	case "Ring 2":
		otherSlot = "Ring 1"
	default:
		return nil, false
	}
	other := itemList[otherSlot]
	if other == nil || strings.Contains(other.Name, "Kalandra's Touch") {
		return nil, true
	}

	srcList := other.modListForSlot(slotNum(slotName))
	mirrored := make([]mod.Mod, 0, len(srcList))
	for _, m := range srcList {
		socketed := false
		for _, tag := range m.Tags() {
			if tag.Type() == mod.TypeSocketedIn {
				socketed = true
				break
			}
		}
		if !socketed {
			mirrored = append(mirrored, mod.SetSource(m.Clone(), mod.Source("Item:-1:"+item.Name)))
		}
	}
	return mirrored, true
}

//pob:block:src/Modules/CalcSetup.lua:1104-1118
func kalandraMirrorItemCounts(env *Environment, itemList map[string]*Item, slotName string, item *Item) {
	if item == nil || !strings.Contains(item.Name, "Kalandra's Touch") {
		return
	}
	var otherSlot string
	switch slotName {
	case "Ring 1":
		otherSlot = "Ring 2"
	case "Ring 2":
		otherSlot = "Ring 1"
	default:
		return
	}
	other := itemList[otherSlot]
	if other == nil || strings.Contains(other.Name, "Kalandra's Touch") {
		return
	}

	for _, pair := range []struct {
		mult string
		has  bool
	}{
		{"CorruptedItem", other.Corrupted},
		{"ShaperItem", other.Influence.Shaper},
		{"ElderItem", other.Influence.Elder},
		{"WarlordItem", other.Influence.Adjudicator},
		{"HunterItem", other.Influence.Basilisk},
		{"CrusaderItem", other.Influence.Crusader},
		{"RedeemerItem", other.Influence.Eyrie},
	} {
		if pair.has {
			env.ItemModDB.Multipliers[pair.mult]++
			env.ItemModDB.Multipliers["Non"+pair.mult]--
		}
	}
	if other.Influence.Elder || other.Influence.Shaper {
		env.ItemModDB.Multipliers["ShaperOrElderItem"]++
	}
	env.ItemModDB.Multipliers[strings.ReplaceAll(other.BaseName, " ", "")+"Equipped"]++
}

//pob:block:src/Modules/CalcSetup.lua:1051-1068
func splitIronMassMods(item *Item, slotName string, srcList []mod.Mod) ([]mod.Mod, bool) {
	if slotName != "Weapon 1" || item.Name != "The Iron Mass, Gladius" {
		return nil, false
	}

	withheld := make([]mod.Mod, 0, len(srcList))
	for _, m := range srcList {
		socketed := false
		for _, tag := range m.Tags() {
			if tag.Type() == mod.TypeSocketedIn {
				socketed = true
				break
			}
		}
		if !socketed {
			withheld = append(withheld, m)
		}
	}
	return withheld, true
}

func splitDancingDervishMods(item *Item, slotName string, srcList []mod.Mod) ([]mod.Mod, []mod.Mod) {
	if slotName != "Weapon 1" || len(item.GrantedSkills) == 0 || item.GrantedSkills[0].SkillID != "UniqueAnimateWeapon" {
		return nil, nil
	}

	withheld := make([]mod.Mod, 0, len(srcList))
	kept := make([]mod.Mod, 0)
	for _, m := range srcList {
		socketed := false
		for _, tag := range m.Tags() {
			if tag.Type() == mod.TypeSocketedIn {
				socketed = true
				break
			}
		}
		if socketed {
			kept = append(kept, m)
		} else {
			withheld = append(withheld, m)
		}
	}
	return withheld, kept
}

//pob:block:src/Modules/CalcSetup.lua:605-608
//pob:block:src/Modules/CalcSetup.lua:760-771
func limitDisabledJewelSockets(env *Environment) map[int]bool {
	if env.Build == nil || env.Build.GetBooleanOption("ignoreJewelLimits") {
		return nil
	}
	specs := env.Build.Tree.Specs
	if len(specs) == 0 {
		return nil
	}
	idx := env.Build.Tree.ActiveSpec - 1
	if idx < 0 || idx >= len(specs) {
		idx = 0
	}
	sockets, byID := overrideSockets(env, specs[idx].Sockets, itemTextByID(env))
	if len(sockets) == 0 {
		return nil
	}

	order := make([]int, 0, len(sockets))
	for i := range sockets {
		order = append(order, i)
	}
	sort.SliceStable(order, func(a, b int) bool { return sockets[order[a]].NodeID < sockets[order[b]].NodeID })

	counts := make(map[string]int)
	var disabled map[int]bool
	for _, i := range order {
		socket := sockets[i]
		if _, allocated := env.AllocatedNodes[strconv.Itoa(socket.NodeID)]; !allocated {
			continue
		}
		text, ok := byID[socket.ItemID]
		if !ok {
			continue
		}
		item := peekItemCached(env.Cache, text)
		if item.Limit == nil || *item.Limit <= 0 {
			continue
		}
		key := item.Title
		if item.BaseName == "Timeless Jewel" {
			key = "Historic"
		}
		if counts[key] >= *item.Limit {
			if disabled == nil {
				disabled = make(map[int]bool, 1)
			}
			disabled[socket.NodeID] = true
			continue
		}
		counts[key]++
	}
	return disabled
}

func mergeTreeJewels(env *Environment, byID map[int]string) {
	specs := env.Build.Tree.Specs
	if len(specs) == 0 {
		return
	}
	idx := env.Build.Tree.ActiveSpec - 1
	if idx < 0 || idx >= len(specs) {
		idx = 0
	}

	limitDisabled := limitDisabledJewelSockets(env)

	sockets, byID := overrideSockets(env, specs[idx].Sockets, byID)
	order := make([]int, 0, len(sockets))
	for i := range sockets {
		order = append(order, i)
	}
	sort.SliceStable(order, func(a, b int) bool { return sockets[order[a]].NodeID < sockets[order[b]].NodeID })

	for _, socketIdx := range order {
		socket := sockets[socketIdx]
		if _, allocated := env.AllocatedNodes[strconv.Itoa(socket.NodeID)]; !allocated {
			continue
		}
		text, ok := byID[socket.ItemID]
		if !ok {
			continue
		}
		item := peekItemCached(env.Cache, text)
		if item.Type != "Jewel" {
			continue
		}
		for _, rarity := range []string{"Magic", "Rare"} {
			if v, ok := jewelDataNumber(item, "corrupted"+rarity+"JewelIncEffect"); ok {
				env.ModDB.Multipliers["Corrupted"+rarity+"JewelEffect"] = v / 100
			}
		}
	}

	for _, socketIdx := range order {
		socket := sockets[socketIdx]
		nodeID := strconv.Itoa(socket.NodeID)
		if _, allocated := env.AllocatedNodes[nodeID]; !allocated {
			continue
		}
		if limitDisabled[socket.NodeID] {
			continue
		}
		text, ok := byID[socket.ItemID]
		if !ok {
			continue
		}
		item := parseItemCached(env.Cache, text)
		if item.Type != "Jewel" {
			continue
		}

		//pob:block:src/Modules/CalcSetup.lua:959-964
		scale := 1.0
		if inc, ok := jewelIncEffectFromClassStart(item); ok {
			scale += float64(float64(env.Spec.DistanceToClassStart[nodeID]) * (inc / 100))
		}

		srcList := asModList(item.Mods)
		if item.Corrupted {
			rarity := string(item.Rarity)
			if rarity != "" {
				rarity = strings.ToUpper(rarity[:1]) + strings.ToLower(rarity[1:])
			}
			if v := env.ModDB.Multipliers["Corrupted"+rarity+"JewelEffect"]; v != 0 {
				scale += v
				srcList = mergedModList(item.Mods)
			}
		}

		updateAbyssJewelConditions(env, item, "")
		env.ItemModDB.ScaleAddList(srcList, scale)

		recordClassRestriction(env, item)
	}
}

//pob:block:src/Modules/CalcSetup.lua:683-695
func updateAbyssJewelConditions(env *Environment, item *Item, parentSlot string) {
	if item.Type != "Jewel" || item.Base == nil || item.Base.SubType != "Abyss" {
		return
	}

	baseName := strings.ReplaceAll(item.BaseName, " ", "")
	cond := "Have" + baseName

	if !env.ItemModDB.Conditions[cond] {
		env.ItemModDB.Conditions[cond] = true
		env.ItemModDB.Multipliers["AbyssJewelType"]++
	}
	if parentSlot != "" {
		env.ItemModDB.Conditions[cond+"In"+parentSlot] = true
		env.ItemModDB.Multipliers[baseName+"In"+parentSlot]++
	}
	env.ItemModDB.Multipliers["AbyssJewel"]++

	switch item.Rarity {
	case RarityUnique, RarityRelic:
		env.ItemModDB.Multipliers["UniqueAbyssJewels"]++
	case RarityRare:
		env.ItemModDB.Multipliers["RareAbyssJewels"]++
	case RarityMagic:
		env.ItemModDB.Multipliers["MagicAbyssJewels"]++
	default:
		env.ItemModDB.Multipliers["NormalAbyssJewels"]++
	}

	env.ItemModDB.Multipliers[baseName]++
}

//pob:src/Modules/ModParser.lua:5201
func jewelDataNumber(item *Item, key string) (float64, bool) {
	for _, m := range item.Mods {
		if m.Name() != "JewelData" {
			continue
		}
		kv, isKV := m.Value().ValueList.(mod.JewelData)
		if !isKV || kv.Key != key {
			continue
		}
		if num, isNum := kv.Value.(float64); isNum {
			return num, true
		}
	}
	return 0, false
}

func jewelIncEffectFromClassStart(item *Item) (float64, bool) {
	for _, m := range item.Mods {
		if m.Name() != "JewelData" {
			continue
		}
		kv, isKV := m.Value().ValueList.(mod.JewelData)
		if !isKV || kv.Key != "jewelIncEffectFromClassStart" {
			continue
		}
		if num, isNum := kv.Value.(float64); isNum {
			return num, true
		}
	}
	return 0, false
}

//pob:block:src/Modules/CalcSetup.lua:1170-1172
func recordClassRestriction(env *Environment, item *Item) {
	if item.ClassRestriction == "" {
		return
	}
	if env.ItemClassRestrictions == nil {
		env.ItemClassRestrictions = map[string]string{}
	}
	env.ItemClassRestrictions[strings.ReplaceAll(item.Title, " ", "")] = item.ClassRestriction
}

//pob:block:src/Modules/CalcSetup.lua:976-981
func assignActorWeaponData(env *Environment) {
	env.Player.WeaponData1 = slotWeaponData(env.Player.ItemList, "Weapon 1", 1)
	if env.Player.WeaponData1 == nil {
		env.Player.WeaponData1 = unarmedWeaponData(env.Spec.ClassID)
	}

	if env.Player.WeaponData1.CountsAsDualWielding {
		env.Player.WeaponData2 = slotWeaponData(env.Player.ItemList, "Weapon 1", 2)
	} else {
		env.Player.WeaponData2 = slotWeaponData(env.Player.ItemList, "Weapon 2", 2)
	}
	if env.Player.WeaponData2 == nil {
		env.Player.WeaponData2 = &SkillData{}
	}
}

//pob:block:src/Modules/CalcSetup.lua:1464
func slotWeaponData(itemList map[string]*ItemData, slotName string, weaponIdx int) *SkillData {
	item := itemList[slotName]
	if item == nil || len(item.WeaponData) < weaponIdx {
		return nil
	}
	sd, _ := item.WeaponData[weaponIdx-1].(*SkillData)
	return sd
}

//pob:block:src/Modules/Data.lua:567-575
func unarmedWeaponData(classID pob.ClassID) *SkillData {
	sd := &SkillData{}
	for k, v := range data.UnarmedWeaponData[classID] {
		utils.Set(sd, k, v)
	}
	return sd
}

//pob:src/Modules/Common.lua:453-463
func mergeDB(dst *moddb.ModDB, src *moddb.ModDB) {
	if src == nil {
		return
	}
	dst.AddDB(src)
	for k, v := range src.Conditions {
		dst.Conditions[k] = v
	}
	for k, v := range src.Multipliers {
		dst.Multipliers[k] = v
	}
}

//pob:block:src/Modules/CalcSetup.lua:807-841
func updateItemCounts(env *Environment, item *Item, slotName string) {
	if item.Type == "Jewel" || item.Type == "Flask" || item.Type == "Tincture" || item.Type == "Graft" {
		return
	}

	var key string
	switch item.Rarity {
	case RarityUnique, RarityRelic:
		//pob:block:src/Modules/CalcSetup.lua:1177-1179
		if item.Foulborn {
			env.ItemModDB.Multipliers["FoulbornUniqueItem"]++
		}
		key = "UniqueItem"
	case RarityRare:
		key = "RareItem"
	case RarityMagic:
		key = "MagicItem"
	default:
		key = "NormalItem"
	}
	env.ItemModDB.Multipliers[key]++
	env.ItemModDB.Conditions[key+"In"+slotName] = true

	if item.Corrupted {
		env.ItemModDB.Multipliers["CorruptedItem"]++
	} else {
		env.ItemModDB.Multipliers["NonCorruptedItem"]++
	}

	if item.Influence.Shaper {
		env.ItemModDB.Multipliers["ShaperItem"]++
		env.ItemModDB.Conditions["ShaperItemIn"+slotName] = true
	} else {
		env.ItemModDB.Multipliers["NonShaperItem"]++
	}

	if item.Influence.Elder {
		env.ItemModDB.Multipliers["ElderItem"]++
		env.ItemModDB.Conditions["ElderItemIn"+slotName] = true
	} else {
		env.ItemModDB.Multipliers["NonElderItem"]++
	}

	for _, inf := range []struct {
		mult string
		has  bool
	}{
		{"WarlordItem", item.Influence.Adjudicator},
		{"HunterItem", item.Influence.Basilisk},
		{"CrusaderItem", item.Influence.Crusader},
		{"RedeemerItem", item.Influence.Eyrie},
	} {
		if inf.has {
			env.ItemModDB.Multipliers[inf.mult]++
		} else {
			env.ItemModDB.Multipliers["Non"+inf.mult]++
		}
	}

	if item.Influence.Shaper || item.Influence.Elder {
		env.ItemModDB.Multipliers["ShaperOrElderItem"]++
	}

	env.ItemModDB.Multipliers[typeItemMultiplierName(item.Type)]++

	if item.Type == "Ring" {
		env.ItemModDB.Multipliers[strings.ReplaceAll(item.BaseName, " ", "")+"Equipped"]++
	}

	updateSocketCounts(env, item, slotName)
}

//pob:block:src/Modules/CalcSetup.lua:1206-1252
func updateSocketCounts(env *Environment, item *Item, slotName string) {
	var socketedGems []pob.Gem
	if env.Build != nil {
		if setIndex := activeSkillSetIndex(env.Build.Skills); setIndex >= 0 {
			for _, group := range env.Build.Skills.SkillSets[setIndex].Skills {
				if socketGroupIsSourced(&group) || !group.Enabled || group.Slot == "" || group.Slot != slotName {
					continue
				}
				for _, gem := range group.Gems {
					gemData, _ := resolveGemInstance(gem)
					if gemData != nil && gem.Enabled {
						socketedGems = append(socketedGems, gem)
					}
				}
			}
		}
	}

	var emptyR, emptyG, emptyB, emptyW int
	gemSockets := 0
	for i, socket := range item.Sockets {
		switch socket.Color {
		case "R", "G", "B", "W":
		default:
			continue
		}
		gemSockets++
		if i+1 > len(socketedGems) {
			switch socket.Color {
			case "R":
				emptyR++
			case "G":
				emptyG++
			case "B":
				emptyB++
			case "W":
				emptyW++
			}
		}
	}

	var colourR, colourG, colourB int
	for i := 0; i < min(gemSockets, len(socketedGems)); i++ {
		tags := socketedGemTags(socketedGems[i])
		if tags["strength"] {
			colourR++
		}
		if tags["dexterity"] {
			colourG++
		}
		if tags["intelligence"] {
			colourB++
		}
	}

	m := env.ItemModDB.Multipliers
	m["SocketedGemsIn"+slotName] = float64(min(gemSockets, len(socketedGems)))
	m["SocketedRedGemsIn"+slotName] = float64(colourR)
	m["SocketedGreenGemsIn"+slotName] = float64(colourG)
	m["SocketedBlueGemsIn"+slotName] = float64(colourB)
	m["EmptySocketIn"+slotName] = float64(min(gemSockets, emptyR+emptyG+emptyB+emptyW))

	m["EmptyRedSocketsInAnySlot"] += float64(emptyR)
	m["EmptyGreenSocketsInAnySlot"] += float64(emptyG)
	m["EmptyBlueSocketsInAnySlot"] += float64(emptyB)
	m["EmptyWhiteSocketsInAnySlot"] += float64(emptyW)
}

func socketedGemTags(gem pob.Gem) map[string]bool {
	gemData, bare := resolveGemInstance(gem)
	if pobGem, ok := pobGemFor(gemData); ok {
		return pobGem.Tags
	}

	effectID := gem.SkillID
	if effectID == "" && bare != nil {
		effectID = bare.ID
	}
	if gameID, ok := raw2.GemGameIDForGrantedEffectID[effectID]; ok {
		if pobGem, ok := raw2.Gems[gameID]; ok {
			return pobGem.Tags
		}
	}
	return nil
}

//pob:src/Modules/CalcSetup.lua:1200
func typeItemMultiplierName(itemType string) string {
	name := strings.ReplaceAll(itemType, " ", "")
	if i := strings.LastIndex(name, "Handed"); i >= 0 {
		name = name[i+len("Handed"):]
	}
	return name + "Item"
}

func slotBonusScale(env *Environment, item *Item) (float64, bool) {
	var name string
	switch item.Type {
	case "Gloves":
		name = "EffectOfBonusesFromGloves"
	case "Boots":
		name = "EffectOfBonusesFromBoots"
	default:
		return 0, false
	}
	if env.InitialNodeModDB == nil {
		return 0, false
	}
	multiplier := calclib.Mod(env.InitialNodeModDB, nil, name)
	if multiplier == 1 {
		return 0, false
	}
	return multiplier - 1, true
}

func mergedModList(srcList []mod.Mod) *moddb.ModList {
	out := moddb.NewModList()
	for _, m := range srcList {
		mergeModIntoList(out, m, false)
	}
	return out
}

func mergeModSkippingNonAdditive(list *moddb.ModList, m mod.Mod) { mergeModIntoList(list, m, true) }

//pob:src/Classes/ModList.lua:69-82
func mergeModIntoList(list *moddb.ModList, m mod.Mod, skipNonAdditive bool) {
	if t := m.Type(); t == mod.TypeBase || t == mod.TypeIncrease || t == mod.TypeMore {
		for _, existing := range list.Mods() {
			if !moddb.CompareModParams(existing, m) {
				continue
			}
			ev, nv := existing.Value(), m.Value()
			if ev == nil || nv == nil ||
				ev.Type() != mod.ModValueMultiTypeFloat || nv.Type() != mod.ModValueMultiTypeFloat {
				continue
			}
			ev.SetFloat(ev.Float() + nv.Float())
			return
		}
	}
	if !skipNonAdditive {
		list.AddMod(m.Clone())
	}
}

//pob:src/Modules/CalcSetup.lua:1126-1128
func quiverBonusMultiplier(env *Environment, itemList map[string]*Item, item *Item) (float64, bool) {
	if item.Type != "Quiver" || env.InitialNodeModDB == nil {
		return 0, false
	}
	weapon := itemList["Weapon 1"]
	treeSum := env.InitialNodeModDB.Sum(mod.TypeIncrease, nil, "EffectOfBonusesFromQuiver")
	if !(weapon != nil && strings.Contains(weapon.Name, "Widowhail")) && treeSum <= 0 {
		return 0, false
	}
	percent := float64(100)
	if weapon != nil {
		percent = asModList(weapon.Mods).Sum(mod.TypeIncrease, nil, "EffectOfBonusesFromQuiver") + treeSum
	}
	return 1 + percent/100, true
}

//pob:block:src/Modules/CalcSetup.lua:520-525
func resolveEquippedSlot(env *Environment, s itemSlot, slotsByName map[string]pob.Slot,
	byID map[int]string, activeWeaponSet int, peek bool,
) (string, *Item, bool) {
	//pob:block:src/Modules/CalcSetup.lua:708-711
	if s.name == "Ring 3" && (env.InitialNodeModDB == nil || !env.InitialNodeModDB.Flag(nil, "AdditionalRingSlot")) {
		return "", nil, false
	}

	item, ok := peekSlotItem(env, s.name, slotsByName, byID)
	if !ok {
		return "", nil, false
	}

	slotName := s.name
	if s.weaponSet != 0 {
		if s.weaponSet != activeWeaponSet {
			return "", nil, false
		}
		if s.weaponSet == 2 {
			slotName = strings.ReplaceAll(slotName, " Swap", "")
		}
	}
	if !peek {
		item = item.Clone()
	}
	return slotName, item, true
}

func slotItem(env *Environment, slotName string, slotsByName map[string]pob.Slot, byID map[int]string) (*Item, bool) {
	item, ok := peekSlotItem(env, slotName, slotsByName, byID)
	return item.Clone(), ok
}

//pob:block:src/Modules/CalcSetup.lua:712-723
func peekSlotItem(env *Environment, slotName string, slotsByName map[string]pob.Slot, byID map[int]string) (*Item, bool) {
	if o := env.override; o != nil && o.RepSlotName != "" {
		if slotName == o.RepSlotName {
			return o.RepItem, o.RepItem != nil
		}
		if o.RepItem != nil && o.RepItem.Base != nil && strings.HasPrefix(o.RepSlotName, "Weapon 1") && strings.HasPrefix(slotName, "Weapon 2") {
			switch o.RepItem.Base.Type {
			case "Staff", "Two Handed Sword", "Two Handed Axe", "Two Handed Mace":
				return nil, false
			}
		}
	}
	slot, ok := slotsByName[slotName]
	if !ok || slot.ItemID == 0 {
		return nil, false
	}
	text, ok := byID[slot.ItemID]
	if !ok {
		return nil, false
	}
	return peekItemCached(env.Cache, text), true
}

//pob:block:src/Modules/CalcSetup.lua:841-905
func disabledItemSlots(env *Environment, slotsByName map[string]pob.Slot, byID map[int]string,
	activeWeaponSet int,
) map[string]bool {
	for _, input := range env.Build.Config.Inputs {
		if input.Name == "ignoreItemDisablers" && input.Boolean != nil && *input.Boolean {
			return nil
		}
	}

	type equipped struct {
		slotName string
		item     *Item
	}
	items := make(map[string]*Item, len(orderedItemSlots))
	ordered := make([]equipped, 0, len(orderedItemSlots))
	for _, s := range orderedItemSlots {
		slotName, item, ok := resolveEquippedSlot(env, s, slotsByName, byID, activeWeaponSet, true)
		if !ok {
			continue
		}
		items[slotName] = item
		ordered = append(ordered, equipped{slotName, item})
	}

	itemDisablers := map[string]string{}
	itemDisabled := map[string]string{}

	treeSource := mod.Source("Tree")
	record := func(disabler string, m mod.Mod) {
		for _, tag := range m.Tags() {
			di, ok := tag.(*mod.DisablesItemTag)
			if !ok {
				continue
			}
			if di.TagExcludeItemType != "" {
				if target := items[di.TagSlotName]; target != nil && target.Type == di.TagExcludeItemType {
					break
				}
			}
			itemDisablers[disabler] = di.TagSlotName
			itemDisabled[di.TagSlotName] = disabler
			break
		}
	}

	if env.InitialNodeModDB != nil {
		for _, res := range env.InitialNodeModDB.Tabulate(mod.TypeDisablerFlag,
			&moddb.ListCfg{Source: &treeSource}, "CanNotUseItem") {
			if res.Mod == nil {
				continue
			}
			record(string(res.Mod.GetSource()), res.Mod)
		}
	}

	for _, e := range ordered {
		for _, m := range e.item.modListForSlot(slotNum(e.slotName)) {
			record(e.slotName, m)
		}
	}

	return resolveDisablerChains(itemDisablers, itemDisabled)
}

//pob:block:src/Modules/CalcSetup.lua:879-904
func resolveDisablerChains(itemDisablers, itemDisabled map[string]string) map[string]bool {
	starts := make([]string, 0, len(itemDisablers))
	for k := range itemDisablers {
		starts = append(starts, k)
	}
	sort.Strings(starts)

	visited := map[string]bool{}
	trueDisabled := map[string]bool{}
	for _, start := range starts {
		if visited[start] {
			continue
		}
		slot := start
		chain := map[string]bool{}
		for itemDisabled[slot] != "" {
			slot = itemDisabled[slot]
			if chain[slot] {
				break
			}
			chain[slot] = true
		}

		for {
			visited[slot] = true
			slot = itemDisablers[slot]
			if slot == "" {
				break
			}
			visited[slot] = true
			trueDisabled[slot] = true
			slot = itemDisablers[slot]
			if slot == "" || visited[slot] {
				break
			}
		}
	}

	return trueDisabled
}
