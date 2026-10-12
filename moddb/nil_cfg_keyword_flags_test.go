package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestNilCfgExcludesKeywordFlaggedMods: PoB's ModStore entry points resolve a nil cfg to keywordFlags = 0.
func TestNilCfgExcludesKeywordFlaggedMods(t *testing.T) {
	plain := mod.NewFloat("LifeRegenPercent", mod.TypeBase, 21)
	totem := mod.NewFloat("LifeRegenPercent", mod.TypeBase, 21).KeywordFlag(mod.KeywordFlagTotem)

	t.Run("ModDB Sum", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(plain)
		db.AddMod(totem)

		testza.AssertEqual(t, float64(21), db.Sum(mod.TypeBase, nil, "LifeRegenPercent"))
	})

	t.Run("ModList Sum", func(t *testing.T) {
		list := NewModList()
		list.AddMod(plain)
		list.AddMod(totem)

		testza.AssertEqual(t, float64(21), list.Sum(mod.TypeBase, nil, "LifeRegenPercent"))
	})

	t.Run("cfg with nil KeywordFlags", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(plain)
		db.AddMod(totem)

		testza.AssertEqual(t, float64(21), db.Sum(mod.TypeBase, &ListCfg{}, "LifeRegenPercent"))
	})

	t.Run("cfg carrying Totem admits both", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(plain)
		db.AddMod(totem)

		kw := mod.KeywordFlagTotem
		testza.AssertEqual(t, float64(42), db.Sum(mod.TypeBase, &ListCfg{KeywordFlags: &kw}, "LifeRegenPercent"))
	})
}

// TestNilCfgExcludesKeywordFlaggedModsAcrossQueries: The same nil-cfg rule governs More.
func TestNilCfgExcludesKeywordFlaggedModsAcrossQueries(t *testing.T) {
	t.Run("More", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("Damage", mod.TypeMore, 50).KeywordFlag(mod.KeywordFlagTotem))

		testza.AssertEqual(t, float64(1), db.More(nil, "Damage"))
	})

	t.Run("Flag", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFlag("Onslaught", true).KeywordFlag(mod.KeywordFlagTotem))

		testza.AssertEqual(t, false, db.Flag(nil, "Onslaught"))
	})

	t.Run("Override", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("Speed", mod.TypeOverride, 7).KeywordFlag(mod.KeywordFlagTotem))

		testza.AssertNil(t, db.Override(nil, "Speed"))
	})

	t.Run("List", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewList("ExtraAura", "x").KeywordFlag(mod.KeywordFlagTotem))

		testza.AssertEqual(t, 0, len(db.List(nil, "ExtraAura")))
	})
}
