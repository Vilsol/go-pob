package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestRegenDataLuaArgvLengthIsPathIndependent: The deterministic LuaJIT sees the same argv and GAMEDATA_DUMP lengths whatever the checkout's path.
func TestRegenDataLuaArgvLengthIsPathIndependent(t *testing.T) {
	if _, err := exec.LookPath("setarch"); err != nil {
		t.Skip("no setarch")
	}
	root := t.TempDir()
	det := filepath.Join(root, ".luajit-det", "bin", "luajit")
	if err := os.MkdirAll(filepath.Dir(det), 0o750); err != nil {
		t.Fatal(err)
	}
	stub := "#!/bin/sh\nprintf '%s %s %s\\n' \"${#1}\" \"${#2}\" \"${#GAMEDATA_DUMP}\" > \"$2\"\n"
	if err := os.WriteFile(det, []byte(stub), 0o700); err != nil { //nolint:gosec // a probe stub
		t.Fatal(err)
	}
	t.Setenv("GOPOB_ROOT", root)

	seen := func(repo string) string {
		t.Helper()
		for _, d := range []string{"oracle", "pob/src"} {
			if err := os.MkdirAll(filepath.Join(repo, d), 0o750); err != nil {
				t.Fatal(err)
			}
		}
		data := filepath.Join(repo, "data.json")
		out := filepath.Join(repo, "out.json")
		if err := os.WriteFile(data, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		r := &rdRun{repo: repo, pob: filepath.Join(repo, "pob")}
		if err := r.runLua("oracle/x.lua", out, data); err != nil {
			t.Fatal(err)
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
		t.Fatalf("argv / GAMEDATA_DUMP lengths moved with the checkout path: %q vs %q", short, long)
	}
}
