package calculator

import (
	"strings"
	"testing"
)

// TestDuringEffectAcceptsTheFlasklessForm pins a ONE-TOKEN divergence from the pin that costs whole modifier lines.
func TestDuringEffectAcceptsTheFlasklessForm(t *testing.T) {
	requireGameData(t, "3.29")

	for _, line := range []string{
		"Enemies Ignited by you during Effect take 10% increased Damage",
		"Adds Knockback during Effect",
		"Culling Strike during Effect",
		"Zealot's Oath during Effect",
		"Ward does not break during Effect",
		"Enemies Ignited by you during Flask Effect take 10% increased Damage",
		"Adds Knockback during Flask Effect",
		"Culling Strike during Flask Effect",
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
