package calculator

import (
	"strings"
	"testing"
)

// TestCombinedResistanceOver75Parses pins ModParser.lua:1527, the COMBINED "per N% Fire, Cold, or Lightning Resistance above 75%" form.
func TestCombinedResistanceOver75Parses(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("10% increased Elemental Damage per 1% Cold Resistance above 75%", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the already-ported per-element sibling does not parse")
	}

	const line = "10% increased Elemental Damage per 1% Fire, Cold, or Lightning Resistance above 75%"
	e := ParseMod(line, false)
	if e == nil {
		t.Fatalf("no entry: %s", line)
	}
	if extra := strings.TrimSpace(modExtra(e)); extra != "" {
		t.Fatalf("leftover %q (line is DROPPED WHOLE): %s", extra, line)
	}
	if len(e.ModList) == 0 {
		t.Fatalf("parsed but produced 0 mods: %s", line)
	}
}
