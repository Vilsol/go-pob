package calculator

import (
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestCompoundDamageTakenAsParses pins ModParser.lua:5304-5307.
func TestCompoundDamageTakenAsParses(t *testing.T) {
	for _, tc := range []struct {
		line string
		want []string
	}{
		{"30% of Fire and Lightning Damage from Hits taken as Cold Damage during Effect",
			[]string{"FireDamageFromHitsTakenAsCold", "LightningDamageFromHitsTakenAsCold"}},
		{"20% of Cold and Lightning Damage taken as Fire Damage",
			[]string{"ColdDamageTakenAsFire", "LightningDamageTakenAsFire"}},
		{"15% of Fire and Cold Damage taken as Lightning Damage",
			[]string{"ColdDamageTakenAsLightning", "FireDamageTakenAsLightning"}},
		{"25% of Physical Damage from Hits taken as Cold Damage",
			[]string{"PhysicalDamageFromHitsTakenAsCold"}},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if strings.Trim(e.Extra, " ") != "" {
				t.Fatalf("unparsed remainder %q -- the WHOLE line is discarded", e.Extra)
			}
			if len(e.ModList) != len(tc.want) {
				t.Fatalf("got %d mods, want %d -- a compound that emits ONE mod has silently "+
					"dropped a damage type", len(e.ModList), len(tc.want))
			}
			got := make([]string, 0, len(e.ModList))
			for _, m := range e.ModList {
				if m.Type() != mod.TypeBase {
					t.Fatalf("%s is %s, want BASE", m.Name(), m.Type())
				}
				got = append(got, m.Name())
			}
			sort.Strings(got)
			want := append([]string(nil), tc.want...)
			sort.Strings(want)
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("got %v, want %v", got, want)
				}
			}
		})
	}

	e := ParseMod("30% of Fire and Lightning Damage from Hits taken as Cold Damage during Effect", false)
	for _, m := range e.ModList {
		if len(m.Tags()) != 1 {
			t.Fatalf("%s carries %d tags, want 1 (Condition:UsingFlask)", m.Name(), len(m.Tags()))
		}
		c, ok := m.Tags()[0].(*mod.ConditionTag)
		if !ok || len(c.VarList) != 1 || c.VarList[0] != "UsingFlask" {
			t.Fatalf("%s tag is %#v, want Condition:UsingFlask", m.Name(), m.Tags()[0])
		}
	}
}
