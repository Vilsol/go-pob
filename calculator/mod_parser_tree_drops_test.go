package calculator

import (
	"strings"
	"testing"
)

// TestTreeDropGrammarGapsParse pins three passive-tree stat lines the parser discarded WHOLE.
func TestTreeDropGrammarGapsParse(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("Regenerate 1 Life per second", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line does not parse; this test cannot attribute a failure")
	}

	for _, tc := range []struct {
		line string
		want string
	}{
		{"Debuffs on you expire 30% faster", "SelfDebuffExpirationRate"},
		{"Debuffs on you expire 15% faster", "SelfDebuffExpirationRate"},
		{"Regenerate 1 Life per second for each 1% Uncapped Fire Resistance", "LifeRegen"},
		{"Hits have 50% chance to ignore Enemy Physical Damage Reduction", "ChanceToIgnoreEnemyPhysicalDamageReduction"},
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
		if len(e.ModList) == 0 {
			t.Errorf("parsed but produced 0 mods: %s", tc.line)
			continue
		}
		var got []string
		for _, m := range e.ModList {
			got = append(got, m.Name())
		}
		if !contains(got, tc.want) {
			t.Errorf("produced %v, want a %q mod: %s", got, tc.want, tc.line)
		}
	}
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}
