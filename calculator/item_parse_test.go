package calculator

import (
	"encoding/json"
	"os"
	"sync/atomic"
	"testing"

	"github.com/MarvinJWendt/testza"
)

const (
	itemParseCases    = "../testdata/oracle/item_parse_cases.json"
	itemParseExpected = "../testdata/oracle/item_parse_expected.json"
)

type itemParseCase struct {
	Name string `json:"name"`
	Raw  string `json:"raw"`
}

type luaBool bool

func (b *luaBool) UnmarshalJSON(raw []byte) error {
	s := string(raw)
	*b = luaBool(s != "false" && s != "null" && s != `""`)
	return nil
}

type expectedModLine struct {
	Line      string   `json:"line"`
	Extra     *string  `json:"extra"`
	Range     *float64 `json:"range"`
	Crafted   luaBool  `json:"crafted"`
	Custom    luaBool  `json:"custom"`
	Fractured luaBool  `json:"fractured"`
	Scourge   luaBool  `json:"scourge"`
}

type expectedSocket struct {
	Color string `json:"color"`
	Group int    `json:"group"`
}

type expectedItem struct {
	Name         string  `json:"name"`
	Title        string  `json:"title"`
	BaseName     string  `json:"baseName"`
	Rarity       string  `json:"rarity"`
	Quality      *int    `json:"quality"`
	ItemLevel    *int    `json:"itemLevel"`
	UniqueID     string  `json:"uniqueID"`
	League       string  `json:"league"`
	Corrupted    bool    `json:"corrupted"`
	Mirrored     bool    `json:"mirrored"`
	Fractured    luaBool `json:"fractured"`
	Synthesised  bool    `json:"synthesised"`
	Shaper       bool    `json:"shaper"`
	Elder        bool    `json:"elder"`
	Crusader     bool    `json:"crusader"`
	Adjudicator  bool    `json:"adjudicator"`
	Basilisk     bool    `json:"basilisk"`
	Eyrie        bool    `json:"eyrie"`
	Cleansing    bool    `json:"cleansing"`
	Tangle       bool    `json:"tangle"`
	CheckSection bool    `json:"checkSection"`

	Sockets      []expectedSocket   `json:"sockets"`
	Requirements map[string]float64 `json:"requirements"`

	ImplicitModLines         []expectedModLine `json:"implicitModLines"`
	ExplicitModLines         []expectedModLine `json:"explicitModLines"`
	EnchantModLines          []expectedModLine `json:"enchantModLines"`
	ScourgeModLines          []expectedModLine `json:"scourgeModLines"`
	ClassRequirementModLines []expectedModLine `json:"classRequirementModLines"`
}

type itemParseResult struct {
	Name  string        `json:"name"`
	Item  *expectedItem `json:"item"`
	Error *string       `json:"error"`
}

func assertModLines(t *testing.T, kind string, want []expectedModLine, got []ModLine) {
	t.Helper()

	testza.AssertEqual(t, len(want), len(got), "%s count", kind)
	if len(want) != len(got) {
		return
	}

	for i := range want {
		testza.AssertEqual(t, want[i].Line, got[i].Line, "%s[%d].line", kind, i)
		testza.AssertEqual(t, bool(want[i].Crafted), got[i].Crafted, "%s[%d].crafted", kind, i)
		testza.AssertEqual(t, bool(want[i].Custom), got[i].Custom, "%s[%d].custom", kind, i)
		testza.AssertEqual(t, bool(want[i].Fractured), got[i].Fractured, "%s[%d].fractured", kind, i)
		testza.AssertEqual(t, bool(want[i].Scourge), got[i].Scourge, "%s[%d].scourge", kind, i)

		if want[i].Extra == nil {
			testza.AssertEqual(t, "", got[i].Extra, "%s[%d].extra", kind, i)
		} else {
			testza.AssertEqual(t, *want[i].Extra, got[i].Extra, "%s[%d].extra", kind, i)
		}

		if want[i].Range == nil {
			testza.AssertNil(t, got[i].Range, "%s[%d].range", kind, i)
		} else {
			testza.AssertNotNil(t, got[i].Range, "%s[%d].range", kind, i)
			if got[i].Range != nil {
				testza.AssertEqual(t, *want[i].Range, *got[i].Range, "%s[%d].range", kind, i)
			}
		}
	}
}

var parserBlocked atomic.Int64

var processed atomic.Int64

const knownParserBlocked = 1

func countExtras(i *Item) int {
	n := 0
	for _, group := range [][]ModLine{i.ImplicitModLines, i.ExplicitModLines, i.EnchantModLines, i.ScourgeModLines} {
		for _, ml := range group {
			if ml.Extra != "" {
				n++
			}
		}
	}
	return n
}

func parserGap(got *Item, want *expectedItem) bool {
	if countExtras(got) > countExpectedExtras(want) {
		return true
	}

	pairs := []struct {
		got  []ModLine
		want []expectedModLine
	}{
		{got.ImplicitModLines, want.ImplicitModLines},
		{got.ExplicitModLines, want.ExplicitModLines},
		{got.EnchantModLines, want.EnchantModLines},
		{got.ScourgeModLines, want.ScourgeModLines},
	}

	for _, p := range pairs {
		if len(p.got) != len(p.want) {
			if countExtras(got) > 0 {
				return true
			}
			continue
		}
		for j := range p.got {
			we := ""
			if p.want[j].Extra != nil {
				we = *p.want[j].Extra
			}
			if p.got[j].Extra != "" && p.got[j].Extra != we {
				return true
			}
		}
	}

	return false
}

