package calculator

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

type listSlot struct {
	name     string
	label    string
	nodeID   int
	selected int
	inactive bool
}

//pob:block:src/Classes/ItemsTab.lua:208-222
func treeSocketIDs() []int {
	var ids []int
	for key, node := range data.TreeVersions[data.LatestTreeVersion].Tree().Nodes {
		if nodeTypeOf(node) != "Socket" {
			continue
		}
		if id, err := strconv.Atoi(key); err == nil {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}

func specAllocated(spec pob.Spec) map[int]bool {
	alloc := map[int]bool{}
	for _, id := range pob.SpecNodes(spec) {
		alloc[int(id)] = true
	}
	return alloc
}

func orderedListSlots(build *pob.PathOfBuilding) []listSlot {
	set, byID := setItems(build)
	out := make([]listSlot, 0, len(orderedItemSlots)+64)
	for _, s := range orderedItemSlots {
		ls := listSlot{name: s.name, label: slotLabel(s), selected: setSlotItemID(set, s.name)}
		if s.parentSlot != "" {
			parent := setSlotItem(set, byID, s.parentSlot)
			ls.inactive = parent == nil || parent.AbyssalSocketCount < float64(s.slotNum)
		}
		out = append(out, ls)
	}
	var spec pob.Spec
	if i := build.ActiveSpecIndex(); i >= 0 {
		spec = build.Tree.Specs[i]
	}
	alloc := specAllocated(spec)
	for _, id := range treeSocketIDs() {
		ls := listSlot{name: "Jewel " + strconv.Itoa(id), label: "Socket", nodeID: id, inactive: !alloc[id]}
		for _, s := range spec.Sockets {
			if s.NodeID == id {
				ls.selected = s.ItemID
			}
		}
		out = append(out, ls)
	}
	return out
}

func itemSetTitle(set pob.ItemSet) string {
	if set.Title == nil {
		return "Default"
	}
	return *set.Title
}

//pob:src/Classes/ItemsTab.lua:2425-2439
func equippedSlotForItem(build *pob.PathOfBuilding, slots []listSlot, itemID int) (*listSlot, *pob.ItemSet) {
	active := strconv.Itoa(build.Items.ActiveItemSet)
	for k := range slots {
		slot := &slots[k]
		if slot.inactive {
			continue
		}
		if slot.selected == itemID {
			return slot, nil
		}
		if slot.nodeID != 0 {
			continue
		}
		for i := range build.Items.ItemSets {
			set := &build.Items.ItemSets[i]
			if set.ID != active && setSlotItemID(*set, slot.name) == itemID {
				return slot, set
			}
		}
	}
	return nil, nil
}

//pob:src/Classes/ItemListControl.lua:198-219
func findSocketedJewel(build *pob.PathOfBuilding, item *Item, itemID int, excludeActive bool) (string, bool) {
	if item == nil || item.Type != "Jewel" {
		return "", false
	}
	equipTree, found := "", false
	matchActive := false
	active := build.ActiveSpecIndex()
	for specID := len(build.Tree.Specs) - 1; specID >= 0; specID-- {
		spec := build.Tree.Specs[specID]
		alloc := specAllocated(spec)
		for _, s := range spec.Sockets {
			if s.ItemID != itemID || !alloc[s.NodeID] {
				continue
			}
			if excludeActive && (specID == active || matchActive) {
				equipTree, found = "", false
				matchActive = true
			} else {
				equipTree, found = pob.SpecTitle(spec), true
			}
		}
	}
	return equipTree, found
}

//pob:src/Classes/ItemListControl.lua:221-240
func findEquippedAbyssJewel(build *pob.PathOfBuilding, item *Item, itemID int, excludeActiveSet bool) (string, bool) {
	if item == nil || item.Base == nil || item.Base.SubType != "Abyss" {
		return "", false
	}
	equipSet, found := "", false
	matchActive := false
	active := strconv.Itoa(build.Items.ActiveItemSet)
	for _, set := range build.Items.ItemSets {
		for _, s := range set.Slots {
			if s.ItemID != itemID {
				continue
			}
			if excludeActiveSet && (set.ID == active || matchActive) {
				equipSet, found = "", false
				matchActive = true
			} else {
				equipSet, found = itemSetTitle(set), true
			}
		}
	}
	return equipSet, found
}

func buildItemText(build *pob.PathOfBuilding, itemID int) (string, bool) {
	for _, it := range build.Items.Items {
		if it.ID == itemID {
			return it.Text, true
		}
	}
	return "", false
}

//pob:block:src/Classes/ItemListControl.lua:331-369
func DeleteItemPrompt(build *pob.PathOfBuilding, itemID int) string {
	text, ok := buildItemText(build, itemID)
	if !ok {
		return ""
	}
	item := parseItemGlobal(text)
	inSet := func(set *pob.ItemSet) string {
		if set == nil {
			return ""
		}
		return " in set '" + itemSetTitle(*set) + "'"
	}
	if slot, set := equippedSlotForItem(build, orderedListSlots(build), itemID); slot != nil {
		return item.Name + " is currently equipped in " + slot.label + inSet(set) + ".\nAre you sure you want to delete it?"
	}
	if title, ok := findEquippedAbyssJewel(build, item, itemID, true); ok {
		return item.Name + " is currently equipped in an Abyssal Socket in set '" + title + "'.\nAre you sure you want to delete it?"
	}
	if tree, ok := findSocketedJewel(build, item, itemID, true); ok {
		return item.Name + " is currently equipped in passive tree '" + tree + "'.\nAre you sure you want to delete it?"
	}
	return ""
}

//pob:block:src/Classes/ItemListControl.lua:23-29
func UnusedItemIDs(build *pob.PathOfBuilding) []int {
	slots := orderedListSlots(build)
	var out []int
	for _, it := range build.Items.Items {
		item := parseItemGlobal(it.Text)
		if slot, _ := equippedSlotForItem(build, slots, it.ID); slot != nil {
			continue
		}
		if _, ok := findEquippedAbyssJewel(build, item, it.ID, false); ok {
			continue
		}
		if _, ok := findSocketedJewel(build, item, it.ID, false); ok {
			continue
		}
		out = append(out, it.ID)
	}
	return out
}

func specRoots(tree *data.Tree, classID pob.ClassID) []int64 {
	start, ok := tree.Nodes[classStartNodeID(tree, classID)]
	if !ok {
		return nil
	}
	var roots []int64
	for _, id := range linkedNodeIDs(start) {
		if n, ok := tree.Nodes[id]; ok && n.Skill != nil && n.AscendancyName == nil {
			roots = append(roots, *n.Skill)
		}
	}
	return roots
}

type socketCut struct {
	socket  int64
	depends []int64
}

//pob:block:src/Classes/ItemsTab.lua:1789-1805
func socketCuts(build *pob.PathOfBuilding, itemID int) map[int][]socketCut {
	cuts := map[int][]socketCut{}
	for k, spec := range build.Tree.Specs {
		if k == build.ActiveSpecIndex() {
			continue
		}
		nodes := pob.SpecNodes(spec)
		for _, s := range spec.Sockets {
			id := int64(s.NodeID)
			if s.ItemID != itemID || !slices.Contains(nodes, id) {
				continue
			}
			shadow, tree := specShadow(build, k)
			shadow.DeallocateNodes([]int64{id})
			pruneOrphansOn(shadow, tree)
			var depends []int64
			for _, n := range nodes {
				if n != id && !slices.Contains(shadow.Build.PassiveNodes, n) {
					depends = append(depends, n)
				}
			}
			cuts[k] = append(cuts[k], socketCut{id, depends})
		}
	}
	return cuts
}

func specShadow(build *pob.PathOfBuilding, k int) (*pob.PathOfBuilding, *data.Tree) {
	shadow := *build
	shadow.Tree.Specs = slices.Clone(build.Tree.Specs)
	shadow.SetActiveSpec(k + 1)
	version := build.Tree.Specs[k].TreeVersion
	if _, ok := data.TreeVersions[version]; !ok {
		version = data.LatestTreeVersion
	}
	return &shadow, data.TreeVersions[version].Tree()
}

//pob:block:src/Classes/ItemListControl.lua:30-41,src/Classes/ItemsTab.lua:1768-1815
func DeleteItems(build *pob.PathOfBuilding, itemIDs []int) bool {
	changed := false
	for i := len(itemIDs) - 1; i >= 0; i-- {
		cuts := socketCuts(build, itemIDs[i])
		socketed := activeSocketed(build, itemIDs[i])
		if !build.DeleteItem(itemIDs[i]) {
			continue
		}
		changed = true
		for k, c := range cuts {
			cutSpecSockets(build, k, c)
		}
		dropClusterRemnants(build, socketed)
	}
	if changed {
		PopulateSlots(build)
	}
	return changed
}

func activeSocketed(build *pob.PathOfBuilding, itemID int) bool {
	i := build.ActiveSpecIndex()
	return i >= 0 && slices.ContainsFunc(build.Tree.Specs[i].Sockets, func(s pob.Socket) bool { return s.ItemID == itemID })
}

//pob:block:src/Classes/PassiveSpec.lua:1723-1790
func dropClusterRemnants(build *pob.PathOfBuilding, socketed bool) {
	version := data.LatestTreeVersion
	tree := data.TreeVersions[version].Tree()
	gone := clusterRemnants(build, build.Build.PassiveNodes, version)
	if len(gone) > 0 || socketed {
		build.DeallocateNodes(gone)
		pruneOrphansOn(build, tree)
	}
}

func cutSpecSockets(build *pob.PathOfBuilding, k int, cuts []socketCut) {
	before := pob.SpecNodes(build.Tree.Specs[k])
	shadow, tree := specShadow(build, k)
	cut := make([]int64, 0, len(cuts))
	var kept []int64
	for _, c := range cuts {
		cut = append(cut, c.socket)
		for _, id := range c.depends {
			if !isClusterNode(tree, id) {
				kept = append(kept, id)
			}
		}
	}
	shadow.DeallocateNodes(cut)
	shadow.DeallocateNodes(append(clusterRemnants(shadow, shadow.Build.PassiveNodes, build.Tree.Specs[k].TreeVersion), kept...))
	pruneOrphansOn(shadow, tree)
	var gone []int64
	for _, id := range before {
		if !slices.Contains(shadow.Build.PassiveNodes, id) && !slices.Contains(kept, id) {
			gone = append(gone, id)
		}
	}
	build.DeallocateSpecNodes(k, gone)
}

func clusterRemnants(build *pob.PathOfBuilding, allocated []int64, version pob.TreeVersion) []int64 {
	if _, ok := data.TreeVersions[version]; !ok {
		version = data.LatestTreeVersion
	}
	tree := data.TreeVersions[version].Tree()
	nodes := buildClusterJewelGraphs(&Environment{Build: build, Cache: NewEnvironmentCache()}, version)
	var gone []int64
	for _, id := range allocated {
		if _, ok := nodes[strconv.FormatInt(id, 10)]; !ok && isClusterNode(tree, id) {
			gone = append(gone, id)
		}
	}
	return gone
}

func isClusterNode(tree *data.Tree, id int64) bool {
	node, static := tree.Nodes[strconv.FormatInt(id, 10)]
	return (!static && id >= clusterNodeIDBase) || (static && node.ExpansionJewel != nil && node.ExpansionJewel.Parent != nil)
}

//pob:block:src/Classes/ItemListControl.lua:47-63
func DeleteAllItems(build *pob.PathOfBuilding) {
	build.DeleteAllItems()
	PopulateSlots(build)
}

//pob:src/Classes/ItemsTab.lua:1731-1765
func SortItems(build *pob.PathOfBuilding) {
	slots := orderedListSlots(build)
	slotOrder := make(map[string]int, len(slots))
	for i, s := range slots {
		slotOrder[s.name] = i + 1
	}
	setOrder := make(map[string]int, len(build.Items.ItemSets))
	for i, s := range build.Items.ItemSets {
		setOrder[s.ID] = i + 1
	}
	type entry struct {
		item  pob.Item
		prim  string
		name  string
		slot  *listSlot
		inSet *pob.ItemSet
	}
	entries := make([]entry, len(build.Items.Items))
	for k, it := range build.Items.Items {
		parsed := parseItemGlobal(it.Text)
		e := entry{item: it, name: parsed.Name}
		if parsed.Base != nil {
			e.prim = parsed.getPrimarySlot()
		}
		e.slot, e.inSet = equippedSlotForItem(build, slots, it.ID)
		entries[k] = e
	}
	less := func(a, b entry) bool {
		if a.prim != b.prim {
			oa, okA := slotOrder[a.prim]
			ob, okB := slotOrder[b.prim]
			if !okA {
				return false
			} else if !okB {
				return true
			}
			return oa < ob
		}
		switch {
		case a.slot != nil && b.slot != nil:
			if a.slot.name != b.slot.name {
				return slotOrder[a.slot.name] < slotOrder[b.slot.name]
			} else if a.inSet != nil && b.inSet == nil {
				return false
			} else if a.inSet == nil && b.inSet != nil {
				return true
			} else if a.inSet != nil && b.inSet != nil {
				return setOrder[a.inSet.ID] < setOrder[b.inSet.ID]
			}
		case a.slot != nil:
			return true
		case b.slot != nil:
			return false
		}
		return a.name < b.name
	}
	slices.SortStableFunc(entries, func(a, b entry) int {
		switch {
		case less(a, b):
			return -1
		case less(b, a):
			return 1
		}
		return 0
	})
	for k, e := range entries {
		build.Items.Items[k] = e.item
	}
}

//pob:block:src/Classes/ItemListControl.lua:290-314
func ToggleEquipItem(build *pob.PathOfBuilding, itemID int, second bool) bool {
	text, ok := buildItemText(build, itemID)
	if !ok {
		return false
	}
	item := parseItemGlobal(text)
	if item.Base == nil {
		return false
	}
	slotName := item.getPrimarySlot()
	i := slices.IndexFunc(orderedItemSlots, func(s itemSlot) bool { return s.name == slotName })
	if i < 0 {
		return false
	}
	set, _ := setItems(build)
	if orderedItemSlots[i].weaponSet == 1 && useSecondWeaponSet(set) {
		slotName += " Swap"
	}
	if second {
		if alt := strings.ReplaceAll(slotName, "1", "2"); IsItemValidForSlot(build, item, alt) {
			slotName = alt
		}
	}
	if setSlotItemID(set, slotName) == itemID {
		build.SetSlotItem(slotName, 0)
	} else {
		build.SetSlotItem(slotName, itemID)
	}
	PopulateSlots(build)
	return true
}

//pob:block:src/Classes/ItemsTab.lua:156-188,src/Classes/ItemSlotControl.lua:98-111
func ShownAbyssalSlots(build *pob.PathOfBuilding) []string {
	var out []string
	for _, s := range orderedItemSlots {
		if s.parentSlot != "" && slotShown(build, s, false) {
			out = append(out, s.name)
		}
	}
	return out
}

//pob:block:src/Classes/ItemsTab.lua:1652-1670
func ActiveTreeSockets(build *pob.PathOfBuilding) []int {
	var spec pob.Spec
	if i := build.ActiveSpecIndex(); i >= 0 {
		spec = build.Tree.Specs[i]
	}
	alloc := specAllocated(spec)
	var out []int
	for _, id := range treeSocketIDs() {
		if alloc[id] {
			out = append(out, id)
		}
	}
	return out
}

//pob:src/Classes/ItemListControl.lua:77-111
func ItemListLoadoutFilters(build *pob.PathOfBuilding) []string {
	list := []string{"Any Loadout", "Current Loadout", "Unused Items"}
	seen := map[string]bool{}
	for _, l := range list {
		seen[l] = true
	}
	add := func(v string) {
		if !seen[v] {
			list = append(list, v)
			seen[v] = true
		}
	}
	for _, l := range Loadouts(build).Loadouts {
		add(l)
	}
	for _, s := range build.Items.ItemSets {
		add(orDefault(s.Title))
	}
	return list
}

var loadoutVersionPrefixRe = regexp.MustCompile(`^\[[^\]]+\]\s*`)

//pob:block:src/Classes/ItemListControl.lua:113-183
func ItemListFiltered(build *pob.PathOfBuilding, filter string) []int {
	var out []int
	switch filter {
	case "", "Any Loadout":
		for _, it := range build.Items.Items {
			out = append(out, it.ID)
		}
		return out
	case "Unused Items":
		return UnusedItemIDs(build)
	}
	setIdx, specIdx := -1, -1
	if filter == "Current Loadout" {
		setIdx, specIdx = activeItemSetIndex(build.Items), build.ActiveSpecIndex()
	} else {
		title := loadoutVersionPrefixRe.ReplaceAllString(filter, "")
		for k, s := range build.Items.ItemSets {
			if setIdx < 0 && orDefault(s.Title) == title {
				setIdx = k
			}
		}
		for k, s := range build.Tree.Specs {
			if specIdx < 0 && orDefault(s.Title) == title {
				specIdx = k
			}
		}
		st := loadoutsOf(build)
		link := ""
		if m := loadoutMatchRe.FindStringSubmatch(filter); m != nil {
			link = m[1]
		}
		if setIdx < 0 {
			if len(build.Items.ItemSets) == 1 {
				setIdx = 0
			} else if l, ok := st.itemLinks[link]; ok && link != "" {
				setIdx = slices.IndexFunc(build.Items.ItemSets, func(s pob.ItemSet) bool { return s.ID == strconv.Itoa(l.setID) })
			}
		}
		if specIdx < 0 {
			if len(build.Tree.Specs) == 1 {
				specIdx = 0
			} else {
				for _, t := range st.treeLinks {
					if link != "" && t.linkID == link {
						specIdx = t.setID - 1
					}
				}
			}
		}
	}
	held := map[int]bool{}
	if setIdx >= 0 && setIdx < len(build.Items.ItemSets) {
		for _, s := range build.Items.ItemSets[setIdx].Slots {
			held[s.ItemID] = true
		}
	}
	if specIdx >= 0 && specIdx < len(build.Tree.Specs) {
		spec := build.Tree.Specs[specIdx]
		alloc := pob.SpecNodes(spec)
		if specIdx == build.ActiveSpecIndex() {
			alloc = build.Build.PassiveNodes
		}
		for _, s := range spec.Sockets {
			if slices.Contains(alloc, int64(s.NodeID)) {
				held[s.ItemID] = true
			}
		}
	}
	for _, it := range build.Items.Items {
		if held[it.ID] {
			out = append(out, it.ID)
		}
	}
	return out
}
