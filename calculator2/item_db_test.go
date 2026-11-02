package calculator2

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/pob2"
)

// itemDBRow is one row of testdata/item_db_pob.jsonl.gz, which testdata/item_db_oracle.lua writes from PoB-PoE2's
// unique database and its list control.
type itemDBRow struct {
	Name        string    `json:"name"`
	Raw         string    `json:"raw"`
	Type        string    `json:"type"`
	PrimarySlot string    `json:"primarySlot"`
	League      string    `json:"league"`
	Source      string    `json:"source"`
	Level       float64   `json:"level"`
	Leagues     []string  `json:"leagues"`
	Types       []string  `json:"types"`
	Context     []float64 `json:"context"`
	Filter      []int     `json:"filter"`
	Search      string    `json:"search"`
	Mode        int       `json:"mode"`
	Listed      []string  `json:"listed"`
}

func readItemDBRows(t *testing.T) []itemDBRow {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "item_db_pob.jsonl.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var rows []itemDBRow
	sc := bufio.NewScanner(zr)
	sc.Buffer(nil, 1<<22)
	for sc.Scan() {
		var r itemDBRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, r)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}

var leadingTagsRe = regexp.MustCompile(`^(\{[^}]*\})+`)

// canonicalBaseLines sorts the tagged base-name lines among themselves: BuildRaw writes them in pairs(baseLines)
// order, Lua's hash order, where Go writes them by name.
func canonicalBaseLines(raw string) string {
	lines := strings.Split(raw, "\n")
	var at []int
	var baseLines []string
	for i, line := range lines {
		if !strings.HasPrefix(line, "{") {
			continue
		}
		if _, ok := poe2.BaseByName(leadingTagsRe.ReplaceAllString(line, "")); ok {
			at = append(at, i)
			baseLines = append(baseLines, line)
		}
	}
	sort.Strings(baseLines)
	for k, i := range at {
		lines[i] = baseLines[k]
	}
	return strings.Join(lines, "\n")
}

func TestUniqueDBMatchesPoB(t *testing.T) {
	db := ItemDB(ItemDBUnique)
	rows := readItemDBRows(t)
	matched, total, runeforged := 0, 0, 0
	var leagues, types []string
	var ctx ItemDBContext
	for _, r := range rows {
		if r.Leagues != nil {
			leagues, types = r.Leagues, r.Types
		}
		if r.Context != nil {
			ctx = ItemDBContext{CharacterLevel: int(r.Context[0]), Str: r.Context[1], Dex: r.Context[2], Int: r.Context[3]}
		}
		if r.Name == "" {
			continue
		}
		total++
		item := db[r.Name]
		if item == nil {
			t.Errorf("%s: not in the database", r.Name)
			continue
		}
		if item.BaseList != nil {
			runeforged++
		}
		got := struct{ Raw, Type, PrimarySlot, League, Source string }{canonicalBaseLines(item.Raw), item.Type, item.GetPrimarySlot(), item.League, item.Source}
		want := struct{ Raw, Type, PrimarySlot, League, Source string }{canonicalBaseLines(r.Raw), r.Type, r.PrimarySlot, r.League, r.Source}
		if got != want || deref(item.Requirements.Level) != r.Level {
			if got.Raw != want.Raw {
				t.Errorf("%s:\n got %q\nwant %q", r.Name, got.Raw, want.Raw)
			} else {
				t.Errorf("%s: %+v level %v, PoB-PoE2 %+v level %v", r.Name, got, deref(item.Requirements.Level), want, r.Level)
			}
			continue
		}
		matched++
	}
	t.Logf("unique database: matched %d / %d (%d with runeforging base variants), %d in ours", matched, total, runeforged, len(db))
	if total == 0 || runeforged == 0 {
		t.Fatal("no database rows, or none runeforged")
	}
	if len(db) != total {
		t.Errorf("ours holds %d uniques, PoB-PoE2's %d", len(db), total)
	}
	gotLeagues, gotTypes := ItemDBLists(ItemDBUnique)
	if strings.Join(gotLeagues, "|") != strings.Join(leagues, "|") || strings.Join(gotTypes, "|") != strings.Join(types, "|") {
		t.Errorf("lists\n got %q %q\nwant %q %q", gotLeagues, gotTypes, leagues, types)
	}
	filters := 0
	for _, r := range rows {
		if r.Filter == nil {
			continue
		}
		filters++
		f := ItemDBFilter{Slot: r.Filter[0], Type: r.Filter[1], League: r.Filter[2], Requirement: r.Filter[3], Obtainable: r.Filter[4], Search: r.Search, SearchMode: r.Mode}
		var got []string
		for _, item := range FilterItemDB(ItemDBUnique, f, ctx) {
			got = append(got, item.Name)
		}
		if strings.Join(got, "\n") != strings.Join(r.Listed, "\n") {
			t.Errorf("filter %+v: %d listed, PoB-PoE2 lists %d\n got %q\nwant %q", f, len(got), len(r.Listed), got, r.Listed)
		}
	}
	if filters == 0 {
		t.Fatal("no filter rows")
	}
}

