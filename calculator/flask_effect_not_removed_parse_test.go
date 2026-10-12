package calculator

import (
	"testing"
)

// TestFlaskEffectNotRemovedLinesParse pins ModParser.lua:5144-5152's three rows against the four LEECH rows they sit beside.
//
//pob:block:src/Modules/ModParser.lua:5144-5152
func TestFlaskEffectNotRemovedLinesParse(t *testing.T) {
	for _, tc := range []struct {
		line string
		flag string
	}{
		{"Life Flask Effects are not removed when Unreserved Life is Filled", "LifeFlaskEffectNotRemoved"},
		{"Mana Flask Effects are not removed when Unreserved Mana is Filled", "ManaFlaskEffectNotRemoved"},
		{"Effect is not removed when Unreserved Mana is Filled", "ManaFlaskEffectNotRemoved"},

		{"Life Leech effects are not removed when Unreserved Life is Filled", "CanLeechLifeOnFullLife"},
		{"Life Leech effects are not removed at Full Life", "CanLeechLifeOnFullLife"},
		{"Energy Shield Leech effects are not removed when Energy Shield is Filled", "CanLeechEnergyShieldOnFullEnergyShield"},
	} {
		t.Run(tc.flag+" | "+tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v mods, want exactly 1", tc.line, e)
			}
			if e.Extra != "" {
				t.Errorf("ParseMod(%q) left Extra %q — processModLine discards the whole line", tc.line, e.Extra)
			}
			if got := e.ModList[0].Name(); got != tc.flag {
				t.Errorf("ParseMod(%q) emitted %q, want %q", tc.line, got, tc.flag)
			}
			if got := e.ModList[0].Value(); got == nil || !got.Flag() {
				t.Errorf("ParseMod(%q) emitted %q with value %v, want a true FLAG", tc.line, tc.flag, got)
			}
		})
	}
}
