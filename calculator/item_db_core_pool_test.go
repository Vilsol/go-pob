package calculator

import "testing"

// TestItemDBCoreDropPool: ItemDBControl.lua:139-140 (#10156): the unique DB's "Core Drop Pool" keeps only uniques with no Source line.
func TestItemDBCoreDropPool(t *testing.T) {
	initItemsTabData(t)
	if ItemDBObtainable[7] != "Core Drop Pool" {
		t.Fatalf("obtainable list %q", ItemDBObtainable)
	}
	core := FilterItemDB(ItemDBUnique, ItemDBFilter{Obtainable: 8}, ItemDBContext{})
	all := FilterItemDB(ItemDBUnique, ItemDBFilter{Obtainable: 2}, ItemDBContext{})
	if len(core) == 0 || len(core) >= len(all) {
		t.Fatalf("core drop pool %d of %d uniques", len(core), len(all))
	}
	for _, item := range core {
		if item.Source != "" {
			t.Fatalf("%s has source %q", item.Name, item.Source)
		}
	}
}
