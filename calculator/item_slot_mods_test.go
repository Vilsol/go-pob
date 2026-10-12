package calculator

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

func slotMultiplierNames(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if strings.Contains(n, "SocketIn") || strings.HasPrefix(n, "Multiplier:QualityOn") {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

func loadItemModListOracle(t *testing.T) ([]itemParseCase, []expectedModList) {
	t.Helper()

	var cases []itemParseCase
	raw, err := os.ReadFile(itemParseCases)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &cases))

	var expected []expectedModList
	raw, err = os.ReadFile(itemModListExpected)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &expected))
	testza.AssertEqual(t, len(cases), len(expected), "cases/expected out of sync; regenerate with `task oracle`")

	return cases, expected
}

// TestItemEmitsSocketAndQualityMultipliers asserts the socket-colour and quality multiplier mods.
func TestItemEmitsSocketAndQualityMultipliers(t *testing.T) {
	cases, expected := loadItemModListOracle(t)

	for idx, c := range cases {
		w := expected[idx]
		if !w.HasBase || w.Error != nil {
			continue
		}

		t.Run(c.Name, func(t *testing.T) {
			item := ParseItem(c.Raw)
			if itemHasUnparsedLine(item) {
				t.Skip("blocked on ModParser coverage")
			}

			wantNames := make([]string, 0, len(w.Mods))
			for _, m := range w.Mods {
				wantNames = append(wantNames, m.Name)
			}
			gotNames := make([]string, 0, len(item.Mods))
			for _, m := range item.Mods {
				gotNames = append(gotNames, m.Name())
			}

			testza.AssertEqual(t, slotMultiplierNames(wantNames), slotMultiplierNames(gotNames),
				"socket/quality multiplier mods for %q", c.Name)
		})
	}
}

// TestQualityMultiplierValueMatchesItemQuality pins the value of the quality multiplier.
func TestQualityMultiplierValueMatchesItemQuality(t *testing.T) {
	cases, expected := loadItemModListOracle(t)

	checked := 0
	for idx, c := range cases {
		w := expected[idx]
		if !w.HasBase || w.Error != nil {
			continue
		}
		for _, want := range w.Mods {
			if !strings.HasPrefix(want.Name, "Multiplier:QualityOn") || want.Value == nil {
				continue
			}
			item := ParseItem(c.Raw)
			if itemHasUnparsedLine(item) {
				continue
			}
			var got *float64
			for _, m := range item.Mods {
				if m.Name() == want.Name {
					v := m.Value().Float()
					got = &v
				}
			}
			testza.AssertNotNil(t, got, "%s: %s not emitted", c.Name, want.Name)
			if got != nil {
				testza.AssertEqual(t, *want.Value, *got, "%s: %s value", c.Name, want.Name)
				checked++
			}
		}
	}
	testza.AssertNotEqual(t, 0, checked, "oracle carried no quality multiplier to check against")
}

func tagStrings(tag mod.Tag) []string {
	v := reflect.ValueOf(tag)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return nil
	}
	var out []string
	s := v.Elem()
	for i := 0; i < s.NumField(); i++ {
		if s.Type().Field(i).PkgPath != "" {
			continue
		}
		f := s.Field(i)
		switch f.Kind() {
		case reflect.String:
			out = append(out, f.String())
		case reflect.Pointer:
			if !f.IsNil() && f.Elem().Kind() == reflect.String {
				out = append(out, f.Elem().String())
			}
		case reflect.Slice:
			if f.Type().Elem().Kind() == reflect.String {
				for j := 0; j < f.Len(); j++ {
					out = append(out, f.Index(j).String())
				}
			}
		}
	}
	return out
}

// TestNoSlotPlaceholderSurvivesBuildModList is the generalised half of the sweep.
func TestNoSlotPlaceholderSurvivesBuildModList(t *testing.T) {
	cases, expected := loadItemModListOracle(t)
	placeholders := []string{"{SlotName}", "{Hand}", "{OtherSlotNum}"}

	for idx, c := range cases {
		w := expected[idx]
		if !w.HasBase || w.Error != nil {
			continue
		}
		item := ParseItem(c.Raw)
		lists := [][]mod.Mod{item.Mods, item.slotModList2, item.BuffModList}
		for _, list := range lists {
			for _, m := range list {
				for _, tag := range m.Tags() {
					for _, s := range tagStrings(tag) {
						for _, p := range placeholders {
							testza.AssertFalse(t, strings.Contains(s, p),
								"%s: mod %s keeps literal %s in a %s tag (%q)", c.Name, m.Name(), p, tag.Type(), s)
						}
					}
				}
			}
		}
	}
}

// TestSlotPlaceholderResolvesPerSlotNum pins the two slot-dependent placeholders.
func TestSlotPlaceholderResolvesPerSlotNum(t *testing.T) {
	const shaperRing = `
Rarity: RARE
Test Ring
Two-Stone Ring
Quality: 0
5% increased Attack Speed if your Other Ring is a Shaper Item
`

	item := ParseItem(shaperRing)
	testza.AssertEqual(t, "Ring", item.Type)

	conditionVars := func(list []mod.Mod) []string {
		var out []string
		for _, m := range list {
			for _, tag := range m.Tags() {
				if c, ok := tag.(*mod.ConditionTag); ok {
					out = append(out, c.VarList...)
				}
			}
		}
		return out
	}

	testza.AssertContains(t, conditionVars(item.Mods), "ShaperItemInRing 2",
		"Ring 1's mods must look at Ring 2")
	testza.AssertContains(t, conditionVars(item.slotModList2), "ShaperItemInRing 1",
		"Ring 2's mods must look at Ring 1")
}
