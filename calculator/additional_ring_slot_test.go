package calculator

import (
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestAdditionalRingSlotRowParses pins ModParser.lua:3056.
func TestAdditionalRingSlotRowParses(t *testing.T) {
	entry := ParseMod("+1 Ring Slot", false)
	if strings.Trim(entry.Extra, " ") != "" {
		t.Fatalf("unparsed remainder %q -- the whole line is discarded", entry.Extra)
	}
	if len(entry.ModList) != 1 {
		t.Fatalf("got %d mods, want 1", len(entry.ModList))
	}
	m := entry.ModList[0]
	if m.Name() != "AdditionalRingSlot" || m.Type() != mod.TypeFlag {
		t.Fatalf("got %s %s, want AdditionalRingSlot FLAG", m.Name(), m.Type())
	}
}

// TestRing3NeedsTheTreeToGrantIt pins CalcSetup.lua:708-711.
func TestRing3NeedsTheTreeToGrantIt(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	cases := map[string]bool{
		"0016-gladiator-dual-strike-of-ambidexterity": true,
		"0052-juggernaut-cleave":                      true,
		"0320-chieftain-shield-charge":                false,
		"0948-elementalist-corrupting-fever-8":        false,
	}

	for build, wantPresent := range cases {
		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			if _, filled := equippedSlotIDs(t, build)["Ring 3"]; !filled {
				t.Fatalf("REFUSING: %s has no Ring 3 item, so this build cannot test the guard", build)
			}
			got := env.Player.ItemList["Ring 3"] != nil
			if got != wantPresent {
				t.Errorf("Ring 3 present = %v, want %v", got, wantPresent)
			}
			if env.Player.ItemList["Ring 1"] == nil {
				t.Error("Ring 1 went missing; the guard is too wide")
			}
		})
	}
}
