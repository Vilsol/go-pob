package calculator

import (
	"strings"
	"testing"
)

func combinedFireDamage(t *testing.T, item *Item) bool {
	t.Helper()
	for _, ml := range item.ExplicitModLines {
		if strings.Contains(ml.Line, "\n") && len(ml.ModList) > 0 {
			return true
		}
	}
	return false
}

// TestLineCombineRespectsModGroup covers Item.lua:1164-1168.
func TestLineCombineRespectsModGroup(t *testing.T) {
	requireGameData(t, "3.29")

	head := "Rarity: RARE\nProbe Ring\nTwo-Stone Ring\n" +
		"Unique ID: 0000\nItem Level: 84\nQuality: 0\nLevelReq: 60\nImplicits: 0\n"

	same := ParseItem(head + "{modGroup:1}Adds 10 to 20\n{modGroup:1}Fire Damage\n")
	if !combinedFireDamage(t, same) {
		t.Fatalf("control: two lines sharing a modGroup were not combined; the fixture no " +
			"longer reaches the combine at item.go:1573")
	}

	diff := ParseItem(head + "{modGroup:1}Adds 10 to 20\n{modGroup:2}Fire Damage\n")
	if combinedFireDamage(t, diff) {
		t.Errorf("two lines with DIFFERENT modGroups were combined; Item.lua:1167-1168 " +
			"requires modLine.modGroup == nextModGroup")
	}
}
