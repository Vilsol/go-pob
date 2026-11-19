package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

// A corpus gate's summary lines, cached by its package's dependency closure (gateClosure) and the corpus and oracle it
// read, so a base is measured once per closure rather than once per commit. Uncommitted work inside the closure is
// measured fresh and never stored: the key cannot see it.

// what is run as "the PoE2 corpus gate": every PoE2 corpus test, the authoritative TestCorpusPoE2VsOracle among them
const gp2DefaultRun = "TestCorpusPoE2"

type gateGame struct {
	name, pkg, run, cache string
	// the local corpus, oracle and dump links the tests read
	data []string
}

var (
	gateGamePoE2 = gateGame{name: "gate-poe2", pkg: "./calculator2/", run: gp2DefaultRun, cache: "gatecache-poe2", data: []string{"testdata/*poe2*"}}
	gateGamePoE1 = gateGame{name: "gate-cached", pkg: "./calculator/", run: "TestCorpus3_29VsOracle", cache: "gatecache-poe1", data: []string{"testdata/corpus-3.*", "testdata/oracle-3.*"}}
)

var gp2Summary = regexp.MustCompile(`^=== [^RPCN]|^=== (?:CORPUS|every)|^\d+ (?:graded|builds)|matched \d+ / \d+|^first failure: |above the floor|^\s*--- (?:PASS|FAIL|SKIP)|^(?:ok|FAIL)\s|^[A-Z_]+: \d+ builds`)

