package calculator

import (
	"strings"
	"testing"
)

// TestEquippedAllHaveDefenceConditionsParse pins the two `if equipped helmet, body armour, gloves, and boots all have <defence>` conditions.
func TestEquippedAllHaveDefenceConditionsParse(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+1% to all maximum Elemental Resistances", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the unconditional control line does not parse; this test cannot " +
			"attribute a failure to the condition")
	}

	for _, line := range []string{
		"+1% to all maximum Elemental Resistances if Equipped Helmet, Body Armour, Gloves, and Boots all have Armour",
		"+15% chance to Suppress Spell Damage if Equipped Helmet, Body Armour, Gloves, and Boots all have Evasion Rating",
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
		if got := len(e.ModList[0].Tags()); got < 4 {
			t.Errorf("got %d tags, want >=4 (one StatThreshold per equipped slot): %s", got, line)
		}
	}
}
