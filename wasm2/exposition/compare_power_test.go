package exposition

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator2"
)

type comparePowerOracleRow struct {
	Category string
	Name     string
	Var      string
	NodeID   int `json:"nodeId"`
	SlotName string
	Impact   float64
	PathDist int
}

// the rows oracle/poe2_compare_power.lua records (`mise run compare-power-poe2`), and each stat's display format's
// smallest non-zero value, below which the report leaves a row out
var comparePowerOracleStats = []struct {
	stat string
	min  float64
}{{"FullDPS", 0.05}, {"TotalEHP", 0.5}, {"Life", 1}, {"Mana", 1}, {"Speed", 0.005}}

// rows where we differ from PoB-PoE2, each with its cause; a pin that starts matching fails
var comparePowerDivergent = map[string]string{
	// Item.lua:2907 leaves activeBondedState nil after BuildModList, so the copy keeps its parsed armour after
	// NormaliseQuality; calculator2 rebuilds the slot lists on the first read and takes the 20% quality
	"TotalEHP Item|Twig Focus, Weapon 2": "Item:BuildModList activeBondedState",
}

var colourCode = regexp.MustCompile(`\^x[0-9A-Fa-f]{6}|\^\d`)

func comparePowerKey(category, name, v string) string {
	if category == "Config" {
		return category + "|" + v
	}
	return category + "|" + name
}

func TestComparePowerBuilderVsPoB(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_compare_power.json"))
	mustOK(t, err)
	var byStat map[string]json.RawMessage
	mustOK(t, json.Unmarshal(raw, &byStat))
	oracle := map[string][]comparePowerOracleRow{}
	for _, s := range comparePowerOracleStats {
		var rows []comparePowerOracleRow
		mustOK(t, json.Unmarshal(byStat[s.stat], &rows))
		oracle[s.stat] = rows
	}

	primary := calculatedFixture(t, "compare_power_primary.xml")
	compared := calculatedFixture(t, "compare_power_compared.xml")
	stats := make([]string, len(comparePowerOracleStats))
	for i, s := range comparePowerOracleStats {
		stats[i] = s.stat
	}
	all := calculator2.ComparePowerCategories{TreeNodes: true, Items: true, SkillGems: true, SupportGems: true, Config: true}
	b, err := primary.NewComparePowerBuilder(compared, stats, all)
	mustOK(t, err)
	for b.StepMs(1000) < 1 {
	}
	report := b.Report()

	for i, s := range comparePowerOracleStats {
		want := map[string]comparePowerOracleRow{}
		for _, r := range oracle[s.stat] {
			want[comparePowerKey(r.Category, r.Name, r.Var)] = r
		}
		got := map[string]calculator2.ComparePowerEntry{}
		for _, e := range report {
			got[comparePowerKey(e.Category, e.Name, e.Var)] = e
		}
		for key, w := range want {
			g, ok := got[key]
			if !ok {
				t.Errorf("%s: PoB-PoE2 reports %s (%v), we do not", s.stat, key, w.Impact)
				continue
			}
			mismatch := math.Abs(g.Impact[i]-w.Impact) > 1e-9*math.Max(1, math.Abs(w.Impact))
			if cause, pinned := comparePowerDivergent[s.stat+" "+key]; pinned {
				if !mismatch {
					t.Errorf("%s: %s now matches PoB-PoE2; drop its pin (%s)", s.stat, key, cause)
				}
			} else if mismatch {
				t.Errorf("%s: %s impact %v, want %v", s.stat, key, g.Impact[i], w.Impact)
			}
			if w.Category == "Tree" && (g.NodeID != w.NodeID || g.PathDist != w.PathDist) {
				t.Errorf("%s: %s node %d path %d, want %d path %d", s.stat, key, g.NodeID, g.PathDist, w.NodeID, w.PathDist)
			}
			if g.SlotName != w.SlotName {
				t.Errorf("%s: %s slot %q, want %q", s.stat, key, g.SlotName, w.SlotName)
			}
		}
		for key, g := range got {
			if _, ok := want[key]; !ok && math.Abs(g.Impact[i]) >= s.min {
				t.Errorf("%s: we report %s at %s, PoB-PoE2 leaves it out", s.stat, key, strconv.FormatFloat(g.Impact[i], 'g', -1, 64))
			}
		}
	}
}