// Every unique a corpus build carries is in the database, under its title, on one of the entry's bases.
func TestCorpusPoE2UniquesInItemDB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("SKIP: reads the local PoE2 corpus; set CORPUS=true")
	}
	dir := filepath.Join("..", "testdata", "corpus-poe2-0.5")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("SKIP: no PoE2 corpus at %s: %v", dir, err)
	}
	byTitle := map[string][]*Item{}
	for _, entry := range ItemDB(ItemDBUnique) {
		byTitle[entry.Title] = append(byTitle[entry.Title], entry)
	}
	modSource := ItemModSource(WithParser(ParseMod))
	builds, resolved, total := 0, 0, 0
	missing := map[string]int{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // the corpus directory
		if err != nil {
			t.Fatal(err)
		}
		build, err := pob2.ParseBuild(raw)
		if err != nil {
			continue
		}
		builds++
		for _, saved := range build.Items.Items {
			item := LoadItem(saved.ID, saved.Raw, ItemVariants{}, nil, modSource)
			if item.Rarity != "UNIQUE" || item.Base == nil {
				continue
			}
			total++
			ok := false
			for _, entry := range byTitle[item.Title] {
				_, onVariant := entry.baseLines[item.BaseName]
				ok = ok || entry.BaseName == item.BaseName || onVariant
			}
			if ok {
				resolved++
			} else {
				missing[item.Name]++
			}
		}
	}
	t.Logf("corpus uniques in the unique database: resolved %d / %d over %d builds", resolved, total, builds)
	names := make([]string, 0, len(missing))
	for n := range missing {
		names = append(names, n)
	}
	sort.Slice(names, func(a, b int) bool { return missing[names[a]] > missing[names[b]] })
	for _, n := range names[:min(len(names), 20)] {
		t.Logf("  %6d  %s", missing[n], n)
	}
	if total == 0 {
		t.Fatal("no corpus uniques")
	}
}

// canonicalOracleRows is the rows with their raw texts' base lines in canonicalBaseLines order, which is all an
// oracle run changes from one process to the next.
func canonicalOracleRows(t *testing.T, body []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	for _, line := range bytes.Split(bytes.TrimSuffix(body, []byte("\n")), []byte("\n")) {
		var row map[string]any
		if err := json.Unmarshal(line, &row); err != nil {
			t.Fatal(err)
		}
		if raw, ok := row["raw"].(string); ok {
			row["raw"] = canonicalBaseLines(raw)
		}
		b, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		out.Write(b)
		out.WriteByte('\n')
	}
	return out.Bytes()
}

// The committed craft, database and stat difference rows are what their oracles write from the pinned PoB-PoE2; CORPUS=true runs them,
// as they start PoB-PoE2.
func TestItemDBOraclesAreCurrent(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true runs PoB-PoE2")
	}
	pob, luajit := pobPoE2Probe(t)
	abs := func(name string) string {
		p, err := filepath.Abs(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, c := range []struct {
		probe, want string
		args        []string
	}{
		{"item_craft_oracle.lua", "item_craft_pob.jsonl", nil},
		{"item_db_oracle.lua", "item_db_pob.jsonl.gz", nil},
		{"item_compare_oracle.lua", "item_compare_pob.jsonl.gz", []string{abs("items_edit.xml"), abs("items_edit_pastes.txt")}},
	} {
		probe, err := filepath.Abs(filepath.Join("testdata", c.probe))
		if err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(t.TempDir(), "out.jsonl")
		cmd := exec.CommandContext(context.Background(), "setarch", append([]string{"-R", luajit, probe, out}, c.args...)...) //nolint:gosec // the test's own probes and fixtures
		cmd.Dir = filepath.Join(pob, "src")
		cmd.Env = append(os.Environ(),
			"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
			"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s failed: %v\n%s", c.probe, err, stderr.String())
		}
		got, err := os.ReadFile(out) //nolint:gosec // the test's own temp dir
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join("testdata", c.want))
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(c.want, ".gz") {
			zr, err := gzip.NewReader(bytes.NewReader(want))
			if err != nil {
				t.Fatal(err)
			}
			if want, err = io.ReadAll(zr); err != nil {
				t.Fatal(err)
			}
		}
		if !bytes.Equal(canonicalOracleRows(t, got), canonicalOracleRows(t, want)) {
			t.Errorf("testdata/%s is stale; rerun testdata/%s", c.want, c.probe)
		}
	}
}
