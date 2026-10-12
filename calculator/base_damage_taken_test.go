package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestBaseDamageTakenMods pins the baseline DamageTaken mods PoB seeds in CalcSetup.lua:64-67.
func TestBaseDamageTakenMods(t *testing.T) {
	db := moddb.NewModDB()
	initModDB(&Environment{}, db)

	var attack, spell, party int
	for _, m := range db.Mods["DamageTaken"] {
		if m.Type() != mod.TypeIncrease || m.GetSource() != "Base" {
			continue
		}
		switch m.Flags() {
		case mod.MFlagAttack:
			attack++
		case mod.MFlagSpell:
			spell++
		}
		if len(m.Tags()) == 2 {
			party++
		}
	}

	testza.AssertEqual(t, 2, attack, "expected two Base DamageTaken INC mods flagged Attack — Intimidated and Party:Intimidated (CalcSetup.lua:64-65)")
	testza.AssertEqual(t, 2, spell, "expected two Base DamageTaken INC mods flagged Spell — Unnerved and Party:Unnerved (CalcSetup.lua:66-67)")
	testza.AssertEqual(t, 2, party, "expected the two PARTY forms to carry both the negated self-condition and the Party condition; one tag means the negation was dropped, which would double the effect on an intimidated bearer")
}
