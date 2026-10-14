package main

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// TestPyPathJoinMatchesCPython: ASKS CPython RATHER THAN ENCODING WHAT CPython IS REMEMBERED TO DO.
func TestPyPathJoinMatchesCPython(t *testing.T) {
	cases := [][]string{
		{".", "x/y"}, {"", "x"}, {"a/", "b"}, {"a", "b"}, {"a//", "b"},
		{"/abs", "b"}, {"a", "/abs"}, {".", "."}, {"..", "x"}, {"./a", "b"},
		{"a", ""}, {"", ""}, {"a/b", "../c"}, {"a", "b", "c"}, {"a", "/b", "c"},
		{"a/", "", "b"}, {".", "testdata/corpus-x"}, {"~/x", "y"},
		{"a", "b/"}, {"trailing/", "/absolute"},
	}
	payload, err := json.Marshal(cases)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "python3", "-c",
		"import json,sys,posixpath\nfor p in json.load(sys.stdin): print(posixpath.join(*p))")
	cmd.Stdin = strings.NewReader(string(payload))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python3: %v", err)
	}
	want := strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n"), "\n")
	if len(want) != len(cases) {
		t.Fatalf("python printed %d lines for %d cases", len(want), len(cases))
	}
	for i, c := range cases {
		if got := pyPathJoin(c...); got != want[i] {
			t.Errorf("pyPathJoin(%q)\n  go     = %q\n  python = %q", c, got, want[i])
		}
	}
}
