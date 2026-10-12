package calculator

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"
)

const itemBasesExpected = "../testdata/oracle/item_bases_expected.json"

type reqTable map[string]int

func (r *reqTable) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '[' {
		*r = reqTable{}
		return nil
	}

	m := map[string]int{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err //nolint:wrapcheck
	}
	*r = m

	return nil
}

type expectedBase struct {
	Name        string             `json:"name"`
	Missing     bool               `json:"missing"`
	Type        string             `json:"type"`
	SubType     string             `json:"subType"`
	SocketLimit int                `json:"socketLimit"`
	Req         reqTable           `json:"req"`
	Armour      map[string]float64 `json:"armour"`
	Weapon      map[string]float64 `json:"weapon"`
	Flask       map[string]float64 `json:"flask"`
	FlaskLife   *int               `json:"flaskLife"`
	FlaskMana   *int               `json:"flaskMana"`
	Tags        []string           `json:"tags"`
	Influence   json.RawMessage    `json:"influenceTags"`
}

func TestItemBasesAgainstPoB(t *testing.T) {
	var expected []expectedBase
	raw, err := os.ReadFile(itemBasesExpected)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &expected))
	testza.AssertGreater(t, len(expected), 0)

	for _, want := range expected {
		testza.AssertFalse(t, want.Missing, "oracle had no base for %q", want.Name)

		t.Run(want.Name, func(t *testing.T) {
			got := ItemBaseByName(want.Name)
			testza.AssertNotNil(t, got, "no item base assembled for %q", want.Name)
			if got == nil {
				return
			}

			testza.AssertEqual(t, want.Type, got.Type, "type")
			testza.AssertEqual(t, want.SocketLimit, got.SocketLimit, "socketLimit")

			testza.AssertEqual(t, want.Req["str"], got.Req.Str, "req.str")
			testza.AssertEqual(t, want.Req["dex"], got.Req.Dex, "req.dex")
			testza.AssertEqual(t, want.Req["int"], got.Req.Int, "req.int")
			testza.AssertEqual(t, want.Req["level"], got.Req.Level, "req.level")

			testza.AssertEqual(t, want.Armour != nil, got.Armour != nil, "has armour data")
			testza.AssertEqual(t, want.Flask != nil, got.Flask != nil, "has flask data")
			testza.AssertEqual(t, want.Weapon != nil, got.Weapon != nil, "has weapon data")

			if want.Weapon != nil && got.Weapon != nil {
				testza.AssertEqual(t, int(want.Weapon["PhysicalMin"]), got.Weapon.PhysicalMin, "weapon PhysicalMin")
				testza.AssertEqual(t, int(want.Weapon["PhysicalMax"]), got.Weapon.PhysicalMax, "weapon PhysicalMax")
				testza.AssertEqual(t, want.Weapon["CritChanceBase"], got.Weapon.CritChanceBase, "weapon CritChanceBase")
				testza.AssertEqual(t, want.Weapon["AttackRateBase"], got.Weapon.AttackRateBase, "weapon AttackRateBase")
				testza.AssertEqual(t, int(want.Weapon["Range"]), got.Weapon.Range, "weapon Range")
			}

			if want.Flask != nil && got.Flask != nil {
				testza.AssertEqual(t, want.Flask["duration"], got.Flask.Duration, "flask duration")
				testza.AssertEqual(t, int(want.Flask["chargesMax"]), got.Flask.ChargesMax, "flask chargesMax")
				testza.AssertEqual(t, int(want.Flask["chargesUsed"]), got.Flask.ChargesUsed, "flask chargesUsed")
				wantLife := 0
				if want.FlaskLife != nil {
					wantLife = *want.FlaskLife
				}
				wantMana := 0
				if want.FlaskMana != nil {
					wantMana = *want.FlaskMana
				}
				testza.AssertEqual(t, wantLife, got.Flask.Life, "flask life")
				testza.AssertEqual(t, wantMana, got.Flask.Mana, "flask mana")
			}
		})
	}
}
