package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestHitsChanceAreaDamageParse covers ModParser.lua:2984 — Saboteur's Bomb Specialist.
func TestHitsChanceAreaDamageParse(t *testing.T) {
	mods, extra := parseMod("Hits have 30% chance to deal 50% more Area Damage", 1)
	testza.AssertEqual(t, "", extra, "the line must parse completely")
	if len(mods) != 1 {
		t.Fatalf("produced %d mods, want 1", len(mods))
	}
	testza.AssertEqual(t, "Damage", mods[0].Name())
	testza.AssertEqual(t, mod.TypeMore, mods[0].Type())
	testza.AssertEqual(t, 15.0, mods[0].Value().ValueFloat)
	testza.AssertTrue(t, mods[0].Flags()&mod.MFlagArea != 0, "must carry the Area flag")
	testza.AssertTrue(t, mods[0].Flags()&mod.MFlagHit != 0, "must carry the Hit flag")
}
