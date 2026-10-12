package calculator

import "testing"

// TestBrineRowsParse pins ModParser.lua:4277 (mace.lua:972) and :3083 (tree.lua:13476).
func TestBrineRowsParse(t *testing.T) {
	requireGameData(t, "3.29")

	e := ParseMod("50% chance to gain a Brine Charge instead of an Endurance Charge", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 2 || e.ModList[0].Name() != "CanGainBrineCharges" ||
		e.ModList[1].Name() != "BrineChargeGainChance" || e.ModList[1].Value().Float() != 50 {
		t.Errorf("brine charge row: got %+v", e)
	}
	e = ParseMod("Drop Brine Ground while moving, lasting 4 seconds", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "CanCreateBrineGround" {
		t.Errorf("brine ground row: got %+v", e)
	}
	e = ParseMod("10% increased Damage per Brine Charge", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "Damage" || len(e.ModList[0].Tags()) != 1 {
		t.Errorf("NEIGHBOUR CONTROL: got %+v", e)
	}
}
