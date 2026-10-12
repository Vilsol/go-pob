package pob

import (
	"testing"
)

func twoItemSets() *PathOfBuilding {
	return &PathOfBuilding{Items: Items{
		ActiveItemSet: 3,
		Items:         []Item{{ID: 1}, {ID: 2}},
		ItemSets: []ItemSet{
			{ID: "3", Slots: []Slot{{Name: "Helmet", ItemID: 1}, {Name: "Flask 1", ItemID: 2, Active: true}}},
			{ID: "2", Slots: []Slot{{Name: "Helmet", ItemID: 2}}},
		},
	}}
}

func TestSetActiveItemSetTakesAnID(t *testing.T) {
	b := twoItemSets()
	b.SetActiveItemSet(2)
	if b.Items.ActiveItemSet != 2 || b.activeItemSet().ID != "2" {
		t.Fatalf("ActiveItemSet = %d (set %q), want 2", b.Items.ActiveItemSet, b.activeItemSet().ID)
	}
}

// TestSetActiveItemSetUnknownIDFallsBackToTheFirstSet: ItemsTab.lua:1586-1588: an id with no set falls back to the first set in order, not to id 1.
func TestSetActiveItemSetUnknownIDFallsBackToTheFirstSet(t *testing.T) {
	b := twoItemSets()
	b.Items.ActiveItemSet = 2
	b.SetActiveItemSet(7)
	if b.Items.ActiveItemSet != 3 {
		t.Fatalf("ActiveItemSet = %d, want 3 (itemSetOrderList[1])", b.Items.ActiveItemSet)
	}
}

// TestSetActiveItemSetCreatesSetOne: ItemsTab.lua:1310-1315: a build always has an item set; with none, set 1 is made and chosen.
func TestSetActiveItemSetCreatesSetOne(t *testing.T) {
	b := &PathOfBuilding{}
	b.SetActiveItemSet(4)
	if len(b.Items.ItemSets) != 1 || b.Items.ItemSets[0].ID != "1" || b.Items.ActiveItemSet != 1 {
		t.Fatalf("got %+v, want one set with id 1, active", b.Items)
	}
}

// TestSetActiveItemSetKeepsEachSetsSlots: ItemsTab.lua:1592-1606: the outgoing set keeps what its slots held.
func TestSetActiveItemSetKeepsEachSetsSlots(t *testing.T) {
	b := twoItemSets()
	b.SetSlotItem("Gloves", 2)
	b.SetActiveItemSet(2)
	b.SetSlotItem("Boots", 1)
	b.SetActiveItemSet(3)

	set3, set2 := b.Items.ItemSets[0], b.Items.ItemSets[1]
	if s := set3.slot("Gloves"); s.ItemID != 2 {
		t.Errorf("set 3 Gloves = %d after a round trip, want 2", s.ItemID)
	}
	if s := set3.slot("Flask 1"); s.ItemID != 2 || !s.Active {
		t.Errorf("set 3 Flask 1 = %+v, want item 2, active", s)
	}
	if s := set3.slot("Boots"); s.ItemID != 0 {
		t.Errorf("set 3 Boots = %d, written while set 2 was active", s.ItemID)
	}
	if s := set2.slot("Boots"); s.ItemID != 1 {
		t.Errorf("set 2 Boots = %d, want 1", s.ItemID)
	}
	if s := b.activeItemSet().slot("Helmet"); s.ItemID != 1 {
		t.Errorf("active Helmet = %d, want set 3's 1", s.ItemID)
	}
}
