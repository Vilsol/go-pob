package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestEvasionAppliesToSpellDamageRowParses: The Unblinking Eye conversion.
func TestEvasionAppliesToSpellDamageRowParses(t *testing.T) {
	mods, extra := parseMod("Increases and Reductions to your Evasion Rating also apply to your Spell Damage", 1)
	testza.AssertEqual(t, "", extra, "an unparsed remainder discards the WHOLE line")
	testza.AssertEqual(t, 1, len(mods))
	if len(mods) != 1 {
		return
	}
	testza.AssertEqual(t, "EvasionAppliesToSpellDamage", mods[0].Name(),
		"the consumer at calc_offence.go reads this exact literal, which is CalcOffence.lua:611's")
	testza.AssertEqual(t, mod.TypeFlag, mods[0].Type())
	testza.AssertTrue(t, mods[0].Value().Flag())
	testza.AssertEqual(t, 0, len(mods[0].Tags()),
		"upstream's row is a bare flag() with no tags — a stray tag would gate a conversion "+
			"upstream applies unconditionally")

	nMods, nExtra := parseMod("Increases and Reductions to Spell Damage also apply to Attacks", 1)
	testza.AssertEqual(t, "", nExtra)
	testza.AssertEqual(t, 2, len(nMods))
	if len(nMods) == 2 {
		testza.AssertEqual(t, "SpellDamageAppliesToAttacks", nMods[0].Name())
		testza.AssertEqual(t, "ImprovedSpellDamageAppliesToAttacks", nMods[1].Name())
	}
}
