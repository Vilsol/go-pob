package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestProjectilesVerbPrefixParsesWhole covers ModParser.lua:1199.
func TestProjectilesVerbPrefixParsesWhole(t *testing.T) {
	t.Run("Projectiles deal", func(t *testing.T) {
		mods, extra := parseMod("Projectiles deal 35% increased Damage with Hits against nearby Enemies", 1)
		testza.AssertEqual(t, "", extra, "the verb must be consumed by the prefix row")
		if len(mods) != 1 {
			t.Fatalf("got %d mods, want 1", len(mods))
		}
		testza.AssertEqual(t, "Damage", mods[0].Name())
		testza.AssertEqual(t, mod.TypeIncrease, mods[0].Type())
		testza.AssertEqual(t, float64(35), mods[0].Value().ValueFloat)
	})

	t.Run("Projectiles have", func(t *testing.T) {
		mods, extra := parseMod("Projectiles have 20% increased Critical Strike Chance", 1)
		testza.AssertEqual(t, "", extra)
		if len(mods) != 1 {
			t.Fatalf("got %d mods, want 1", len(mods))
		}
		testza.AssertEqual(t, "CritChance", mods[0].Name())
	})

	t.Run("bare projectiles row survives", func(t *testing.T) {
		mods, extra := parseMod("Projectiles Pierce an additional Target", 1)
		testza.AssertEqual(t, "", extra)
		if len(mods) != 1 {
			t.Fatalf("got %d mods, want 1", len(mods))
		}
		testza.AssertEqual(t, "PierceCount", mods[0].Name())
	})
}
