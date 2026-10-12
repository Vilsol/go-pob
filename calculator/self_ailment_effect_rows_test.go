package calculator

import (
	"slices"
	"strings"
	"testing"
)

// TestSelfAilmentEffectRowsParse covers the THREE assertions that `TestSelfNonDamagingAilmentEffectLinesParse` in self_ailment_effect_parse_test.go does not make about the same two rows.
func TestSelfAilmentEffectRowsParse(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 ||
		strings.TrimSpace(modExtra(e)) != "" {
		t.Fatal("REFUSING: the control line `+10 to Strength` did not parse cleanly, so nothing " +
			"below can be attributed to the rows this test is about")
	}

	parse := func(t *testing.T, line string) []string {
		t.Helper()
		e := ParseMod(line, false)
		if e == nil {
			t.Fatalf("ParseMod(%q) returned nil", line)
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Fatalf("leftover %q — an unparsed remainder DISCARDS THE WHOLE LINE: %s", extra, line)
		}
		names := make([]string, 0, len(e.ModList))
		for _, m := range e.ModList {
			names = append(names, m.Name())
		}
		return names
	}

	for _, line := range []string{
		"20% increased Effect of non-Damaging Ailments you inflict",
		"20% increased Effect of non-Damaging Ailments",
	} {
		t.Run(line, func(t *testing.T) {
			got := parse(t, line)
			if !slices.Contains(got, "EnemyChillEffect") {
				t.Errorf("%q -> %v, want the ENEMY-side names; the on-you row has swallowed it",
					line, got)
			}
			for _, n := range got {
				if strings.HasPrefix(n, "Self") {
					t.Errorf("%q emitted %s: an enemy-side row must not produce a Self* name", line, n)
				}
			}
		})
	}
}
