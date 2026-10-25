package calculator2

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// pobPoE2Probe locates the PoB-PoE2 pin and the deterministic luajit, or skips.
func pobPoE2Probe(t *testing.T) (string, string) {
	t.Helper()
	common, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		t.Skipf("SKIP: not in a git checkout: %v", err)
	}
	main := filepath.Dir(strings.TrimSpace(string(common)))
	pob := os.Getenv("POB2_DIR")
	if pob == "" {
		pob = filepath.Join(main, "..", "PathOfBuilding-PoE2")
	}
	luajit := filepath.Join(main, ".luajit-det", "bin", "luajit")
	for _, p := range []string{filepath.Join(pob, "src", "HeadlessWrapper.lua"), luajit} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("SKIP: %s is absent", p)
		}
	}
	return pob, luajit
}

// runItemOracle loads the given build XMLs' items in PoB-PoE2 and returns its rows.
func runItemOracle(t *testing.T, xmls []string) []*oracleItem {
	t.Helper()
	pob, luajit := pobPoE2Probe(t)
	probe, err := filepath.Abs(filepath.Join("testdata", "item_oracle.lua"))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "items.jsonl")
	args := append([]string{"-R", luajit, probe, out}, xmls...)
	cmd := exec.CommandContext(context.Background(), "setarch", args...)
	cmd.Dir = filepath.Join(pob, "src")
	cmd.Env = append(os.Environ(),
		"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
		"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("the item oracle failed: %v\n%s", err, stderr.String())
	}
	b, err := os.ReadFile(out) //nolint:gosec // the test's own temp dir
	if err != nil {
		t.Fatal(err)
	}
	var rows []*oracleItem
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 64<<20), 64<<20)
	for sc.Scan() {
		row := &oracleItem{}
		if err := json.Unmarshal(sc.Bytes(), row); err != nil {
			t.Fatalf("bad oracle row: %v\n%.300s", err, sc.Text())
		}
		rows = append(rows, row)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}

type itemGrade struct {
	items, agree int
	classes      map[string]int
	first        map[string]string
	bases        map[string]int
	causes       map[string]int
	unexplained  int
}

func gradeItems(rows []*oracleItem, dump io.Writer) itemGrade {
	g := itemGrade{classes: map[string]int{}, first: map[string]string{}, bases: map[string]int{}, causes: map[string]int{}}
	for _, row := range rows {
		item, src := loadOracleItem(row)
		bad := compareOracleItem(row, item, src)
		g.items++
		if len(bad) == 0 {
			g.agree++
		} else {
			g.bases[strOr(row.BaseName)]++
			if cause := knownItemCause(row, item, src, bad); cause != "" {
				g.causes[cause]++
			} else {
				g.unexplained++
			}
		}
		if dump != nil && len(bad) > 0 {
			for class, msg := range bad {
				fmt.Fprintf(dump, "%s\t%d\t%s\t%s\t%s\n", row.Build, row.ID, strOr(row.BaseName), class, msg)
			}
		}
		for class, msg := range bad {
			g.classes[class]++
			if _, seen := g.first[class]; !seen {
				g.first[class] = row.Build + " item " + strconv.Itoa(row.ID) + ": " + msg
			}
		}
	}
	return g
}

func (g itemGrade) report(t *testing.T) {
	t.Helper()
	t.Logf("items: %d agree / %d; of the rest %d have no known cause", g.agree, g.items, g.unexplained)
	causes := make([]string, 0, len(g.causes))
	for c := range g.causes {
		causes = append(causes, c)
	}
	sort.Strings(causes)
	for _, c := range causes {
		t.Logf("  known cause %6d  %s", g.causes[c], c)
	}
	classes := make([]string, 0, len(g.classes))
	for c := range g.classes {
		classes = append(classes, c)
	}
	sort.Strings(classes)
	for _, c := range classes {
		t.Logf("  disagree %-14s %6d  first: %.600s", c, g.classes[c], g.first[c])
	}
	bases := make([]string, 0, len(g.bases))
	for b := range g.bases {
		bases = append(bases, b)
	}
	sort.Slice(bases, func(a, b int) bool { return g.bases[bases[a]] > g.bases[bases[b]] })
	for n, b := range bases {
		if n == 8 {
			break
		}
		t.Logf("  disagreeing base %-40q %d", b, g.bases[b])
	}
}

// TestItemsVsPoBPoE2Corpus loads the items of the first ITEM_BUILDS (default 100) PoE2 corpus
// builds in both PoB-PoE2 and Go, with PoB's own parse of every line as the mod source on both
// sides, and compares the item: base, rarity, mod lines, slot mod lists and defences.
func TestItemsVsPoBPoE2Corpus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("SKIP: reads the local PoE2 corpus; set CORPUS=true")
	}
	dir := filepath.Join("..", "testdata", "corpus-poe2-0.5")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("SKIP: no PoE2 corpus at %s: %v", dir, err)
	}
	n := 100
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
	rows := runItemOracle(t, xmls)
	if len(rows) == 0 {
		t.Fatalf("the oracle printed no items for %d builds", len(xmls))
	}
	var dump io.Writer
	if p := os.Getenv("ITEM_DUMP"); p != "" {
		f, err := os.Create(p) //nolint:gosec // the caller's path
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		dump = f
	}
	g := gradeItems(rows, dump)
	g.report(t)
	if g.unexplained > 0 {
		t.Errorf("%d of %d items disagree with PoB-PoE2 for no known cause", g.unexplained, g.items)
	}
}
