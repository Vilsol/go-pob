package pob

import (
	"slices"
	"strconv"
	"strings"
)

//pob:block:src/Classes/ItemsTab.lua:156-177,src/Classes/ItemsTab.lua:187-200
func weaponSetOf(slotName string) int {
	if !strings.HasPrefix(slotName, "Weapon ") {
		return 0
	}
	if strings.Contains(slotName, " Swap") {
		return 2
	}
	return 1
}

//pob:block:src/Classes/ItemsTab.lua:236-275
func (b *PathOfBuilding) SetUseSecondWeaponSet(second bool) bool {
	set := b.activeItemSet()
	if (set.UseSecondWeaponSet != nil && *set.UseSecondWeaponSet) == second {
		return false
	}
	set.UseSecondWeaponSet = &second
	b.Items.UseSecondWeaponSet = &second
	from, to := 2, 1
	if second {
		from, to = 1, 2
	}
	if len(b.Skills.SkillSets) == 0 {
		return true
	}
	groups := b.Skills.SkillSets[b.ActiveSkillSetIndex()].Skills
	main := b.Build.MainSocketGroup
	if main < 1 || main > len(groups) || groups[main-1].Slot == "" || weaponSetOf(groups[main-1].Slot) != from {
		return true
	}
	for index, group := range groups {
		if group.Slot != "" && weaponSetOf(group.Slot) == to {
			b.Build.MainSocketGroup = index + 1
			break
		}
	}
	return true
}

func (b *PathOfBuilding) nextItemSetID() int {
	id := 1
	for b.itemSetIndex(id) >= 0 {
		id++
	}
	return id
}

func (b *PathOfBuilding) itemSetIndex(itemSetID int) int {
	return slices.IndexFunc(b.Items.ItemSets, func(s ItemSet) bool {
		id, err := strconv.Atoi(s.ID)
		return err == nil && id == itemSetID
	})
}

//pob:block:src/Classes/ItemSetListControl.lua:41-44,src/Classes/ItemSetListControl.lua:54-61
func (b *PathOfBuilding) NewItemSet(title string) int {
	b.activeItemSet()
	id := b.nextItemSetID()
	b.Items.ItemSets = append(b.Items.ItemSets, ItemSet{ID: strconv.Itoa(id), Title: &title})
	return id
}

//pob:block:src/Classes/ItemSetListControl.lua:17-25,src/Classes/ItemSetListControl.lua:54-61
func (b *PathOfBuilding) CopyItemSet(itemSetID int, title string) int {
	i := b.itemSetIndex(itemSetID)
	if i < 0 {
		return 0
	}
	from := b.Items.ItemSets[i]
	id := b.nextItemSetID()
	set := ItemSet{ID: strconv.Itoa(id), Title: &title, Slots: slices.Clone(from.Slots)}
	if from.UseSecondWeaponSet != nil {
		second := *from.UseSecondWeaponSet
		set.UseSecondWeaponSet = &second
	}
	b.Items.ItemSets = append(b.Items.ItemSets, set)
	return id
}

//pob:block:src/Classes/ItemSetListControl.lua:54-55
func (b *PathOfBuilding) RenameItemSet(itemSetID int, title string) {
	if i := b.itemSetIndex(itemSetID); i >= 0 {
		b.Items.ItemSets[i].Title = &title
	}
}

//pob:block:src/Classes/ItemSetListControl.lua:123-138
func (b *PathOfBuilding) DeleteItemSet(itemSetID int) bool {
	if len(b.Items.ItemSets) <= 1 {
		return false
	}
	i := b.itemSetIndex(itemSetID)
	if i < 0 {
		return false
	}
	b.Items.ItemSets = slices.Delete(b.Items.ItemSets, i, i+1)
	if itemSetID != b.Items.ActiveItemSet {
		return false
	}
	next, _ := strconv.Atoi(b.Items.ItemSets[max(0, i-1)].ID)
	b.SetActiveItemSet(next)
	return true
}

