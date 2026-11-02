package calculator2

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// itemRangeRow is one row of testdata/item_range_pob.jsonl, which testdata/item_range_oracle.lua writes from
// PoB-PoE2's item editor: an item opened, a range slider moved or the quality set, and the item after it.
type itemRangeRow struct {
	Case         string   `json:"case"`
	Step         string   `json:"step"`
	Raw          string   `json:"raw"`
	Quality      *float64 `json:"quality"`
	QualityShown bool     `json:"qualityShown"`
	RangeLines   []struct {
		Line  string  `json:"line"`
		Range float64 `json:"range"`
	} `json:"rangeLines"`
}

// TestItemRangeLinesMatchPoB replays PoB-PoE2's modifier range sliders and quality box over pasted items and
// uniques: each step's item text, range lines and quality equal PoB-PoE2's.
func TestItemRangeLinesMatchPoB(t *testing.T) {
	pasteText, err := os.ReadFile(filepath.Join("testdata", "item_affix_pastes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	pasted := strings.Split(strings.TrimSuffix(string(pasteText), "\n"), "\n=====\n")
	f, err := os.Open(filepath.Join("testdata", "item_range_pob.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tab := &ItemsTab{modSource: ItemModSource(WithParser(ParseMod))}
	steps := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var r itemRangeRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		parts := strings.Fields(r.Step)
		switch parts[0] {
		case "open":
			raw := ""
			if n, ok := strings.CutPrefix(r.Case, "paste "); ok {
				k, _ := strconv.Atoi(n)
				raw = pasted[k-1]
			} else if u := ItemDB(ItemDBUnique)[r.Case]; u != nil {
				raw = u.Raw
			} else {
				t.Fatalf("%s is not in the unique database", r.Case)
			}
			item := ParseItem(raw, tab.modSource)
			item.NormaliseQuality()
			tab.SetDisplayItem(item)
		case "range":
			line, _ := strconv.Atoi(parts[1])
			val, _ := strconv.ParseFloat(parts[2], 64)
			if err := tab.SetDisplayItemRangeLine(line, val); err != nil {
				t.Fatalf("%s %s: %v", r.Case, r.Step, err)
			}
		case "quality":
			q, _ := strconv.Atoi(parts[1])
			if err := tab.SetDisplayItemQuality(q); err != nil {
				t.Fatalf("%s %s: %v", r.Case, r.Step, err)
			}
		}
		steps++
		if got := tab.DisplayItem.BuildRaw(); got != r.Raw {
			t.Errorf("%s %s: item text %s", r.Case, r.Step, firstDiffLine(got, r.Raw))
		}
		lines := tab.DisplayItemRangeLines()
		if len(lines) != len(r.RangeLines) {
			t.Errorf("%s %s: %d range lines, PoB-PoE2's %d", r.Case, r.Step, len(lines), len(r.RangeLines))
		} else {
			for i, l := range lines {
				if want := r.RangeLines[i]; tab.DisplayItem.RangeLines[i].Line != want.Line || l.Range != want.Range {
					t.Errorf("%s %s: range line %d %q %v, PoB-PoE2's %q %v", r.Case, r.Step, i+1, tab.DisplayItem.RangeLines[i].Line, l.Range, want.Line, want.Range)
				}
			}
		}
		if tab.DisplayItemQualityShown() != r.QualityShown {
			t.Errorf("%s %s: quality shown %v, PoB-PoE2's %v", r.Case, r.Step, tab.DisplayItemQualityShown(), r.QualityShown)
		}
	}
	if steps == 0 {
		t.Fatal("no steps")
	}
	t.Logf("%d steps", steps)
}
