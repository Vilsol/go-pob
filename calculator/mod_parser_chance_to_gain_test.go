package calculator

import (
	"strings"
	"testing"
)

// TestChanceToGainFormParses pins ModParser.lua:99, `["^([%+%-]?%d+)%% chance to gain "] = "FLAG"`.
func TestChanceToGainFormParses(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("5% chance to Freeze", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the plain `N%% chance` form no longer parses; this test cannot " +
			"distinguish a missing row from a shadowed one")
	}

	for _, tc := range []struct {
		line string
		want string
	}{
		{"10% chance to gain Onslaught for 4 seconds on Kill", "Condition:Onslaught"},
		{"5% chance to gain Phasing for 4 seconds on Kill", "Condition:Phasing"},
		{"10% chance to gain Arcane Surge when you Kill an Enemy", "Condition:ArcaneSurge"},
	} {
		e := ParseMod(tc.line, false)
		if e == nil {
			t.Errorf("no entry: %s", tc.line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q (line is DROPPED WHOLE): %s", extra, tc.line)
			continue
		}
		var got []string
		for _, m := range e.ModList {
			got = append(got, m.Name())
		}
		if !contains(got, tc.want) {
			t.Errorf("produced %v, want %s: %s", got, tc.want, tc.line)
		}
	}
}
