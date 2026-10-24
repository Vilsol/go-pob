package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func rfHereWithin(t *testing.T) string {
	t.Helper()
	done := make(chan string, 1)
	go func() { done <- rfHere() }()
	select {
	case got := <-done:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("rfHere did not return with GOPOB_ROOT unset")
		return ""
	}
}

func TestRegenFixturesRootFoundWithoutGopobRoot(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "oracle"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "oracle", "build_stats.lua"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPOB_ROOT", "")
	t.Chdir(sub)

	if got := rfHereWithin(t); got != root {
		t.Fatalf("rfHere = %q, want %q", got, root)
	}
}

func TestRegenFixturesRootTerminatesWithoutAnchor(t *testing.T) {
	wd, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPOB_ROOT", "")
	t.Chdir(wd)

	if got := rfHereWithin(t); got != wd {
		t.Fatalf("rfHere = %q, want the cwd %q", got, wd)
	}
}

// TestRegenFixturesLuaArgvLengthIsPathIndependent: The probe sees the same argv lengths whatever the checkout's and the fixture's paths.
func TestRegenFixturesLuaArgvLengthIsPathIndependent(t *testing.T) {
	t.Parallel()
	requireWorks(t, probeSetarch)
	stub := filepath.Join(t.TempDir(), "luajit")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '%s %s %s\\n' \"${#1}\" \"${#2}\" \"${#3}\" > \"$3\"\n"), 0o700); err != nil { //nolint:gosec // a probe stub
		t.Fatal(err)
	}
	seen := func(here string) string {
		t.Helper()
		for _, d := range []string{"oracle", "pob/src", "testdata/oracle"} {
			if err := os.MkdirAll(filepath.Join(here, d), 0o750); err != nil {
				t.Fatal(err)
			}
		}
		in := filepath.Join(here, "testdata/oracle/x_cases.json")
		out := filepath.Join(here, "testdata/oracle/x_expected.json")
		if !rfRun("x", filepath.Join(here, "pob"), here, stub, []string{in}, out) {
			t.Fatal("rfRun failed")
		}
		b, err := os.ReadFile(out) //nolint:gosec // the test's own temp tree
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	short := seen(filepath.Join(t.TempDir(), "r"))
	long := seen(filepath.Join(t.TempDir(), "a-much-longer-checkout-path", "repo"))
	if short == "" || short != long {
		t.Fatalf("argv lengths moved with the checkout path: %q vs %q", short, long)
	}
}
