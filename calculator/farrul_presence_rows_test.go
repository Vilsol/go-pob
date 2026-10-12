package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestFarrulPresenceRowsParse pins ModParser.lua:5087-5101, the three bloodline notables of the Farrul ascendancy.
func TestFarrulPresenceRowsParse(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		line string
		want []struct {
			name string
			val  float64
		}
	}{
		{"Raised Beast Spectres have Farrul's Farric Presence", []struct {
			name string
			val  float64
		}{{"Accuracy", 80}, {"CritChance", 120}, {"ReduceCritExtraDamage", 100}}},
		{"Raised Beast Spectres have Farrul's Fertile Presence", []struct {
			name string
			val  float64
		}{{"Damage", 100}, {"LifeRegenPercent", 3}, {"StunImmune", 0}}},
		{"Raised Beast Spectres have Farrul's Wild Presence", []struct {
			name string
			val  float64
		}{{"Speed", 20}, {"MovementSpeed", 20}, {"MinimumActionSpeed", 100}}},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e.Extra != "" {
				t.Fatalf("leftover %q -- an unparsed remainder DISCARDS THE WHOLE LINE", e.Extra)
			}
			if len(e.ModList) != 3 {
				t.Fatalf("got %d mods, want 3: %+v", len(e.ModList), e.ModList)
			}
			for i, w := range tc.want {
				m := e.ModList[i]
				if m.Name() != "ExtraAura" {
					t.Fatalf("mod %d name = %q, want ExtraAura", i, m.Name())
				}
				ea, ok := m.Value().List().(mod.ExtraAura)
				if !ok {
					t.Fatalf("mod %d payload is %T, want mod.ExtraAura", i, m.Value().List())
				}
				if !ea.FromAllies {
					t.Fatalf("mod %d: FromAllies is false; upstream sets fromAllies on all nine", i)
				}
				if ea.OnlyAllies {
					t.Fatalf("mod %d: OnlyAllies is true -- that is a DIFFERENT field and it "+
						"suppresses the self arm entirely", i)
				}
				if ea.Mod.Name() != w.name {
					t.Fatalf("mod %d inner name = %q, want %q", i, ea.Mod.Name(), w.name)
				}
				if w.val != 0 && ea.Mod.Value().Float() != w.val {
					t.Fatalf("mod %d inner value = %v, want %v", i, ea.Mod.Value().Float(), w.val)
				}
				if len(m.Tags()) != 1 {
					t.Fatalf("mod %d: %d tags, want exactly 1 (Condition:HaveBeastSpectre) -- "+
						"an untagged ExtraAura applies with no spectre raised", i, len(m.Tags()))
				}
			}
		})
	}
}
