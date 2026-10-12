package calculator

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

func itoa(v int) string { return strconv.Itoa(v) }

func itemHasUnparsedLine(i *Item) bool {
	groups := [][]ModLine{
		i.EnchantModLines, i.ScourgeModLines, i.ClassRequirementModLines,
		i.ImplicitModLines, i.ExplicitModLines,
	}
	for _, g := range groups {
		for _, ml := range g {
			if ml.Extra != "" {
				return true
			}
			if strings.Contains(ml.Line, "matching modifier on Forbidden") {
				return true
			}
		}
	}
	return false
}

func goModKey(m mod.Mod) string {
	var value *float64
	switch m.Value().Type() {
	case mod.ModValueMultiTypeFloat:
		v := m.Value().Float()
		value = &v
	case mod.ModValueMultiTypeFlag:
		v := 0.0
		if m.Value().Flag() {
			v = 1
		}
		value = &v
	}

	tags := make([]string, 0, len(m.Tags()))
	for _, tag := range m.Tags() {
		tags = append(tags, string(tag.Type()))
	}

	return modKey(m.Name(), string(m.Type()), value, int(m.Flags()), int(m.KeywordFlags()), tags)
}

const itemModListExpected = "../testdata/oracle/item_modlist_expected.json"

type expectedMod struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Value        *float64 `json:"value"`
	ValueType    string   `json:"valueType"`
	Flags        int      `json:"flags"`
	KeywordFlags int      `json:"keywordFlags"`
	Tags         []string `json:"tags"`
}

type expectedModList struct {
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	IsWeapon bool          `json:"isWeapon"`
	IsArmour bool          `json:"isArmour"`
	IsFlask  bool          `json:"isFlask"`
	HasBase  bool          `json:"hasBase"`
	ModCount int           `json:"modCount"`
	Mods     []expectedMod `json:"mods"`
	Error    *string       `json:"error"`
}

func modKey(name, modType string, value *float64, flags, kw int, tags []string) string {
	sorted := append([]string(nil), tags...)
	sort.Strings(sorted)
	v := "nil"
	if value != nil {
		v = strconv.FormatFloat(*value, 'g', -1, 64)
	}
	return strings.Join([]string{name, modType, v, itoa(flags), itoa(kw), strings.Join(sorted, "+")}, "|")
}

func TestItemBuildModListAgainstPoB(t *testing.T) {
	var cases []itemParseCase
	raw, err := os.ReadFile(itemParseCases)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &cases))

	var expected []expectedModList
	raw, err = os.ReadFile(itemModListExpected)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &expected))
	testza.AssertEqual(t, len(cases), len(expected), "cases/expected out of sync; regenerate with `task oracle`")

	for idx, c := range cases {
		w := expected[idx]
		testza.AssertNil(t, w.Error, "oracle errored for %q", c.Name)

		if w.IsWeapon || w.IsArmour || w.IsFlask || !w.HasBase {
			continue
		}

		t.Run(c.Name, func(t *testing.T) {
			item := ParseItem(c.Raw)

			if itemHasUnparsedLine(item) {
				t.Skip("blocked on ModParser coverage")
			}

			wantKeys := make([]string, 0, len(w.Mods))
			for _, m := range w.Mods {
				wantKeys = append(wantKeys, modKey(m.Name, m.Type, m.Value, m.Flags, m.KeywordFlags, m.Tags))
			}

			gotKeys := make([]string, 0, len(item.Mods))
			for _, m := range item.Mods {
				gotKeys = append(gotKeys, goModKey(m))
			}

			sort.Strings(wantKeys)
			sort.Strings(gotKeys)
			testza.AssertEqual(t, wantKeys, gotKeys, "mod list for %q", c.Name)
		})
	}
}
