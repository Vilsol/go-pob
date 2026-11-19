package calculator2

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"maps"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// itemCompareRow is one row of testdata/item_compare_pob.jsonl.gz, which testdata/item_compare_oracle.lua writes
// from PoB-PoE2's item tooltips over testdata/items_edit.xml: the base pass first, then one row per item.
type itemCompareRow struct {
	Base  map[string]any `json:"base"`
	Item  string         `json:"item"`
	Lines []string       `json:"lines"`
	Calcs []struct {
		Slot    string         `json:"slot"`
		RepItem bool           `json:"repItem"`
		Toggle  bool           `json:"toggle"`
		Output  map[string]any `json:"output"`
	} `json:"calcs"`
}

func readItemCompareRows(t *testing.T) []itemCompareRow {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "item_compare_pob.jsonl.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var rows []itemCompareRow
	sc := bufio.NewScanner(zr)
	sc.Buffer(nil, 1<<24)
	for sc.Scan() {
		var r itemCompareRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, r)
	}
	return rows
}

var colorCodeRe = regexp.MustCompile(`\^x[0-9A-Fa-f]{6}|\^\d`)

// outputDiffs lists the stats where ours differs from PoB-PoE2's numeric output, the missing ones on either side
// included.
func outputDiffs(got map[string]float64, want map[string]any) []string {
	var diffs []string
	for k, v := range want {
		w, ok := v.(float64)
		if !ok {
			continue
		}
		g, has := got[k]
		if !has || math.Abs(g-w) > 1e-6*math.Max(1, math.Abs(w)) {
			diffs = append(diffs, k)
		}
	}
	sort.Strings(diffs)
	return diffs
}

