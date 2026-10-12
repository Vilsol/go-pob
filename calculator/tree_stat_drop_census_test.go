package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/moddb"
)

// TestTreeStatDropCensus ranks the passive-tree stat lines that the mod parser fails to consume.
func TestTreeStatDropCensus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	{
		probe := &Environment{DebugErrors: make([]string, 0)}
		processStats(probe, moddb.NewModList(), "0", "Control", []string{"+10 to Strength"})
		if len(probe.DebugErrors) != 0 {
			t.Fatalf("REFUSING: a line that parses cleanly produced %d DebugErrors (%v); this "+
				"census would report false drops", len(probe.DebugErrors), probe.DebugErrors)
		}
		processStats(probe, moddb.NewModList(), "0", "Control", []string{"Zz Not A Real Stat Line Zz"})
		if len(probe.DebugErrors) != 1 {
			t.Fatalf("REFUSING: an unparseable line produced %d DebugErrors, want 1; the census "+
				"cannot see a drop and would report a clean bill of health", len(probe.DebugErrors))
		}
	}

	corpusDir, names := corpusNamesForLoadedData(t)

	type drop struct {
		occurrences int
		builds      map[string]bool
		extra       string
		kind        string
	}

	var mu sync.Mutex
	drops := map[string]*drop{}
	byKind := map[string]int{}
	builtCount := 0

	runConcurrently(len(names), 12, func(i int, cache *EnvironmentCache) {
		name := names[i]
		file, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			return
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			return
		}
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		if env == nil {
			return
		}

		mu.Lock()
		builtCount++
		for _, e := range env.DebugErrors {
			kind, extra, line, ok := parseTreeStatDebugError(e)
			if !ok {
				continue
			}
			byKind[kind]++
			d, ok := drops[line]
			if !ok {
				d = &drop{builds: map[string]bool{}, extra: extra, kind: kind}
				drops[line] = d
			}
			d.occurrences++
			d.builds[name] = true
		}
		mu.Unlock()
	})

	if builtCount < len(names)/2 {
		t.Fatalf("REFUSING: only %d of %d builds produced an environment; this census would be "+
			"measuring a build failure, not a parser gap", builtCount, len(names))
	}

	type row struct {
		line        string
		kind        string
		extra       string
		occurrences int
		builds      int
	}
	rows := make([]row, 0, len(drops))
	totalOccurrences := 0
	for line, d := range drops {
		rows = append(rows, row{line, d.kind, d.extra, d.occurrences, len(d.builds)})
		totalOccurrences += d.occurrences
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].occurrences != rows[j].occurrences {
			return rows[i].occurrences > rows[j].occurrences
		}
		return rows[i].line < rows[j].line
	})

	t.Logf("TREE STAT DROP CENSUS over %d builds (of %d in the corpus)", builtCount, len(names))
	t.Logf("  distinct dropped lines: %d", len(rows))
	t.Logf("  total drop occurrences: %d", totalOccurrences)
	kinds := make([]string, 0, len(byKind))
	for k := range byKind {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(i, j int) bool { return byKind[kinds[i]] > byKind[kinds[j]] })
	for _, k := range kinds {
		t.Logf("  %-40s %6d occurrences", k, byKind[k])
	}

	t.Log("TOP 60 DROPPED LINES BY OCCURRENCE")
	t.Logf("  %6s %6s  %s", "occ", "builds", "line -> leftover Extra")
	for i, r := range rows {
		if i >= 60 {
			break
		}
		t.Logf("  %6d %6d  %q -> %q", r.occurrences, r.builds, r.line, r.extra)
	}

	if dir := os.Getenv("CENSUS_OUT"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("CENSUS_OUT: %v", err)
		}
		var b strings.Builder
		b.WriteString("occurrences\tbuilds\tkind\tline\textra\n")
		for _, r := range rows {
			fmt.Fprintf(&b, "%d\t%d\t%s\t%s\t%s\n", r.occurrences, r.builds, r.kind, r.line, r.extra)
		}
		if err := os.WriteFile(filepath.Join(dir, "tree_stat_drops.tsv"), []byte(b.String()), 0o600); err != nil {
			t.Fatalf("CENSUS_OUT: %v", err)
		}
		t.Logf("wrote %s", filepath.Join(dir, "tree_stat_drops.tsv"))
	}
}

func parseTreeStatDebugError(msg string) (string, string, string, bool) {
	var kind, extra, line string
	const prefix = "Error parsing "
	if !strings.HasPrefix(msg, prefix) {
		return "", "", "", false
	}
	rest := msg[len(prefix):]
	sep := strings.Index(rest, " mod: ")
	if sep < 0 {
		return "", "", "", false
	}
	kind = rest[:sep]
	rest = rest[sep+len(" mod: "):]

	statMarker := strings.LastIndex(rest, ", stat text: ")
	if statMarker < 0 {
		return "", "", "", false
	}
	extra = strings.TrimSpace(rest[:statMarker])
	rest = rest[statMarker+len(", stat text: "):]

	withMarker := strings.LastIndex(rest, ", with ")
	if withMarker < 0 {
		return "", "", "", false
	}
	line = rest[:withMarker]

	if p := strings.Index(kind, " ("); p >= 0 {
		kind = kind[:p]
	}
	return kind, extra, line, true
}
