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

// The PoE2 corpus gate's summary lines, cached by the committed gate inputs (every tracked file but the frontend, desktop, CI, Nix and docs) and the corpus and oracle it read, so a base is
// measured once per commit rather than once per change on top of it. A dirty tree is measured fresh and never stored: the tree hash
// cannot see uncommitted work.

// what is run as "the PoE2 corpus gate": every PoE2 corpus test, the authoritative TestCorpusPoE2VsOracle among them
const gp2DefaultRun = "TestCorpusPoE2"

// the local PoE2 corpus, oracle and dump links the tests read
const gp2Inputs = "testdata/*poe2*"

var gp2Summary = regexp.MustCompile(`^=== [^RPCN]|^=== (?:CORPUS|every)|^\d+ (?:graded|builds)|matched \d+ / \d+|^first failure: |above the floor|^\s*--- (?:PASS|FAIL|SKIP)|^(?:ok|FAIL)\s|^[A-Z_]+: \d+ builds`)

func gp2Git(repo string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // git with this tool's own arguments
	cmd.Dir = repo
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// gp2DataKey names the PoE2 corpus, oracle and dumps by where they resolve to, their size and modification time and
// entry count, and for a small directory (an oracle) each entry's size and time
func gp2DataKey(repo string) (string, error) {
	links, err := filepath.Glob(filepath.Join(repo, gp2Inputs))
	if err != nil || len(links) == 0 {
		return "", fmt.Errorf("nothing matches %s", gp2Inputs)
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
	repo    string
	pattern string
	out     io.Writer
	// runs the gate and returns its whole output
	run func() (string, error)
}

func gp2RunGoTest(repo, pattern string) func() (string, error) {
	return func() (string, error) {
		cmd := exec.CommandContext(context.Background(), "go", "test", "./calculator2/", "-run", pattern, "-count=1", "-v", "-timeout", "60m") //nolint:gosec // the caller's -run pattern
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "CORPUS=true")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
}

func (e gp2Env) gate(force bool) int {
	tree, err := gateInputsKey(e.repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING: gate-poe2: no HEAD inputs: "+err.Error())
		return root.RefusalCode
	}
	data, err := gp2DataKey(e.repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING: gate-poe2: the local PoE2 corpus is missing, so there is no number to report: "+err.Error())
		return root.RefusalCode
	}
	status, err := gp2Git(e.repo, "status", "--porcelain")
	if err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING: gate-poe2: git status: "+err.Error())
		return root.RefusalCode
	}
	common, err := gp2Git(e.repo, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING: gate-poe2: no git dir: "+err.Error())
		return root.RefusalCode
	}
	dir := filepath.Join(common, "gatecache-poe2")
	key := gp2Key(tree, e.pattern, data)
	file := filepath.Join(dir, key+".txt")
	dirty := status != ""

	if !dirty && !force {
		if b, err := os.ReadFile(file); err == nil && len(b) > 0 { //nolint:gosec // this repository's own cache
			_, _ = e.out.Write(b)
			_, _ = fmt.Fprintf(e.out, "(cached) -run %s measured at gate inputs %s; FORCE_GATES=1 or --force re-runs; whole output %s\n", e.pattern, tree[:12], filepath.Join(dir, key+".log"))
			return 0
		}
	}
	if dirty {
		_, _ = fmt.Fprintln(e.out, "(uncached) the working tree is dirty: measuring fresh, not storing")
	}

	out, runErr := e.run()
	body := strings.Join(gp2Filter(out), "\n") + "\n"
	_, _ = io.WriteString(e.out, body)
	if runErr != nil {
		log := filepath.Join(os.TempDir(), "gate-poe2-"+tree[:12]+".log")
		_ = os.WriteFile(log, []byte(out), 0o600)
		_, _ = fmt.Fprintf(e.out, "GATE FAILED: %v (not stored); the whole output is in %s\n", runErr, log)
		return 1
	}
	if !strings.Contains(body, "matched ") {
		fmt.Fprintln(os.Stderr, "REFUSING: gate-poe2: the gate printed no matched line (did it skip?); nothing stored")
		return root.RefusalCode
	}
	if dirty {
		return 0
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		fmt.Fprintln(os.Stderr, "gate-poe2: not stored: "+err.Error())
		return 0
	}
	_ = os.WriteFile(filepath.Join(dir, key+".log"), []byte(out), 0o600)
	if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "gate-poe2: not stored: "+err.Error())
	}
	_, _ = fmt.Fprintf(e.out, "(stored) tree %s\n", tree[:12])
	return 0
}

func gatePoE2(argv []string) int {
	force := os.Getenv("FORCE_GATES") != ""
	pattern := gp2DefaultRun
	for i := 0; i < len(argv); i++ {
		switch a := argv[i]; {
		case a == "--force":
			force = true
		case a == "--run" && i+1 < len(argv):
			i++
			pattern = argv[i]
		default:
			fmt.Fprintln(os.Stderr, "REFUSING: gate-poe2: unknown argument "+a+"; usage: gate-poe2 [--force] [--run <go test -run pattern>]")
			return 64
		}
	}
	repo, err := root.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING: gate-poe2: "+err.Error())
		return root.RefusalCode
	}
	return gp2Env{repo: repo, pattern: pattern, out: os.Stdout, run: gp2RunGoTest(repo, pattern)}.gate(force)
}
