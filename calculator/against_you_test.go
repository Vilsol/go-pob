package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestAgainstYouRoutesToTheEnemy pins ModParser.lua:1282.
func TestAgainstYouRoutesToTheEnemy(t *testing.T) {
	for _, line := range []string{
		"Adds 4 to 9 Physical Damage to Attacks against you",
		"Adds 10 to 20 Cold Damage to Hits against you",
		"Adds 3 to 7 Fire Damage to Attacks against you",
	} {
		t.Run(line, func(t *testing.T) {
			e := ParseMod(line, false)
			if strings.Trim(e.Extra, " ") != "" {
				t.Fatalf("unparsed remainder %q -- the WHOLE line is discarded", e.Extra)
			}
			if len(e.ModList) != 2 {
				t.Fatalf("got %d mods, want 2 (a min and a max)", len(e.ModList))
			}
			for _, m := range e.ModList {
				if m.Name() != "EnemyModifier" || m.Type() != mod.TypeList {
					t.Fatalf("got %s %s, want EnemyModifier LIST — the mod is being applied to US",
						m.Name(), m.Type())
				}
			}
		})
	}

	for _, line := range []string{
		"Attacks against you always Hit",
		"Hits against you are always Critical Strikes",
		"Hits against you Overwhelm 10% of Physical Damage Reduction",
		"Hits against you have 30% chance to be Critical Strikes",
	} {
		t.Run("control/"+line, func(t *testing.T) {
			e := ParseMod(line, false)
			if strings.Trim(e.Extra, " ") == line {
				t.Fatalf("the whole line went unparsed -- the unanchored row shadowed its handler")
			}
		})
	}

	e := ParseMod("Nearby Enemies have 25% increased Damage", false)
	if strings.Trim(e.Extra, " ") != "" || len(e.ModList) != 1 {
		t.Fatalf("control: `^nearby enemies have ` broke: extra=%q mods=%d", e.Extra, len(e.ModList))
	}
}
