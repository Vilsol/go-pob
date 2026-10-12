package calculator

import (
	"testing"
)

// TestPerfectAgonyDotMultiplierChainIsWhole is the positive successor to the tripwire that stood here until 2026-08-30.
func TestPerfectAgonyDotMultiplierChainIsWhole(t *testing.T) {
	requireGameData(t, "3.29")

	const line = "Damage over Time Multiplier for Ailments is equal to Critical Strike Multiplier"

	if control := ParseMod("10% increased Critical Strike Multiplier", false); control == nil || len(control.ModList) == 0 {
		t.Fatal("REFUSING: the control line parsed to nothing, so this probe cannot distinguish " +
			"an absent row from a broken call")
	}

	e := ParseMod(line, false)
	if e == nil {
		t.Fatalf("ParseMod returned nothing for %q -- the Perfect Agony parser row has been "+
			"lost. Its consumer in calc_offence.go is then correct and inert, which no "+
			"missing-thing instrument here can see.", line)
	}

	if e.Extra != "" {
		t.Errorf("ParseMod left extra %q on %q -- an unparsed remainder discards the whole line",
			e.Extra, line)
	}

	for _, m := range e.ModList {
		if m.Name() == "DotMultiplierIsCritMultiplier" {
			return
		}
	}
	t.Errorf("ParseMod produced %d mods for %q but none named the Perfect Agony ailment "+
		"dot-multiplier flag its consumer reads", len(e.ModList), line)
}
