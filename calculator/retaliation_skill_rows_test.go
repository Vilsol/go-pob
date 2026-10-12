package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestRetaliationSkillRowsParse covers the three Retaliation rows of ModParser.lua.
func TestRetaliationSkillRowsParse(t *testing.T) {
	requireGameData(t, "3.29")

	t.Run("deal, through the preFlagList prefix", func(t *testing.T) {
		mods, extra := parseMod("Retaliation Skills deal 16% increased Damage", 1)
		testza.AssertEqual(t, "", extra)
		if len(mods) != 1 {
			t.Fatalf("got %d mods, want 1", len(mods))
		}
		testza.AssertEqual(t, "Damage", mods[0].Name())
		testza.AssertEqual(t, mod.TypeIncrease, mods[0].Type())
		testza.AssertEqual(t, float64(16), mods[0].Value().ValueFloat)
		if len(mods[0].Tags()) != 1 {
			t.Fatalf("want exactly the SkillType tag, got %d tags", len(mods[0].Tags()))
		}
		testza.AssertEqual(t, mod.TypeSkillType, mods[0].Tags()[0].Type())
	})

	t.Run("have, same prefix", func(t *testing.T) {
		mods, extra := parseMod("Retaliation Skills have 15% increased Area of Effect", 1)
		testza.AssertEqual(t, "", extra)
		if len(mods) != 1 {
			t.Fatalf("got %d mods, want 1", len(mods))
		}
		testza.AssertEqual(t, "AreaOfEffect", mods[0].Name())
	})

	t.Run("speed needs its own row and writes two mods", func(t *testing.T) {
		mods, extra := parseMod("Retaliation Skills have 6% increased Speed", 1)
		testza.AssertEqual(t, "", extra)
		if len(mods) != 2 {
			t.Fatalf("got %d mods, want 2 (Speed and WarcrySpeed)", len(mods))
		}
		testza.AssertEqual(t, "Speed", mods[0].Name())
		testza.AssertEqual(t, "WarcrySpeed", mods[1].Name())
	})

	t.Run("cost of retaliation skills", func(t *testing.T) {
		mods, extra := parseMod("10% reduced Cost of Retaliation Skills", 1)
		testza.AssertEqual(t, "", extra)
		if len(mods) != 1 {
			t.Fatalf("got %d mods, want 1", len(mods))
		}
		testza.AssertEqual(t, "Cost", mods[0].Name())
		testza.AssertEqual(t, float64(-10), mods[0].Value().ValueFloat)
	})

	t.Run("Fortify on Melee Hit is dropped by BOTH sides", func(t *testing.T) {
		mods, extra := parseMod("Retaliation Skills Fortify on Melee Hit", 1)
		testza.AssertNotEqual(t, "", extra, "PoB leaves a remainder here too")
		testza.AssertEqual(t, 0, len(mods))
	})

	t.Run("SkillType.Retaliation resolves", func(t *testing.T) {
		testza.AssertEqual(t, data.SkillType("Retaliation"), data.SkillTypeRetaliation)
	})
}
