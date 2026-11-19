package calculator

import "testing"

// Item.lua:299-308 search crucible lines with the other non-explicit lines and skip disabled lines.
func TestCfgModifierMatchesLineSet(t *testing.T) {
	requireGameData(t, "3.30")
	disabled := ParseItem("Rarity: RARE\nTest\nGold Ring\nImplicits: 0\n{disabled}+10% to Fire Resistance")
	if disabled.CfgModifierMatches("fire resistance", "ring 1") {
		t.Errorf("a disabled line matched")
	}
	crucible := ParseItem("Rarity: RARE\nTest\nGold Ring\nImplicits: 0\n{crucible}+10% to Fire Resistance")
	if len(crucible.CrucibleModLines) != 1 {
		t.Fatalf("REFUSING: crucible lines %d", len(crucible.CrucibleModLines))
	}
	if !crucible.CfgModifierMatches("fire resistance", "ring 1") {
		t.Errorf("a crucible line did not match")
	}
	if crucible.CfgModifierMatches("explicit fire resistance", "ring 1") {
		t.Errorf("a crucible line matched an explicit search")
	}
}

func TestActorItemModifierMatchesLineSet(t *testing.T) {
	requireGameData(t, "3.30")
	a := &Actor{ItemList: map[string]*ItemData{
		"Ring 1": itemToItemData(ParseItem("Rarity: RARE\nTest\nGold Ring\nImplicits: 0\n{disabled}+10% to Fire Resistance")),
		"Ring 2": itemToItemData(ParseItem("Rarity: RARE\nTest\nGold Ring\nImplicits: 0\n{crucible}+10% to Cold Resistance")),
	}}
	if a.ItemModifierMatches("Ring 1", "fire resistance") {
		t.Errorf("a disabled line matched")
	}
	if !a.ItemModifierMatches("Ring 2", "cold resistance") {
		t.Errorf("a crucible line did not match")
	}
}