// CompareTab.lua:3670-3675, 3856-3907, 3936-3966 (PoB-PoE2): the Items view's rows and the Equip button's comparison
func TestCompareItemsVsPoB(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_compare_power.json"))
	mustOK(t, err)
	var oracle struct {
		Items []struct {
			Slot, Label, P, C string
			PWarn, CWarn      bool
		}
		Equip map[string]struct {
			Header string
			Delta  map[string]float64
		}
	}
	mustOK(t, json.Unmarshal(raw, &oracle))
	if len(oracle.Equip) < 4 {
		t.Fatalf("the oracle records %d equip comparisons", len(oracle.Equip))
	}

	primary := calculatedFixture(t, "compare_power_primary.xml")
	compared := calculatedFixture(t, "compare_power_compared.xml")
	view, err := primary.CompareItems(compared)
	mustOK(t, err)
	name := func(v *ItemView) string {
		if v == nil {
			return ""
		}
		var parts []string
		for _, l := range v.Name {
			parts = append(parts, colourCode.ReplaceAllString(l, ""))
		}
		return strings.Join(parts, ", ")
	}
	rows := append(view.Slots, view.Jewels...)
	if len(rows) != len(oracle.Items) {
		t.Fatalf("%d rows, want %d", len(rows), len(oracle.Items))
	}
	for i, w := range oracle.Items {
		g := rows[i]
		if g.Slot != w.Slot || g.Label != w.Label || name(g.A) != w.P || name(g.B) != w.C || g.AWarn != w.PWarn || g.BWarn != w.CWarn {
			t.Errorf("row %d: %s %q %q/%q warn %v/%v, want %s %q %q/%q warn %v/%v", i, g.Slot, g.Label, name(g.A), name(g.B), g.AWarn, g.BWarn,
				w.Slot, w.Label, w.P, w.C, w.PWarn, w.CWarn)
		}
	}
	for slot, w := range oracle.Equip {
		v, err := primary.CompareEquipItem(compared, slot)
		mustOK(t, err)
		if v == nil || len(v.Blocks) != 1 {
			t.Fatalf("%s: no comparison", slot)
		}
		b := v.Blocks[0]
		if b.Header != w.Header {
			t.Errorf("%s: header %q, want %q", slot, b.Header, w.Header)
		}
		for stat, d := range w.Delta {
			got := b.Compare.Output[stat] - v.Base.Output[stat]
			mismatch := math.Abs(got-d) > 1e-9*math.Max(1, math.Abs(d))
			if pinned := slot == "Weapon 2" && stat == "TotalEHP"; pinned {
				// the Twig Focus pin above: the same copy, the same cause
				if !mismatch {
					t.Errorf("%s: %s now matches PoB-PoE2; drop its pin", slot, stat)
				}
			} else if mismatch {
				t.Errorf("%s: %s %v, want %v", slot, stat, got, d)
			}
		}
	}
}

// CompareTab.lua:3494-3659 (PoB-PoE2): the expanded mode's lines of each side's item
func TestCompareItemCardsVsPoB(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_compare_power.json"))
	mustOK(t, err)
	var oracle struct {
		Cards map[string]struct{ A, B []string }
	}
	mustOK(t, json.Unmarshal(raw, &oracle))
	if len(oracle.Cards) < 4 {
		t.Fatalf("the oracle records %d slots' cards", len(oracle.Cards))
	}
	primary := calculatedFixture(t, "compare_power_primary.xml")
	compared := calculatedFixture(t, "compare_power_compared.xml")
	for slot, w := range oracle.Cards {
		got, err := primary.CompareItemCards(compared, slot)
		mustOK(t, err)
		if !slices.Equal(got.A, w.A) || !slices.Equal(got.B, w.B) {
			t.Errorf("%s:\n got %q\n     %q\nwant %q\n     %q", slot, got.A, got.B, w.A, w.B)
		}
	}
}

