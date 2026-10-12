package calculator

import (
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestUnboundRowsCarryCondition pins ModParser.lua:3031-3036 (Avatar of the Wilds, TreeData/3_29/tree.lua).
func TestUnboundRowsCarryCondition(t *testing.T) {
	for line, want := range map[string]int{
		"Your Hits always inflict Freeze, Shock and Ignite while Unbound": 3,
		"100% more Elemental Damage while Unbound":                        1,
	} {
		e := ParseMod(strings.ToLower(line), false)
		if e == nil || e.Extra != "" || len(e.ModList) != want {
			t.Errorf("%q: got %+v, want %d mods parsed whole", line, e, want)
			continue
		}
		for _, m := range e.ModList {
			ok := false
			for _, tag := range m.Tags() {
				if c, isCond := tag.(*mod.ConditionTag); isCond && slices.Contains(c.VarList, "Unbound") {
					ok = true
				}
			}
			if !ok {
				t.Errorf("%q: %s lacks Condition:Unbound", line, m.Name())
			}
		}
	}
}
