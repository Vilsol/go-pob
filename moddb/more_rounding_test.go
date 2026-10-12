package moddb

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMoreIsRoundedPerName pins ModDB.lua:163-188.
func TestMoreIsRoundedPerName(t *testing.T) {
	newDB := func(mods ...mod.Mod) *ModDB {
		db := NewModDB()
		for _, m := range mods {
			db.AddMod(m)
		}
		return db
	}

	t.Run("a single more is rounded to two decimals", func(t *testing.T) {
		db := newDB(mod.NewFloat("Damage", mod.TypeMore, 12.5))
		if got := db.More(nil, "Damage"); got != 1.13 {
			t.Errorf("More = %v, want 1.13 (1.125 rounded); the mod COUNT is not what matters", got)
		}
	})

	t.Run("a product already on two decimals is untouched", func(t *testing.T) {
		db := newDB(mod.NewFloat("Damage", mod.TypeMore, 10))
		if got := db.More(nil, "Damage"); got != 1.1 {
			t.Errorf("More = %v, want 1.1 exactly", got)
		}
	})

	t.Run("each NAME rounds separately", func(t *testing.T) {
		db := newDB(
			mod.NewFloat("Damage", mod.TypeMore, 12.5),
			mod.NewFloat("Speed", mod.TypeMore, 12.5),
		)
		got := db.More(nil, "Damage", "Speed")
		a, b := 1.13, 1.13
		if want := a * b; got != want {
			t.Errorf("More over two names = %v, want %v; upstream folds one rounded modResult per "+
				"NAME, it does not round the combined product", got, want)
		}
	})

	t.Run("two mods on ONE name round once, together", func(t *testing.T) {
		db := newDB(
			mod.NewFloat("Damage", mod.TypeMore, 30),
			mod.NewFloat("Damage", mod.TypeMore, 25),
		)
		if got := db.More(nil, "Damage"); got != 1.63 {
			t.Errorf("More = %v, want 1.63 (1.625 rounded once, at the end of the name)", got)
		}
	})

	t.Run("SupportManaMultiplier takes the high-precision floor instead", func(t *testing.T) {
		db := newDB(mod.NewFloat("SupportManaMultiplier", mod.TypeMore, 12.5))
		if got := db.More(nil, "SupportManaMultiplier"); got != 1.125 {
			t.Errorf("More = %v, want 1.125 kept at precision 4, not rounded to 1.13", got)
		}
	})
}
