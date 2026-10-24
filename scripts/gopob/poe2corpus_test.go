package main

import (
	"archive/zip"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPoe2CorpusErrorKindDropsLocationAndNumbers(t *testing.T) {
	t.Parallel()
	got := p2cErrorKind("calc", "Modules/CalcSetup.lua:1743: attempt to index field 'x' (a nil value) at 12")
	if want := "calc: attempt to index field 'x' (a nil value) at N"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if a, b := p2cErrorKind("dump", "a.lua:1: boom 7"), p2cErrorKind("dump", "b.lua:99: boom 8"); a != b {
		t.Fatalf("one error shape gave two kinds: %q, %q", a, b)
	}
}

func TestPoe2CorpusExtractSortsLimitsAndSkipsNested(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	zp := filepath.Join(d, "c.zip")
	f, err := os.Create(zp)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, n := range []string{"c3.json", "c1.json", "sub/c0.json", "readme.txt", "c2.json"} {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"character":{}}`))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	out := filepath.Join(d, "json")
	got, err := p2cExtract(zp, out, 2)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"c1", "c2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	entries, _ := os.ReadDir(out)
	if len(entries) != 2 {
		t.Fatalf("extracted %d files, want 2", len(entries))
	}
}

// TestPoe2CorpusRefusesBeforeCreatingAnything: AN INSTRUMENT THAT CANNOT ANSWER MUST REFUSE, NEVER RETURN EMPTY.
func TestPoe2CorpusRefusesBeforeCreatingAnything(t *testing.T) {
	d := t.TempDir()
	here, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	pob := filepath.Join(d, "pob")
	if err := os.MkdirAll(filepath.Join(pob, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(pob, "src", "HeadlessWrapper.lua"), nil, 0o600)
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "x"}} {
		if out, err := exec.Command("git", append([]string{"-C", pob}, args...)...).CombinedOutput(); err != nil { //nolint:gosec,noctx // a fixed git command over the test's temp dir
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	zp := filepath.Join(d, "c.zip")
	_ = os.WriteFile(zp, nil, 0o600)
	full := filepath.Join(d, "full")
	_ = os.MkdirAll(full, 0o750)
	_ = os.WriteFile(filepath.Join(full, "x"), nil, 0o600)

	cases := []struct {
		name, luajit, zip, pob, out string
		code                        int
	}{
		{"missing zip", "/x/.luajit-det/bin/luajit", filepath.Join(d, "none.zip"), pob, filepath.Join(d, "o1"), 2},
		{"no headless PoB", "/x/.luajit-det/bin/luajit", zp, d, filepath.Join(d, "o2"), 2},
		{"stock luajit", "/usr/bin/luajit", zp, pob, filepath.Join(d, "o3"), 2},
		{"non-empty out", "/x/.luajit-det/bin/luajit", zp, pob, full, 3},
	}
	for _, c := range cases {
		t.Setenv("LUAJIT", c.luajit)
		if got := poe2Corpus([]string{c.pob, here, c.zip, c.out, "poe2-0.5"}); got != c.code {
			t.Errorf("%s: exit %d, want %d", c.name, got, c.code)
		}
		if c.out != full {
			if _, err := os.Stat(c.out); err == nil {
				t.Errorf("%s: created %s before refusing", c.name, c.out)
			}
		}
	}
	stamp, err := p2cStamp(pob, filepath.Join(here, "oracle", "poe2_import.lua"), filepath.Join(here, "oracle", "dump_build.lua"))
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(full, ".poe2-corpus-stamp"), []byte(stamp), 0o600)
	t.Setenv("LUAJIT", "/x/.luajit-det/bin/luajit")
	if got := poe2Corpus([]string{pob, here, zp, full, "poe2-0.5"}); got != 4 {
		t.Errorf("a stamped root resumes and reaches the (empty) zip: exit %d, want 4", got)
	}
	_ = os.WriteFile(filepath.Join(full, ".poe2-corpus-stamp"), []byte("converter other\n"), 0o600)
	if got := poe2Corpus([]string{pob, here, zp, full, "poe2-0.5"}); got != 3 {
		t.Errorf("a root stamped by another converter: exit %d, want 3", got)
	}
	if got := poe2Corpus([]string{pob}); got != 2 {
		t.Errorf("short argv: exit %d, want 2", got)
	}
}
