package calculator

import (
	"strings"
	"testing"
)

// TestDropBatch4Parses pins two more PoB-confirmed drops with consumers already in place.
func TestDropBatch4Parses(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 ||
		strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line does not parse")
	}

	for _, tc := range []struct{ line, want string }{
		{"Minions Recover 2% of their Life when they Block", "MinionModifier"},
		{"Nearby Enemies are Intimidated while you have Rage", "EnemyModifier"},
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
			t.Errorf("produced %v, want %q: %s", got, tc.want, tc.line)
		}
	}
}
