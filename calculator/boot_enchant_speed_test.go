package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestKilledRecentlyBootEnchantIsFlagged pins ModParser.lua:4357-4360.
func TestKilledRecentlyBootEnchantIsFlagged(t *testing.T) {
	const line = "10% increased Attack and Cast Speed if you've Killed Recently"
	mods, extra := parseMod(line, 1)
	if extra != "" {
		t.Fatalf("REFUSING: %q left %q unparsed, so this test cannot see the defect", line, extra)
	}
	var flags []mod.MFlag
	for _, m := range mods {
		if m.Name() != "Speed" {
			continue
		}
		for _, tg := range m.Tags() {
			c, ok := tg.(*mod.ConditionTag)
			if ok && len(c.VarList) == 1 && c.VarList[0] == "KilledRecently" {
				flags = append(flags, m.Flags())
			}
		}
	}
	if len(flags) != 2 {
		t.Fatalf("%d Condition:KilledRecently Speed mod(s) with flags %v, want 2 (one Attack, "+
			"one Cast). One unflagged mod matches every cfg, including passes with neither flag.",
			len(flags), flags)
	}
	var haveAttack, haveCast bool
	for _, f := range flags {
		haveAttack = haveAttack || f == mod.MFlagAttack
		haveCast = haveCast || f == mod.MFlagCast
	}
	if !haveAttack || !haveCast {
		t.Errorf("flags %v, want exactly one Attack (%#x) and one Cast (%#x)",
			flags, mod.MFlagAttack, mod.MFlagCast)
	}
}
