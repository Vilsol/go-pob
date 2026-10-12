package raw

import "testing"

// TestNoManaCostIsMinus100 pins the one transcription error scripts/dataraw_mirror.py found in the global stat map.
func TestNoManaCostIsMinus100(t *testing.T) {
	entry, ok := SkillStatMap["no_mana_cost"]
	if !ok {
		t.Fatal("SkillStatMap has no no_mana_cost entry")
	}
	if entry.Value == nil {
		t.Fatal("no_mana_cost has no value; PoB sets value = -100")
	}
	if *entry.Value != -100 {
		t.Fatalf("no_mana_cost value = %v, want -100 (SkillStatMap.lua:335)", *entry.Value)
	}
}
