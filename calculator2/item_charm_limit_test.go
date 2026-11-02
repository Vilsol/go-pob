package calculator2

import (
	"strings"
	"testing"
)

// A belt base carries charmLimit = 0, so a belt pasted without its Charm Slots line still counts the slots its
// implicit grants; PoB-PoE2's unique database writes Headhunter with "Charm Slots: 2"
// (testdata/item_db_pob.jsonl.gz).
func TestBeltCharmSlotsFromItsBase(t *testing.T) {
	item := ParseItem("Rarity: UNIQUE\nHeadhunter\nHeavy Belt\nImplicits: 2\n(20-30)% increased Stun Threshold\nHas (1-3) Charm Slot\n+(40-60) to maximum Life", ItemModSource(WithParser(ParseMod)))
	if item.CharmLimit == nil || *item.CharmLimit != 2 {
		t.Fatalf("charm limit %v, PoB-PoE2 2", item.CharmLimit)
	}
	if !strings.Contains(item.BuildRaw(), "\nCharm Slots: 2\n") {
		t.Errorf("BuildRaw writes no Charm Slots line:\n%s", item.BuildRaw())
	}
}
