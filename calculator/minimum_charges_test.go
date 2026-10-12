package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestWhileAtMinimumChargesParses pins ModParser.lua:1692-1694.
func TestWhileAtMinimumChargesParses(t *testing.T) {
	for _, tc := range []struct {
		line  string
		stat  string
		upper bool
	}{
		{"Determination has 50% increased Aura Effect while at Minimum Endurance Charges", "EnduranceChargesMin", true},
		{"Determination has 50% increased Aura Effect while at Minimum Power Charges", "PowerChargesMin", true},
		{"Determination has 50% increased Aura Effect while at Minimum Frenzy Charges", "FrenzyChargesMin", true},
		{"Anger has 50% increased Aura Effect while at maximum Endurance Charges", "EnduranceChargesMax", false},
		{"Anger has 50% increased Aura Effect while at maximum Power Charges", "PowerChargesMax", false},
	} {
		t.Run(tc.stat, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if strings.Trim(e.Extra, " ") != "" {
				t.Fatalf("unparsed remainder %q -- the WHOLE line is discarded", e.Extra)
			}
			if len(e.ModList) != 1 {
				t.Fatalf("got %d mods, want 1", len(e.ModList))
			}
			m := e.ModList[0]
			if m.Name() != "AuraEffect" {
				t.Fatalf("got %s, want AuraEffect", m.Name())
			}
			var found *mod.StatThresholdTag
			for _, tag := range m.Tags() {
				if st, ok := tag.(*mod.StatThresholdTag); ok {
					found = st
				}
			}
			if found == nil {
				t.Fatalf("no StatThreshold tag on %d tags", len(m.Tags()))
			}
			if found.TagThresholdStat != tc.stat {
				t.Fatalf("threshold stat %q, want %q", found.TagThresholdStat, tc.stat)
			}
			if found.TagUpper != tc.upper {
				t.Fatalf("upper=%v, want %v", found.TagUpper, tc.upper)
			}
		})
	}
}
