package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestFortificationAboveTwentyParsesWhole pins ModParser.lua:1535.
func TestFortificationAboveTwentyParsesWhole(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
		typ  mod.Type
	}{
		{"3% more Armour and Evasion Rating per Fortification above 20", "ArmourAndEvasion", mod.TypeMore},
		{"3% increased Attack Speed per Fortification above 20", "Speed", mod.TypeIncrease},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if strings.Trim(e.Extra, " ") != "" {
				t.Fatalf("unparsed remainder %q -- the WHOLE line is discarded", e.Extra)
			}
			if len(e.ModList) != 1 {
				t.Fatalf("got %d mods, want 1", len(e.ModList))
			}
			m := e.ModList[0]
			if m.Name() != tc.name || m.Type() != tc.typ {
				t.Fatalf("got %s %s, want %s %s", m.Name(), m.Type(), tc.name, tc.typ)
			}
			if len(m.Tags()) != 1 {
				t.Fatalf("got %d tags, want 1", len(m.Tags()))
			}
			ps, ok := m.Tags()[0].(*mod.PerStatTag)
			if !ok {
				t.Fatalf("tag is %T, want *mod.PerStatTag", m.Tags()[0])
			}
			if len(ps.StatList) != 1 || ps.StatList[0] != "FortificationStacksOver20" {
				t.Fatalf("tag reads %v, want [FortificationStacksOver20]", ps.StatList)
			}
		})
	}

	e := ParseMod("3% increased Attack Speed per Fortification", false)
	if strings.Trim(e.Extra, " ") != "" || len(e.ModList) != 1 {
		t.Fatalf("control: the plain `per fortification` form broke: extra=%q mods=%d", e.Extra, len(e.ModList))
	}
	ps, ok := e.ModList[0].Tags()[0].(*mod.PerStatTag)
	if !ok || len(ps.StatList) != 1 || ps.StatList[0] != "FortificationStacks" {
		t.Fatalf("control: the plain form no longer reads FortificationStacks (%#v)", e.ModList[0].Tags()[0])
	}
}
