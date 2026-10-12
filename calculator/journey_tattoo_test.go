package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestJourneyTattooRowsAreWiredToTheirOwnMultiplier: The Journey Tattoo family (ModParser.lua:5800-5812) had no rows at all.
func TestJourneyTattooRowsAreWiredToTheirOwnMultiplier(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		line string
		stat string
		mult string
	}{
		{"+40 to maximum Life per allocated Journey Tattoo of the Body", "Life", "JourneyTattooBody"},
		{"+40 to maximum Energy Shield per allocated Journey Tattoo of the Soul", "EnergyShield", "JourneyTattooSoul"},
		{"+40 to maximum Mana per allocated Journey Tattoo of the Mind", "Mana", "JourneyTattooMind"},
	} {
		t.Run(tc.stat, func(t *testing.T) {
			t.Parallel()

			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 2 {
				t.Fatalf("ParseMod(%q) produced %v, want two mods", tc.line, e)
			}
			if e.Extra != "" {
				t.Fatalf("leftover Extra %q — the line would be discarded entirely", e.Extra)
			}

			one := moddb.NewModList()
			for _, m := range e.ModList {
				one.AddMod(m)
			}
			if got := one.Sum(mod.TypeBase, nil, tc.stat); got != 40 {
				t.Errorf("one allocated tattoo: %s = %v, want 40 — the Multiplier tag and the "+
					"Multiplier:%s the same row emits are not wired to each other", tc.stat, got, tc.mult)
			}

			three := moddb.NewModList()
			for i := 0; i < 3; i++ {
				for _, m := range ParseMod(tc.line, false).ModList {
					three.AddMod(m)
				}
			}
			if got := three.Sum(mod.TypeBase, nil, tc.stat); got != 360 {
				t.Errorf("three allocated tattoos: %s = %v, want 360 (3 nodes x 3 multiplier)",
					tc.stat, got)
			}

			bare := moddb.NewModList()
			for _, m := range e.ModList {
				if m.Name() == tc.stat {
					bare.AddMod(m)
				}
			}
			if got := bare.Sum(mod.TypeBase, nil, tc.stat); got != 0 {
				t.Errorf("with no Multiplier:%s the %s mod contributed %v, want 0 — the tag is inert",
					tc.mult, tc.stat, got)
			}
		})
	}
}
