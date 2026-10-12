package pob

import (
	"slices"
	"testing"
)

func setIDs(b *PathOfBuilding) []string {
	out := make([]string, len(b.Items.ItemSets))
	for i, s := range b.Items.ItemSets {
		out[i] = s.ID
	}
	return out
}

// TestNewItemSetTakesTheLowestFreeID: ItemsTab.lua:1566-1573: New takes the lowest id no set holds, and the list appends it (ItemSetListControl.lua:54).
func TestNewItemSetTakesTheLowestFreeID(t *testing.T) {
	b := twoItemSets()
	if id := b.NewItemSet("pvp"); id != 1 {
		t.Fatalf("id = %d, want 1 (the sets hold 3 and 2)", id)
	}
	if id := b.NewItemSet("boss"); id != 4 {
		t.Fatalf("id = %d, want 4", id)
	}
	if got := setIDs(b); !slices.Equal(got, []string{"3", "2", "1", "4"}) {
		t.Fatalf("order = %v, want [3 2 1 4]", got)
	}
	if s := b.Items.ItemSets[2]; s.Title == nil || *s.Title != "pvp" || len(s.Slots) != 0 {
		t.Fatalf("new set = %+v, want an empty set titled pvp", s)
	}
	if b.Items.ActiveItemSet != 3 {
		t.Fatalf("New switched the active set to %d", b.Items.ActiveItemSet)
	}
}

// TestCopyItemSetCopiesSlotsAndWeaponSet: ItemSetListControl.lua:14-22: Copy is a copy of the selected set — its slots, flask states and weapon set.
func TestCopyItemSetCopiesSlotsAndWeaponSet(t *testing.T) {
	b := twoItemSets()
	on := true
	b.Items.ItemSets[0].UseSecondWeaponSet = &on
	id := b.CopyItemSet(3, "copy")
	if id != 1 {
		t.Fatalf("id = %d, want 1", id)
	}
	c := b.Items.ItemSets[2]
	if c.Title == nil || *c.Title != "copy" || c.UseSecondWeaponSet == nil || !*c.UseSecondWeaponSet {
		t.Fatalf("copy = %+v", c)
	}
	if !slices.Equal(c.Slots, b.Items.ItemSets[0].Slots) {
		t.Fatalf("slots = %+v, want %+v", c.Slots, b.Items.ItemSets[0].Slots)
	}
	c.Slots[0].ItemID = 9
	*c.UseSecondWeaponSet = false
	if b.Items.ItemSets[0].Slots[0].ItemID != 1 || !*b.Items.ItemSets[0].UseSecondWeaponSet {
		t.Fatal("the copy shares its slots or weapon set with the original")
	}
	if b.CopyItemSet(7, "none") != 0 || len(b.Items.ItemSets) != 3 {
		t.Fatal("copying a set that does not exist added one")
	}
}

func TestRenameItemSet(t *testing.T) {
	b := twoItemSets()
	b.RenameItemSet(2, "mapping")
	if s := b.Items.ItemSets[1]; s.Title == nil || *s.Title != "mapping" {
		t.Fatalf("set 2 = %+v", s)
	}
}

// TestDeleteItemSet: ItemSetListControl.lua:119-134: never the last set; deleting the active set activates the one before it in the list, or the new first.
func TestDeleteItemSet(t *testing.T) {
	b := twoItemSets()
	b.NewItemSet("c")
	if b.DeleteItemSet(2) || b.Items.ActiveItemSet != 3 {
		t.Fatalf("deleting an inactive set changed the active set to %d", b.Items.ActiveItemSet)
	}
	b.SetActiveItemSet(1)
	if !b.DeleteItemSet(1) || b.Items.ActiveItemSet != 3 {
		t.Fatalf("active = %d, want 3 (the set before the deleted one)", b.Items.ActiveItemSet)
	}
	b.NewItemSet("d")
	if !b.DeleteItemSet(3) || b.Items.ActiveItemSet != 1 {
		t.Fatalf("active = %d, want 1 (the new first)", b.Items.ActiveItemSet)
	}
	if b.DeleteItemSet(1) || len(b.Items.ItemSets) != 1 {
		t.Fatal("the last set was deleted")
	}
	b = twoItemSets()
	b.NewItemSet("c")
	b.SetActiveItemSet(2)
	if !b.DeleteItemSet(2) || b.Items.ActiveItemSet != 3 {
		t.Fatalf("active = %d, want 3 (list[index - 1])", b.Items.ActiveItemSet)
	}
}

