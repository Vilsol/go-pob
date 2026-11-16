package mcp

import (
	"testing"
)

// A write that moves many stats keeps every watched stat and the DPS, EHP and Life rows; only other rows are trimmed.
func TestDeltaKeepsWatchDPSEHPAndLifeWhenTrimmed(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	watch := []string{"FireResistTotal", "ColdResistTotal", "LightningResistTotal", "ChaosResistTotal", "Mana", "Armour",
		"Evasion", "EnergyShield", "Str", "Dex", "Int", "BlockChance"}
	res := assertSize(t, s, "edit_build", map[string]any{"handle": h, "dry_run": true, "watch": watch, "ops": ops(
		map[string]any{"op": "set_config", "var": "customMods", "value": "+30% to all Elemental Resistances\n+20% to Chaos Resistance\n" +
			"+500 to maximum Life\n+500 to maximum Mana\n+500 to Armour\n+500 to Evasion Rating\n+500 to maximum Energy Shield\n" +
			"50% increased Damage\n+50 to all Attributes\n+10% Chance to Block Attack Damage\n20% increased Attack Speed\n20% increased Cast Speed"})}, capWrite)
	seen := map[string]bool{}
	for _, r := range asList(res["delta"]) {
		seen[r.(map[string]any)["stat"].(string)] = true
	}
	un, _ := res["unchanged"].(map[string]any)
	for _, k := range append([]string{mainDPS, "TotalEHP", "Life"}, watch...) {
		if _, same := un[k]; !seen[k] && !same {
			t.Errorf("%s is neither in delta nor unchanged (more %v): %v", k, res["more"], res["delta"])
		}
	}
}
