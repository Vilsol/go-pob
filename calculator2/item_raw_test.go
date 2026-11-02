package calculator2

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

type buildRawRow struct {
	Build string `json:"build"`
	ID    int    `json:"id"`
	Raw   string `json:"raw"`
}

// runBuildRawOracle is testdata/item_build_raw_oracle.lua over the given build XMLs.
func runBuildRawOracle(t *testing.T, xmls []string) []buildRawRow {
	t.Helper()
	pob, luajit := pobPoE2Probe(t)
	probe, err := filepath.Abs(filepath.Join("testdata", "item_build_raw_oracle.lua"))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "raw.jsonl")
	args := append([]string{"-R", luajit, probe, out}, xmls...)
	cmd := exec.CommandContext(context.Background(), "setarch", args...)
	cmd.Dir = filepath.Join(pob, "src")
	cmd.Env = append(os.Environ(),
		"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
		"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("the BuildRaw oracle failed: %v\n%s", err, stderr.String())
	}
	return readBuildRawRows(t, out)
}

func readBuildRawRows(t *testing.T, path string) []buildRawRow {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // test fixture or the test's own temp dir
	if err != nil {
		t.Fatal(err)
	}
	var rows []buildRawRow
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 16<<20), 16<<20)
	for sc.Scan() {
		var row buildRawRow
		if err := json.Unmarshal(sc.Bytes(), &row); err != nil {
			t.Fatalf("bad oracle row: %v", err)
		}
		row.Raw = oracleBytes(row.Raw)
		rows = append(rows, row)
	}
	return rows
}

// oracleBytes undoes the oracle's encoder, which writes each byte above 127 as its own \u00XX.
func oracleBytes(s string) string {
	b := make([]byte, 0, len(s))
	for _, r := range s {
		if r > 255 {
			return s
		}
		b = append(b, byte(r)) //nolint:gosec // r is at most 255 here
	}
	return string(b)
}

// baseLineOrder sorts each run of variant-tagged lines: PoB-PoE2 writes the base lines among them in its hash order.
func baseLineOrder(raw string) string {
	lines := strings.Split(raw, "\n")
	for i := 0; i < len(lines); i++ {
		j := i
		for j < len(lines) && isTaggedBaseLine(lines[j]) {
			j++
		}
		sort.Strings(lines[i:j])
		if j > i {
			i = j
		}
	}
	return strings.Join(lines, "\n")
}

func isTaggedBaseLine(line string) bool {
	return strings.HasPrefix(line, "{variant:") || strings.HasPrefix(line, "{base:") || strings.HasPrefix(line, "{version:") || strings.HasPrefix(line, "{group:")
}

// buildRawDiffs compares BuildRaw for every item of the given builds with PoB-PoE2's rows.
func buildRawDiffs(t *testing.T, xmls []string, rows []buildRawRow) (int, []string) {
	t.Helper()
	want := map[string]string{}
	for _, r := range rows {
		want[r.Build+"\x00"+strconv.Itoa(r.ID)] = r.Raw
	}
	n := 0
	var bad []string
	for _, x := range xmls {
		raw, err := os.ReadFile(x) //nolint:gosec // test input
		if err != nil {
			t.Fatal(err)
		}
		build, err := pob2.ParseBuild(raw)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(x)
		for _, it := range build.Items.Items {
			ranges := make([]ItemModRange, 0, len(it.ModRanges))
			for _, r := range it.ModRanges {
				ranges = append(ranges, ItemModRange{ID: r.ID, Range: r.Range})
			}
			item := LoadItem(it.ID, it.Raw, ItemVariants{Variant: it.Variant, VariantAlt: it.VariantAlt, VariantAlt2: it.VariantAlt2,
				VariantAlt3: it.VariantAlt3, VariantAlt4: it.VariantAlt4, VariantAlt5: it.VariantAlt5}, ranges, ItemModSource(WithParser(ParseMod)))
			if item.Base == nil {
				continue
			}
			n++
			w, ok := want[name+"\x00"+strconv.Itoa(it.ID)]
			if !ok {
				bad = append(bad, fmt.Sprintf("%s item %d: PoB-PoE2 has no row", name, it.ID))
				continue
			}
			if got := item.BuildRaw(); baseLineOrder(got) != baseLineOrder(w) {
				bad = append(bad, fmt.Sprintf("%s item %d:\n%s\nPoB-PoE2:\n%s", name, it.ID, got, w))
			}
		}
	}
	return n, bad
}

