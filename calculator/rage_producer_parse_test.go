package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestRageProducersParse pins the ModParser lines that publish Condition:CanGainRage.
func TestRageProducersParse(t *testing.T) {
	for _, tc := range []struct {
		line          string
		wantRageRegen bool
	}{
		{"Regenerate 1 Rage per second for every 100 Life Recovery per second from Regeneration", true},
		{"1 Rage Regenerated for every 25 Mana Regeneration per Second", true},
		{"When you lose Temporal Chains you gain maximum Rage", false},

		{"Gain 5 Rage when you Kill an Enemy", false},
		{"Gain 5 Rage when you use a Warcry", false},
		{"You and nearby Party Members gain 5 Rage when you Warcry", false},
		{"Gain 1 Rage on Hit with Attacks, no more than once every 0.3 seconds", false},
		{"Warcries grant 10 Rage per 5 Power if you have less than 25 Rage", false},
		{"Gain 1 Rage on Critical Hit with attacks, no more than once every 0.5 seconds", false},
		{"Attacks with Axes or Swords grant 1 Rage on Hit, no more than once every second", false},
		{"While a Unique Enemy is in your Presence, Gain 1 Rage on Hit with Attacks, no more than once every 0.3 seconds", false},
		{"While a Pinnacle Atlas Boss is in your Presence, Gain 1 Rage on Hit with Attacks, no more than once every 0.3 seconds", false},
	} {
		entry := ParseMod(tc.line, false)
		if entry == nil || entry.ModList == nil {
			t.Errorf("%q: no modList at all", tc.line)
			continue
		}

		var sawFlag, sawRegen bool
		for _, m := range entry.ModList {
			switch m.Name() {
			case "Condition:CanGainRage":
				sawFlag = m.Type() == mod.TypeFlag
			case "RageRegen":
				sawRegen = m.Type() == mod.TypeBase
			}
		}

		if !sawFlag {
			t.Errorf("%q: no Condition:CanGainRage FLAG (got %d mods, extra %q)",
				tc.line, len(entry.ModList), entry.Extra)
		}
		if sawRegen != tc.wantRageRegen {
			t.Errorf("%q: RageRegen BASE present=%v, want %v", tc.line, sawRegen, tc.wantRageRegen)
		}
	}
}
