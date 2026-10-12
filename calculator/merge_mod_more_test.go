package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestMergeModFoldsMore pins ModList.lua:70, whose type list is BASE, INC **and MORE**.
func TestMergeModFoldsMore(t *testing.T) {
	cases := []struct {
		name     string
		modType  mod.Type
		wantLen  int
		wantVal  float64
		whyFolds string
	}{
		{"MORE folds", mod.TypeMore, 1, 40, "ModList.lua:70 names MORE alongside BASE and INC"},
		{"BASE folds", mod.TypeBase, 1, 40, "the control: folding must already work"},
		{"INC folds", mod.TypeIncrease, 1, 40, "the control: folding must already work"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			list := moddb.NewModList()
			mergeMod(list, mod.NewFloat("Damage", c.modType, 20).Source("Test"))
			mergeMod(list, mod.NewFloat("Damage", c.modType, 20).Source("Test"))

			mods := list.Mods()
			if len(mods) != c.wantLen {
				t.Fatalf("%s: list holds %d mods, want %d — %s", c.modType, len(mods), c.wantLen, c.whyFolds)
			}
			if got := mods[0].Value().Float(); got != c.wantVal {
				t.Errorf("%s: folded value = %v, want %v", c.modType, got, c.wantVal)
			}
		})
	}

	t.Run("FLAG still appends", func(t *testing.T) {
		list := moddb.NewModList()
		mergeMod(list, mod.NewFlag("Onslaught", true).Source("Test"))
		mergeMod(list, mod.NewFlag("Onslaught", true).Source("Test"))
		if got := len(list.Mods()); got != 2 {
			t.Errorf("list holds %d mods, want 2 — ModList.lua:70 folds only BASE, INC and MORE", got)
		}
	})
}