// The fixture's items are written as PoB-PoE2 writes them; testdata/item_build_raw_pob.jsonl is its oracle's rows.
func TestBuildRawMatchesPoB(t *testing.T) {
	fixture := filepath.Join("testdata", "items_edit.xml")
	rows := readBuildRawRows(t, filepath.Join("testdata", "item_build_raw_pob.jsonl"))
	n, bad := buildRawDiffs(t, []string{fixture}, rows)
	if n == 0 || n != len(rows) {
		t.Fatalf("%d items graded, %d oracle rows", n, len(rows))
	}
	for _, b := range bad {
		t.Error(b)
	}
}

// TestBuildRawVsPoBPoE2Corpus writes every item of the first ITEM_BUILDS (default 300) PoE2 corpus builds with
// BuildRaw on both sides.
func TestBuildRawVsPoBPoE2Corpus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("SKIP: reads the local PoE2 corpus; set CORPUS=true")
	}
	dir := filepath.Join("..", "testdata", "corpus-poe2-0.5")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("SKIP: no PoE2 corpus at %s: %v", dir, err)
	}
	n := 300
	if s := os.Getenv("ITEM_BUILDS"); s != "" {
		if n, err = strconv.Atoi(s); err != nil {
			t.Fatalf("ITEM_BUILDS=%q: %v", s, err)
		}
	}
	var xmls []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xml") && len(xmls) < n {
			p, err := filepath.Abs(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			xmls = append(xmls, p)
		}
	}
	if len(xmls) == 0 {
		t.Skipf("SKIP: no XMLs in %s", dir)
	}
	rows := runBuildRawOracle(t, xmls)
	graded, bad := buildRawDiffs(t, xmls, rows)
	t.Logf("BuildRaw: %d / %d items identical", graded-len(bad), graded)
	unexplained := 0
	for _, b := range bad {
		// a socketed rune line scaled by its item: the data has no ModScalability row for it, as the item grade knows
		if knownRuneScalability(b) {
			t.Logf("known cause: %s", strings.SplitN(b, "\n", 2)[0])
			continue
		}
		if unexplained++; unexplained <= 10 {
			t.Error(b)
		}
	}
}

// knownRuneScalability is a diff whose only differing lines are scaled rune lines.
func knownRuneScalability(diff string) bool {
	parts := strings.SplitN(diff, "\nPoB-PoE2:\n", 2)
	if len(parts) != 2 {
		return false
	}
	got, want := strings.Split(parts[0], "\n")[1:], strings.Split(parts[1], "\n")
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] && !strings.HasPrefix(want[i], "{enchant}{rune}") {
			return false
		}
	}
	return true
}

// TestItemsSaveVsPoBPoE2Corpus saves the items of the first ITEM_BUILDS (default 300) PoE2 corpus builds as an edit
// does, every item through BuildRaw, and calculates them again: every output is unchanged.
func TestItemsSaveVsPoBPoE2Corpus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("SKIP: reads the local PoE2 corpus; set CORPUS=true")
	}
	dir := filepath.Join("..", "testdata", "corpus-poe2-0.5")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("SKIP: no PoE2 corpus at %s: %v", dir, err)
	}
	n := 300
	if s := os.Getenv("ITEM_BUILDS"); s != "" {
		if n, err = strconv.Atoi(s); err != nil {
			t.Fatalf("ITEM_BUILDS=%q: %v", s, err)
		}
	}
	tree := loadTestTree(t)
	builds, changed, failed := 0, 0, 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".xml") || builds >= n {
			continue
		}
		builds++
		raw, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // corpus input
		if err != nil {
			t.Fatal(err)
		}
		build, err := pob2.ParseBuild(raw)
		if err != nil {
			failed++
			continue
		}
		tab := NewItemsTab(build, tree, nil)
		before, err := BuildOutput(build, Options{Tree: tree, Items: tab})
		if err != nil {
			failed++
			continue
		}
		tab.Save(build)
		after, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
		if err != nil {
			t.Errorf("%s: %v after the save", e.Name(), err)
			continue
		}
		var diffs []string
		for k, v := range before.Player.Output {
			if w := after.Player.Output[k]; !(v == w || (v != v && w != w) || math.Abs(v-w) <= 1e-9*math.Max(1, math.Abs(v))) {
				diffs = append(diffs, fmt.Sprintf("%s %v -> %v", k, v, w))
			}
		}
		if len(diffs) > 0 {
			changed++
			sort.Strings(diffs)
			if changed <= 10 {
				t.Errorf("%s: %d outputs moved: %.300s", e.Name(), len(diffs), strings.Join(diffs, "; "))
			}
		}
	}
	t.Logf("%d builds, %d failed to calculate, %d changed by the save", builds, failed, changed)
}
