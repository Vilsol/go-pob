package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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
