package calculator

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestResistanceToDefenceLinesParseToFlags: ModParser.lua:4805-4812 parse these lines to flags.
func TestResistanceToDefenceLinesParseToFlags(t *testing.T) {
	requireGameData(t, "3.29")
	for line, flag := range map[string]string{
		"Maximum Energy Shield is increased by Chance to Block Spell Damage": "EnergyShieldIncreasedByChanceToBlockSpellDamage",
		"Maximum Energy Shield is increased by Chaos Resistance":             "EnergyShieldIncreasedByChaosResistance",
		"Armour is increased by Uncapped Fire Resistance":                    "ArmourIncreasedByUncappedFireRes",
		"Armour is increased by Overcapped Fire Resistance":                  "ArmourIncreasedByOvercappedFireRes",
		"Evasion Rating is increased by Uncapped Cold Resistance":            "EvasionRatingIncreasedByUncappedColdRes",
		"Evasion Rating is increased by Overcapped Cold Resistance":          "EvasionRatingIncreasedByOvercappedColdRes",
	} {
		entry := ParseMod(line, false)
		if entry == nil || entry.Extra != "" || len(entry.ModList) != 1 {
			t.Errorf("%s: parse %+v", line, entry)
			continue
		}
		if m := entry.ModList[0]; m.Name() != flag || m.Type() != mod.TypeFlag {
			t.Errorf("%s: %s %s, PoB FLAG %s", line, m.Name(), m.Type(), flag)
		}
	}
}

func TestOvercappedFireResistanceArmourRow(t *testing.T) {
	assertTablesMatch(t, "13276-ladder-cuba_5717.cubamrc.xml", func(c modSectionCase) bool {
		return slices.Equal(c.ModName, []string{"Armour", "ArmourAndEvasion", "Defences"})
	})
}
