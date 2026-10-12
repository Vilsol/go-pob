package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestRadiantFaithGrantsBothLines pins ModParser.lua:2863-2865 — Guardian's Radiant Faith, tree node 42264.
func TestRadiantFaithGrantsBothLines(t *testing.T) {
	for _, tc := range []struct {
		line    string
		carrier string
		granted string
		want    float64
	}{
		{"Grants Armour equal to 25% of your Reserved Mana to you and nearby Allies",
			"GrantReservedManaAsAura", "Armour", 0.25},
		{"Grants maximum Energy Shield equal to 10% of your Reserved Mana to you and nearby Allies",
			"GrantReservedManaAsAura", "EnergyShield", 0.10},
		{"Grants Armour equal to 20% of your Reserved Life to you and nearby Allies",
			"GrantReservedLifeAsAura", "Armour", 0.20},
	} {
		t.Run(tc.granted+"/"+tc.carrier, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if strings.Trim(e.Extra, " ") != "" {
				t.Fatalf("unparsed remainder %q -- the WHOLE line is discarded", e.Extra)
			}
			if len(e.ModList) != 1 {
				t.Fatalf("got %d mods, want 1", len(e.ModList))
			}
			m := e.ModList[0]
			if m.Name() != tc.carrier || m.Type() != mod.TypeList {
				t.Fatalf("got %s %s, want %s LIST", m.Name(), m.Type(), tc.carrier)
			}
			granted, ok := m.Value().List().(mod.GrantReservedPoolAsAura)
			if !ok {
				t.Fatalf("carrier value is %T, want mod.GrantReservedPoolAsAura", m.Value().List())
			}
			if granted.Mod.Name() != tc.granted {
				t.Fatalf("grants %s, want %s", granted.Mod.Name(), tc.granted)
			}
			if got := granted.Mod.Value().Float(); got != tc.want {
				t.Fatalf("grants %v per point, want %v", got, tc.want)
			}
		})
	}
}
