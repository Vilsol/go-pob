package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestPyHomeMatchesPython: "ABSENT", "EMPTY" AND "SET" ARE THREE STATES AND python ANSWERS DIFFERENTLY IN ALL THREE.
func TestPyHomeMatchesPython(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  []string
	}{
		{"home-set", []string{"HOME=/tmp/somewhere"}},
		{"home-set-with-trailing-slashes", []string{"HOME=/tmp/somewhere//"}},
		{"home-empty", []string{"HOME="}},
		{"home-absent", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := pythonSays(t, tc.env, "from pathlib import Path; print(Path.home(), end='')")
			got := withEnv(t, tc.env, func() string {
				h, ok := pyPathHome()
				if !ok {
					return "<unresolvable>"
				}
				return h
			})
			if got != want {
				t.Errorf("pyPathHome() = %q, python's Path.home() = %q", got, want)
			}

			for _, in := range []string{"~/x", "~"} {
				wantX := pythonSays(t, tc.env,
					"import os,sys; print(os.path.expanduser(sys.argv[1]), end='')", in)
				gotX := withEnv(t, tc.env, func() string { return pyExpandUser(in) })
				if gotX != wantX {
					t.Errorf("pyExpandUser(%q) = %q, python's expanduser = %q", in, gotX, wantX)
				}
			}
		})
	}
}

// TestPyExpandUserLeavesOtherPathsAlone: A NON-~ PATH MUST COME BACK BYTE-FOR-BYTE, INCLUDING THE SHAPES filepath.Clean WOULD EAT.
func TestPyExpandUserLeavesOtherPathsAlone(t *testing.T) {
	for _, p := range []string{"", ".", "./x", "a//b", "a/../b", "/abs/", "x~y", "-~"} {
		want := pythonSays(t, []string{"HOME=/tmp/h"},
			"import os,sys; print(os.path.expanduser(sys.argv[1]), end='')", p)
		got := withEnv(t, []string{"HOME=/tmp/h"}, func() string { return pyExpandUser(p) })
		if got != want {
			t.Errorf("pyExpandUser(%q) = %q, python = %q", p, got, want)
		}
	}
}

func withEnv(t *testing.T, env []string, f func() string) string {
	t.Helper()
	t.Setenv("HOME", "")
	if env == nil {
		_ = os.Unsetenv("HOME")
	} else {
		for _, kv := range env {
			k, v, _ := strings.Cut(kv, "=")
			t.Setenv(k, v)
		}
	}
	return f()
}

func pythonSays(t *testing.T, env []string, code string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "python3", //nolint:gosec // a literal program and code
		append([]string{"-c", code}, args...)...)
	base := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "HOME=") {
			base = append(base, kv)
		}
	}
	cmd.Env = append(base, env...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python could not answer (%v); the expectation cannot be faked: %v", code, err)
	}
	return string(out)
}
