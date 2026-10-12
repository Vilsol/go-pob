package calculator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

type pinModLine struct {
	I         int      `json:"i"`
	WalkIndex int      `json:"walkIndex"`
	Line      string   `json:"line"`
	Range     *float64 `json:"range"`
	IsExtra   bool     `json:"isExtra"`
	ModCount  int      `json:"modCount"`
	Crafted   bool     `json:"crafted"`
	Implicit  bool     `json:"implicit"`
}

type pinItem struct {
	ID     int                     `json:"id"`
	Name   string                  `json:"name"`
	Title  string                  `json:"title"`
	Rarity string                  `json:"rarity"`
	Base   string                  `json:"base"`
	Lists  map[string][]pinModLine `json:"lists"`
}

type pinBuild struct {
	Build string    `json:"build"`
	Items []pinItem `json:"items"`
	Error string    `json:"error"`
}

var walkListOrder = []string{
	"buffModLines",
	"enchantModLines",
	"scourgeModLines",
	"implicitModLines",
	"explicitModLines",
	"crucibleModLines",
}

func oursLists(i *Item) map[string][]ModLine {
	return map[string][]ModLine{
		"buffModLines":     nil,
		"enchantModLines":  i.EnchantModLines,
		"scourgeModLines":  i.ScourgeModLines,
		"implicitModLines": i.ImplicitModLines,
		"explicitModLines": i.ExplicitModLines,
		"crucibleModLines": nil,
	}
}

var rolledRangeRE = regexp.MustCompile(`\(-?\d+\.?\d*--?\d+\.?\d*\)`)

type flatLine struct {
	List  string
	I     int
	Line  string
	Range *float64
}

func flattenPin(it pinItem) []flatLine {
	out := make([]flatLine, 0, 16)
	for _, name := range walkListOrder {
		for _, ml := range it.Lists[name] {
			out = append(out, flatLine{List: name, I: ml.I, Line: ml.Line, Range: ml.Range})
		}
	}
	return out
}

const buffPlaceholder = "\x00buffModLine"

func flattenOurs(i *Item) []flatLine {
	lists := oursLists(i)
	out := make([]flatLine, 0, 16)
	for n := 0; n < i.buffModLineCount(); n++ {
		out = append(out, flatLine{List: "buffModLines", I: n + 1, Line: buffPlaceholder})
	}
	for _, name := range walkListOrder {
		for idx, ml := range lists[name] {
			out = append(out, flatLine{List: name, I: idx + 1, Line: ml.Line, Range: ml.Range})
		}
	}
	return out
}

func rangeStr(r *float64) string {
	if r == nil {
		return "nil"
	}
	return fmt.Sprintf("%g", *r)
}

