package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestNilCfgExcludesFlaggedMods: The flags predicate has the identical nil-cfg rule as the keyword one.
func TestNilCfgExcludesFlaggedMods(t *testing.T) {
	plain := mod.NewFloat("Damage", mod.TypeIncrease, 10)
	spell := mod.NewFloat("Damage", mod.TypeIncrease, 10).Flag(mod.MFlagSpell)

	t.Run("ModDB Sum", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(plain)
		db.AddMod(spell)

		testza.AssertEqual(t, float64(10), db.Sum(mod.TypeIncrease, nil, "Damage"))
	})

	t.Run("ModList Sum", func(t *testing.T) {
		list := NewModList()
		list.AddMod(plain)
		list.AddMod(spell)

		testza.AssertEqual(t, float64(10), list.Sum(mod.TypeIncrease, nil, "Damage"))
	})

	t.Run("cfg with nil Flags", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(plain)
		db.AddMod(spell)

		testza.AssertEqual(t, float64(10), db.Sum(mod.TypeIncrease, &ListCfg{}, "Damage"))
	})

	t.Run("cfg carrying Spell admits both", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(plain)
		db.AddMod(spell)

		flags := mod.MFlagSpell | mod.MFlagCast
		testza.AssertEqual(t, float64(20), db.Sum(mod.TypeIncrease, &ListCfg{Flags: &flags}, "Damage"))
	})

	t.Run("cfg carrying only Attack rejects the Spell mod", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(plain)
		db.AddMod(spell)

		flags := mod.MFlagAttack
		testza.AssertEqual(t, float64(10), db.Sum(mod.TypeIncrease, &ListCfg{Flags: &flags}, "Damage"))
	})
}

// TestNilCfgExcludesFlaggedModsAcrossQueries: The same nil-cfg rule governs More.
func TestNilCfgExcludesFlaggedModsAcrossQueries(t *testing.T) {
	t.Run("More", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("Damage", mod.TypeMore, 50).Flag(mod.MFlagSpell))

		testza.AssertEqual(t, float64(1), db.More(nil, "Damage"))
	})

	t.Run("Flag", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFlag("Onslaught", true).Flag(mod.MFlagSpell))

		testza.AssertEqual(t, false, db.Flag(nil, "Onslaught"))
	})

	t.Run("Override", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("Speed", mod.TypeOverride, 7).Flag(mod.MFlagSpell))

		testza.AssertNil(t, db.Override(nil, "Speed"))
	})

	t.Run("List", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewList("ExtraAura", "x").Flag(mod.MFlagSpell))

		testza.AssertEqual(t, 0, len(db.List(nil, "ExtraAura")))
	})
}
