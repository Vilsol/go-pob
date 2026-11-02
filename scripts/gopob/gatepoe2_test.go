package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const gp2Output = `=== RUN   TestCorpusPoE2VsOracle

=== CORPUS PoE2 0.5 vs ORACLE PoE2 0.5 (skills off, defence slice) ===
9990 graded, 0 failed to calculate
matched 180123 / 181000  (99.5155%)   NaN/Inf on our side: 0
  stat                     seen       ok  nonzero  nonzeroOK
  Life                     9990     9980     9990       9980

=== every numeric oracle output (report only; a stat we do not produce counts as 0) ===
matched 900 / 1000  (90.0000%)   nonzero matched 400 / 500  (80.0000%)   12 distinct stats
--- PASS: TestCorpusPoE2VsOracle (41.00s)
`

const (
	gp2Corpus = "testdata/corpus-poe2-0.5-skillsoff"
	gp2Oracle = "testdata/oracle-poe2-0.5/output-poe2-0.5-skillsoff.jsonl.gz"
)

func gp2Repo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-q")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("testdata/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{gp2Corpus, filepath.Dir(gp2Oracle)} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, gp2Oracle), []byte("oracle"), 0o600); err != nil {
		t.Fatal(err)
	}
	run("add", ".gitignore")
	run("commit", "-q", "-m", "base")
	return dir
}

type gp2Fake struct {
	calls int
	out   string
	err   error
}

func (f *gp2Fake) env(repo string, w *bytes.Buffer) gp2Env {
	return gp2Env{repo: repo, pattern: gp2DefaultRun, out: w, run: func() (string, error) { f.calls++; return f.out, f.err }}
}

func TestGatePoE2CachesACleanTreeOnce(t *testing.T) {
	repo := gp2Repo(t)
	f := &gp2Fake{out: gp2Output}
	var first, second bytes.Buffer
	if rc := f.env(repo, &first).gate(false); rc != 0 {
		t.Fatalf("first run rc=%d: %s", rc, first.String())
	}
	if rc := f.env(repo, &second).gate(false); rc != 0 {
		t.Fatalf("second run rc=%d: %s", rc, second.String())
	}
	if f.calls != 1 {
		t.Fatalf("the gate ran %d times over two clean runs, want 1", f.calls)
	}
	if !strings.Contains(second.String(), "(cached)") || !strings.Contains(second.String(), "nonzero matched 400 / 500") {
		t.Fatalf("second run did not serve the stored lines:\n%s", second.String())
	}
	if strings.Contains(second.String(), "Life") {
		t.Fatalf("the per-stat table was stored; only the summary lines belong in the cache:\n%s", second.String())
	}
	var forced bytes.Buffer
	_ = f.env(repo, &forced).gate(true)
	if f.calls != 2 {
		t.Fatalf("--force served the cache")
	}
}

func TestGatePoE2NeverStoresADirtyTree(t *testing.T) {
	repo := gp2Repo(t)
	if err := os.WriteFile(filepath.Join(repo, "new.go"), []byte("package x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &gp2Fake{out: gp2Output}
	for range 2 {
		var w bytes.Buffer
		if rc := f.env(repo, &w).gate(false); rc != 0 {
			t.Fatalf("rc=%d: %s", rc, w.String())
		}
		if !strings.Contains(w.String(), "(uncached)") {
			t.Fatalf("a dirty run did not say so:\n%s", w.String())
		}
	}
	if f.calls != 2 {
		t.Fatalf("the gate ran %d times over two dirty runs, want 2", f.calls)
	}
	if err := os.Remove(filepath.Join(repo, "new.go")); err != nil {
		t.Fatal(err)
	}
	var w bytes.Buffer
	_ = f.env(repo, &w).gate(false)
	if f.calls != 3 {
		t.Fatalf("a dirty run stored its numbers for the clean tree")
	}
}

func TestGatePoE2KeysOnTheDataAndThePattern(t *testing.T) {
	repo := gp2Repo(t)
	f := &gp2Fake{out: gp2Output}
	var w bytes.Buffer
	_ = f.env(repo, &w).gate(false)
	if err := os.WriteFile(filepath.Join(repo, gp2Corpus, "build.xml"), []byte("<x/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = f.env(repo, &w).gate(false)
	if f.calls != 2 {
		t.Fatalf("a changed corpus was served the old corpus's numbers")
	}
	if err := os.WriteFile(filepath.Join(repo, gp2Oracle), []byte("a regenerated oracle"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = f.env(repo, &w).gate(false)
	if f.calls != 3 {
		t.Fatalf("a changed oracle file was served the old oracle's numbers")
	}
	e := f.env(repo, &w)
	e.pattern = "^TestCorpusPoE2VsOracle$"
	_ = e.gate(false)
	if f.calls != 4 {
		t.Fatalf("another -run pattern was served the default pattern's numbers")
	}
}

func TestGatePoE2RefusesWhatItCannotStore(t *testing.T) {
	repo := gp2Repo(t)
	failed := &gp2Fake{out: gp2Output, err: errors.New("exit status 1")}
	var w bytes.Buffer
	if rc := failed.env(repo, &w).gate(false); rc != 1 || !strings.Contains(w.String(), "GATE FAILED") {
		t.Fatalf("a failed gate: rc=%d\n%s", rc, w.String())
	}
	skipped := &gp2Fake{out: "--- SKIP: TestCorpusPoE2VsOracle\n"}
	if rc := skipped.env(repo, &w).gate(false); rc != 3 {
		t.Fatalf("a skipped gate: rc=%d, want the refusal code", rc)
	}
	ok := &gp2Fake{out: gp2Output}
	_ = ok.env(repo, &w).gate(false)
	if ok.calls != 1 {
		t.Fatalf("a failed or skipped run was stored")
	}
	if err := os.RemoveAll(filepath.Join(repo, "testdata")); err != nil {
		t.Fatal(err)
	}
	if rc := ok.env(repo, &w).gate(false); rc != 3 {
		t.Fatalf("no PoE2 corpus: rc=%d, want the refusal code", rc)
	}
}
