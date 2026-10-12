package calculator

import (
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestYouInflictOnShockedChilledTags pins ModParser.lua:1979/:1986.
func TestYouInflictOnShockedChilledTags(t *testing.T) {
	for _, tc := range []struct{ line, cond string }{
		{"25% increased Effect of Cold Ailments you inflict on Shocked Enemies", "Shocked"},
		{"25% increased Effect of Lightning Ailments you inflict on Chilled Enemies", "Chilled"},
		{"40% increased Damage with Ignite inflicted on Chilled Enemies", "Chilled"},
	} {
		e := ParseMod(strings.ToLower(tc.line), false)
		if e == nil || e.Extra != "" || len(e.ModList) == 0 {
			t.Errorf("%q: got %+v, want parsed whole", tc.line, e)
			continue
		}
		for _, m := range e.ModList {
			found := false
			for _, tag := range m.Tags() {
				if ac, ok := tag.(*mod.ActorConditionTag); ok && slices.Contains(ac.VariableList, tc.cond) {
					found = true
				}
			}
			if !found {
				t.Errorf("%q: %s has no enemy ActorCondition %s", tc.line, m.Name(), tc.cond)
			}
		}
	}
}
