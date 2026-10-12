package calculator

import (
	"sort"
	"strings"
	"testing"
)

// TestKiarasImmunityLineParses pins the one row of the flask-immunity block whose key was narrowed to the literal "immune".
func TestKiarasImmunityLineParses(t *testing.T) {
	const line = "Immunity to Freeze, Chill, Curses and Stuns during Flask Effect"

	entry := ParseMod(line, false)
	if rem := strings.Trim(entry.Extra, " "); rem != "" {
		t.Fatalf("unparsed remainder %q for %q — the whole line is discarded, not just the remainder", rem, line)
	}

	got := make([]string, 0, len(entry.ModList))
	for _, m := range entry.ModList {
		got = append(got, m.Name())
	}
	sort.Strings(got)
	want := []string{"ChillImmune", "CurseImmune", "FreezeImmune", "StunImmune"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}
