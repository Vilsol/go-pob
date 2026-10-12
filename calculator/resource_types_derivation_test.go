package calculator

import (
	"reflect"
	"testing"
)

// TestRegenTypesCoversFormerHandWrittenRows is the search the declared gap at regenTypes never named.
func TestRegenTypesCoversFormerHandWrittenRows(t *testing.T) {
	formerlyHandWritten := map[string][]string{
		"life":                           {"LifeRegen"},
		"maximum life":                   {"LifeRegen"},
		"life and mana":                  {"LifeRegen", "ManaRegen"},
		"mana":                           {"ManaRegen"},
		"maximum mana":                   {"ManaRegen"},
		"energy shield":                  {"EnergyShieldRegen"},
		"maximum energy shield":          {"EnergyShieldRegen"},
		"maximum mana and energy shield": {"ManaRegen", "EnergyShieldRegen"},
		"rage":                           {"RageRegen"},
	}

	for line, want := range formerlyHandWritten {
		got, ok := regenTypes[line]
		if !ok {
			t.Errorf("regenTypes lost the row %q the hand-written table had", line)
			continue
		}
		if !reflect.DeepEqual(got.names, want) {
			t.Errorf("regenTypes[%q] = %v, hand-written table had %v", line, got.names, want)
		}
	}
}

// TestResourceTypesCarriesTheMaximumPrefixedRows pins the second half of ModParser.lua:5950-5976.
func TestResourceTypesCarriesTheMaximumPrefixedRows(t *testing.T) {
	if len(resourceTypes) != len(resourceTypesBase)*2 {
		t.Fatalf("resourceTypes has %d rows, want twice the %d-row literal", len(resourceTypes), len(resourceTypesBase))
	}

	for line, names := range resourceTypesBase {
		maximum, ok := resourceTypes["maximum "+line]
		if !ok {
			t.Errorf("resourceTypes is missing the maximum-prefixed row for %q", line)
			continue
		}
		if !reflect.DeepEqual(maximum, names) {
			t.Errorf("resourceTypes[%q] = %v, want %v", "maximum "+line, maximum, names)
		}
	}

	for _, tc := range []struct {
		table map[string]modNameListType
		line  string
		want  []string
	}{
		{regenTypes, "maximum life and energy shield", []string{"LifeRegen", "EnergyShieldRegen"}},
		{degenTypes, "maximum mana and life", []string{"LifeDegen", "ManaDegen"}},
		{costTypes, "maximum energy shield", []string{"EnergyShieldCost"}},
		{baseCostTypes, "maximum rage", []string{"RageCostNoMult"}},
	} {
		got, ok := tc.table[tc.line]
		if !ok {
			t.Errorf("table is missing %q", tc.line)
			continue
		}
		if !reflect.DeepEqual(got.names, tc.want) {
			t.Errorf("table[%q] = %v, want %v", tc.line, got.names, tc.want)
		}
	}
}
