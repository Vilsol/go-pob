package main

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// TestPyReprMatchesCPython: pyRepr HAS SEVEN CALLERS AND ITS OUTPUT IS COMPARED BYTE FOR BYTE.
func TestPyReprMatchesCPython(t *testing.T) {
	cases := []string{
		"plain",
		"",
		"Lioneye's Fall",
		`has "double"`,
		`both ' and "`,
		"back\\slash",
		"tab\there", "nl\nhere", "cr\rhere",
		"ünïcode", "emoji🔥",
		"\x00\x01\x7f",
		" nbsp",
		" line-sep",
		"​zero-width",
		"mixed 'q' \x07 ünï",
		"\U000E0001", "\U00100000",
	}
	payload, err := json.Marshal(cases)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "python3", "-c",
		"import json,sys\nfor s in json.load(sys.stdin): print(repr(s))")
	cmd.Stdin = strings.NewReader(string(payload))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python3: %v", err)
	}
	want := strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n"), "\n")
	if len(want) != len(cases) {
		t.Fatalf("python printed %d lines for %d cases", len(want), len(cases))
	}
	for i, s := range cases {
		if got := pyRepr(s); got != want[i] {
			t.Errorf("pyRepr(%q)\n  go     = %s\n  python = %s", s, got, want[i])
		}
	}
}

// TestPyNumberReprMatchesCPython: THE FLOAT SIDE HAS A BOUNDARY NO REAL CORPUS REACHES.
func TestPyNumberReprMatchesCPython(t *testing.T) {
	cases := []string{
		"5", "-5", "0", "-0", "5.0", "-0.0", "1.5000", "123456789012345678901234567890",
		"1e15", "1e16", "9.9e15", "1.5e16", "2e9", "1e-4", "1e-5", "2.674e-4",
		"2880521056.8914", "0.00026740809142486", "1e100", "1e-300",
	}
	payload, err := json.Marshal(cases)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "python3", "-c",
		"import json,sys\nfor s in json.load(sys.stdin): print(repr(json.loads(s)))")
	cmd.Stdin = strings.NewReader(string(payload))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python3: %v", err)
	}
	want := strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n"), "\n")
	for i, s := range cases {
		if got := pyNumberRepr(json.Number(s)); got != want[i] {
			t.Errorf("pyNumberRepr(%s)\n  go     = %s\n  python = %s", s, got, want[i])
		}
	}
}
