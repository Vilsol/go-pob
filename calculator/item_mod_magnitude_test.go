package calculator

import (
	"reflect"
	"testing"
)

// TestModMagnitudeLinesParse: Item.lua:1249-1283 reads three line forms into modMagnitude entries.
func TestModMagnitudeLinesParse(t *testing.T) {
	initBenchData(t)
	text := `Rarity: Unique
Test Amulet
Onyx Amulet
Implicits: 0
40% increased Explicit Modifier magnitudes
Implicit Modifier magnitudes are doubled
25% reduced effect of Suffixes`
	item := ParseItem(text)
	two := 2.0
	want := []ModMagnitudeMod{
		{Quality: 40, ModType: "explicit"},
		{Quality: 100, Multiplier: &two, ModType: "implicit"},
		{Quality: -25, Tags: []string{"suffix"}},
	}
	if !reflect.DeepEqual(item.ModMagnitudeMods, want) {
		t.Fatalf("got %+v, want %+v", item.ModMagnitudeMods, want)
	}
}

// TestReminderTextIsSkipped: Item.lua:540-544 skips reminder text.
func TestReminderTextIsSkipped(t *testing.T) {
	initBenchData(t)
	text := `Rarity: Rare
Test Amulet
Onyx Amulet
Implicits: 0
+10 to Strength
(Reminder text that
spans two lines)
(25% of Damage is reminder too)
+12 to Dexterity`
	item := ParseItem(text)
	got := make([]string, 0, len(item.ExplicitModLines))
	for _, ml := range item.ExplicitModLines {
		got = append(got, ml.Line)
	}
	if want := []string{"+10 to Strength", "+12 to Dexterity"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("explicit lines %q, want %q", got, want)
	}
}