//pob:block:src/Classes/ListControl.lua:441-449,src/Classes/ItemSetListControl.lua:112-114
func (b *PathOfBuilding) MoveItemSet(from, to int) {
	n := len(b.Items.ItemSets)
	if from < 1 || from > n || to < 1 || to > n || from == to {
		return
	}
	set := b.Items.ItemSets[from-1]
	b.Items.ItemSets = slices.Insert(slices.Delete(b.Items.ItemSets, from-1, from), to-1, set)
}

func (b *PathOfBuilding) DeallocateSpecNodes(specIndex int, nodeIDs []int64) {
	if specIndex < 0 || specIndex >= len(b.Tree.Specs) || len(nodeIDs) == 0 {
		return
	}
	if specIndex == b.ActiveSpecIndex() {
		b.DeallocateNodes(nodeIDs)
		return
	}
	spec := &b.Tree.Specs[specIndex]
	parts := make([]string, 0, 100)
	for _, n := range SpecNodes(*spec) {
		if !slices.Contains(nodeIDs, n) {
			parts = append(parts, strconv.FormatInt(n, 10))
		}
	}
	spec.NodesAttr = strings.Join(parts, ",")
	if spec.MasteryEffects != "" {
		sel := SpecMasterySelections(*spec)
		kept := slices.DeleteFunc(slices.Clone(sel), func(m MasterySelection) bool { return slices.Contains(nodeIDs, m.Node) })
		if len(kept) != len(sel) {
			spec.MasteryEffects = formatMasterySelections(kept)
		}
	}
}

//pob:src/Classes/ItemsTab.lua:1857-1904
func (b *PathOfBuilding) DeleteItem(itemID int) bool {
	i := slices.IndexFunc(b.Items.Items, func(it Item) bool { return it.ID == itemID })
	if i < 0 {
		return false
	}
	for s := range b.Items.ItemSets {
		for k := range b.Items.ItemSets[s].Slots {
			if b.Items.ItemSets[s].Slots[k].ItemID == itemID {
				b.Items.ItemSets[s].Slots[k].ItemID = 0
			}
		}
	}
	b.Items.Items = slices.Delete(b.Items.Items, i, i+1)
	for k := range b.Tree.Specs {
		b.Tree.Specs[k].Sockets = slices.DeleteFunc(b.Tree.Specs[k].Sockets, func(s Socket) bool { return s.ItemID == itemID })
	}
	return true
}

//pob:block:src/Classes/ItemListControl.lua:54-64
func (b *PathOfBuilding) DeleteAllItems() {
	set := b.activeItemSet()
	for k := range set.Slots {
		set.Slots[k].ItemID = 0
	}
	for k := range b.Tree.Specs {
		b.Tree.Specs[k].Sockets = []Socket{}
	}
	b.Items.Items = []Item{}
}

func (b *PathOfBuilding) StatDifferencesShown() bool {
	return b.Items.ShowStatDifferences == nil || *b.Items.ShowStatDifferences
}

func (b *PathOfBuilding) SetShowStatDifferences(on bool) {
	b.Items.ShowStatDifferences = &on
}

//pob:block:src/Classes/ListControl.lua:441-449,src/Classes/ItemListControl.lua:295-297
func (b *PathOfBuilding) MoveItem(id, beforeID int) {
	from := slices.IndexFunc(b.Items.Items, func(it Item) bool { return it.ID == id })
	if from < 0 || id == beforeID {
		return
	}
	item := b.Items.Items[from]
	b.Items.Items = slices.Delete(b.Items.Items, from, from+1)
	to := slices.IndexFunc(b.Items.Items, func(it Item) bool { return it.ID == beforeID })
	if to < 0 {
		to = len(b.Items.Items)
	}
	b.Items.Items = slices.Insert(b.Items.Items, to, item)
}
