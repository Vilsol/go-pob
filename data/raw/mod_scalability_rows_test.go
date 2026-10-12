package raw

import (
	"reflect"
	"testing"
)

// TestModScalabilityRowsDecode: the table is emitted as encoded rows.
func TestModScalabilityRowsDecode(t *testing.T) {
	for key, want := range map[string][]ModScalabilityEntry{
		"":                                     {},
		"#":                                    {{IsScalable: false, Formats: []string{"tree_expansion_jewel_passive"}}},
		"# Accuracy Rating per 2 Intelligence": {{IsScalable: true}},
		"# to Level of all # Gems":             {{IsScalable: true}, {IsScalable: false, Formats: []string{"display_indexable_skill"}}},
		"#% reduced Global Attack Speed per Level": {{IsScalable: true, Formats: []string{"negate", "milliseconds_to_seconds"}}},
		"# Lightning Damage taken per second per Power Charge if\nyour Skills have dealt a Critical Strike Recently": {
			{IsScalable: true, Formats: []string{"per_minute_to_per_second"}},
		},
		"Zealotry has #% reduced Aura Effect": {{IsScalable: true, Formats: []string{"negate"}}},
	} {
		got, ok := ModScalability[key]
		if !ok {
			t.Errorf("%q: absent", key)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%q: got %#v, want %#v", key, got, want)
		}
	}
	if len(ModScalability) != len(modScalabilityRows) {
		t.Errorf("%d keys from %d rows: a key repeats", len(ModScalability), len(modScalabilityRows))
	}
}

// TestItemModStatOrderRowsDecode: the table is emitted as an array of rows; each field must land in its own place and no key may repeat.
func TestItemModStatOrderRowsDecode(t *testing.T) {
	for key, want := range map[string]ItemModStatOrderEntry{
		"# added passive skill is a jewel socket":                        {Exact: 0, Normalised: 8082},
		"# cinderflame, applying -#% to fire resistance":                 {Exact: 0, Normalised: 6680.1},
		"50% increased effect of shrine buffs on you":                    {Exact: 2850, Normalised: 0},
		"zombies' slam attack has 100% increased cooldown recovery rate": {Exact: 8252.1, Normalised: 0},
	} {
		if got, ok := ItemModStatOrder[key]; !ok || got != want {
			t.Errorf("%q: got %+v (present %v), want %+v", key, got, ok, want)
		}
	}
	if len(ItemModStatOrder) != len(itemModStatOrderRows) {
		t.Errorf("%d keys from %d rows: a key repeats", len(ItemModStatOrder), len(itemModStatOrderRows))
	}
}
