package calculator

import (
	"strings"
	"testing"
)

// TestDropBatch3Parses pins seven more PoB-confirmed drops, all with literal patterns in the pin.
func TestDropBatch3Parses(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 ||
		strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line does not parse; this test cannot attribute a failure")
	}

	if e := ParseMod("You take 10% reduced Extra Damage from Critical Strikes", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the already-ported sibling row does not parse")
	}

	for _, tc := range []struct {
		line string
		want string
	}{
		{"Monsters cannot Block your Attacks", "EnemyModifier"},
		{"You cannot be Cursed with Silence", "SilenceImmune"},
		{"Skills Cost Energy Shield instead of Mana or Life", "CostESInsteadOfManaOrLife"},
		{"+1% to all maximum Elemental Resistances if you have Reserved Life and Mana", "ElementalResistMax"},
		{"12% increased maximum Life and Mana if your equipped Staff has a Red and Blue Socket", "Life"},
		{"6% increased Cast Speed for each different Non-Instant Spell you've Cast Recently", "Speed"},
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
