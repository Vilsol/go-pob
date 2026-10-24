package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestDumpAloneOutputDiffCountsBothSidesAndNestedValues: A KEY ON ONE SIDE ONLY IS A DIFFERENCE.
func TestDumpAloneOutputDiffCountsBothSidesAndNestedValues(t *testing.T) {
	t.Parallel()
	committed := map[string]any{"Life": 100.0, "Same": "x", "OnlyCommitted": 1.0,
		"Nested": map[string]any{"a": 1.0}}
	alone := map[string]any{"Life": 101.0, "Same": "x", "OnlyAlone": true,
		"Nested": map[string]any{"a": 2.0}}
	diff, total := daOutputDiff(committed, alone)
	want := []string{"Life", "Nested", "OnlyAlone", "OnlyCommitted"}
	if !reflect.DeepEqual(diff, want) || total != 5 {
		t.Fatalf("diff %v total %d, want %v total 5", diff, total, want)
	}
	if d, _ := daOutputDiff(committed, committed); len(d) != 0 {
		t.Fatalf("a table differs from itself: %v", d)
	}
}

func TestDumpAloneListDropsExtensionsBlanksAndComments(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "list")
	if err := os.WriteFile(p, []byte("# suspects\n1037-a.xml\n\n/abs/1190-b.json\n1211-c\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := daReadList(p)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"1037-a", "1190-b", "1211-c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// TestDumpAloneRefusesWhatItCannotCompare: AN INSTRUMENT THAT CANNOT ANSWER MUST REFUSE, NEVER RETURN EMPTY.
func TestDumpAloneRefusesWhatItCannotCompare(t *testing.T) {
	d := t.TempDir()
	pob := filepath.Join(d, "pob")
	builds := filepath.Join(d, "builds")
	committed := filepath.Join(d, "committed")
	for _, p := range []string{filepath.Join(pob, "src"), builds, committed} {
		if err := os.MkdirAll(p, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	list := filepath.Join(d, "list")
	empty := filepath.Join(d, "empty")
	_ = os.WriteFile(list, []byte("0001-x\n"), 0o600)
	_ = os.WriteFile(empty, []byte("# nothing\n"), 0o600)
	_ = os.WriteFile(filepath.Join(builds, "0001-x.xml"), []byte("<x/>"), 0o600)
	t.Setenv("LUAJIT", "/nonexistent/.luajit-det/bin/luajit")

	out := func() string { return filepath.Join(t.TempDir(), "out") }
	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{"wrong-arity", []string{pob}, 2},
		{"no-pob-src", []string{d, d, builds, committed, out(), list}, 2},
		{"empty-list", []string{pob, d, builds, committed, out(), empty}, 4},
		{"unreadable-list", []string{pob, d, builds, committed, out(), filepath.Join(d, "nope")}, 4},
		{"no-committed-dump", []string{pob, d, builds, committed, out(), list}, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := dumpAlone(tc.args); got != tc.code {
				t.Fatalf("exit %d, want %d", got, tc.code)
			}
		})
	}
	t.Run("unset-luajit", func(t *testing.T) {
		t.Setenv("LUAJIT", "")
		if got := dumpAlone([]string{pob, d, builds, committed, out(), list}); got != 2 {
			t.Fatalf("exit %d, want 2", got)
		}
	})
	t.Run("non-empty-out", func(t *testing.T) {
		o := out()
		_ = os.MkdirAll(o, 0o750)
		_ = os.WriteFile(filepath.Join(o, "someone-elses"), nil, 0o600)
		if got := dumpAlone([]string{pob, d, builds, committed, o, list}); got != 3 {
			t.Fatalf("exit %d, want 3", got)
		}
	})
}