// TestMoveItemSet: ListControl.lua:428-436: a drag lands the set at the position it ends at; the active set is an id and stays.
func TestMoveItemSet(t *testing.T) {
	b := twoItemSets()
	b.NewItemSet("c")
	b.MoveItemSet(1, 3)
	if got := setIDs(b); !slices.Equal(got, []string{"2", "1", "3"}) {
		t.Fatalf("order = %v, want [2 1 3]", got)
	}
	b.MoveItemSet(3, 1)
	if got := setIDs(b); !slices.Equal(got, []string{"3", "2", "1"}) {
		t.Fatalf("order = %v, want [3 2 1]", got)
	}
	if b.Items.ActiveItemSet != 3 {
		t.Fatalf("active = %d, want 3", b.Items.ActiveItemSet)
	}
}

// TestDeleteItemAndDeallocateAnInactiveSpec: ItemsTab.lua:1789-1805 reaches every tree.
func TestDeleteItemAndDeallocateAnInactiveSpec(t *testing.T) {
	b := twoItemSets()
	b.Tree = Tree{ActiveSpec: 1, Specs: []Spec{
		{NodesAttr: "5,6", Sockets: []Socket{{NodeID: 6, ItemID: 2}}},
		{NodesAttr: "7,8,9,10", MasteryEffects: "{9,100},{10,200}", Sockets: []Socket{{NodeID: 8, ItemID: 2}, {NodeID: 11, ItemID: 1}}},
	}}
	b.Build.PassiveNodes = []int64{5, 6}
	if !b.DeleteItem(2) || len(b.Items.Items) != 1 {
		t.Fatal("item 2 was not deleted")
	}
	if b.Items.ItemSets[0].Slots[1].ItemID != 0 || b.Items.ItemSets[1].Slots[0].ItemID != 0 {
		t.Fatalf("sets still hold item 2: %+v", b.Items.ItemSets)
	}
	if len(b.Tree.Specs[0].Sockets) != 0 || !slices.Equal(b.Tree.Specs[1].Sockets, []Socket{{NodeID: 11, ItemID: 1}}) {
		t.Fatalf("sockets = %+v / %+v", b.Tree.Specs[0].Sockets, b.Tree.Specs[1].Sockets)
	}
	b.DeallocateSpecNodes(1, []int64{8, 9})
	if s := b.Tree.Specs[1]; s.NodesAttr != "7,10" || s.MasteryEffects != "{10,200}" {
		t.Fatalf("inactive spec = %q / %q", s.NodesAttr, s.MasteryEffects)
	}
	b.DeallocateSpecNodes(0, []int64{6})
	if !slices.Equal(b.Build.PassiveNodes, []int64{5}) || b.Tree.Specs[0].NodesAttr != "5" {
		t.Fatalf("active spec = %v / %q", b.Build.PassiveNodes, b.Tree.Specs[0].NodesAttr)
	}
	if b.DeleteItem(2) {
		t.Fatal("deleting a missing item reported a change")
	}
}

// TestMoveItem: ListControl.lua:428-436 on the item list: a dragged item lands just before the row it is dropped on, or last.
func TestMoveItem(t *testing.T) {
	b := &PathOfBuilding{}
	for _, s := range []string{"a", "b", "c", "d"} {
		b.AddItem(s)
	}
	ids := func() []int {
		out := make([]int, 0, len(b.Items.Items))
		for _, it := range b.Items.Items {
			out = append(out, it.ID)
		}
		return out
	}
	b.MoveItem(4, 2)
	if got := ids(); !slices.Equal(got, []int{1, 4, 2, 3}) {
		t.Fatalf("order = %v, want [1 4 2 3]", got)
	}
	b.MoveItem(1, 3)
	if got := ids(); !slices.Equal(got, []int{4, 2, 1, 3}) {
		t.Fatalf("order = %v, want [4 2 1 3]", got)
	}
	b.MoveItem(4, 0)
	if got := ids(); !slices.Equal(got, []int{2, 1, 3, 4}) {
		t.Fatalf("order = %v, want [2 1 3 4]", got)
	}
	b.MoveItem(9, 1)
	b.MoveItem(2, 2)
	if got := ids(); !slices.Equal(got, []int{2, 1, 3, 4}) {
		t.Fatalf("an unknown or self move changed the order: %v", got)
	}
}

// TestApplyNewBuildDefaults: Build.lua:79-84, SkillsTab.lua:92.
func TestApplyNewBuildDefaults(t *testing.T) {
	for _, c := range []struct{ level, quality, wantLevel, wantQuality int }{{1, 0, 1, 0}, {90, 20, 90, 20}, {150, 40, 100, 23}, {0, 5, 1, 5}} {
		b := &PathOfBuilding{}
		b.ApplyNewBuildDefaults(c.level, c.quality)
		if b.Build.Level != c.wantLevel || b.Build.CharacterLevelAutoMode != (c.wantLevel == 1) || b.Skills.DefaultGemQuality == nil || *b.Skills.DefaultGemQuality != c.wantQuality {
			t.Errorf("defaults %d / %d: level %d auto %v quality %v", c.level, c.quality, b.Build.Level, b.Build.CharacterLevelAutoMode, b.Skills.DefaultGemQuality)
		}
	}
}
