package calculator2

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// itemAffixRow is one row of testdata/item_affix_pob.jsonl.gz, which testdata/item_affix_oracle.lua writes from
// PoB-PoE2's Items tab: a craft or affix step and the editor's affix controls after it, or a pasted item.
type itemAffixRow struct {
	Case       string `json:"case"`
	Step       string `json:"step"`
	Raw        string `json:"raw"`
	AffixLimit int    `json:"affixLimit"`
	Controls   []struct {
		OutputTable string  `json:"outputTable"`
		OutputIndex int     `json:"outputIndex"`
		SelIndex    int     `json:"selIndex"`
		SliderShown bool    `json:"sliderShown"`
		SliderVal   float64 `json:"sliderVal"`
		DivCount    int     `json:"divCount"`
		List        []struct {
			Label     string   `json:"label"`
			ModList   []string `json:"modList"`
			ModID     string   `json:"modId"`
			HaveRange bool     `json:"haveRange"`
		} `json:"list"`
	} `json:"controls"`
}

func readItemAffixRows(t *testing.T) []itemAffixRow {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "item_affix_pob.jsonl.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var rows []itemAffixRow
	sc := bufio.NewScanner(zr)
	sc.Buffer(nil, 4<<20)
	for sc.Scan() {
		var r itemAffixRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, r)
	}
	return rows
}

func affixEntryText(label string, modList []string, modID string, haveRange bool) string {
	return fmt.Sprintf("%q %v %q %v", label, modList, modID, haveRange)
}

// affixControlText lists a control's entries in its own order. PoB-PoE2 sorts them with an unstable sort over
// pairs() order, so entries with equal stat orders and levels come in an order that varies between runs; ordered
// lists the entries sorted instead, and the selection by its entry rather than its index.
func affixControlText(i int, table string, index, sel int, shown bool, val float64, div int, entries []string, ordered bool) string {
	var b strings.Builder
	selected := ""
	if sel > 1 {
		selected = entries[sel-2]
	}
	// a hidden slider keeps the tier count of an earlier selection
	if !shown {
		div = 0
	}
	if ordered {
		entries = slices.Clone(entries)
		sort.Strings(entries)
		sel = 0
	}
	fmt.Fprintf(&b, "%d %s %d sel=%d %s slider=%v %.10g div=%d\n", i, table, index, sel, selected, shown, val, div)
	for _, e := range entries {
		b.WriteString("  " + e + "\n")
	}
	return b.String()
}

func affixControlsText(tab *ItemsTab, ordered bool) string {
	var b strings.Builder
	for i, c := range tab.AffixControls() {
		var entries []string
		for _, e := range c.List[1:] {
			entries = append(entries, affixEntryText(e.Label, e.ModList, e.ModID, e.HaveRange))
		}
		b.WriteString(affixControlText(i+1, c.OutputTable, c.OutputIndex, c.SelIndex, c.SliderShown, c.SliderVal, c.DivCount, entries, ordered))
	}
	return b.String()
}

func (r *itemAffixRow) controlsText(ordered bool) string {
	var b strings.Builder
	for i, c := range r.Controls {
		var entries []string
		for _, e := range c.List[1:] {
			entries = append(entries, affixEntryText(e.Label, e.ModList, e.ModID, e.HaveRange))
		}
		b.WriteString(affixControlText(i+1, c.OutputTable, c.OutputIndex, c.SelIndex, c.SliderShown, c.SliderVal, c.DivCount, entries, ordered))
	}
	return b.String()
}

func firstDiffLine(got, want string) string {
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(g) || i < len(w); i++ {
		var gl, wl string
		if i < len(g) {
			gl = g[i]
		}
		if i < len(w) {
			wl = w[i]
		}
		if gl != wl {
			return fmt.Sprintf("line %d:\n got  %s\n want %s", i+1, gl, wl)
		}
	}
	return ""
}

// TestItemAffixesMatchPoB replays PoB-PoE2's affix drop-downs and roll sliders: every step's item text and every
// control's list, selection and slider equal PoB-PoE2's, and pasted advanced-copy items keep their affixes.
func TestItemAffixesMatchPoB(t *testing.T) {
	rows := readItemAffixRows(t)
	pasteText, err := os.ReadFile(filepath.Join("testdata", "item_affix_pastes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	pasted := strings.Split(strings.TrimSuffix(string(pasteText), "\n"), "\n=====\n")
	tab := &ItemsTab{modSource: ItemModSource(WithParser(ParseMod))}
	steps, pastes, sameOrder := 0, 0, 0
	var prev *itemAffixRow
	for n := range rows {
		r := &rows[n]
		parts := strings.Fields(r.Step)
		switch parts[0] {
		case "craft":
			if strings.HasPrefix(r.Case, "paste") {
				tab.DisplayItem.Craft()
				break
			}
			rarity, _ := strconv.Atoi(r.Case[:1])
			item, ok := CraftItem(rarity, "", r.Case[2:], tab.modSource)
			if !ok {
				t.Fatalf("%s: not crafted", r.Case)
			}
			tab.SetDisplayItem(item)
		case "select":
			control, _ := strconv.Atoi(parts[1])
			entry, _ := strconv.Atoi(parts[2])
			val, _ := strconv.ParseFloat(parts[3], 64)
			// PoB-PoE2's entry by its content: equal entries' order varies between its runs
			want := prev.Controls[control-1].List[entry-1]
			for k, e := range tab.AffixControls()[control-1].List {
				if e.Label == want.Label && slices.Equal(e.ModList, want.ModList) {
					entry = k + 1
				}
			}
			if err := tab.SelectDisplayItemAffix(control, entry, val); err != nil {
				t.Fatalf("%s %s: %v", r.Case, r.Step, err)
			}
		case "slide":
			control, _ := strconv.Atoi(parts[1])
			val, _ := strconv.ParseFloat(parts[2], 64)
			if err := tab.SetDisplayItemAffixRange(control, val); err != nil {
				t.Fatalf("%s %s: %v", r.Case, r.Step, err)
			}
		case "paste":
			pastes++
			k, _ := strconv.Atoi(strings.TrimPrefix(r.Case, "paste "))
			item := ParseItem(pasted[k-1], tab.modSource)
			item.NormaliseQuality()
			tab.SetDisplayItem(item)
		}
		prev = r
		steps++
		if got := tab.DisplayItem.BuildRaw(); got != r.Raw {
			t.Errorf("%s %s: item text %s", r.Case, r.Step, firstDiffLine(got, r.Raw))
		}
		if tab.DisplayItem.AffixLimit != r.AffixLimit {
			t.Errorf("%s %s: affixLimit %d, PoB-PoE2's %d", r.Case, r.Step, tab.DisplayItem.AffixLimit, r.AffixLimit)
		}
		if !strings.HasPrefix(r.Case, "paste") {
			if got, want := affixControlsText(tab, true), r.controlsText(true); got != want {
				t.Errorf("%s %s: affix controls %s", r.Case, r.Step, firstDiffLine(got, want))
			} else if g, w := affixControlsText(tab, false), r.controlsText(false); g == w {
				sameOrder++
			}
		}
	}
	t.Logf("%d steps, %d pastes; %d steps' controls in PoB-PoE2's own order", steps, pastes, sameOrder)
	if steps == 0 || pastes == 0 {
		t.Fatalf("%d steps, %d pastes", steps, pastes)
	}
}
