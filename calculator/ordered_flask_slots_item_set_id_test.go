package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestOrderedFlaskSlotsResolvesActiveItemSetByID pins CalcPerform.lua:1548's flask walk to the ACTIVE item set.
func TestOrderedFlaskSlotsResolvesActiveItemSetByID(t *testing.T) {
	requireGameData(t, "3.29")

	ruby := "Rarity: NORMAL\nRuby Flask\n"
	granite := "Rarity: NORMAL\nGranite Flask\n"
	env := &Environment{Build: &pob.PathOfBuilding{Items: pob.Items{
		ActiveItemSet: 1,
		Items:         []pob.Item{{ID: 1, Text: ruby}, {ID: 2, Text: granite}},
		ItemSets: []pob.ItemSet{
			{ID: "2", Slots: []pob.Slot{{ItemID: 2, Name: "Flask 1"}}},
			{ID: "1", Slots: []pob.Slot{{ItemID: 1, Name: "Flask 1"}}},
		},
	}}}

	got := orderedFlaskSlots(env)
	if len(got) != 1 {
		t.Fatalf("want 1 flask from the active set, got %d", len(got))
	}
	if got[0].item.BaseName != "Ruby Flask" {
		t.Errorf("active set id 1 holds the Ruby Flask; got %q (the set at POSITION 0)", got[0].item.BaseName)
	}
}
