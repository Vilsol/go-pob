package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestDiagBuildSources diffs the SET of mod SOURCES between PoB's dump and ours for one build and one actor.
func TestDiagBuildSources(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	dir := diagDumpsDir(t)
	name := os.Getenv("BUILD")
	if name == "" {
		t.Skip("BUILD unset")
	}
	actor := os.Getenv("ACTOR")
	if actor == "" {
		actor = "player"
	}

	dump, healthy := diagLoadDump(dir, name)
	if dump == nil {
		t.Fatalf("no dump for %s in %s", name, dir)
	}
	requireHealthyDump(t, name, healthy)
	file, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatal(err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	pobMods, ourDB, label := diagActorModDB(dump, env, actor)

	names := map[string]string{}
	pob := map[string]bool{}
	for _, mods := range pobMods {
		for _, m := range mods {
			key := normalizeSource(m.Source)
			pob[key] = true
			if i := strings.Index(m.Source, "("); i >= 0 {
				names[key] = strings.TrimSuffix(m.Source[i+1:], ")")
			}
		}
	}
	ours := map[string]bool{}
	if ourDB != nil {
		for _, mods := range ourDB.Mods {
			for _, m := range mods {
				ours[normalizeSource(string(m.GetSource()))] = true
			}
		}
	}

	fmt.Printf("\n=== SOURCE SET / %s / actor=%s (healthy dump=%v) — pob %d, ours %d ===\n",
		name, label, healthy, len(pob), len(ours))
	if ourDB == nil {
		fmt.Printf("we build no %s actor for this build; every PoB source below is unreachable\n", label)
	}

	if ourDB != nil {
		var placeholderMods, totalMods int
		for _, mods := range ourDB.Mods {
			for _, m := range mods {
				totalMods++
				if s := normalizeSource(string(m.GetSource())); s == "" || strings.Contains(s, "<none>") {
					placeholderMods++
				}
			}
		}
		if placeholderMods > 0 {
			fmt.Printf("⚠️  ATTRIBUTION: %d of our %d mods carry the placeholder source `Other:<none>`.\n"+
				"    Those mods EXIST but we did not record where they came from, so PoB's real source\n"+
				"    for each shows up below as ONLY-POB. Treat ONLY-POB as an UPPER BOUND on missing\n"+
				"    mods, never as a count — and on the minion actor prefer TestDiagBuildMods\n"+
				"    ACTOR=%s, which is source-blind when that fits better.\n",
				placeholderMods, totalMods, actor)
		}
	}

	report := func(title string, have, lack map[string]bool) int {
		var rows []string
		for s := range have {
			if !lack[s] {
				if n, ok := names[s]; ok {
					rows = append(rows, fmt.Sprintf("%-44s %s", s, n))
					continue
				}
				rows = append(rows, s)
			}
		}
		sort.Strings(rows)
		fmt.Printf("%s (%d):\n", title, len(rows))
		for _, r := range rows {
			fmt.Println("  ", r)
		}
		return len(rows)
	}

	onlyPoB := report("ONLY POB — a source we never built, so every mod on it is missing", pob, ours)
	onlyOurs := report("ONLY OURS — a source PoB never built, so every mod on it is invented", ours, pob)
	if onlyPoB == 0 && onlyOurs == 0 {
		fmt.Println("source sets are identical; the divergence is inside a shared source — use TestDiagBuildMods NAME='*'")
	}
}
