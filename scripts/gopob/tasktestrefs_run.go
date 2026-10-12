package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ttCheckNameSet(root string, resolved map[string]bool, updateNames bool) int {
	namesPath := filepath.Join(root, ttNamesFile)
	if updateNames {
		var b strings.Builder
		for _, n := range ttSortedKeys(resolved) {
			b.WriteString(n + "\n")
		}
		ttWriteText(namesPath, b.String(), "the name-set baseline")
		fmt.Printf("recorded %d resolved name(s) in %s\n", len(resolved), ttNamesFile)
		return 0
	}
	if st, err := os.Stat(namesPath); err != nil || !st.Mode().IsRegular() {
		fmt.Fprintf(os.Stderr, "REFUSING: %s is missing; the name-set guard cannot "+
			"answer. Run with --update-names to seed it.\n", ttNamesFile)
		return 3
	}
	baseline := map[string]bool{}
	for _, ln := range ttSplitLines(ttReadText(namesPath, "the name-set baseline")) {
		if s := strings.TrimSpace(ln); s != "" {
			baseline[s] = true
		}
	}
	if len(baseline) == 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: %s is empty, so the guard would pass vacuously.\n",
			ttNamesFile)
		return 3
	}
	var lost, gained []string
	for n := range baseline {
		if !resolved[n] {
			lost = append(lost, n)
		}
	}
	for n := range resolved {
		if !baseline[n] {
			gained = append(gained, n)
		}
	}
	sort.Strings(lost)
	sort.Strings(gained)
	if len(gained) > 0 {
		fmt.Printf("name set GAINED %d: %s\n", len(gained), strings.Join(gained, ", "))
		fmt.Printf("  -> record with: go run ./scripts/gopob task-test-refs --update-names\n")
	}
	if len(lost) > 0 {
		fmt.Fprintf(os.Stderr,
			"\nNAME SET SHRANK: %d name(s) no longer referenced by any live command\n\n", len(lost))
		for _, n := range lost {
			fmt.Fprintf(os.Stderr, "  %s\n", n)
		}
		fmt.Fprint(os.Stderr,
			"\nA name leaving this set means some driver stopped naming it — usually because a\n"+
				"refactor moved the literal behind a variable, which narrows this checker without\n"+
				"failing anything. Follow the value to its new home, or record the loss deliberately\n"+
				"with --update-names.\n")
		return 1
	}
	return 0
}

func ttSortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

var ttPlanted []string

func ttSelftestDir(root string) string {
	d := filepath.Join(root, "scripts")
	if v := os.Getenv("SELFTEST_DIR"); v != "" {
		abs, err := filepath.Abs(v)
		if err == nil {
			d = abs
		} else {
			d = v
		}
		if resolved, err := filepath.EvalSymlinks(d); err == nil {
			d = resolved
		}
	}
	if err := os.MkdirAll(d, 0o750); err != nil { //nolint:gosec // SELFTEST_DIR is the operator's, as in the original
		ttRefuse(fmt.Sprintf("%s cannot be created for the selftest plant: %s", d, errnoText(err)))
	}
	return d
}

func ttUnplant() {
	for _, f := range ttPlanted {
		_ = os.Remove(f)
	}
	ttPlanted = nil
}

func ttRelToRoot(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return p
	}
	return filepath.ToSlash(rel)
}

func ttResolves(tests map[string]bool, name string) bool {
	for t := range tests {
		if t == name || strings.HasPrefix(t, name) {
			return true
		}
	}
	return false
}

type ttFinding struct{ rel, name string }

