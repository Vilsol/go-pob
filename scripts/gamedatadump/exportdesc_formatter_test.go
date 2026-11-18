package main

import (
	"slices"
	"testing"
)

// statdesc.lua:399-403: a formatter's stat value is a row number (Lua's 1-based index) in its name table, a range
// describes its lowest value, and a value with no row stays a number.
func TestDescriptionFormatterNamesAValue(t *testing.T) {
	saved := descriptionFormatterNames
	t.Cleanup(func() { descriptionFormatterNames = saved })
	descriptionFormatterNames = map[string][]string{"passive_jewel_radius": {"Small", "Medium", "Large"}}

	desc := &edDescriptor{stats: []string{"local_jewel_variable_ring_radius_value"}, order: 1, variants: []sdVariant{{
		Limit: [][2]any{{"#", "#"}},
		Text:  "Only affects Passives in {0} Ring",
		Specs: []sdSpec{{K: "passive_jewel_radius", V: float64(1)}, {K: "reminderstring", V: "ReminderTextJewelRadiusRing"}},
	}}}
	e := &exportDescriber{descriptors: map[string]*edDescriptor{"local_jewel_variable_ring_radius_value": desc}}
	for _, c := range []struct {
		value [2]float64
		want  string
	}{
		{[2]float64{1, 1}, "Only affects Passives in Small Ring"},
		{[2]float64{3, 3}, "Only affects Passives in Large Ring"},
		{[2]float64{2, 3}, "Only affects Passives in Medium Ring"},
		{[2]float64{4, 4}, "Only affects Passives in 4 Ring"},
	} {
		out, _, err := e.describeStats(map[string][2]float64{"local_jewel_variable_ring_radius_value": c.value})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(out, []string{c.want}) {
			t.Errorf("%v: got %q, want %q", c.value, out, c.want)
		}
	}
}
