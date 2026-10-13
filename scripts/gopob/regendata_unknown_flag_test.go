package main

import (
	"strings"
	"testing"
)

// TestRegenDataRefusesAnUnknownFlag pins the arm that stops `regen-data --help` REGENERATING EVERY COMMITTED DATA TABLE.
func TestRegenDataRefusesAnUnknownFlag(t *testing.T) {
	for _, argv := range [][]string{
		{"--help"},
		{"-h"},
		{"--dry-run"},
		{"--chekc"},
		{"--only", "gems", "--verify"},
	} {
		t.Run(strings.Join(argv, " "), func(t *testing.T) {
			var stderr strings.Builder
			restore := rdStderrForTest(&stderr)
			defer restore()

			if _, ok := rdParseArgs(&rdRun{}, argv, "/pob"); ok {
				t.Fatalf("regen-data %v parsed clean — it accepted a flag it does not implement and\n"+
					"  would proceed to REGENERATE EVERY COMMITTED DATA TABLE. An unrecognised flag\n"+
					"  must refuse before anything is written.", argv)
			}
			if got := stderr.String(); !strings.Contains(got, "unknown option") {
				t.Fatalf("regen-data %v refused without naming the cause; stderr was:\n%s", argv, got)
			}
		})
	}
}

// TestRegenDataStillAcceptsItsRealFlags is the boundary companion.
func TestRegenDataStillAcceptsItsRealFlags(t *testing.T) {
	for _, argv := range [][]string{
		{"--check"},
		{"--check", "--only", "gems"},
		{"--check", "--only=gems"},
		{"--check", "--pob", "/somewhere"},
		{"--check", "--pob=/somewhere"},
		{},
	} {
		t.Run(strings.Join(argv, " "), func(t *testing.T) {
			var stderr strings.Builder
			restore := rdStderrForTest(&stderr)
			defer restore()

			if _, ok := rdParseArgs(&rdRun{}, argv, "/pob"); !ok {
				t.Fatalf("regen-data %v was refused, but every flag in it is one the tool "+
					"implements", argv)
			}
			if got := stderr.String(); strings.Contains(got, "unknown option") {
				t.Fatalf("regen-data %v was refused as an unknown option, but every flag in it is\n"+
					"  one the tool implements; stderr was:\n%s", argv, got)
			}
		})
	}
}
