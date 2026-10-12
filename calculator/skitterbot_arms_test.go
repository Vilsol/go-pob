package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSkitterbotUniqueRowsParse: The three ModParser.lua:3827-3829 rows had no counterpart here.
func TestSkitterbotUniqueRowsParse(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
		typ  mod.Type
	}{
		{"Summon Skitterbots also Summons a Scorching Skitterbot", "ScorchingSkitterbot", mod.TypeFlag},
		{"Summoned Skitterbots' Auras affect you as well as Enemies", "SkitterbotAffectPlayer", mod.TypeFlag},
		{"25% increased Effect of Non-Damaging Ailments inflicted by Summoned Skitterbots", "SkitterbotAilmentEffect", mod.TypeIncrease},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || len(e.ModList) == 0 {
			t.Errorf("ParseMod(%q) returned NOTHING — an unparsed remainder discards the WHOLE line", tc.line)
			continue
		}
		found := false
		for _, m := range e.ModList {
			if m.Name() == tc.name {
				found = true
				if m.Type() != tc.typ {
					t.Errorf("%s: type %v, want %v", tc.name, m.Type(), tc.typ)
				}
			}
		}
		if !found {
			t.Errorf("ParseMod(%q) produced %d mod(s) but none named %s", tc.line, len(e.ModList), tc.name)
		}
	}
}
