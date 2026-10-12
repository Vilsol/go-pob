package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestPreFlagListPrefixesParseWholeLines covers the rows added to preFlagList from a name-set diff against the pin.
func TestPreFlagListPrefixesParseWholeLines(t *testing.T) {
	for _, line := range []string{
		"Each Rage also grants +2% to Fire Damage Over Time Multiplier",
		"Every Rage also grants 1% of Physical Damage as Extra Fire Damage",
		"Auras grant 10% increased Damage",
		"Totems have 20% increased Movement Speed",
		"Skills used by Totems have 10% increased Cast Speed",
		"Vaal Skills require 20% reduced Souls Per Use",
		"Non-Aura Vaal Skills require 20% reduced Souls Per Use",
		"Non-Minion Skills have 10% increased Area of Effect",
		"Banner Skills have 20% increased Duration",
		"Melee Weapon Hits deal 10% increased Damage",
		"For each nearby corpse, 1% increased Attack Speed",
		"Consecrated Ground you create grants 10% increased Damage",
	} {
		t.Run(line, func(t *testing.T) {
			e := ParseMod(line, false)
			if e == nil || len(e.ModList) == 0 {
				t.Fatalf("ParseMod(%q) produced no mods; the prefix is unhandled and the whole line is discarded", line)
			}
			if e.Extra != "" {
				t.Errorf("ParseMod(%q) left Extra %q -- an unparsed remainder discards the whole line", line, e.Extra)
			}
		})
	}
}

// TestPreFlagListPrefixesCarryTheirTag is the second half.
func TestPreFlagListPrefixesCarryTheirTag(t *testing.T) {
	const line = "Each Rage also grants +2% to Fire Damage Over Time Multiplier"

	e := ParseMod(line, false)
	if e == nil || len(e.ModList) != 1 {
		t.Fatalf("ParseMod(%q) produced %v, want exactly one mod", line, e)
	}

	tags := e.ModList[0].Tags()
	if len(tags) == 0 {
		t.Fatalf("ParseMod(%q) produced a mod with no tags; the prefix must contribute a "+
			"Multiplier, and without it the mod applies at full value unconditionally", line)
	}

	found := false
	for _, tag := range tags {
		m, ok := tag.(*mod.MultiplierTag)
		if !ok {
			continue
		}
		for _, v := range m.VariableList {
			if v == "RageEffect" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("ParseMod(%q) tags = %v, want one carrying the RageEffect multiplier", line, tags)
	}
}

// TestPreFlagListPrefixesAreNotNarrowerThanThePin covers the second half of the same diff: rows that were PRESENT and matched less than the pin's.
func TestPreFlagListPrefixesAreNotNarrowerThanThePin(t *testing.T) {
	for _, line := range []string{
		"Skills used by Traps gain 10% increased Area of Effect",
		"Skills used by Mines gain 10% increased Area of Effect",
		"Skills which Throw Traps have 10% increased Cooldown Recovery Rate",
		"Raised Spectres have 20% increased Movement Speed",
	} {
		t.Run(line, func(t *testing.T) {
			e := ParseMod(line, false)
			if e == nil || len(e.ModList) == 0 {
				t.Fatalf("ParseMod(%q) produced no mods", line)
			}
			if e.Extra != "" {
				t.Errorf("ParseMod(%q) left Extra %q -- the prefix matched less than the pin's "+
					"and the unparsed remainder discards the whole line", line, e.Extra)
			}
		})
	}
}
