package calculator

import "testing"

// TestEmperorsVigilanceBypassLinesParse pins the two Emperor's Vigilance rows.
func TestEmperorsVigilanceBypassLinesParse(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
	}{
		{"Damage taken from Blocked Hits cannot bypass Energy Shield", "BlockedDamageDoesntBypassES"},
		{"Damage taken from Unblocked hits always bypasses Energy Shield", "UnblockedDamageDoesBypassES"},
		{"Chaos Damage taken does not bypass Energy Shield while not on Low Mana", "ChaosNotBypassEnergyShield"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v mods, want 1", tc.line, e)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — processModLine discards the whole line", e.Extra)
			}
			if got := e.ModList[0].Name(); got != tc.name {
				t.Errorf("mod name = %q, want %q", got, tc.name)
			}
		})
	}

	for _, line := range []string{
		"Chaos Damage does not bypass Energy Shield",
		"Chaos Damage taken does not bypass Energy Shield",
	} {
		if e := ParseMod(line, false); e == nil || len(e.ModList) != 1 || e.Extra != "" {
			t.Errorf("control %q: ParseMod produced %v, want exactly 1 mod and no Extra", line, e)
		}
	}
}
