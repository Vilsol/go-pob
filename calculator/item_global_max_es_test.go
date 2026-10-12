package calculator

import (
	"regexp"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func hasGlobalTag(m mod.Mod) bool {
	for _, tag := range m.Tags() {
		if _, ok := tag.(*mod.GlobalTag); ok {
			return true
		}
	}
	return false
}

// TestIncreasedMaximumEnergyShieldIsGlobal covers ModParser.lua:2400, whose own comment is.
// "Override as increased maximum is always global".
func TestIncreasedMaximumEnergyShieldIsGlobal(t *testing.T) {
	requireGameData(t, "3.29")

	local := ParseMod("126% increased Energy Shield", false)
	if len(local.ModList) != 1 || local.Extra != "" {
		t.Fatalf("control: %d mods, extra %q", len(local.ModList), local.Extra)
	}
	if hasGlobalTag(local.ModList[0]) {
		t.Errorf("control: the LOCAL wording gained a Global tag; it must still be drained into the item")
	}

	got := ParseMod("15% increased maximum Energy Shield", false)
	if len(got.ModList) != 1 || got.Extra != "" {
		t.Fatalf("%d mods, extra %q", len(got.ModList), got.Extra)
	}
	m := got.ModList[0]
	if m.Name() != "EnergyShield" || m.Type() != mod.TypeIncrease || m.Value().Float() != 15 {
		t.Errorf("parsed as %s %s %v, want EnergyShield INC 15", m.Name(), m.Type(), m.Value().Float())
	}
	if !hasGlobalTag(m) {
		t.Errorf("no Global tag: calcLocal will drain this into the item's own recompute, " +
			"raising that item and lowering the total")
	}

	const raw = "Rarity: RARE\nPlague Skin\nTwilight Regalia\n" +
		"Energy Shield: 1225\nEnergyShieldBasePercentile: 0.9988\n" +
		"Unique ID: 0000\nItem Level: 91\nQuality: 30\nSockets: B-R-W-G-B-B\nLevelReq: 84\nImplicits: 1\n" +
		"15% increased maximum Energy Shield\n" +
		"+115 to maximum Energy Shield\n126% increased Energy Shield\n"
	item := ParseItem(regexp.MustCompile(`\s+$`).ReplaceAllString(strings.TrimSpace(raw), ""))
	if item.ArmourData == nil {
		t.Fatalf("REFUSING: fixture produced no armourData")
	}
	if got := item.ArmourData["EnergyShield"]; got != 1225 {
		t.Errorf("item energy shield = %v, want 1225 (its own stated value; PoB's breakdown for "+
			"1023 gives Body Armour base=1225)", got)
	}
}
