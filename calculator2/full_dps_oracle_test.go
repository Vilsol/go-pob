package calculator2

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/pob2"
)

// fullDPSOracleRow is one line oracle/poe2_full_dps.lua writes.
type fullDPSOracleRow struct {
	Build      string   `json:"build"`
	Error      string   `json:"error"`
	FullDPS    *float64 `json:"FullDPS"`
	FullDotDPS *float64 `json:"FullDotDPS"`
	Skills     []struct {
		Name      string  `json:"name"`
		DPS       float64 `json:"dps"`
		Count     float64 `json:"count"`
		Trigger   string  `json:"trigger"`
		SkillPart string  `json:"skillPart"`
		Source    string  `json:"source"`
	} `json:"skills"`
	StatBox [][2]string         `json:"statBox"`
	Saved   []map[string]string `json:"saved"`
}

var includeInFullDPSAttr = regexp.MustCompile(` includeInFullDPS="[^"]*"`)

// includeAllInFullDPS is the rewrite oracle/poe2_full_dps.lua makes: every socket group included in Full DPS.
func includeAllInFullDPS(xml []byte) []byte {
	s := includeInFullDPSAttr.ReplaceAllString(string(xml), "")
	return []byte(strings.ReplaceAll(s, "<Skill ", `<Skill includeInFullDPS="true" `))
}

func readFullDPSOracle(r io.Reader) ([]fullDPSOracleRow, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 16<<20), 16<<20)
	var rows []fullDPSOracleRow
	for sc.Scan() {
		var row fullDPSOracleRow
		if err := json.Unmarshal(sc.Bytes(), &row); err != nil {
			return nil, fmt.Errorf("row %d: %w", len(rows)+1, err)
		}
		rows = append(rows, row)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read oracle: %w", err)
	}
	return rows, nil
}

func fullDPSBuild(t testing.TB, tree *tree2.PassiveTree, raw []byte) (*Environment, error) {
	t.Helper()
	build, err := pob2.ParseBuild(includeAllInFullDPS(raw))
	if err != nil {
		return nil, err
	}
	var env *Environment
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		env, err = BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	}()
	return env, err
}

// skillRowKeys are the rows as name, count, skill part and source with the DPS, sorted; PoB's sidebar sorts its
// list in place by dps * count, which leaves ties in no fixed order.
func skillRowKeys(rows []SkillDPS) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = fmt.Sprintf("%s|%g|%s|%s|%s", r.Name, r.Count, r.SkillPart, r.Source, r.Trigger)
	}
	sort.Strings(out)
	return out
}

// fullDPSDiff is the first difference between our Full DPS and the oracle row, or "".
func fullDPSDiff(env *Environment, want fullDPSOracleRow) string {
	out := env.Player.Output
	if want.FullDPS == nil || !poe2OracleClose(out["FullDPS"], *want.FullDPS, poe2OracleTolerance) {
		return fmt.Sprintf("FullDPS %v, PoB %v", out["FullDPS"], fullDPSDeref(want.FullDPS))
	}
	if want.FullDotDPS == nil || !poe2OracleClose(out["FullDotDPS"], *want.FullDotDPS, poe2OracleTolerance) {
		return fmt.Sprintf("FullDotDPS %v, PoB %v", out["FullDotDPS"], fullDPSDeref(want.FullDotDPS))
	}
	wantRows := make([]SkillDPS, len(want.Skills))
	for i, s := range want.Skills {
		wantRows[i] = SkillDPS{Name: s.Name, DPS: s.DPS, Count: s.Count, Trigger: s.Trigger, SkillPart: s.SkillPart, Source: s.Source}
	}
	got, exp := skillRowKeys(env.Player.SkillDPS), skillRowKeys(wantRows)
	if strings.Join(got, "\n") != strings.Join(exp, "\n") {
		return fmt.Sprintf("rows\n  ours %q\n  PoB  %q", got, exp)
	}
	// the oracle's rows are in the order the sidebar's sort leaves them
	sorted := append([]SkillDPS(nil), env.Player.SkillDPS...)
	SortSkillDPS(sorted)
	for i, g := range sorted {
		w := wantRows[i]
		if g.Name != w.Name || g.Count != w.Count || !poe2OracleClose(g.DPS, w.DPS, poe2OracleTolerance) {
			return fmt.Sprintf("row %d %s x%g dps %v, PoB %s x%g dps %v", i+1, g.Name, g.Count, g.DPS, w.Name, w.Count, w.DPS)
		}
	}
	return ""
}

