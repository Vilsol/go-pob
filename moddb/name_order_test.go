package moddb

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestListAndOverrideFollowNameOrder pins the loop NESTING of ModList.lua:173-216.
func TestListAndOverrideFollowNameOrder(t *testing.T) {
	t.Run("Override returns the first override in NAME order", func(t *testing.T) {
		list := NewModList()
		list.AddMod(mod.NewFloat("B", mod.TypeOverride, 2))
		list.AddMod(mod.NewFloat("A", mod.TypeOverride, 1))

		got := list.Override(nil, "A", "B")
		if got == nil {
			t.Fatal("Override returned nil with two matching OVERRIDE mods present")
		}
		if got.Float() != 1 {
			t.Errorf("Override returned %v, want 1 — ModList.lua:174-176 loops the queried NAMES "+
				"outside the mod list, so the first override of the FIRST NAME wins, not the "+
				"first override in list order", got.Float())
		}
	})

	t.Run("List groups results by NAME, not by list order", func(t *testing.T) {
		list := NewModList()
		list.AddMod(mod.NewList("B", "b-payload"))
		list.AddMod(mod.NewList("A", "a-payload"))

		got := list.List(nil, "A", "B")
		if len(got) != 2 {
			t.Fatalf("List returned %d values, want 2", len(got))
		}
		if got[0] != "a-payload" || got[1] != "b-payload" {
			t.Errorf("List returned %v, want [a-payload b-payload] — ModList.lua:196-198 loops the "+
				"queried NAMES outside the mod list, so every mod of the first name is emitted "+
				"before any mod of the second", got)
		}
	})

	t.Run("CONTROL: a single name is unaffected by the nesting", func(t *testing.T) {
		list := NewModList()
		list.AddMod(mod.NewList("A", "first"))
		list.AddMod(mod.NewList("A", "second"))

		got := list.List(nil, "A")
		if len(got) != 2 || got[0] != "first" || got[1] != "second" {
			t.Errorf("CONTROL: List(A) returned %v, want [first second] — with one name both "+
				"nestings emit in list order, so this failing means the cause is not the one "+
				"this test names", got)
		}
	})
}

// TestMeleeProximityLeavesTheGapBetween15And16Untouched pins ModStore.lua:580-586's THREE arms.
func TestMeleeProximityLeavesTheGapBetween15And16Untouched(t *testing.T) {
	newDB := func() *ModDB {
		db := NewModDB()
		db.AddMod(mod.NewFloat("Damage", mod.TypeMore, 30).
			Tag(mod.MeleeProximity([]int{1, 0})))
		return db
	}

	if got := newDB().More(&ListCfg{SkillDist: 15.5, SkillDistSet: true}, "Damage"); got != 1.3 {
		t.Errorf("More at skillDist 15.5 = %v, want 1.3 — ModStore.lua:580-586 has no arm for "+
			"15 < d < 16, so the value crosses the tag UNCHANGED; a `default:` catching the gap "+
			"scales it by 0 and deletes the modifier", got)
	}

	if got := newDB().More(&ListCfg{SkillDist: 15, SkillDistSet: true}, "Damage"); got != 1.3 {
		t.Errorf("More at skillDist 15 = %v, want 1.3 (full potency arm, :580)", got)
	}
	if got := newDB().More(&ListCfg{SkillDist: 40, SkillDistSet: true}, "Damage"); got != 1 {
		t.Errorf("More at skillDist 40 = %v, want 1 — the :585 arm zeroes the multiplier, so the "+
			"MORE collapses to the identity", got)
	}
}
