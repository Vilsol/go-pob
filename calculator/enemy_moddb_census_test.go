package calculator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestEnemyModDBCensus diffs OUR enemy ModDB against PoB's, by (mod name, source), per build.
func TestEnemyModDBCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census")
	dumpDir := os.Getenv("DUMPS")
	if dumpDir == "" {
		t.Skip("set DUMPS=<dir of PoB per-build dumps>")
	}
	requireGameData(t, "3.29")

	files, err := filepath.Glob(filepath.Join(dumpDir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: %s holds no .json dumps — empty must mean 'I looked and found "+
			"nothing', never 'I did not look' (%v)", dumpDir, err)
	}
	limit := len(files)
	if v := os.Getenv("LIMIT"); v != "" {
		n, perr := fmt.Sscanf(v, "%d", &limit)
		if n != 1 || perr != nil || limit <= 0 {
			t.Fatalf("REFUSING: LIMIT=%q is not a positive integer", v)
		}
		if limit > len(files) {
			limit = len(files)
		}
	}
	sort.Strings(files)
	files = files[:limit]

	corpus := corpusDirForLoadedData(t)
	type dump struct {
		Build      string `json:"build"`
		EnemyModDB map[string][]struct {
			Source string `json:"source"`
		} `json:"enemyModDB"`
	}

	missingBuilds := map[string]int{}
	extraBuilds := map[string]int{}
	byName := map[string]int{}
	compared, unusable := 0, 0

	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			unusable++
			continue
		}
		var d dump
		if json.Unmarshal(raw, &d) != nil || d.Build == "" || len(d.EnemyModDB) == 0 {
			unusable++
			continue
		}
		xml, err := os.ReadFile(filepath.Join(corpus, d.Build))
		if err != nil {
			unusable++
			continue
		}
		b, err := builds.ParseBuild(xml)
		if err != nil {
			unusable++
			continue
		}
		env := (&Calculator{PoB: b}).BuildOutput(OutputModeMain)
		if env == nil || env.Enemy == nil || env.Enemy.ModDB == nil {
			unusable++
			continue
		}
		compared++

		mine := map[string]bool{}
		for name, mods := range env.Enemy.ModDB.Mods {
			for _, m := range mods {
				mine[name+" | "+string(m.GetSource())] = true
			}
		}
		theirs := map[string]bool{}
		for name, rows := range d.EnemyModDB {
			for _, r := range rows {
				theirs[name+" | "+r.Source] = true
			}
		}
		for k := range theirs {
			if !mine[k] {
				missingBuilds[k]++
				byName[strings.SplitN(k, " | ", 2)[0]]++
			}
		}
		for k := range mine {
			if !theirs[k] {
				extraBuilds[k]++
			}
		}
	}

	if compared == 0 {
		t.Fatalf("REFUSING: compared 0 builds across %d dumps (%d unusable)", len(files), unusable)
	}

	top := func(title string, m map[string]int, n int) {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if m[keys[i]] != m[keys[j]] {
				return m[keys[i]] > m[keys[j]]
			}
			return keys[i] < keys[j]
		})
		fmt.Printf("\n%s (top %d of %d distinct)\n", title, n, len(keys))
		for i, k := range keys {
			if i >= n {
				break
			}
			fmt.Printf("  %5d  %s\n", m[k], k)
		}
	}

	fmt.Printf("=== ENEMY MODDB CENSUS === %d builds compared, %d unusable\n", compared, unusable)
	top("PoB HAS IT, WE DO NOT — builds affected, by (name | source)", missingBuilds, 30)
	top("WE HAVE IT, PoB DOES NOT — the over-application direction", extraBuilds, 15)
	top("PoB HAS IT, WE DO NOT — collapsed to the MOD NAME", byName, 25)
	fmt.Printf("\n⚠️ A source-string mismatch (\"Config\" vs \"EnemyConfig\") shows in BOTH lists and " +
		"is cosmetic. Read the NAME table for whether a value is actually absent.\n")
}