// CompareTab.lua:4545-4561 (PoB-PoE2): whether each Calcs entry's mod rows match between the two builds
func TestCompareModsEqualVsPoB(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_compare_power.json"))
	mustOK(t, err)
	var oracle struct {
		ModRows []struct {
			ModName                        []string
			Multi                          bool
			ModType, Cfg, ModSource, Actor string
			Enemy, Match                   bool
		}
	}
	mustOK(t, json.Unmarshal(raw, &oracle))
	sds := make([]calculator2.ModSectionData, len(oracle.ModRows))
	differ := 0
	for i, r := range oracle.ModRows {
		sds[i] = calculator2.ModSectionData{ModName: r.ModName, Multi: r.Multi, ModType: r.ModType, Cfg: r.Cfg, ModSource: r.ModSource, Enemy: r.Enemy, Actor: r.Actor}
		if !r.Match {
			differ++
		}
	}
	if differ < 10 || differ == len(sds) {
		t.Fatalf("%d of %d entries differ in the oracle; the fixture pair should give both", differ, len(sds))
	}
	primary := calculatedFixture(t, "compare_power_primary.xml")
	compared := calculatedFixture(t, "compare_power_compared.xml")
	got, err := primary.CompareModsEqual(compared, sds)
	mustOK(t, err)
	for i, r := range oracle.ModRows {
		if got[i] != r.Match {
			t.Errorf("%v %s cfg %q: match %v, want %v", r.ModName, r.ModType, r.Cfg, got[i], r.Match)
		}
	}
}

// CompareCalcsHelpers.lua:99-146 (PoB-PoE2): the Calcs view tooltip's mod rows of each build
func TestCalcsCompareModRowsVsPoB(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_compare_power.json"))
	mustOK(t, err)
	type row struct{ Key, Display, SourceType, SourceName, ModName string }
	var oracle struct {
		ModRows []struct {
			ModName                        []string
			Multi                          bool
			ModType, Cfg, ModSource, Actor string
			Enemy                          bool
			PRows, CRows                   []row
		}
	}
	mustOK(t, json.Unmarshal(raw, &oracle))
	primary := calculatedFixture(t, "compare_power_primary.xml")
	compared := calculatedFixture(t, "compare_power_compared.xml")
	key := func(r row) string {
		return strings.Join([]string{r.Key, r.Display, r.SourceType, r.SourceName, r.ModName}, "|")
	}
	rowsOf := func(c *Calculator, sd calculator2.ModSectionData) []string {
		got, err := c.CalcsCompareModRows(sd)
		mustOK(t, err)
		out := make([]string, len(got))
		for i, g := range got {
			out[i] = key(row{g.Key, g.Display, g.SourceType, g.SourceName, g.ModName})
		}
		slices.Sort(out)
		return out
	}
	checked := 0
	for _, r := range oracle.ModRows {
		if len(r.PRows)+len(r.CRows) == 0 {
			continue
		}
		checked++
		sd := calculator2.ModSectionData{ModName: r.ModName, Multi: r.Multi, ModType: r.ModType, Cfg: r.Cfg, ModSource: r.ModSource, Enemy: r.Enemy, Actor: r.Actor}
		for side, c := range map[string]*Calculator{"A": primary, "B": compared} {
			want := r.PRows
			if side == "B" {
				want = r.CRows
			}
			ws := make([]string, len(want))
			for i, w := range want {
				ws[i] = key(w)
			}
			slices.Sort(ws)
			if got := rowsOf(c, sd); !slices.Equal(got, ws) {
				t.Errorf("%v %s %s:\n got %q\nwant %q", r.ModName, r.ModType, side, got, ws)
			}
		}
	}
	if checked < 20 {
		t.Fatalf("only %d entries have rows", checked)
	}
}
