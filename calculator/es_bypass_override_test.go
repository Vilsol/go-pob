package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestAllDamageBypassesESIsOverride pins ModParser.lua:2377-2383.
func TestAllDamageBypassesESIsOverride(t *testing.T) {
	e := ParseMod("All Damage taken bypasses Energy Shield", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 5 {
		t.Fatalf("ParseMod = %v, want 5 mods and no Extra", e)
	}
	seen := map[string]bool{}
	for _, m := range e.ModList {
		if m.Type() != mod.TypeOverride || m.Value().Float() != 100 {
			t.Errorf("%s: type %s value %v, want OVERRIDE 100", m.Name(), m.Type(), m.Value().Float())
		}
		seen[m.Name()] = true
	}
	for _, dt := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		if !seen[dt+"EnergyShieldBypass"] {
			t.Errorf("missing %sEnergyShieldBypass", dt)
		}
	}

	for _, tc := range []struct {
		line  string
		count int
		value float64
	}{
		{"33% of Non-Chaos Damage taken bypasses Energy Shield", 4, 33},
		{"Physical Damage taken bypasses Energy Shield", 1, 100},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || e.Extra != "" || len(e.ModList) != tc.count {
			t.Fatalf("control %q: ParseMod = %v, want %d mods", tc.line, e, tc.count)
		}
		for _, m := range e.ModList {
			if m.Type() != mod.TypeBase || m.Value().Float() != tc.value {
				t.Errorf("control %q: %s %s %v, want BASE %v", tc.line, m.Name(), m.Type(), m.Value().Float(), tc.value)
			}
		}
	}
}

// TestESBypassReadsOverride pins CalcDefence.lua:2286, `modDB:Override(nil, T.."EnergyShieldBypass") or modDB:Sum("BASE", nil, T.."EnergyShieldBypass")`.
func TestESBypassReadsOverride(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0012-occultist-unknown.xml")
	if o := env.Player.ModDB.Override(nil, "PhysicalEnergyShieldBypass"); o == nil {
		t.Fatal("premise broken: no PhysicalEnergyShieldBypass OVERRIDE on the player modDB")
	}
	for _, dt := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		if got := env.Player.Output[dt+"EnergyShieldBypass"]; got != 100 {
			t.Errorf("%sEnergyShieldBypass = %v, want 100 (PoB dump)", dt, got)
		}
	}
}
