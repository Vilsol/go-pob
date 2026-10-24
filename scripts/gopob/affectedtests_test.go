package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

func atRealSelect(t *testing.T, changed ...string) atSelection {
	t.Helper()
	repo, err := root.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if repo, err = filepath.EvalSymlinks(repo); err != nil {
		t.Fatal(err)
	}
	pkgs, err := atListPackages(repo)
	if err != nil {
		t.Fatal(err)
	}
	return atSelect(repo, pkgs, changed)
}

func atPicked(sel atSelection) map[string]string {
	m := map[string]string{}
	for _, p := range sel.picked {
		m[atShort(p.pkg.ImportPath)] = p.reason
	}
	return m
}

// TestAffectedTestsModChangeMatchesGoListDepsTest grades the selection against an independent reach: `go list -deps -test <pkg>` per package.
func TestAffectedTestsModChangeMatchesGoListDepsTest(t *testing.T) {
	t.Parallel()
	sel := atRealSelect(t, "mod/base_mod.go")
	got := atPicked(sel)
	repo, _ := root.Resolve()
	modPath := root.Module + "/mod"
	nonImporters := 0
	for _, p := range atListedForTest(t) {
		cmd := exec.CommandContext(t.Context(), "go", "list", "-deps", "-test", p)
		cmd.Dir = repo
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("go list -deps -test %s: %v", p, err)
		}
		reaches := false
		for _, l := range strings.Split(string(out), "\n") {
			if l == modPath {
				reaches = true
			}
		}
		if !reaches {
			nonImporters++
		}
		if _, picked := got[atShort(p)]; picked != reaches {
			t.Errorf("%s: picked %v, go list -deps -test reaches mod %v", p, picked, reaches)
		}
	}
	if nonImporters == 0 || len(got) < 3 {
		t.Fatalf("vacuous: %d picked, %d packages do not reach mod", len(got), nonImporters)
	}
	if got["calculator"] != "imports mod" {
		t.Errorf("calculator reason %q", got["calculator"])
	}
}

func atListedForTest(t *testing.T) []string {
	t.Helper()
	repo, _ := root.Resolve()
	cmd := exec.CommandContext(t.Context(), "go", "list", "./...")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(out))
}

func TestAffectedTestsFrontendAndRootTestdataSelectNone(t *testing.T) {
	t.Parallel()
	layout := "frontend/src/routes/+layout" + ".svelte"
	sel := atRealSelect(t, layout, "frontend/e2e/load/cert.go", "testdata/corpus-x/1"+".xml")
	if len(sel.picked) != 0 {
		t.Fatalf("picked %v", atPicked(sel))
	}
	if sel.notes["frontend/e2e/load/cert.go"] == "no Go package owns it" || sel.notes["frontend/e2e/load/cert.go"] == "" {
		t.Errorf("a build-constrained .go file needs its own note, got %q", sel.notes["frontend/e2e/load/cert.go"])
	}
}

func TestAffectedTestsDeletedFileAndDirectoryStillResolve(t *testing.T) {
	t.Parallel()
	sel := atRealSelect(t, "utils/no_such_file_deleted.go", "storage/gone/deeper/x.go", "calculator/testdata/gone.json")
	for f, want := range map[string]string{
		"utils/no_such_file_deleted.go": root.Module + "/utils",
		"storage/gone/deeper/x.go":      root.Module + "/storage",
		"calculator/testdata/gone.json": root.Module + "/calculator",
	} {
		if sel.mapped[f] != want {
			t.Errorf("%s -> %q, want %q", f, sel.mapped[f], want)
		}
	}
}

func TestAffectedTestsUnownedFileSelectsThePackagesThatNameIt(t *testing.T) {
	t.Parallel()
	got := atPicked(atRealSelect(t, "mise.toml"))
	if r := got["scripts/gopob/internal/toolchain"]; !strings.Contains(r, "mise.toml") {
		t.Fatalf("toolchain reads mise.toml in its tests; got %v", got)
	}
	if _, ok := got["mod"]; ok {
		t.Fatalf("mod does not name mise.toml; got %v", got)
	}
}

func TestAffectedTestsGoModSelectsEverything(t *testing.T) {
	t.Parallel()
	sel := atRealSelect(t, "go.sum")
	if len(sel.picked) != sel.allPkgs || sel.allPkgs < 10 {
		t.Fatalf("picked %d of %d", len(sel.picked), sel.allPkgs)
	}
}

func TestAffectedTestsRefusesAnUnresolvableBase(t *testing.T) {
	t.Setenv("BASE", "no-such-ref-affected-tests")
	if rc := affectedTests([]string{"--dry"}); rc != root.RefusalCode {
		t.Fatalf("rc %d, want %d", rc, root.RefusalCode)
	}
}
