package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestUnlinkedSocketMultiplier covers Item.lua:2222-2228 and its consumer at ModParser.lua:1440.
func TestUnlinkedSocketMultiplier(t *testing.T) {
	requireGameData(t, "3.29")

	unlinkedFor := func(sockets string) (float64, bool) {
		t.Helper()
		raw := "Rarity: RARE\nProbe Blade\nSiege Axe\n" +
			"Unique ID: 0000\nItem Level: 84\nQuality: 0\nSockets: " + sockets + "\nLevelReq: 70\nImplicits: 0\n"
		item := ParseItem(raw)
		if item.Base == nil {
			t.Fatalf("REFUSING: fixture resolved no base")
		}
		for _, m := range item.Mods {
			if m.Name() == "Multiplier:UnlinkedSocketInWeapon 1" {
				return m.Value().Float(), true
			}
		}
		return 0, false
	}

	if got, ok := unlinkedFor("R-G B W"); !ok {
		t.Errorf("no Multiplier:UnlinkedSocketInWeapon 1 was emitted at all")
	} else if got != 2 {
		t.Errorf("unlinked sockets = %v, want 2 (two groups of exactly one)", got)
	}

	if got, ok := unlinkedFor("R-G-B"); !ok {
		t.Errorf("a fully linked item emitted no multiplier; PoB emits BASE 0")
	} else if got != 0 {
		t.Errorf("fully linked unlinked-count = %v, want 0", got)
	}

	entry := ParseMod("10% increased Damage for each unlinked Socket in Equipped Two Handed Weapon", false)
	if entry.Extra != "" || len(entry.ModList) != 1 {
		t.Fatalf("the consumer line gave %d mods, extra %q", len(entry.ModList), entry.Extra)
	}
	m := entry.ModList[0]
	if m.Name() != "Damage" || m.Type() != mod.TypeIncrease || m.Value().Float() != 10 {
		t.Errorf("consumer parsed as %s %s %v, want Damage INC 10", m.Name(), m.Type(), m.Value().Float())
	}
	var mult *mod.MultiplierTag
	var cond bool
	for _, tag := range m.Tags() {
		switch tg := tag.(type) {
		case *mod.MultiplierTag:
			mult = tg
		case *mod.ConditionTag:
			cond = true
		}
	}
	if mult == nil || len(mult.VariableList) != 1 || mult.VariableList[0] != "UnlinkedSocketInWeapon 1" {
		t.Errorf("consumer carries multiplier %v, want UnlinkedSocketInWeapon 1 -- the two ends must "+
			"agree on the NAME or the pair is inert", mult)
	}
	if !cond {
		t.Errorf("consumer carries no Condition tag; PoB gates it on UsingTwoHandedWeapon")
	}
}
