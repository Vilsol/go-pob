package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMinionDamageConversionLinesParse pins ModParser.lua:4433, the GENERIC minion conversion row.
func TestMinionDamageConversionLinesParse(t *testing.T) {
	for _, tc := range []struct {
		line string
		conv string
		val  float64
		mult string
	}{
		{"Minions convert 100% of Fire Damage to Chaos Damage", "FireDamageConvertToChaos", 100, ""},
		{"Minions convert 25% of Physical Damage to Lightning Damage per Socketed Blue Gem", "PhysicalDamageConvertToLightning", 25, "SocketedBlueGemsIn{SlotName}"},
		{"Minions convert 25% of Physical Damage to Chaos Damage per Empty Socket", "PhysicalDamageConvertToChaos", 25, "EmptySocketIn{SlotName}"},
		{"Minions convert 25% of Physical Damage to Fire Damage per Red Socket", "PhysicalDamageConvertToFire", 25, "RedSocketIn{SlotName}"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v, want 1 mod", tc.line, e)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — processModLine discards the whole line", e.Extra)
			}
			m := e.ModList[0]
			if m.Name() != "MinionModifier" {
				t.Fatalf("mod name = %q, want MinionModifier", m.Name())
			}
			mm, ok := m.Value().List().(mod.MinionModifier)
			if !ok {
				t.Fatalf("value is %T, want mod.MinionModifier", m.Value().List())
			}
			if mm.Mod.Name() != tc.conv {
				t.Errorf("inner mod = %q, want %q", mm.Mod.Name(), tc.conv)
			}
			if mm.Mod.Type() != mod.TypeBase || mm.Mod.Value().Float() != tc.val {
				t.Errorf("inner mod = %v %v, want BASE %v", mm.Mod.Type(), mm.Mod.Value().Float(), tc.val)
			}
			got := ""
			for _, tg := range m.Tags() {
				if mt, ok := tg.(*mod.MultiplierTag); ok && len(mt.VariableList) > 0 {
					got = mt.VariableList[0]
				}
			}
			if got != tc.mult {
				t.Errorf("multiplier var = %q, want %q", got, tc.mult)
			}
		})
	}
}
