package calculator

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"testing"

	"github.com/MarvinJWendt/testza"
)

const itemDataExpected = "../testdata/oracle/item_data_expected.json"

type dataMap map[string]float64

func (d *dataMap) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '[' {
		*d = dataMap{}
		return nil
	}
	m := map[string]float64{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err //nolint:wrapcheck
	}
	*d = m
	return nil
}

type expectedItemData struct {
	Name       string  `json:"name"`
	IsWeapon   bool    `json:"isWeapon"`
	IsArmour   bool    `json:"isArmour"`
	IsFlask    bool    `json:"isFlask"`
	WeaponData dataMap `json:"weaponData"`
	ArmourData dataMap `json:"armourData"`
	FlaskData  dataMap `json:"flaskData"`
	Error      *string `json:"error"`
}

func assertDataEqual(t *testing.T, kind string, want dataMap, got map[string]float64) {
	t.Helper()

	keys := make([]string, 0, len(want))
	for k := range want {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		g, ok := got[k]
		testza.AssertTrue(t, ok, "%s missing key %q", kind, k)
		if !ok {
			continue
		}
		if math.Abs(want[k]-g) > 1e-6 {
			testza.AssertEqual(t, want[k], g, "%s[%q]", kind, k)
		}
	}
}

func TestItemDataAgainstPoB(t *testing.T) {
	var cases []itemParseCase
	raw, err := os.ReadFile(itemParseCases)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &cases))

	var expected []expectedItemData
	raw, err = os.ReadFile(itemDataExpected)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &expected))
	testza.AssertEqual(t, len(cases), len(expected), "cases/expected out of sync; regenerate with `task oracle`")

	for idx, c := range cases {
		w := expected[idx]
		if w.Error != nil || !(w.IsWeapon || w.IsArmour || w.IsFlask) {
			continue
		}

		t.Run(c.Name, func(t *testing.T) {
			item := ParseItem(c.Raw)

			switch {
			case w.IsWeapon:
				got := map[string]float64{}
				if item.WeaponData != nil {
					got = item.WeaponData[1]
				}
				assertDataEqual(t, "weaponData", w.WeaponData, got)
			case w.IsArmour:
				assertDataEqual(t, "armourData", w.ArmourData, item.ArmourData)
			case w.IsFlask:
				assertDataEqual(t, "flaskData", w.FlaskData, item.FlaskData)
			}
		})
	}
}
