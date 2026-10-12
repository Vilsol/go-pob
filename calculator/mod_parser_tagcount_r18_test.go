package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestTagCountRowsR18: Three rows the TAG-COUNT differential.
func TestTagCountRowsR18(t *testing.T) {
	tagsOf := func(t *testing.T, m mod.Mod) []mod.Tag {
		t.Helper()
		return m.Tags()
	}

	t.Run("minion impale scales per socketed Ghastly Eye Jewel", func(t *testing.T) {
		mods, extra := parseMod("Minions have 30% chance to Impale on Attack Hit per socketed Ghastly Eye Jewel", 1)
		testza.AssertEqual(t, "", extra)
		if len(mods) != 1 {
			t.Fatalf("want 1 mod, got %d", len(mods))
		}
		testza.AssertEqual(t, "MinionModifier", mods[0].Name())
		tags := tagsOf(t, mods[0])
		if len(tags) != 1 {
			t.Fatalf("want exactly the GhastlyEyeJewelIn{SlotName} multiplier, got %d tags: %+v", len(tags), tags)
		}
		mult, ok := tags[0].(*mod.MultiplierTag)
		if !ok {
			t.Fatalf("want a MultiplierTag, got %T", tags[0])
		}
		testza.AssertTrue(t, len(mult.VariableList) == 1 && mult.VariableList[0] == "GhastlyEyeJewelIn{SlotName}", "multiplier must be GhastlyEyeJewelIn{SlotName}, got %+v", mult.VariableList)
	})

	t.Run("totem arm of the Hierophant regen loses its PerStat upstream", func(t *testing.T) {
		mods, extra := parseMod("You and your Totems Regenerate 0.5% of Life per second for each Summoned Totem", 1)
		testza.AssertEqual(t, "", extra)
		if len(mods) != 2 {
			t.Fatalf("want 2 mods, got %d", len(mods))
		}
		testza.AssertEqual(t, "LifeRegenPercent", mods[0].Name())
		testza.AssertEqual(t, "LifeRegenPercent", mods[1].Name())
		testza.AssertEqual(t, 1, len(tagsOf(t, mods[0])), "the player arm keeps its PerStat")
		testza.AssertEqual(t, mod.KeywordFlag(0), mods[0].KeywordFlags())
		testza.AssertEqual(t, 0, len(tagsOf(t, mods[1])), "the TOTEM arm carries no tags upstream — createMod eats it")
		testza.AssertEqual(t, mod.KeywordFlagTotem, mods[1].KeywordFlags())
	})
}

// TestTagCountOverRestrictionsR18 covers the two rows in the same differential where OURS carries a tag upstream does not.
func TestTagCountOverRestrictionsR18(t *testing.T) {
	t.Run("socketed-spells-on-Focus support is gated on the socket only", func(t *testing.T) {
		for _, line := range []string{
			"Trigger Socketed Spells when you Focus",
			"Trigger Socketed Spells when you Focus, with a 0.25 second Cooldown",
		} {
			mods, extra := parseMod(line, 1)
			testza.AssertEqual(t, "", extra, line)
			if len(mods) != 1 {
				t.Fatalf("%q: want 1 mod, got %d", line, len(mods))
			}
			testza.AssertEqual(t, "ExtraSupport", mods[0].Name())
			tags := mods[0].Tags()
			if len(tags) != 1 {
				t.Fatalf("%q: want only the SocketedIn tag, got %d: %+v", line, len(tags), tags)
			}
			if _, ok := tags[0].(*mod.SocketedInTag); !ok {
				t.Fatalf("%q: want a SocketedInTag, got %T", line, tags[0])
			}
		}
	})

	t.Run("cold exposure whole-line row beats the with-cold-damage suffix", func(t *testing.T) {
		mods, extra := parseMod("10% Chance to Inflict Cold Exposure on Hit with Cold Damage", 1)
		testza.AssertEqual(t, "", extra)
		if len(mods) != 1 {
			t.Fatalf("want 1 mod, got %d", len(mods))
		}
		testza.AssertEqual(t, "ColdExposureChance", mods[0].Name())
		testza.AssertEqual(t, 0, len(mods[0].Tags()), "upstream's whole-line row carries no tags")

		ctrl, extra := parseMod("30% increased Critical Strike Chance with Cold Damage", 1)
		testza.AssertEqual(t, "", extra)
		if len(ctrl) != 1 {
			t.Fatalf("control: want 1 mod, got %d", len(ctrl))
		}
		ctrlTags := ctrl[0].Tags()
		if len(ctrlTags) != 1 {
			t.Fatalf("control: the suffix must still tag lines it owns, got %d tags: %+v", len(ctrlTags), ctrlTags)
		}
		cond, ok := ctrlTags[0].(*mod.ConditionTag)
		if !ok {
			t.Fatalf("control: want a ConditionTag, got %T", ctrlTags[0])
		}
		testza.AssertTrue(t, len(cond.VarList) == 1 && cond.VarList[0] == "ColdHasDamage", "control condition, got %+v", cond.VarList)
	})
}
