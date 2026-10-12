package calculator

import (
	"strings"
	"testing"
)

// TestWardDoesNotBreakEmitsUnbrokenWard pins ModParser.lua:4972.
func TestWardDoesNotBreakEmitsUnbrokenWard(t *testing.T) {
	requireGameData(t, "3.29")

	e := ParseMod("Ward does not Break during Effect", false)
	if e == nil || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the line does not parse cleanly, so nothing below is attributable")
	}

	got := make([]string, 0, len(e.ModList))
	for _, m := range e.ModList {
		got = append(got, m.Name())
	}
	if !contains(got, "Condition:WardNotBreak") {
		t.Errorf("produced %v, missing Condition:WardNotBreak; the existing readers would go dark", got)
	}
	if !contains(got, "Condition:UnbrokenWard") {
		t.Errorf("produced %v, missing Condition:UnbrokenWard (ModParser.lua:4972 emits both)", got)
	}
}
