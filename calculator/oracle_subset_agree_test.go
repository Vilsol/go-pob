package calculator

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/internal/oraclebin"
)

// TestOracleSubsetAgreesWithFullLoad grades the subset path against the full one.
func TestOracleSubsetAgreesWithFullLoad(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	full, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	_, names := corpusNamesForLoadedData(t)
	var ascii, nonASCII string
	for _, n := range names {
		isASCII := true
		for i := 0; i < len(n); i++ {
			if n[i] >= 0x80 {
				isASCII = false
				break
			}
		}
		if _, ok := full[n]; !ok {
			continue
		}
		if isASCII && ascii == "" {
			ascii = n
		}
		if !isASCII && nonASCII == "" {
			nonASCII = n
		}
	}
	if ascii == "" {
		t.Fatalf("REFUSING: no ASCII-named build in the blob — the sample cannot be built")
	}
	if nonASCII == "" {
		t.Fatalf("REFUSING: no non-ASCII-named build resolved against the blob, so the escaping " +
			"case this test exists for would go unexercised. 915 of 8648 names carry one.")
	}

	sample := []string{ascii, nonASCII, names[0], names[len(names)-1]}
	seen := map[string]bool{}
	uniq := sample[:0]
	for _, s := range sample {
		if !seen[s] {
			seen[s] = true
			uniq = append(uniq, s)
		}
	}

	got, err := oracleRecordsForBuilds(uniq...)
	if err != nil {
		t.Fatalf("REFUSING: subset load failed for %v: %v", uniq, err)
	}
	if len(got) != len(uniq) {
		t.Fatalf("subset returned %d records for %d builds", len(got), len(uniq))
	}
	for _, name := range uniq {
		if !reflect.DeepEqual(got[name], full[name]) {
			t.Errorf("%s: subset record differs from the full load", name)
		}
	}
	t.Logf("subset == full on %d (ascii %q, non-ascii %q)", len(uniq), ascii, nonASCII)
}

// TestOracleSubsetRefusesAnUnknownBuild pins the refusal.
func TestOracleSubsetRefusesAnUnknownBuild(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	_, err := oracleRecordsForBuilds("no-such-build-zzz.xml")
	if err == nil {
		t.Fatal("subset load returned no error for a build that is not in the blob")
	}
	var missing *oraclebin.MissingBuildsError
	if !errors.As(err, &missing) {
		t.Fatalf("want a *oraclebin.MissingBuildsError, got %T: %v", err, err)
	}
	if len(missing.Missing) != 1 || missing.Missing[0] != "no-such-build-zzz.xml" {
		t.Errorf("the refusal must NAME the build; got %v", missing.Missing)
	}
	if _, err := oracleRecordsForBuilds(); err == nil {
		t.Fatal("subset load returned no error when asked for no builds at all")
	}
}

// BenchmarkOracleLoad is the before/after for the diag path.
func BenchmarkOracleLoad(b *testing.B) {
	if os.Getenv("CORPUS") != "true" {
		b.Skip("set CORPUS=true")
	}
	path, err := oracleBlobForVersion(raw.LatestVersion)
	if err != nil {
		b.Fatalf("REFUSING: %v", err)
	}
	first, last := benchCorpusEndNames(b)

	b.Run("Full", func(b *testing.B) {
		for b.Loop() {
			rec, err := loadOutputOracleUncached(path)
			if err != nil || len(rec) == 0 {
				b.Fatalf("%d records, err %v", len(rec), err)
			}
		}
	})

	b.Run("SubsetFirst", func(b *testing.B) {
		for b.Loop() {
			rec, err := oraclebin.LoadSubset(path, []string{first})
			if err != nil || len(rec) != 1 {
				b.Fatalf("%d records, err %v", len(rec), err)
			}
		}
	})

	b.Run("SubsetLast", func(b *testing.B) {
		for b.Loop() {
			rec, err := oraclebin.LoadSubset(path, []string{last})
			if err != nil || len(rec) != 1 {
				b.Fatalf("%d records, err %v", len(rec), err)
			}
		}
	})
}

func benchCorpusEndNames(b *testing.B) (string, string) {
	b.Helper()
	dir, err := corpusDirForVersion(raw.LatestVersion)
	if err != nil {
		b.Fatalf("REFUSING: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		b.Fatalf("REFUSING: %v", err)
	}
	var names []string
	for _, e := range entries {
		if n := e.Name(); len(n) > 4 && n[len(n)-4:] == ".xml" {
			names = append(names, n)
		}
	}
	if len(names) < 2 {
		b.Fatalf("REFUSING: %s holds %d builds; the first/last split needs at least 2", dir, len(names))
	}
	first, last := names[0], names[len(names)-1]
	path, err := oracleBlobForVersion(raw.LatestVersion)
	if err != nil {
		b.Fatalf("REFUSING: %v", err)
	}
	if _, err := oraclebin.LoadSubset(path, []string{first, last}); err != nil {
		b.Fatalf("REFUSING: the corpus ends do not resolve in the blob: %v", err)
	}
	return first, last
}