func TestItemComparesMatchPoB(t *testing.T) {
	rows := readItemCompareRows(t)
	pastes := readItemsEditPastes(t)
	raw, err := os.ReadFile(filepath.Join("testdata", "items_edit.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	tab := NewItemsTab(build, loadTestTree(t), nil)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t), Items: tab})
	if err != nil {
		t.Fatal(err)
	}
	misc := NewMiscCalculator(env)
	if d := outputDiffs(misc.Base.Player.Output, rows[0].Base); len(d) > 0 {
		t.Errorf("base pass differs on %d stats: %.300v", len(d), d)
	}
	pasted := func(i int) *Item {
		item := ParseItem(pastes[i-1], tab.modSource)
		item.NormaliseQuality()
		item.BuildModList()
		return item
	}
	items := map[string]*Item{
		"paste helmet":    pasted(1),
		"paste wand":      pasted(2),
		"paste flask":     pasted(4),
		"equipped helmet": tab.Items[3],
		"Andvarius":       ItemDB(ItemDBUnique)["Andvarius, Gold Ring"],
		"Headhunter":      ItemDB(ItemDBUnique)["Headhunter, Heavy Belt"],
		"Arakaali's Gift": ItemDB(ItemDBUnique)["Arakaali's Gift, Antidote Charm"],
	}
	blocks, stats := 0, 0
	for _, r := range rows[1:] {
		item := items[r.Item]
		if item == nil {
			t.Fatalf("no item for %q", r.Item)
		}
		compares := tab.ItemCompares(item, misc, false)
		// the headers PoB-PoE2 shows are those of the blocks with a stat difference
		var wantHeaders []string
		for i := 0; i < len(r.Lines); i++ {
			line := colorCodeRe.ReplaceAllString(r.Lines[i], "")
			if strings.HasSuffix(line, "will give you:") {
				if i+1 < len(r.Lines) && strings.HasPrefix(r.Lines[i+1], "(replacing ") {
					line += "\n" + colorCodeRe.ReplaceAllString(r.Lines[i+1], "")
				}
				wantHeaders = append(wantHeaders, line)
			}
		}
		var gotHeaders []string
		for _, c := range compares {
			blocks++
			// the last calculation for the slot is the one its block shows
			var want map[string]any
			for _, calc := range r.Calcs {
				if calc.Slot == c.Slot && (calc.Toggle == (c.Slot == "")) {
					want = calc.Output
				}
			}
			if want == nil {
				t.Errorf("%s: PoB-PoE2 ran no calculation for %q", r.Item, c.Slot)
				continue
			}
			stats += len(want)
			if d := outputDiffs(c.Env.Player.Output, want); len(d) > 0 {
				t.Errorf("%s in %q: %d stats differ: %.300v", r.Item, c.Slot, len(d), d)
			}
			if len(outputDiffs(c.Env.Player.Output, rows[0].Base)) > 0 {
				gotHeaders = append(gotHeaders, c.Header)
			}
		}
		if strings.Join(gotHeaders, "|") != strings.Join(wantHeaders, "|") {
			t.Errorf("%s: headers\n got %q\nwant %q", r.Item, gotHeaders, wantHeaders)
		}
	}
	t.Logf("%d blocks, %d stats graded", blocks, stats)
	if blocks == 0 {
		t.Fatal("no blocks graded")
	}
}

// PoB-PoE2's spec:CollectGrantedPassiveNodesFromItems over the custom-mods warrior grants nothing with no override,
// and node 62152 with { repSlotName = "Amulet", repItem = <amulet allocating a sinister jewel socket> }
// (PassiveSpec.lua:1218-1220).
func TestOverrideItemGrantsPassiveSocket(t *testing.T) {
	raw, err := os.ReadFile("testdata/defence_custom_mods.xml")
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	tab := NewItemsTab(build, loadTestTree(t), nil)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t), Items: tab})
	if err != nil {
		t.Fatal(err)
	}
	item := ParseItem("Rarity: RARE\nTest Pendant\nGold Amulet\nAllocates 1 Sinister Jewel Socket", tab.modSource)
	item.BuildModList()
	for _, c := range []struct {
		override calcOverride
		want     []int
	}{
		{calcOverride{}, nil},
		{calcOverride{RepSlotName: "Amulet", RepItem: item}, []int{62152}},
	} {
		granted := collectGrantedPassiveNodesFromItems(env.Spec, tab, env.Spec.AllocNodes, false, c.override, newModList(), 1)
		if got := slices.Sorted(maps.Keys(granted)); !slices.Equal(got, c.want) {
			t.Errorf("override %q: granted %v, PoB-PoE2 %v", c.override.RepSlotName, got, c.want)
		}
	}
}

// PoB-PoE2's calcFunc({ repSlotName = "Body Armour", repItem = <armour with a disabled bonded rune line and 100%
// increased effect of socketed augments> }) over the custom-mods warrior: the line has no bonded list, so its
// scaled mods land in the base list (Item.lua:2838), Life 547 -> 570; without the line Life stays 547.
func TestDisabledBondedRuneLineScalesIntoBaseList(t *testing.T) {
	raw, err := os.ReadFile("testdata/defence_custom_mods.xml")
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	tab := NewItemsTab(build, loadTestTree(t), nil)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t), Items: tab})
	if err != nil {
		t.Fatal(err)
	}
	misc := NewMiscCalculator(env)
	for _, c := range []struct {
		raw  string
		life float64
	}{
		{"Rarity: RARE\nTest Shell\nVaal Cuirass\nSockets: S\nRune: Iron Rune\nImplicits: 1\n{enchant}{rune}{disabled}Bonded: +40 to maximum Life\n100% increased effect of Socketed Augment Items", 570},
		{"Rarity: RARE\nTest Shell\nVaal Cuirass\nSockets: S\nRune: Iron Rune\nImplicits: 0\n100% increased effect of Socketed Augment Items", 547},
	} {
		item := ParseItem(c.raw, tab.modSource)
		item.BuildModList()
		if got := misc.Calc(ItemOverride{RepSlotName: "Body Armour", RepItem: item}).Player.Output["Life"]; got != c.life {
			t.Errorf("Life %v, PoB-PoE2 %v for\n%s", got, c.life, c.raw)
		}
	}
}
