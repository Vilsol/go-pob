package calculator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

const blockChanceAnchorCase = "2.xml#10 Aegis Aurora"

func TestShieldBlockChanceAppliesIncreaseAndFloor(t *testing.T) {
	var cases []itemParseCase
	raw, err := os.ReadFile(itemParseCases)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &cases))

	var expected []expectedItemData
	raw, err = os.ReadFile(itemDataExpected)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &expected))

	idx := -1
	for i, c := range cases {
		if c.Name == blockChanceAnchorCase {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatalf("anchor fixture %q is gone; this test can no longer exercise the mechanism", blockChanceAnchorCase)
	}

	anchor, ok := (expected[idx].ArmourData)["BlockChance"]
	if !ok || anchor <= 0 {
		t.Fatalf("PoB's dump has no positive BlockChance for %q; anchor is invalid", blockChanceAnchorCase)
	}

	base := ParseItem(cases[idx].Raw)
	testza.AssertEqual(t, anchor, base.ArmourData["BlockChance"],
		"unmodified BlockChance already disagrees with PoB; fix that before trusting the cases below")

	for _, inc := range []int{5, 51, 75} {
		t.Run(fmt.Sprintf("inc=%d%%", inc), func(t *testing.T) {
			scaled := anchor * (1 + float64(inc)/100)
			want := math.Floor(scaled)

			if want == anchor {
				t.Fatalf("inc=%d%% does not move BlockChance off %v; case is vacuous", inc, anchor)
			}

			line := fmt.Sprintf("%d%% increased Block Chance", inc)

			e := ParseMod(line, false)
			if strings.TrimSpace(e.Extra) != "" {
				t.Fatalf("%q no longer parses cleanly (Extra=%q); this case would be testing a "+
					"dropped mod rather than the increase/floor", line, e.Extra)
			}
			if len(e.ModList) != 1 || e.ModList[0].Name() != "BlockChance" || e.ModList[0].Type() != mod.TypeIncrease {
				t.Fatalf("%q no longer yields a single BlockChance INC mod; case is invalid", line)
			}

			raw := injectMod(t, cases[idx].Raw, line)
			got := ParseItem(raw).ArmourData["BlockChance"]

			if got != want {
				t.Errorf("BlockChance = %v, want %v\n"+
					"  PoB: m_floor((base+BASE) * (1+INC/100)) = floor(%v * %.2f) = floor(%v) = %v\n"+
					"  (got == %v would mean the increase is dropped; got == %v would mean floor became round)",
					got, want, anchor, 1+float64(inc)/100, scaled, want, anchor, math.Round(scaled))
			}
		})
	}
}

func injectMod(t *testing.T, raw, line string) string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "Implicits:") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("item text has no `Implicits:` line; cannot place %q reliably", line)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if lines[i] == "Corrupted" {
			end = i
			break
		}
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:end]...)
	out = append(out, line)
	out = append(out, lines[end:]...)
	return strings.Join(out, "\n")
}
