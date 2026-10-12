package calculator

import (
	"encoding/json"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
)

type qualityMultiplierCase struct {
	name            string
	localIncPattern *regexp.Regexp
	keys            []string
}

var (
	reQualityLine = regexp.MustCompile(`(?m)^Quality: \+?(\d+)`)

	qualityMultiplierCases = []qualityMultiplierCase{
		{
			name:            "2.xml#10 Aegis Aurora",
			localIncPattern: regexp.MustCompile(`(?m)^(\d+)% increased Armour and Energy Shield$`),
			keys:            []string{"Armour", "EnergyShield"},
		},
		{
			name:            "12.xml#2 Starkonja's Head",
			localIncPattern: regexp.MustCompile(`(?m)^(\d+)% increased Evasion Rating$`),
			keys:            []string{"Evasion"},
		},
		{
			name:            "12.xml#6 Bloodseeker",
			localIncPattern: regexp.MustCompile(`(?m)^(\d+)% increased Physical Damage$`),
			keys:            []string{"PhysicalMin", "PhysicalMax"},
		},
	}
)

func TestItemQualityIsASeparateMultiplier(t *testing.T) {
	var cases []itemParseCase
	raw, err := os.ReadFile(itemParseCases)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &cases))

	var expected []expectedItemData
	raw, err = os.ReadFile(itemDataExpected)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &expected))
	testza.AssertEqual(t, len(cases), len(expected), "cases/expected out of sync; regenerate with `task oracle`")

	byName := map[string]int{}
	for i, c := range cases {
		byName[c.Name] = i
	}

	for _, qc := range qualityMultiplierCases {
		t.Run(qc.name, func(t *testing.T) {
			idx, ok := byName[qc.name]
			if !ok {
				t.Fatalf("fixture %q is gone; this test can no longer exercise the mechanism", qc.name)
			}
			c, w := cases[idx], expected[idx]

			qm := reQualityLine.FindStringSubmatch(c.Raw)
			if qm == nil {
				t.Fatalf("fixture %q has no Quality line; the quality scalar would be 0 and the case vacuous", qc.name)
			}
			quality, err := strconv.Atoi(qm[1])
			testza.AssertNoError(t, err)
			if quality == 0 {
				t.Fatalf("fixture %q has Quality 0; the two formulas are identical and the case vacuous", qc.name)
			}

			localInc := 0
			for _, m := range qc.localIncPattern.FindAllStringSubmatch(c.Raw, -1) {
				v, err := strconv.Atoi(m[1])
				testza.AssertNoError(t, err)
				localInc += v
			}
			if localInc == 0 {
				t.Fatalf("fixture %q no longer carries a local increase matching %s; "+
					"with I=0 the two formulas agree exactly and the case is vacuous", qc.name, qc.localIncPattern)
			}

			summed := 1 + float64(localInc+quality)/100
			separate := (1 + float64(localInc)/100) * (1 + float64(quality)/100)
			if math.Abs(summed-separate) < 1e-9 {
				t.Fatalf("fixture %q: I=%d Q=%d make the formulas agree; case is vacuous", qc.name, localInc, quality)
			}

			item := ParseItem(c.Raw)
			got := map[string]float64{}
			switch {
			case w.IsWeapon:
				if item.WeaponData != nil {
					got = item.WeaponData[1]
				}
			case w.IsArmour:
				got = item.ArmourData
			default:
				t.Fatalf("fixture %q is neither weapon nor armour", qc.name)
			}

			want := w.WeaponData
			if w.IsArmour {
				want = w.ArmourData
			}

			for _, k := range qc.keys {
				wv, ok := want[k]
				if !ok {
					t.Fatalf("PoB's dump has no %q for %q; fixture or oracle changed", k, qc.name)
				}
				if wv == 0 {
					t.Fatalf("PoB's %q is 0 for %q; case cannot discriminate", k, qc.name)
				}
				gv := got[k]
				if math.Abs(wv-gv) > 1e-6 {
					t.Errorf("%s[%q] = %v, want %v (PoB)\n"+
						"  I=%d%% Q=%d%%: summed (1+(I+Q)/100)=%.4f vs separate (1+I/100)(1+Q/100)=%.4f, ratio %.6f",
						strings.ToLower(k[:1])+k[1:], k, gv, wv,
						localInc, quality, summed, separate, summed/separate)
				}
			}
		})
	}
}