func TestItemModLineOrderDiff(t *testing.T) {
	pinDir := os.Getenv("PIN")
	if pinDir == "" {
		t.Skip("set PIN=<dir> to the output of oracle/item_modline_order.lua")
	}
	entries, err := os.ReadDir(pinDir)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		t.Fatalf("REFUSING: no .json pin dumps under %s -- an instrument that cannot answer must refuse, never return empty", pinDir)
	}
	sort.Strings(files)

	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	var (
		buildsSeen, buildsErr           int
		itemsSeen, itemsOrderDiff       int
		itemsListDiff, itemsSetDiff     int
		itemsRangeDiff, itemsWithRanges int
		rangePairs, rangePairsDiff      int
		modRangeWouldMove               int
		sample                          []string
		causeBuffLines                  int
		causeLineCount                  int
		causePermuted                   int
		causeJoin                       int
		itemsIdentical                  int

		linesShared, linesWrongToday  int
		linesRanged, linesRangedWrong int
	)
	detail := 0
	if v := os.Getenv("DETAIL"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &detail)
	}
	detailCause := os.Getenv("DETAILCAUSE")

	for _, f := range files {
		blob, err := os.ReadFile(filepath.Join(pinDir, f))
		if err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		var pb pinBuild
		if err := json.Unmarshal(blob, &pb); err != nil {
			t.Fatalf("REFUSING: %s: %v", f, err)
		}
		if pb.Error != "" {
			buildsErr++
			continue
		}
		buildsSeen++

		xmlName := strings.TrimSuffix(f, ".json") + ".xml"
		xmlBytes, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), xmlName))
		if err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		build, err := builds.ParseBuild(xmlBytes)
		if err != nil {
			t.Fatalf("REFUSING: %s: %v", xmlName, err)
		}
		byID := map[int]string{}
		for _, it := range build.Items.Items {
			byID[it.ID] = it.Text
		}

		for _, pi := range pb.Items {
			text, ok := byID[pi.ID]
			if !ok {
				continue
			}
			ours := ParseItem(text)
			itemsSeen++

			pinFlat := flattenPin(pi)
			ourFlat := flattenOurs(ours)

			pinPos := map[string]int{}
			for idx, fl := range pinFlat {
				if _, dup := pinPos[fl.Line]; !dup {
					pinPos[fl.Line] = idx
				}
			}
			ourPos := map[string]int{}
			for idx, fl := range ourFlat {
				if fl.Line == buffPlaceholder {
					continue
				}
				if _, dup := ourPos[fl.Line]; !dup {
					ourPos[fl.Line] = idx
				}
			}

			setDiff := false
			for line := range pinPos {
				if _, ok := ourPos[line]; !ok {
					setDiff = true
				}
			}
			for line := range ourPos {
				if _, ok := pinPos[line]; !ok {
					setDiff = true
				}
			}
			if setDiff {
				itemsSetDiff++
			}

			orderDiff, listDiff := false, false
			for line, pIdx := range pinPos {
				oIdx, ok := ourPos[line]
				if !ok {
					continue
				}
				if pinFlat[pIdx].I != ourFlat[oIdx].I {
					orderDiff = true
				}
				if pinFlat[pIdx].List != ourFlat[oIdx].List {
					listDiff = true
				}
			}
			if orderDiff {
				itemsOrderDiff++
			}
			if listDiff {
				itemsListDiff++
			}

			ids := parseModRangeElements(text)
			if len(ids) > 0 {
				itemsWithRanges++
			}

			moved := false
			for _, mr := range ids {
				id := mr.ID
				if id < 1 {
					continue
				}
				rangePairs++
				var pinLine, ourLine string
				if id <= len(pinFlat) {
					pinLine = pinFlat[id-1].Line
				}
				if id <= len(ourFlat) {
					ourLine = ourFlat[id-1].Line
				}
				if pinLine != ourLine {
					rangePairsDiff++
					moved = true
				}
			}
			if len(pi.Lists["buffModLines"]) > 0 {
				causeBuffLines++
			}
			if len(pinFlat) != len(ourFlat) {
				causeLineCount++
			} else if !setDiff && orderDiff {
				causePermuted++
			}
			if len(pinFlat) == len(ourFlat) && !setDiff && !orderDiff && !listDiff {
				itemsIdentical++
			}

			if moved {
				modRangeWouldMove++
				if len(sample) < 40 {
					sample = append(sample, fmt.Sprintf("%s item %d %q (%s)", pb.Build, pi.ID, pi.Title, pi.Rarity))
				}
				if detail > 0 && (detailCause == "" ||
					(detailCause == "buff" && len(pi.Lists["buffModLines"]) > 0) ||
					(detailCause == "count" && len(pinFlat) != len(ourFlat) && len(pi.Lists["buffModLines"]) == 0) ||
					(detailCause == "permute" && len(pinFlat) == len(ourFlat) && !setDiff && orderDiff)) {
					detail--
					t.Logf("---- %s item %d %q (%s) advancedCopy=%v crafted=%v", pb.Build, pi.ID, pi.Title, pi.Rarity, ours.AdvancedCopy, ours.Crafted)
					n := len(pinFlat)
					if len(ourFlat) > n {
						n = len(ourFlat)
					}
					for k := 0; k < n; k++ {
						var p, o string
						if k < len(pinFlat) {
							p = fmt.Sprintf("%s[%d] r=%s %s", pinFlat[k].List, pinFlat[k].I, rangeStr(pinFlat[k].Range), pinFlat[k].Line)
						}
						if k < len(ourFlat) {
							o = fmt.Sprintf("%s[%d] r=%s %s", ourFlat[k].List, ourFlat[k].I, rangeStr(ourFlat[k].Range), ourFlat[k].Line)
						}
						mark := "  "
						if p != o {
							mark = "!!"
						}
						t.Logf("   %s %-2d PIN %-70s | OURS %s", mark, k+1, p, o)
					}
				}
			}

			rangeDiffHere := false
			for line, pIdx := range pinPos {
				oIdx, ok := ourPos[line]
				if !ok {
					continue
				}
				linesShared++
				want := rangeStr(pinFlat[pIdx].Range)
				if want != rangeStr(ourFlat[oIdx].Range) {
					linesWrongToday++
					rangeDiffHere = true
				}
				if rolledRangeRE.MatchString(pinFlat[pIdx].Line) {
					linesRanged++
					if want != rangeStr(ourFlat[oIdx].Range) {
						linesRangedWrong++
					}
				}
			}
			if rangeDiffHere {
				itemsRangeDiff++
			}

			if len(ourFlat) > len(pinFlat) {
				for _, fl := range pinFlat {
					if strings.Contains(fl.Line, "\n") {
						causeJoin++
						break
					}
				}
			}
		}
	}

	t.Logf("builds %d graded, %d errored", buildsSeen, buildsErr)
	t.Logf("items %d compared", itemsSeen)
	t.Logf("  line-SET differs (a line one side has and the other does not): %d", itemsSetDiff)
	t.Logf("  LIST differs (same line, different mod-line list):             %d", itemsListDiff)
	t.Logf("  ORDER differs (same line, different index in its list):        %d", itemsOrderDiff)
	t.Logf("  RANGE differs on a shared line:                                %d", itemsRangeDiff)
	t.Logf("<ModRange> walk:")
	t.Logf("  items carrying at least one <ModRange>:                        %d", itemsWithRanges)
	t.Logf("  <ModRange> ids seen:                                           %d", rangePairs)
	t.Logf("  ids addressing a DIFFERENT line under our order:               %d", rangePairsDiff)
	t.Logf("  items where at least one id moves:                             %d", modRangeWouldMove)
	t.Logf("CAUSE BUCKETS (independent, an item can be in several):")
	t.Logf("  pin has buffModLines (we hold them as a count, not lines):     %d", causeBuffLines)
	t.Logf("  the two sides disagree on the mod-line COUNT:                  %d", causeLineCount)
	t.Logf("  same set and count, PERMUTED order:                            %d", causePermuted)
	t.Logf("  byte-identical walk (list, index and line all agree):          %d", itemsIdentical)
	t.Logf("  the pin JOINED a wrapped mod line and we did not:              %d", causeJoin)
	t.Logf("THE DECISION NUMBERS -- per mod line both sides have (%d):", linesShared)
	t.Logf("  range wrong (ours vs the pin, post-<ModRange>-walk):           %d", linesWrongToday)
	t.Logf("  of those lines, ones carrying an actual (min-max) roll:        %d", linesRanged)
	t.Logf("  RANGED lines whose range is wrong (the only ones that move a value): %d", linesRangedWrong)
	for _, s := range sample {
		t.Logf("    %s", s)
	}
}
