package moddb

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestConvertMod pins ModStore.lua:127-132 over ModList.lua:54-67 and ModDB.lua:76-106.
func TestConvertMod(t *testing.T) {
	newChain := func() (*ModList, *ModDB) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("FireMin", mod.TypeBase, 10).Source("Item:1"))
		list := NewModList()
		list.Parent = db
		list.AddMod(mod.NewFloat("PhysicalMin", mod.TypeBase, 3).Source("Tree:2"))
		return list, db
	}
	convert := func(list *ModList, oldName, newName string, src mod.Source, v float64) {
		list.ConvertMod(oldName, mod.NewFloat(newName, mod.TypeBase, v).Source(src))
	}

	t.Run("parent ModDB mod moves bucket", func(t *testing.T) {
		list, db := newChain()
		convert(list, "FireMin", "ColdMin", "Item:1", 10)
		if got := db.Sum(mod.TypeBase, nil, "FireMin"); got != 0 {
			t.Fatalf("FireMin left in the parent: %v", got)
		}
		if got := list.Sum(mod.TypeBase, nil, "ColdMin"); got != 10 {
			t.Fatalf("ColdMin = %v, want 10 (and not 20: the mod moves, it is not added)", got)
		}
		if got := len(db.Mods["ColdMin"]); got != 1 {
			t.Fatalf("parent ColdMin bucket holds %d mods, want 1", got)
		}
	})
	t.Run("own ModList mod replaced in place", func(t *testing.T) {
		list, _ := newChain()
		convert(list, "PhysicalMin", "ColdMin", "Tree:2", 3)
		if list.Sum(mod.TypeBase, nil, "PhysicalMin") != 0 || list.Sum(mod.TypeBase, nil, "ColdMin") != 3 {
			t.Fatalf("physical %v cold %v", list.Sum(mod.TypeBase, nil, "PhysicalMin"), list.Sum(mod.TypeBase, nil, "ColdMin"))
		}
	})
	t.Run("source mismatch adds instead of converting", func(t *testing.T) {
		list, db := newChain()
		convert(list, "FireMin", "ColdMin", "Item:9", 10)
		if db.Sum(mod.TypeBase, nil, "FireMin") != 10 {
			t.Fatal("a mod from another source was converted")
		}
		if list.Sum(mod.TypeBase, nil, "ColdMin") != 10 {
			t.Fatal("the unmatched mod was not added")
		}
	})
}
