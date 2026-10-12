package calculator

import (
	"strings"
	"testing"
)

// TestRadiusAlsoGrantJewelFuncsParse pins the nine `Passive Skills in Radius also grant …` rows.
func TestRadiusAlsoGrantJewelFuncsParse(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("Passive Skills in Radius also grant: 10% increased Unarmed Attack Speed with Melee Skills", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: an already-ported radius row does not parse; the jewel-func path is " +
			"broken generally and this test cannot attribute a failure")
	}

	for _, line := range []string{
		"Passive Skills in Radius also grant 10% increased Global Critical Strike Chance",
		"Passive Skills in Radius also grant +20 to Maximum Life",
		"Passive Skills in Radius also grant +15 to Maximum Mana",
		"Passive Skills in Radius also grant 3% increased Energy Shield",
		"Passive Skills in Radius also grant 4% increased Armour",
		"Passive Skills in Radius also grant 5% increased Evasion Rating",
		"Passive Skills in Radius also grant +4 to all Attributes",
		"Passive Skills in Radius also grant +7% to Chaos Resistance",
		"Passive Skills in Radius also grant 6% increased Physical Damage",
		"Passive Skills in Radius also grant 6% increased Fire Damage",
		"Passive Skills in Radius also grant 6% increased Cold Damage",
		"Passive Skills in Radius also grant 6% increased Chaos Damage",
		"Passive Skills in Radius also grant 6% increased Lightning Damage",
	} {
		e := ParseMod(line, false)
		if e == nil {
			t.Errorf("no entry: %s", line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q (line is DROPPED WHOLE): %s", extra, line)
			continue
		}
		if len(e.ModList) == 0 {
			t.Errorf("parsed but produced 0 mods: %s", line)
			continue
		}
		var got []string
		for _, m := range e.ModList {
			got = append(got, m.Name())
		}
		if !contains(got, "JewelFunc") {
			t.Errorf("produced %v, want a JewelFunc mod: %s", got, line)
		}
	}
}
