package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gateClosureRepo(t *testing.T) (string, func(path, body string)) {
	t.Helper()
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...) //nolint:gosec // git with the test's own arguments
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	return repo, func(path, body string) {
		t.Helper()
		full := filepath.Join(repo, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		git("add", path)
		git("commit", "-qm", path)
	}
}

func TestGateClosureKeyMovesOnlyWithItsGate(t *testing.T) {
	repo, write := gateClosureRepo(t)
	poe1 := newGateClosure("calculator", "moddb")
	poe2 := newGateClosure("calculator2", "moddb")
	keys := func() (string, string) {
		t.Helper()
		k1, err := poe1.key(repo)
		if err != nil {
			t.Fatal(err)
		}
		k2, err := poe2.key(repo)
		if err != nil {
			t.Fatal(err)
		}
		return k1, k2
	}
	for _, p := range []string{"calculator/calc.go", "calculator2/calc.go", "moddb/db.go"} {
		write(p, "package x\n")
	}
	type step struct {
		path       string
		poe1, poe2 bool
	}
	for _, s := range []step{
		{"README.md", false, false},
		{"calculator/NOTES.md", false, false},
		{"frontend/src/app.html", false, false},
		{"desktop/main.go", false, false},
		{"calculator2/calc.go", false, true},
		{"calculator/calc.go", true, false},
		{"moddb/db.go", true, true},
		{"moddb/testdata/fixture.json", true, true},
		{"testdata/oracle.json", true, true},
		{"go.mod", true, true},
		{"scripts/pob_pin_poe2.json", true, true},
	} {
		b1, b2 := keys()
		write(s.path, "changed by "+s.path+"\n")
		a1, a2 := keys()
		if (a1 != b1) != s.poe1 || (a2 != b2) != s.poe2 {
			t.Errorf("committing %s: PoE1 key moved %v (want %v), PoE2 key moved %v (want %v)", s.path, a1 != b1, s.poe1, a2 != b2, s.poe2)
		}
	}
}

func TestGateClosureDirtyOnlyInside(t *testing.T) {
	repo, write := gateClosureRepo(t)
	write("calculator2/calc.go", "package x\n")
	c := newGateClosure("calculator2")
	dirty := func() bool {
		t.Helper()
		d, err := c.dirty(repo)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	if dirty() {
		t.Fatal("a clean tree reads as dirty")
	}
	put := func(p string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, p)), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, p), []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	put("frontend/new/page.svelte")
	if dirty() {
		t.Fatal("an untracked frontend file made calculator2's closure dirty")
	}
	put("calculator2/new/sub/x.go")
	if dirty() {
		t.Fatal("a file in a package outside the closure made it dirty")
	}
	put("calculator2/new.go")
	if !dirty() {
		t.Fatal("an untracked file in a closure package left it clean")
	}
}

func TestGateClosureOfTheRealGates(t *testing.T) {
	repo := repoRoot(t)
	poe2, err := gateClosureOf(repo, "./calculator2/")
	if err != nil {
		t.Fatal(err)
	}
	poe1, err := gateClosureOf(repo, "./calculator/")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"calculator2", "moddb"} {
		if !poe2.dirs[want] {
			t.Errorf("calculator2's closure lacks %s", want)
		}
	}
	for _, want := range []string{"calculator", "moddb", "data/raw"} {
		if !poe1.dirs[want] {
			t.Errorf("calculator's closure lacks %s", want)
		}
	}
	if poe2.dirs["calculator"] || poe1.dirs["calculator2"] || poe2.dirs["frontend"] {
		t.Errorf("a closure names a package its gate does not link: poe1 %v, poe2 %v", poe1.dirs, poe2.dirs)
	}
}

func TestGateInputsKeyIgnoresFrontendAndDocs(t *testing.T) {
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...) //nolint:gosec // git with the test's own arguments
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(repo, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		git("add", path)
		git("commit", "-qm", path)
	}
	key := func() string {
		t.Helper()
		k, err := gateInputsKey(repo)
		if err != nil {
			t.Fatal(err)
		}
		return k
	}
	git("init", "-q")
	write("calculator/calc.go", "package calculator\n")
	base := key()
	for _, path := range []string{"README.md", "frontend/src/app.html", ".github/workflows/push.yml", "desktop/main.go", "nix/package.nix"} {
		write(path, "x\n")
		if got := key(); got != base {
			t.Fatalf("committing %s moved the key: %s -> %s", path, base, got)
		}
	}
	// the control: a gate input moves it
	write("calculator/calc.go", "package calculator\n\nconst x = 1\n")
	if key() == base {
		t.Fatal("changing calculator/calc.go left the key unchanged")
	}
	moved := key()
	write("testdata/oracle.json", "{}\n")
	if key() == moved {
		t.Fatal("adding testdata/oracle.json left the key unchanged")
	}
}
