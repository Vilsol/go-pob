package parsecache

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	t.Setenv("PARSE_CACHE", t.TempDir())
	s, err := Open()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	return s
}

func countEntries(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".gpbc") {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	return n
}

// TestStoreServesTheParseItStored is the cache's whole contract: a hit and a miss must produce the same build.
func TestStoreServesTheParseItStored(t *testing.T) {
	s := testStore(t)
	xmls := sampleXML(t)[:20]

	for _, x := range xmls {
		want, err := builds.ParseBuild(x)
		if err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		miss, err := s.Parse(x)
		if err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		if !reflect.DeepEqual(want, miss) {
			t.Fatal("cache miss returned a different build from builds.ParseBuild")
		}
	}
	if got := countEntries(t, s.Dir()); got != len(xmls) {
		t.Fatalf("stored %d entries for %d builds — the miss path did not write", got, len(xmls))
	}

	for i, x := range xmls {
		hit, err := s.Parse(x)
		if err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		want, _ := builds.ParseBuild(x)
		if !reflect.DeepEqual(want, hit) {
			t.Fatalf("build %d: cache hit differs from a fresh parse", i)
		}
	}
	if got := countEntries(t, s.Dir()); got != len(xmls) {
		t.Fatalf("%d entries after the second pass, want %d — a hit was treated as a miss", got, len(xmls))
	}
}

// TestKeyIsContentNotName is rule 1 of this work, as a check.
func TestKeyIsContentNotName(t *testing.T) {
	s := testStore(t)
	x := sampleXML(t)[0]

	if _, err := s.Parse(x); err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	edited := append([]byte(nil), x...)
	edited = append(edited, ' ')
	if s.path(x) == s.path(edited) {
		t.Fatal("an edited build addresses the same entry as the original")
	}
	if _, err := s.Parse(edited); err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	if got := countEntries(t, s.Dir()); got != 2 {
		t.Fatalf("%d entries for two different documents, want 2", got)
	}
}

// TestFingerprintSaltsTheKey covers the half content-addressing cannot.
func TestFingerprintSaltsTheKey(t *testing.T) {
	s := testStore(t)
	x := sampleXML(t)[0]
	other := &Store{dir: s.dir, fp: s.fp + "-different-parser"}
	if s.path(x) == other.path(x) {
		t.Fatal("the parser fingerprint does not reach the key")
	}
}

// TestFingerprintCoversTheParserSources.
func TestFingerprintCoversTheParserSources(t *testing.T) {
	for _, want := range []string{"builds", "builds/parsecache", "pob"} {
		found := false
		for _, got := range fingerprintSources {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q is not fingerprinted: an edit to it would be served from the old cache", want)
		}
	}

	root, err := checkoutRoot()
	if err != nil {
		t.Skipf("%v", err)
	}
	for _, f := range []string{"builds/parser.go", "pob/types_saved.go", "builds/parsecache/codec.go"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Errorf("%s is gone: the fingerprint no longer covers what it claims", f)
		}
	}
}

// TestOpenRefusesWhenDisabled — PARSE_CACHE=off must be a REFUSAL the caller can see.
func TestOpenRefusesWhenDisabled(t *testing.T) {
	t.Setenv("PARSE_CACHE", "off")
	if _, err := Open(); !errors.Is(err, ErrDisabled) {
		t.Fatalf("PARSE_CACHE=off returned %v, want ErrDisabled", err)
	}
}

// TestCorruptEntryFallsBackToParsing — every failure on the cache path must mean "no cached answer", never "no answer".
func TestCorruptEntryFallsBackToParsing(t *testing.T) {
	s := testStore(t)
	x := sampleXML(t)[0]
	if _, err := s.Parse(x); err != nil {
		t.Fatalf("REFUSING: %v", err)
	}

	p := s.path(x)
	blob, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	for name, bad := range map[string][]byte{
		"truncated": blob[:len(blob)/2],
		"garbage":   []byte("not a cache entry at all"),
		"empty":     {},
	} {
		if err := os.WriteFile(p, bad, 0o600); err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		got, err := s.Parse(x)
		if err != nil {
			t.Fatalf("%s entry made Parse fail: %v", name, err)
		}
		want, _ := builds.ParseBuild(x)
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("%s entry produced a different build", name)
		}
	}
}
