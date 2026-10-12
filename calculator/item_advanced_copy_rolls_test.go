package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestAdvancedCopyRollRangesMatchesThePin pins Item.lua:1121-1160.
func TestAdvancedCopyRollRangesMatchesThePin(t *testing.T) {
	half, point4, point6, one := 0.5, 0.4, 0.6, 1.0
	for _, c := range []struct {
		in, want string
		rng      *float64
	}{
		{"Maximum number of Summoned Reapers (Animated Weapons-Holy Armaments) is Doubled", "Maximum number of Summoned Reapers is Doubled", nil},
		{"Cannot have Minions other than Summoned Reapers (Animated Weapons-Holy Armaments)", "Cannot have Minions other than Summoned Reapers", nil},
		{"Auras from Chaos Skills at 15(10-15)% of their value, up to a maximum of 150%", "Auras from Chaos Skills at (10-15)% of their value, up to a maximum of 150%", &one},
		{"Adds 12(10-15) to 30(25-35) Fire Damage", "Adds 12 to 30 Fire Damage", nil},
		{"Adds 12(10-15) to 29(25-35) Fire Damage", "Adds (10-15) to (25-35) Fire Damage", &point4},
		{"-7(-10--5)% to Fire Resistance", "+(-10--5)% to Fire Resistance", &point6},
		{"+60(40-50) to maximum Life", "+60 to maximum Life", nil},
		{"+45(50-40) to maximum Life", "+(40-50) to maximum Life", &half},
		{"(Some-Thing) nested (a (b-c) d) end", " nested end", nil},
		{"+(10-20) to Strength", "+(10-20) to Strength", nil},
		{"Grants 3.5(2.5-4.5) Life per Enemy Hit", "Grants (2.5-4.5) Life per Enemy Hit", &half},
		{"x (a-b)(c-d) y", "x y", nil},
	} {
		got, rng := advancedCopyRollRanges(c.in)
		if got != c.want {
			t.Errorf("%q: line = %q, want %q", c.in, got, c.want)
		}
		switch {
		case (rng == nil) != (c.rng == nil):
			t.Errorf("%q: range = %v, want %v", c.in, rng, c.rng)
		case rng != nil && *rng != *c.rng:
			t.Errorf("%q: range = %v, want %v", c.in, *rng, *c.rng)
		}
	}
}

// TestAdvancedCopyEnumAndBaseValueReachTheModList drives both halves through ParseItem.
func TestAdvancedCopyEnumAndBaseValueReachTheModList(t *testing.T) {
	t.Run("value(base) keeps the value", func(t *testing.T) {
		item := ParseItem("Rarity: RARE\nFixture\nTitan Greaves\nImplicits: 0\n{tags:defences,armour,energyshield}1070(1000)% increased Armour and Energy Shield\n")
		if len(item.ExplicitModLines) != 1 {
			t.Fatalf("REFUSING: %d explicit lines, want 1", len(item.ExplicitModLines))
		}
		if got := item.ExplicitModLines[0].Line; got != "1070% increased Armour and Energy Shield" {
			t.Errorf("line = %q", got)
		}
		if extra := item.ExplicitModLines[0].Extra; extra != "" {
			t.Errorf("line did not parse: extra %q", extra)
		}
	})

	t.Run("value(min-max) derives the range that rolls the mod", func(t *testing.T) {
		item := ParseItem("Rarity: UNIQUE\nFixture\nTitan Greaves\nImplicits: 0\n+48(40-50) to maximum Life\n")
		if got := sumItemMod(item, "Life", mod.TypeBase); got != 48 {
			t.Errorf("Life BASE = %v, want 48 (range 0.8 from 48 in 40-50)", got)
		}
	})

	t.Run("Dark Monarch's enum line parses", func(t *testing.T) {
		item := ParseItem("Rarity: UNIQUE\nThe Dark Monarch\nTitan Greaves\nImplicits: 0\nMaximum number of Summoned Reapers (Animated Weapons-Holy Armaments) is Doubled\n+25% movement speed\n")
		var found bool
		for _, ml := range item.ExplicitModLines {
			if ml.Line == "Maximum number of Summoned Reapers is Doubled" {
				found = true
				if ml.Extra != "" || len(ml.ModList) == 0 {
					t.Errorf("enum-stripped line did not parse: extra %q, %d mods", ml.Extra, len(ml.ModList))
				}
			}
		}
		if !found {
			t.Errorf("no enum-stripped line; explicit lines %q", explicitLines(item))
		}
	})
}
