package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestStrengthOfBloodKeystoneParses pins ModParser.lua:2196-2199, the two rows behind the Kaom timeless-jewel keystone "Strength of Blood".
func TestStrengthOfBloodKeystoneParses(t *testing.T) {
	t.Run("additional PDR per leech rate", func(t *testing.T) {
		entry := ParseMod("2% additional Physical Damage Reduction for every 3% Life Recovery per second from Leech", false)
		if strings.Trim(entry.Extra, " ") != "" {
			t.Fatalf("unparsed remainder %q — the whole line is discarded", entry.Extra)
		}
		if len(entry.ModList) != 1 {
			t.Fatalf("got %d mods, want 1", len(entry.ModList))
		}
		m := entry.ModList[0]
		if m.Name() != "PhysicalDamageReduction" || m.Type() != mod.TypeBase || m.Value().Float() != 2 {
			t.Fatalf("got %s %s %v, want PhysicalDamageReduction BASE 2",
				m.Name(), m.Type(), m.Value().Float())
		}
		var perStat, leeching bool
		for _, tag := range m.Tags() {
			switch tg := tag.(type) {
			case *mod.PerStatTag:
				perStat = len(tg.StatList) == 1 && tg.StatList[0] == "MaxLifeLeechRatePercent" && tg.Divide == 3
			case *mod.ConditionTag:
				leeching = len(tg.VarList) == 1 && tg.VarList[0] == "Leeching"
			}
		}
		if !perStat {
			t.Errorf("missing PerStat{stat: MaxLifeLeechRatePercent, div: 3}; tags = %+v", m.Tags())
		}
		if !leeching {
			t.Errorf("missing Condition:Leeching; tags = %+v", m.Tags())
		}
	})

	t.Run("non-instant leech is not applied", func(t *testing.T) {
		for line, want := range map[string]string{
			"Life Recovery from Non-Instant Leech is not applied":   "UnaffectedByNonInstantLifeLeech",
			"Mana Recovery from Non-Instant Leech is not applied":   "UnaffectedByNonInstantManaLeech",
			"Energy Recovery from Non-Instant Leech is not applied": "UnaffectedByNonInstantEnergyLeech",
		} {
			entry := ParseMod(line, false)
			if strings.Trim(entry.Extra, " ") != "" {
				t.Errorf("%q: unparsed remainder %q", line, entry.Extra)
				continue
			}
			if len(entry.ModList) != 1 || entry.ModList[0].Name() != want ||
				entry.ModList[0].Type() != mod.TypeFlag {
				t.Errorf("%q: got %v, want one %s FLAG", line, entry.ModList, want)
			}
		}
	})
}