func fullDPSDeref(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

// TestCorpusPoE2FullDPSVsOracle grades Full DPS over a sample of the PoE2 corpus with every socket group included
// against PoB-PoE2 (mise run oracle-poe2-full-dps).
func TestCorpusPoE2FullDPSVsOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 corpus")
	}
	corpusDir := filepath.Join("..", "testdata", "corpus-poe2-0.5")
	oraclePath := filepath.Join("..", "testdata", "oracle-poe2-0.5-fulldps", "full-dps-poe2-0.5.jsonl.gz")
	if _, err := os.Stat(oraclePath); err != nil {
		t.Skipf("no PoE2 Full DPS oracle at %s (mise run oracle-poe2-full-dps)", oraclePath)
	}
	if _, err := os.Stat(corpusDir); err != nil {
		t.Skipf("no PoE2 corpus at %s", corpusDir)
	}
	f, err := os.Open(oraclePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := readFullDPSOracle(gz)
	if err != nil {
		t.Fatal(err)
	}
	tree := loadTestTree(t)

	type result struct {
		skipped, failed       bool
		diff                  string
		nonzero, dot, dotSeen bool
		rowsOK, rows          int
	}
	results := make([]result, len(rows))
	tasks := make(chan int, len(rows))
	for i := range rows {
		tasks <- i
	}
	close(tasks)
	var wg sync.WaitGroup
	for range min(poe2Workers(), len(rows)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range tasks {
				res := &results[i]
				if rows[i].Error != "" {
					res.skipped = true
					continue
				}
				raw, err := os.ReadFile(filepath.Join(corpusDir, rows[i].Build))
				if err != nil {
					res.failed, res.diff = true, err.Error()
					continue
				}
				env, err := fullDPSBuild(t, tree, raw)
				if err != nil {
					res.failed, res.diff = true, err.Error()
					continue
				}
				res.diff = fullDPSDiff(env, rows[i])
				res.nonzero = rows[i].FullDPS != nil && *rows[i].FullDPS != 0
				res.dotSeen = rows[i].FullDotDPS != nil && *rows[i].FullDotDPS != 0
				res.dot = res.dotSeen && poe2OracleClose(env.Player.Output["FullDotDPS"], *rows[i].FullDotDPS, poe2OracleTolerance)
				res.rows = len(rows[i].Skills)
				ours := map[string]int{}
				for _, k := range skillRowKeys(env.Player.SkillDPS) {
					ours[k]++
				}
				for _, s := range rows[i].Skills {
					k := skillRowKeys([]SkillDPS{{Name: s.Name, Count: s.Count, SkillPart: s.SkillPart, Source: s.Source, Trigger: s.Trigger}})[0]
					if ours[k] > 0 {
						ours[k]--
						res.rowsOK++
					}
				}
			}
		}()
	}
	wg.Wait()

	var graded, matched, nonzero, nonzeroOK, skipped, failed, dotSeen, dotOK, rowsOK, rowsAll int
	var shown int
	for i, r := range results {
		switch {
		case r.skipped:
			skipped++
			continue
		case r.failed:
			failed++
		}
		graded++
		rowsOK += r.rowsOK
		rowsAll += r.rows
		if r.dotSeen {
			dotSeen++
			if r.dot {
				dotOK++
			}
		}
		if r.nonzero {
			nonzero++
		}
		if r.diff == "" {
			matched++
			if r.nonzero {
				nonzeroOK++
			}
		} else if shown < 15 {
			shown++
			t.Logf("%s: %s", rows[i].Build, r.diff)
		}
	}
	t.Logf("=== CORPUS PoE2 0.5 Full DPS (every socket group included) vs PoB-PoE2 ===")
	t.Logf("%d graded, %d failed to calculate, %d PoB could not load (left out)", graded, failed, skipped)
	t.Logf("builds: matched %d / %d   nonzero matched %d / %d", matched, graded, nonzeroOK, nonzero)
	t.Logf("FullDotDPS nonzero matched %d / %d   SkillDPS rows (name, count, part, source) matched %d / %d", dotOK, dotSeen, rowsOK, rowsAll)
	if graded == 0 {
		t.Fatal("REFUSING: nothing graded")
	}
}

// fullDPSFixturePath is a fixture build the committed oracle names by its file name.
func fullDPSFixturePath(name string) string {
	for _, dir := range []string{"testdata", filepath.Join("..", "wasm2", "exposition", "testdata")} {
		if p := filepath.Join(dir, name); fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// TestFullDPSMatchesPoB grades Full DPS over the fixture builds with every socket group included against PoB-PoE2
// (mise run full-dps-poe2-fixtures).
func TestFullDPSMatchesPoB(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "testdata", "oracle", "poe2_full_dps.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := readFullDPSOracle(f)
	if err != nil {
		t.Fatal(err)
	}
	tree := loadTestTree(t)
	var withSkills int
	for _, row := range rows {
		t.Run(row.Build, func(t *testing.T) {
			if row.Error != "" {
				t.Fatalf("PoB-PoE2 failed: %s", row.Error)
			}
			path := fullDPSFixturePath(row.Build)
			if path == "" {
				t.Fatalf("no fixture %s", row.Build)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			env, err := fullDPSBuild(t, tree, raw)
			if err != nil {
				t.Fatal(err)
			}
			if diff := fullDPSDiff(env, row); diff != "" {
				t.Error(diff)
			}
		})
		if len(row.Skills) > 0 {
			withSkills++
		}
	}
	if withSkills < 20 {
		t.Fatalf("only %d fixture builds have a Full DPS row; the oracle is not exercising the port", withSkills)
	}
}