func ttRun(root string, updateNames bool) int {
	tests := ttKnownTests(root)
	if len(tests) == 0 {
		fmt.Fprintln(os.Stderr, "REFUSING: found no Test functions at all; the scan is broken")
		return 3
	}

	driverFiles := ttDrivers(root)
	if len(driverFiles) == 0 {
		fmt.Fprintln(os.Stderr, "REFUSING: discovered no drivers to scan")
		return 3
	}
	selftest := os.Getenv("SELFTEST") != ""
	if selftest {
		plant := filepath.Join(ttSelftestDir(root), "zz_task_test_refs_selftest.sh")
		ttWriteText(plant, "go test ./calculator/ -run TestThisDoesNotExistAnywhere\n",
			"the selftest plant")
		ttPlanted = append(ttPlanted, plant)
		driverFiles = append(driverFiles, plant)
	}

	var bad, mentions []ttFinding
	resolved := map[string]bool{}
	checked := 0

	gateTables := []string{""}
	if selftest {
		planted := filepath.Join(ttSelftestDir(root), "zz_gate_versions_selftest.tsv")
		ttWriteText(planted, "9.99\tTestGateThatDoesNotExist\tTestNoHardcodedVersionPaths\t1\n",
			"the planted gate-version table")
		ttPlanted = append(ttPlanted, planted)
		gateTables = append(gateTables, planted)
	}
	for _, table := range gateTables {
		for _, name := range ttGateVersionTests(root, table) {
			checked++
			if !ttResolves(tests, name) {
				where := "gate-versions"
				if table != "" {
					where = ttRelToRoot(root, table)
				}
				bad = append(bad, ttFinding{where, name})
			} else {
				resolved[name] = true
			}
		}
	}
	for _, rel := range driverFiles {
		path := ttJoinRoot(root, rel)
		if _, err := os.Stat(path); err != nil {
			fmt.Fprintf(os.Stderr, "REFUSING: %s is missing; this check cannot answer\n", rel)
			return 3
		}
		for _, ref := range ttReferenced(rel, ttReadText(path, "a discovered driver")) {
			if !ref.live {
				if !ttResolves(tests, ref.name) {
					mentions = append(mentions, ttFinding{rel, ref.name})
				}
				continue
			}
			checked++
			if !ttResolves(tests, ref.name) {
				bad = append(bad, ttFinding{rel, ref.name})
			} else {
				resolved[ref.name] = true
			}
		}
	}

	if selftest {
		if len(bad) == 0 {
			fmt.Fprintln(os.Stderr, "SELFTEST FAILED: the planted dead reference was not caught")
			return 3
		}
		fmt.Printf("SELFTEST OK: caught %d planted dead reference(s)\n", len(bad))
		return 0
	}

	if len(bad) > 0 {
		fmt.Fprintf(os.Stderr, "DEAD TEST REFERENCES: %d\n\n", len(bad))
		for _, f := range bad {
			fmt.Fprintf(os.Stderr, "  %s: -run %s matches no test function\n", f.rel, f.name)
		}
		fmt.Fprint(os.Stderr,
			"\n`go test -run` matching nothing exits 0 having evaluated nothing, so this driver\n"+
				"reports on an empty set rather than failing. Point it at a live test or delete it.\n"+
				"If the name is deliberately historical, express it as a MENTION — an inline `code\n"+
				"span` in markdown prose, or a # comment in a script — not as a command.\n")
		return 1
	}

	if len(mentions) > 0 {
		fmt.Printf("historical mentions (not failures): %d\n", len(mentions))
		for _, f := range mentions {
			fmt.Printf("  %s: %s named in prose\n", f.rel, f.name)
		}
	}

	if rc := ttCheckNameSet(root, resolved, updateNames); rc != 0 {
		return rc
	}
	fmt.Printf("task test refs OK: %d -run use(s) across %d discovered driver(s), "+
		"%d test functions, %d historical mention(s)\n",
		checked, len(driverFiles), len(tests), len(mentions))
	return 0
}

func ttJoinRoot(root, rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(root, rel)
}

func taskTestRefs(argv []string) int {
	updateNames := false
	for _, a := range argv {
		if a == "--update-names" {
			updateNames = true
		}
	}
	defer ttUnplant()
	return ttRun(pcRepoRoot(), updateNames)
}
