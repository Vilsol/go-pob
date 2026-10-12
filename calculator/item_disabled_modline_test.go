package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func hasPhysicalDamageInc(item *Item) bool {
	for _, m := range item.Mods {
		if m.Name() == "PhysicalDamage" && m.Type() == mod.TypeIncrease {
			return true
		}
	}
	return false
}

// TestDisabledModLineIsNotApplied covers Item.lua:109-125's `disabled` lineFlag and the early return in BuildModList's processModLine at Item.lua:2511-2514.
func TestDisabledModLineIsNotApplied(t *testing.T) {
	requireGameData(t, "3.29")

	head := "Rarity: RARE\nProbe Ring\nTwo-Stone Ring\n" +
		"Unique ID: 0000\nItem Level: 84\nQuality: 0\nLevelReq: 60\nImplicits: 0\n"

	enabled := ParseItem(head + "21% increased Global Physical Damage\n")
	enabled.BuildModList()
	if !hasPhysicalDamageInc(enabled) {
		t.Fatalf("control: the untagged line produced no PhysicalDamage INC mod, so this " +
			"fixture cannot show what {disabled} suppresses")
	}

	disabled := ParseItem(head + "{disabled}21% increased Global Physical Damage\n")
	disabled.BuildModList()
	if hasPhysicalDamageInc(disabled) {
		t.Errorf("a {disabled} mod line was applied; Item.lua:2512 returns early from " +
			"processModLine on modLine.disabled")
	}
}
