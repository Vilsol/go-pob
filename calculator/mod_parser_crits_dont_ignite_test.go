package calculator

import (
	"strings"
	"testing"
)

// TestCritsDontInherentlyIgniteParses pins ModParser.lua:3790.
func TestCritsDontInherentlyIgniteParses(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("Critical Strikes do not inherently Freeze", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the already-ported freeze sibling does not parse; the family is " +
			"broken generally and this test cannot attribute a failure")
	}

	const line = "Critical Strikes do not inherently Ignite"
	e := ParseMod(line, false)
	if e == nil {
		t.Fatalf("no entry: %s", line)
	}
	if extra := strings.TrimSpace(modExtra(e)); extra != "" {
		t.Fatalf("leftover %q (line is DROPPED WHOLE): %s", extra, line)
	}
	got := make([]string, 0, len(e.ModList))
	for _, m := range e.ModList {
		got = append(got, m.Name())
	}
	if !contains(got, "CritsDontAlwaysIgnite") {
		t.Fatalf("produced %v, want CritsDontAlwaysIgnite", got)
	}
}
