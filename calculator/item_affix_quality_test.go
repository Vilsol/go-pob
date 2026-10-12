package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func TestSetDefaultItemAffixQuality(t *testing.T) {
	const text = "Rarity: RARE\nAffix Quality Fixture\nTitan Greaves\nImplicits: 0\n+(10-20) to maximum Life\n"
	t.Cleanup(func() { SetDefaultItemAffixQuality(0.5) })

	before := ParseItem(text)
	before.BuildModList()
	if got := sumItemMod(before, "Life", mod.TypeBase); got != 15 {
		t.Fatalf("Life BASE at the default 0.5 = %v, want 15", got)
	}
	peeked := PeekItem(text)
	dbBefore := ItemDB(ItemDBUnique)

	SetDefaultItemAffixQuality(1)

	after := ParseItem(text)
	after.BuildModList()
	if got := sumItemMod(after, "Life", mod.TypeBase); got != 20 {
		t.Errorf("Life BASE of a new item at 1 = %v, want 20", got)
	}
	before.BuildModList()
	if got := sumItemMod(before, "Life", mod.TypeBase); got != 15 {
		t.Errorf("Life BASE of the item parsed before = %v, want 15 (it keeps its range)", got)
	}
	if PeekItem(text) == peeked {
		t.Error("the global item cache still holds the item parsed at 0.5")
	}
	if len(dbBefore) == 0 {
		t.Fatal("REFUSING: the unique DB is empty, so its reload cannot be seen")
	}
	if dbAfter := ItemDB(ItemDBUnique); len(dbAfter) == 0 || dbAfter[0] == dbBefore[0] {
		t.Error("the unique DB was not reloaded")
	}

	SetDefaultItemAffixQuality(0.456)
	if *defaultItemAffixQuality != 0.46 {
		t.Errorf("0.456 → %v, want 0.46", *defaultItemAffixQuality)
	}
	SetDefaultItemAffixQuality(3)
	if *defaultItemAffixQuality != 1 {
		t.Errorf("3 → %v, want 1", *defaultItemAffixQuality)
	}
}
