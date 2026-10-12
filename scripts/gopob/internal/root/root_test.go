package root

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMod(t *testing.T, dir, mod string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+mod+"\n\ngo 1.26.2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestResolvesInANonGitDirectoryCopy: THE CONTROL THAT MADE THIS RULE NECESSARY: a plain directory copy with NO git repository must resolve.
func TestResolvesInANonGitDirectoryCopy(t *testing.T) {
	t.Setenv("GOPOB_ROOT", "")
	sb := t.TempDir()
	writeMod(t, sb, Module)
	deep := filepath.Join(sb, "scripts", "gopob")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(sb, ".git")); !os.IsNotExist(err) {
		t.Fatalf("fixture is not git-free, so it cannot prove the point")
	}
	got, err := From(deep)
	if err != nil {
		t.Fatalf("refused a valid non-git checkout: %v", err)
	}
	if want, _ := filepath.EvalSymlinks(sb); want != "" {
		if g, _ := filepath.EvalSymlinks(got); g != want {
			t.Fatalf("got %q, want %q", g, want)
		}
	}
}

func TestRefusesOutsideAnyCheckout(t *testing.T) {
	t.Setenv("GOPOB_ROOT", "")
	sb := t.TempDir()
	if _, err := From(sb); err == nil {
		t.Fatal("resolved a directory with no go-pob go.mod above it")
	}
}

// TestRefusesAForeignModule: A go.mod for SOME OTHER module must not satisfy the marker; otherwise any Go checkout resolves.
func TestRefusesAForeignModule(t *testing.T) {
	t.Setenv("GOPOB_ROOT", "")
	sb := t.TempDir()
	writeMod(t, sb, "example.com/not-go-pob")
	if _, err := From(sb); err == nil {
		t.Fatal("accepted a foreign module as the go-pob root")
	}
}

func TestEnvOverrideWinsAndIsValidated(t *testing.T) {
	good := t.TempDir()
	writeMod(t, good, Module)
	elsewhere := t.TempDir()

	t.Setenv("GOPOB_ROOT", good)
	got, err := From(elsewhere)
	if err != nil {
		t.Fatalf("GOPOB_ROOT was not honoured: %v", err)
	}
	if g, _ := filepath.EvalSymlinks(got); g != mustEval(good) {
		t.Fatalf("got %q, want %q", got, good)
	}

	t.Setenv("GOPOB_ROOT", elsewhere)
	if _, err := From(good); err == nil {
		t.Fatal("an invalid GOPOB_ROOT fell back instead of refusing")
	}
}

func mustEval(p string) string { e, _ := filepath.EvalSymlinks(p); return e }

// TestRefusalTextNamesNoDirectory: The refusal TEXT is part of the contract.
func TestRefusalTextNamesNoDirectory(t *testing.T) {
	for _, s := range []string{Refusal, RefusalEnv} {
		if filepath.IsAbs(s) || containsPathish(s) {
			t.Fatalf("refusal text carries a path: %q", s)
		}
	}
}

func containsPathish(s string) bool {
	for _, tok := range []string{"/home/", "/tmp/", "\\"} {
		for i := 0; i+len(tok) <= len(s); i++ {
			if s[i:i+len(tok)] == tok {
				return true
			}
		}
	}
	return false
}
