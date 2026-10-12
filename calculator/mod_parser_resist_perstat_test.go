package calculator

import (
	"strings"
	"testing"
)

// TestPerElementalResistancePerStatParses pins ModParser.lua:1520-1522.
func TestPerElementalResistancePerStatParses(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+5% to Fire Resistance per 1% Chaos Resistance", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the already-ported chaos sibling does not parse; the family is " +
			"broken generally and this test cannot attribute a failure")
	}

	for _, line := range []string{
		"+1% to Chaos Resistance per 1% Fire Resistance",
		"+1% to Chaos Resistance per 1% Cold Resistance",
		"+1% to Chaos Resistance per 1% Lightning Resistance",
	} {
		e := ParseMod(line, false)
		if e == nil {
			t.Errorf("no entry: %s", line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q (line is DROPPED WHOLE): %s", extra, line)
			continue
		}
		if len(e.ModList) == 0 {
			t.Errorf("parsed but produced 0 mods: %s", line)
		}
	}
}
