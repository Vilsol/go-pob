package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	rpDrift = regexp.MustCompile(
		`KNOWN DIVERGENCE (\S+) (\S+) = ([-\d.eE+]+), pinned at go-pob's ([-\d.eE+]+)`)
	rpClosed = regexp.MustCompile(
		`KNOWN DIVERGENCE (\S+) (\S+) now MATCHES (?:the pinned )?PoB \(([-\d.eE+]+)\)`)
	rpANSI   = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	rpFunc   = regexp.MustCompile(`(?m)^func (Test\w+)\(`)
	rpAssert = regexp.MustCompile(`assertDivergesFromOracle(Dump)?\(`)
)

func rpRepoRoot() (string, bool) {
	out, ok, _ := awGit("", "rev-parse", "--show-toplevel")
	return out, ok
}

func rpPinFiles(root string) []string {
	var out []string
	//nolint:gosec // the repo root comes from `git rev-parse`, as in the original
	_ = filepath.WalkDir(filepath.Join(root, "calculator"), func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr // grep -r skips what it cannot read, silently
		}
		b, err := os.ReadFile(p) //nolint:gosec // a path this walk enumerated
		if err == nil && rpAssert.Match(b) {
			out = append(out, p)
		}
		return nil
	})
	return out
}

func rpRunPins(root string) string {
	names := rpPinFiles(root)
	if len(names) == 0 {
		return ""
	}
	funcs := map[string]bool{}
	for _, path := range names {
		b, err := os.ReadFile(path) //nolint:gosec // a path this walk enumerated
		if err != nil {
			continue
		}
		src := string(b)
		locs := rpFunc.FindAllStringSubmatchIndex(src, -1)
		for i, m := range locs {
			end := len(src)
			if i+1 < len(locs) {
				end = locs[i+1][0]
			}
			if strings.Contains(src[m[1]:end], "assertDivergesFromOracle") {
				funcs[src[m[2]:m[3]]] = true
			}
		}
	}
	if len(funcs) == 0 {
		return ""
	}
	sorted := make([]string, 0, len(funcs))
	for f := range funcs {
		sorted = append(sorted, f)
	}
	sort.Strings(sorted)

	cmd := exec.Command("go", "test", "-count=1", "-v", "./calculator/", //nolint:gosec,noctx // as in the original
		"-run", strings.Join(sorted, "|"))
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CORPUS=true")
	out, _ := cmd.CombinedOutput()
	return rpANSI.ReplaceAllString(string(out), "")
}

type rpKey struct{ build, stat string }

func repin(argv []string) int {
	_ = argv

	root, ok := rpRepoRoot()
	if !ok {
		fmt.Fprintln(os.Stderr,
			"REFUSING: not inside a git checkout; repin rewrites pins in tracked files")
		return 1
	}
	check := os.Getenv("CHECK")
	dryRun := check != "" && check != "0" && check != "false"

	out := rpRunPins(root)
	if out == "" {
		fmt.Println("no divergence pins found")
		return 0
	}

	drifted := map[rpKey][2]string{}
	order := []rpKey{}
	for _, m := range rpDrift.FindAllStringSubmatch(out, -1) {
		k := rpKey{m[1], m[2]}
		if _, seen := drifted[k]; !seen {
			order = append(order, k)
		}
		drifted[k] = [2]string{m[4], m[3]}
	}
	closed := rpClosed.FindAllStringSubmatch(out, -1)

	if len(drifted) == 0 && len(closed) == 0 {
		fmt.Println("all divergence pins are on their measured values — nothing to do")
		return 0
	}

	changed := 0
	for _, path := range rpPinFiles(root) {
		b, err := os.ReadFile(path) //nolint:gosec // a path this walk enumerated
		if err != nil {
			continue
		}
		src := string(b)
		orig := src
		for _, k := range order {
			v := drifted[k]
			pat := regexp.MustCompile(
				`(assertDivergesFromOracle(?:Dump)?\(\s*t,\s*"` + regexp.QuoteMeta(k.build) +
					`",\s*"` + regexp.QuoteMeta(k.stat) + `",\s*)([-\d.eE+]+)`)
			n := 0
			src = pat.ReplaceAllStringFunc(src, func(m string) string {
				n++
				return pat.FindStringSubmatch(m)[1] + v[1]
			})
			if n > 0 {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					rel = path
				}
				fmt.Printf("  %s: %s %s  %s -> %s\n", rel, k.build, k.stat, v[0], v[1])
				changed += n
			}
		}
		if src != orig && !dryRun {
			_ = os.WriteFile(path, []byte(src), 0o600) //nolint:gosec // rewriting a tracked source file in place
		}
	}

	for _, c := range closed {
		fmt.Printf("  CLOSED (not rewritten): %s %s now matches PoB (%s)\n", c[1], c[2], c[3])
		fmt.Println("    promote the call to assertMatchesOracle and say in the comment what closed it")
	}

	if dryRun {
		fmt.Printf("\nCHECK=1: %d pin(s) would be rewritten, %d closed; nothing written\n",
			changed, len(closed))
	} else {
		fmt.Printf("\nrewrote %d pin(s); %d closed pin(s) need a human\n", changed, len(closed))
		if changed > 0 {
			fmt.Println("re-run the pin tests to confirm green, then commit with the measurement in the message")
		}
	}
	return 0
}
