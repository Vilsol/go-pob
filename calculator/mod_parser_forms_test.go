package calculator

import (
	"strings"
	"testing"
)

// TestMissingParserForms pins five rows of ModParser.lua's `formList` that were never ported.
func TestMissingParserForms(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("Regenerate 2 Life per second", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: an already-ported form does not parse; the form scan is broken " +
			"generally and this test cannot attribute a failure")
	}

	for _, tc := range []struct {
		form string
		line string
		want string
	}{
		{"GRANTS", "Grants 3 Life per Enemy Hit", "LifeOnHit"},
		{"LOSE", "Lose 7 Mana per Enemy Hit", "ManaOnHit"},
	} {
		t.Run(tc.form, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil {
				t.Fatalf("no entry: %s", tc.line)
			}
			if extra := strings.TrimSpace(modExtra(e)); extra != "" {
				t.Fatalf("leftover %q (line is DROPPED WHOLE): %s", extra, tc.line)
			}
			var got []string
			for _, m := range e.ModList {
				got = append(got, m.Name())
			}
			if !contains(got, tc.want) {
				t.Fatalf("produced %v, want %s: %s", got, tc.want, tc.line)
			}
		})
	}
}