func gp2Git(repo string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // git with this tool's own arguments
	cmd.Dir = repo
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// gp2DataKey names the corpus, oracle and dumps by where they resolve to, their size and modification time and
// entry count, and for a small directory (an oracle) each entry's size and time
func gp2DataKey(repo string, globs []string) (string, error) {
	var links []string
	for _, g := range globs {
		m, _ := filepath.Glob(filepath.Join(repo, g))
		links = append(links, m...)
	}
	if len(links) == 0 {
		return "", fmt.Errorf("nothing matches %s", strings.Join(globs, " "))
	}
	var b strings.Builder
	for _, link := range links {
		p, err := filepath.EvalSymlinks(link)
		if err != nil {
			return "", fmt.Errorf("%s: %w", link, err)
		}
		fi, err := os.Stat(p)
		if err != nil {
			return "", fmt.Errorf("%s: %w", link, err)
		}
		fmt.Fprintf(&b, "%s\t%s\t%d\t%d\n", filepath.Base(link), p, fi.Size(), fi.ModTime().UnixNano())
		if !fi.IsDir() {
			continue
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return "", fmt.Errorf("%s: %w", link, err)
		}
		fmt.Fprintf(&b, "\t%d entries\n", len(entries))
		if len(entries) > 64 {
			continue
		}
		for _, e := range entries {
			if ei, err := e.Info(); err == nil {
				fmt.Fprintf(&b, "\t%s\t%d\t%d\n", e.Name(), ei.Size(), ei.ModTime().UnixNano())
			}
		}
	}
	return b.String(), nil
}

func gp2Key(tree, run, data string) string {
	h := sha256.Sum256([]byte(tree + "\n" + run + "\n" + data))
	return hex.EncodeToString(h[:12])
}

// gp2Filter returns the summary lines of the gate's output
func gp2Filter(out string) []string {
	var keep []string
	for _, ln := range strings.Split(out, "\n") {
		if gp2Summary.MatchString(ln) {
			keep = append(keep, ln)
		}
	}
	return keep
}

type gp2Env struct {
	game    gateGame
	repo    string
	pattern string
	out     io.Writer
	// runs the gate and returns its whole output
	run     func() (string, error)
	closure func() (gateClosure, error)
}

func gp2RunGoTest(repo, pkg, pattern string) func() (string, error) {
	return func() (string, error) {
		cmd := exec.CommandContext(context.Background(), "go", "test", pkg, "-run", pattern, "-count=1", "-v", "-timeout", "60m") //nolint:gosec // the caller's -run pattern
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "CORPUS=true")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
}

func (e gp2Env) gate(force bool) int {
	name := e.game.name
	closure, err := e.closure()
	if err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING: "+name+": no dependency closure: "+err.Error())
		return root.RefusalCode
	}
	tree, err := closure.key(e.repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING: "+name+": no HEAD inputs: "+err.Error())
		return root.RefusalCode
	}
	data, err := gp2DataKey(e.repo, e.game.data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING: "+name+": the local corpus is missing, so there is no number to report: "+err.Error())
		return root.RefusalCode
	}
	dirty, err := closure.dirty(e.repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING: "+name+": git status: "+err.Error())
		return root.RefusalCode
	}
	common, err := gp2Git(e.repo, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING: "+name+": no git dir: "+err.Error())
		return root.RefusalCode
	}
	dir := filepath.Join(common, e.game.cache)
	key := gp2Key(tree, e.pattern, data)
	file := filepath.Join(dir, key+".txt")

	if !dirty && !force {
		if b, err := os.ReadFile(file); err == nil && len(b) > 0 { //nolint:gosec // this repository's own cache
			_, _ = e.out.Write(b)
			_, _ = fmt.Fprintf(e.out, "(cached) -run %s measured at dependency closure %s; FORCE_GATES=1 or --force re-runs; whole output %s\n", e.pattern, tree[:12], filepath.Join(dir, key+".log"))
			return 0
		}
	}
	if dirty {
		_, _ = fmt.Fprintln(e.out, "(uncached) uncommitted work touches "+e.game.pkg+"'s dependency closure: measuring fresh, not storing")
	}

	out, runErr := e.run()
	body := strings.Join(gp2Filter(out), "\n") + "\n"
	_, _ = io.WriteString(e.out, body)
	if runErr != nil {
		log := filepath.Join(os.TempDir(), name+"-"+tree[:12]+".log")
		_ = os.WriteFile(log, []byte(out), 0o600)
		_, _ = fmt.Fprintf(e.out, "GATE FAILED: %v (not stored); the whole output is in %s\n", runErr, log)
		return 1
	}
	if !strings.Contains(body, "matched ") {
		fmt.Fprintln(os.Stderr, "REFUSING: "+name+": the gate printed no matched line (did it skip?); nothing stored")
		return root.RefusalCode
	}
	if dirty {
		return 0
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		fmt.Fprintln(os.Stderr, name+": not stored: "+err.Error())
		return 0
	}
	_ = os.WriteFile(filepath.Join(dir, key+".log"), []byte(out), 0o600)
	if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, name+": not stored: "+err.Error())
	}
	_, _ = fmt.Fprintf(e.out, "(stored) dependency closure %s\n", tree[:12])
	return 0
}

func gatePoE2(argv []string) int { return gateCachedMain(gateGamePoE2, argv) }

func gateCached(argv []string) int { return gateCachedMain(gateGamePoE1, argv) }

func gateCachedMain(g gateGame, argv []string) int {
	force := os.Getenv("FORCE_GATES") != ""
	pattern := g.run
	for i := 0; i < len(argv); i++ {
		switch a := argv[i]; {
		case a == "--force":
			force = true
		case a == "--run" && i+1 < len(argv):
			i++
			pattern = argv[i]
		default:
			fmt.Fprintln(os.Stderr, "REFUSING: "+g.name+": unknown argument "+a+"; usage: "+g.name+" [--force] [--run <go test -run pattern>]")
			return 64
		}
	}
	repo, err := root.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING: "+g.name+": "+err.Error())
		return root.RefusalCode
	}
	return gp2Env{game: g, repo: repo, pattern: pattern, out: os.Stdout, run: gp2RunGoTest(repo, g.pkg, pattern),
		closure: func() (gateClosure, error) { return gateClosureOf(repo, g.pkg) }}.gate(force)
}
