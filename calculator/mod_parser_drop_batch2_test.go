package calculator

import (
	"strings"
	"testing"
)

// TestDropBatch2Parses pins three more lines the drop censuses found and the pinned PoB parses.
func TestDropBatch2Parses(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 ||
		strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line does not parse; this test cannot attribute a failure")
	}

	for _, tc := range []struct {
		line string
		want string
	}{
		{"Recover Energy Shield equal to 2% of Armour when you Block", "EnergyShieldOnBlock"},
		{"Recover Energy Shield equal to 3% of Evasion Rating when you Block", "EnergyShieldOnBlock"},
		{"Enemies Ignited by you during Effect have Malediction", "EnemyModifier"},
		{"Enemies Blinded by you have Malediction", "EnemyModifier"},
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
