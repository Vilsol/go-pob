package calculator

import (
	"strings"
	"testing"
)

// TestModTagListBatch pins a batch of ModParser.lua `modTagList` rows that were never ported.
func TestModTagListBatch(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("10% increased Damage per Power Charge", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: an already-ported modTagList row no longer parses; this test cannot " +
			"attribute a failure to the new rows")
	}

	for _, tc := range []struct {
		shape string
		line  string
		want  string
	}{
		{"Multiplier", "+40 to maximum Life for each Empty Red Socket on any Equipped Item", "Life"},
		{"Condition", "10% increased Movement Speed on Kill", "MovementSpeed"},
		{"PerStat/div", "1% increased Armour per 100 Armour on Equipped Boots", "Armour"},
		{"PerStat", "1% increased Attack Speed per Dexterity", "Speed"},
		{"consume", "10% chance to gain Onslaught for 4 seconds on Kill", "Condition:Onslaught"},
	} {
		t.Run(tc.shape, func(t *testing.T) {
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
