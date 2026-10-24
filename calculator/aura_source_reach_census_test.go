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

// TestAuraSourceReachCensus answers one question over a LIST of builds at once.
func TestAuraSourceReachCensus(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireOperatorRun(t, "report-only census")
	dir := diagDumpsDir(t)
	source := os.Getenv("SOURCE")
	if source == "" {
		source = "Skill:Envy"
	}
	corpus := corpusDirForLoadedData(t)

	var names []string
	if list := os.Getenv("BUILDS"); list != "" {
		raw, err := os.ReadFile(list)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			names = append(names, strings.TrimSuffix(line, ".xml"))
		}
	} else {
		entries, err := os.ReadDir(corpus)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".xml") {
				names = append(names, strings.TrimSuffix(e.Name(), ".xml"))
			}
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatalf("no builds selected (BUILDS=%q, corpus=%s)", os.Getenv("BUILDS"), corpus)
	}

	var both, pobOnly, oursOnly, neither, skipped int
	for _, name := range names {
		dump, healthy := diagLoadDump(dir, name)
		if dump == nil || !healthy {
			skipped++
			continue
		}
		file, err := os.ReadFile(filepath.Join(corpus, name+".xml"))
		if err != nil {
			skipped++
			continue
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			skipped++
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		pobMods, ourDB, _ := diagActorModDB(dump, env, "player")

		pobHas := false
		for _, mods := range pobMods {
			for _, m := range mods {
				if normalizeSource(m.Source) == source {
					pobHas = true
				}
			}
		}
		oursHas := false
		if ourDB != nil {
			for _, mods := range ourDB.Mods {
				for _, m := range mods {
					if normalizeSource(string(m.GetSource())) == source {
						oursHas = true
					}
				}
			}
		}
		switch {
		case pobHas && oursHas:
			both++
		case pobHas:
			pobOnly++
			fmt.Printf("POB-ONLY %s\n", name)
		case oursHas:
			oursOnly++
			fmt.Printf("OURS-ONLY %s\n", name)
		default:
			neither++
		}
	}
	fmt.Printf("\n=== %s on the PLAYER over %d builds: both %d, POB-ONLY %d, OURS-ONLY %d, neither %d, skipped %d ===\n",
		source, len(names), both, pobOnly, oursOnly, neither, skipped)
}