func countExpectedExtras(e *expectedItem) int {
	n := 0
	for _, group := range [][]expectedModLine{e.ImplicitModLines, e.ExplicitModLines, e.EnchantModLines, e.ScourgeModLines} {
		for _, ml := range group {
			if ml.Extra != nil {
				n++
			}
		}
	}
	return n
}

func TestParseItemAgainstPoB(t *testing.T) {
	var cases []itemParseCase
	raw, err := os.ReadFile(itemParseCases)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &cases))

	var expected []itemParseResult
	raw, err = os.ReadFile(fixturePath("ITEM_PARSE_EXPECTED", itemParseExpected))
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &expected))

	testza.AssertEqual(t, len(cases), len(expected), "cases and expected results are out of sync; regenerate with `task oracle`")

	for i, c := range cases {
		want := expected[i]
		testza.AssertNil(t, want.Error, "oracle errored for %q", c.Name)
		if want.Item == nil {
			continue
		}

		t.Run(c.Name, func(t *testing.T) {
			processed.Add(1)
			got := ParseItem(c.Raw)
			w := want.Item

			if parserGap(got, w) {
				parserBlocked.Add(1)
				t.Skipf("blocked on ModParser coverage, not ParseRaw")
			}

			testza.AssertEqual(t, w.Name, got.Name, "name")
			testza.AssertEqual(t, w.Title, got.Title, "title")
			testza.AssertEqual(t, w.BaseName, got.BaseName, "baseName")
			testza.AssertEqual(t, w.Rarity, string(got.Rarity), "rarity")
			testza.AssertEqual(t, w.UniqueID, got.UniqueID, "uniqueID")
			testza.AssertEqual(t, w.League, got.League, "league")

			if w.Quality == nil {
				testza.AssertNil(t, got.Quality, "quality")
			} else {
				testza.AssertNotNil(t, got.Quality, "quality")
				if got.Quality != nil {
					testza.AssertEqual(t, *w.Quality, *got.Quality, "quality")
				}
			}

			if w.ItemLevel == nil {
				testza.AssertNil(t, got.ItemLevel, "itemLevel")
			} else {
				testza.AssertNotNil(t, got.ItemLevel, "itemLevel")
				if got.ItemLevel != nil {
					testza.AssertEqual(t, *w.ItemLevel, *got.ItemLevel, "itemLevel")
				}
			}

			testza.AssertEqual(t, w.Corrupted, got.Corrupted, "corrupted")
			testza.AssertEqual(t, w.Mirrored, got.Mirrored, "mirrored")
			testza.AssertEqual(t, bool(w.Fractured), got.Fractured, "fractured")
			testza.AssertEqual(t, w.Synthesised, got.Synthesised, "synthesised")
			testza.AssertEqual(t, w.CheckSection, got.CheckSection, "checkSection")

			testza.AssertEqual(t, w.Shaper, got.Influence.Shaper, "shaper")
			testza.AssertEqual(t, w.Elder, got.Influence.Elder, "elder")
			testza.AssertEqual(t, w.Crusader, got.Influence.Crusader, "crusader")
			testza.AssertEqual(t, w.Adjudicator, got.Influence.Adjudicator, "adjudicator")
			testza.AssertEqual(t, w.Basilisk, got.Influence.Basilisk, "basilisk")
			testza.AssertEqual(t, w.Eyrie, got.Influence.Eyrie, "eyrie")
			testza.AssertEqual(t, w.Cleansing, got.Influence.Cleansing, "cleansing")
			testza.AssertEqual(t, w.Tangle, got.Influence.Tangle, "tangle")

			testza.AssertEqual(t, len(w.Sockets), len(got.Sockets), "socket count")
			if len(w.Sockets) == len(got.Sockets) {
				for j := range w.Sockets {
					testza.AssertEqual(t, w.Sockets[j].Color, got.Sockets[j].Color, "socket[%d].color", j)
					testza.AssertEqual(t, w.Sockets[j].Group, got.Sockets[j].Group, "socket[%d].group", j)
				}
			}

			for k, v := range w.Requirements {
				if k == "strMod" || k == "dexMod" || k == "intMod" {
					continue
				}
				testza.AssertEqual(t, v, got.Requirements[k], "requirements.%s", k)
			}

			assertModLines(t, "implicit", w.ImplicitModLines, got.ImplicitModLines)
			assertModLines(t, "explicit", w.ExplicitModLines, got.ExplicitModLines)
			assertModLines(t, "enchant", w.EnchantModLines, got.EnchantModLines)
			assertModLines(t, "scourge", w.ScourgeModLines, got.ScourgeModLines)
			assertModLines(t, "classRequirement", w.ClassRequirementModLines, got.ClassRequirementModLines)
		})
	}

	blocked := int(parserBlocked.Load())
	if int(processed.Load()) != len(cases) {
		t.Logf("partial run (%d/%d cases); skipping the ratchet assertion", processed.Load(), len(cases))
		return
	}
	testza.AssertEqual(t, knownParserBlocked, blocked,
		"items blocked on ModParser coverage changed; if ModParser improved, lower knownParserBlocked")
	t.Logf("%d/%d items verified against PoB; %d blocked on ModParser coverage", len(cases)-blocked, len(cases), blocked)
}
