package calculator

import (
	"bytes"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"unicode"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/internal/oraclebin"
)

// TestOracleBinMatchesJSONL is the whole licence for the .orb side-car to exist.
func TestOracleBinMatchesJSONL(t *testing.T) {
	t.Parallel()
	blob, err := oracleBlobForVersion(raw.LatestVersion)
	if err != nil {
		t.Fatal(err)
	}
	binPath := oraclebin.BinPath(blob)
	blobBytes, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("REFUSING: no side-car at %s: %v — run `task oracle-bin`", binPath, err)
	}
	binRecs, digest, err := oraclebin.Decode(blobBytes)
	if err != nil {
		t.Fatalf("decode %s: %v", binPath, err)
	}
	want, err := oraclebin.DigestFile(blob)
	if err != nil {
		t.Fatal(err)
	}
	if digest != want {
		t.Fatalf("STALE: %s was built from %x but %s is %x — run `task oracle-bin`",
			binPath, digest[:8], blob, want[:8])
	}

	jsonlRecs, err := oraclebin.DecodeJSONL(blob)
	if err != nil {
		t.Fatal(err)
	}
	if len(jsonlRecs) != len(binRecs) {
		t.Fatalf("build count: jsonl %d, bin %d", len(jsonlRecs), len(binRecs))
	}

	compared := 0
	for name, want := range jsonlRecs {
		got, ok := binRecs[name]
		if !ok {
			t.Fatalf("%s: missing from the side-car", name)
		}
		if got.Build != want.Build || got.TreeSources != want.TreeSources {
			t.Fatalf("%s: build %q/%q treeSources %d/%d",
				name, want.Build, got.Build, want.TreeSources, got.TreeSources)
		}
		for i, pair := range oracleBinTablePairs(&want, &got) {
			if (pair.want == nil) != (pair.got == nil) {
				t.Fatalf("%s table %d: nil-ness differs (jsonl nil=%v, bin nil=%v)",
					name, i, pair.want == nil, pair.got == nil)
			}
			if len(pair.want) != len(pair.got) {
				t.Fatalf("%s table %d: %d keys in the jsonl, %d in the side-car",
					name, i, len(pair.want), len(pair.got))
			}
			for k, wv := range pair.want {
				gv, ok := pair.got[k]
				if !ok {
					t.Fatalf("%s table %d: %q missing from the side-car", name, i, k)
				}
				if math.Float64bits(gv) != math.Float64bits(wv) {
					t.Fatalf("%s table %d %q: jsonl %x (%v), bin %x (%v)",
						name, i, k, math.Float64bits(wv), wv, math.Float64bits(gv), gv)
				}
				compared++
			}
		}
	}
	if compared < 1_000_000 {
		t.Fatalf("REFUSING: only %d pairs compared — the walk is not reaching the tables", compared)
	}
	t.Logf("%d builds, %d pairs compared bit-for-bit", len(jsonlRecs), compared)
}

type oracleBinPair struct{ want, got map[string]float64 }

func oracleBinTablePairs(a, b *oracleRecord) []oracleBinPair {
	return []oracleBinPair{
		{a.Output, b.Output},
		{a.MainHand, b.MainHand},
		{a.OffHand, b.OffHand},
		{a.EnemyOutput, b.EnemyOutput},
		{a.Minion, b.Minion},
		{a.MinionMain, b.MinionMain},
		{a.MinionOff, b.MinionOff},
	}
}

// TestOracleBinRoundTripsEdgeValues covers what the committed blob cannot.
func TestOracleBinRoundTripsEdgeValues(t *testing.T) {
	vals := []float64{
		0, math.Copysign(0, -1), 1, -1, math.NaN(), math.Inf(1), math.Inf(-1),
		math.MaxFloat64, math.SmallestNonzeroFloat64, 0.1 + 0.2,
	}
	in := map[string]oracleRecord{}
	for i, v := range vals {
		name := string(rune('a'+i)) + ".xml"
		in[name] = oracleRecord{
			Build:       name,
			TreeSources: i - 3,
			Output:      map[string]float64{"A": v, "B": -v, "ZZZ": v},
			MainHand:    map[string]float64{},
			EnemyOutput: map[string]float64{"E": v},
		}
	}
	var buf bytes.Buffer
	if err := oraclebin.Encode(&buf, in, [32]byte{7}); err != nil {
		t.Fatal(err)
	}
	out, digest, err := oraclebin.Decode(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if digest != ([32]byte{7}) {
		t.Fatalf("digest not round-tripped: %x", digest[:4])
	}
	if len(out) != len(in) {
		t.Fatalf("%d records in, %d out", len(in), len(out))
	}
	for name, want := range in {
		got := out[name]
		if got.TreeSources != want.TreeSources {
			t.Fatalf("%s: treeSources %d -> %d", name, want.TreeSources, got.TreeSources)
		}
		if got.MainHand == nil {
			t.Fatalf("%s: an EMPTY MainHand decoded as nil", name)
		}
		if got.OffHand != nil || got.Minion != nil {
			t.Fatalf("%s: a nil table decoded as non-nil", name)
		}
		for k, wv := range want.Output {
			if math.Float64bits(got.Output[k]) != math.Float64bits(wv) {
				t.Fatalf("%s %q: %x -> %x", name, k, math.Float64bits(wv),
					math.Float64bits(got.Output[k]))
			}
		}
	}
}

// TestOracleBinEncodeIsDeterministic pins the property `gzip -9 -n` gives the JSONL.
func TestOracleBinEncodeIsDeterministic(t *testing.T) {
	in := map[string]oracleRecord{}
	for i := 0; i < 40; i++ {
		name := string(rune('a'+i%26)) + string(rune('a'+i/26)) + ".xml"
		tbl := map[string]float64{}
		for j := 0; j < 40; j++ {
			tbl[string(rune('A'+j%26))+string(rune('a'+j/26))] = float64((i * j) % 7)
		}
		in[name] = oracleRecord{Build: name, TreeSources: i, Output: tbl}
	}
	var a, b bytes.Buffer
	for _, w := range []*bytes.Buffer{&a, &b} {
		if err := oraclebin.Encode(w, in, [32]byte{}); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("two encodings of the same map differ — map iteration is reaching the output")
	}
}

// TestOracleBinSubsetMatchesFull grades the SUBSET seam against the full decode.
func TestOracleBinSubsetMatchesFull(t *testing.T) {
	blob, err := oracleBlobForVersion(raw.LatestVersion)
	if err != nil {
		t.Fatal(err)
	}
	full, err := oraclebin.Load(blob)
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0, len(full))
	for n := range full {
		names = append(names, n)
	}
	sort.Strings(names)

	nonASCII := ""
	for _, n := range names {
		for _, r := range n {
			if r > unicode.MaxASCII {
				nonASCII = n
				break
			}
		}
		if nonASCII != "" {
			break
		}
	}
	if nonASCII == "" {
		t.Fatal("REFUSING: no non-ASCII build name in the blob — this test's whole point is gone")
	}
	want := []string{names[0], names[len(names)/2], names[len(names)-1], nonASCII}

	stage := filepath.Join(t.TempDir(), filepath.Base(blob))
	src, err := os.ReadFile(blob)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stage, src, 0o600); err != nil {
		t.Fatal(err)
	}

	for _, arm := range []struct{ name, path string }{
		{"sidecar", blob},
		{"jsonl-fallback", stage},
	} {
		t.Run(arm.name, func(t *testing.T) {
			got, err := oraclebin.LoadSubset(arm.path, want)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(want) {
				t.Fatalf("asked for %d builds, got %d", len(want), len(got))
			}
			compared := 0
			for _, n := range want {
				g, ok := got[n]
				if !ok {
					t.Fatalf("%q missing from the subset", n)
				}
				w := full[n]
				if g.Build != w.Build || g.TreeSources != w.TreeSources {
					t.Fatalf("%q: build %q/%q treeSources %d/%d", n, w.Build, g.Build,
						w.TreeSources, g.TreeSources)
				}
				for i, pair := range oracleBinTablePairs(&w, &g) {
					if (pair.want == nil) != (pair.got == nil) || len(pair.want) != len(pair.got) {
						t.Fatalf("%q table %d: shape differs", n, i)
					}
					for k, wv := range pair.want {
						if math.Float64bits(pair.got[k]) != math.Float64bits(wv) {
							t.Fatalf("%q table %d %q: %x vs %x", n, i, k,
								math.Float64bits(wv), math.Float64bits(pair.got[k]))
						}
						compared++
					}
				}
			}
			if compared < 1000 {
				t.Fatalf("REFUSING: only %d pairs compared — the walk is not reaching the tables",
					compared)
			}

			var missing *oraclebin.MissingBuildsError
			_, err = oraclebin.LoadSubset(arm.path, []string{names[0], "no-such-build-zzz.xml"})
			if !errors.As(err, &missing) {
				t.Fatalf("an absent build must refuse with *MissingBuildsError, got %v", err)
			}
			if len(missing.Missing) != 1 || missing.Missing[0] != "no-such-build-zzz.xml" {
				t.Fatalf("refusal names %v, want just the absent build", missing.Missing)
			}
			if _, err := oraclebin.LoadSubset(arm.path, nil); err == nil {
				t.Fatal("an empty want list must refuse")
			}
			if _, err := oraclebin.LoadSubset(arm.path, []string{""}); err == nil {
				t.Fatal("an empty build name must refuse")
			}
		})
	}
}

// BenchmarkOracleSubset measures what the build DIRECTORY buys.
func BenchmarkOracleSubset(b *testing.B) {
	blob := oracleBenchPath(b)
	full, err := oraclebin.Load(blob)
	if err != nil {
		b.Fatal(err)
	}
	names := make([]string, 0, len(full))
	for n := range full {
		names = append(names, n)
	}
	sort.Strings(names)

	stage := filepath.Join(b.TempDir(), filepath.Base(blob))
	src, err := os.ReadFile(blob)
	if err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(stage, src, 0o600); err != nil {
		b.Fatal(err)
	}

	for _, arm := range []struct{ name, path string }{
		{"sidecar", blob},
		{"jsonl", stage},
	} {
		for _, pick := range []struct {
			name string
			at   int
		}{{"First", 0}, {"Last", len(names) - 1}} {
			b.Run(arm.name+pick.name, func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					m, err := oraclebin.LoadSubset(arm.path, []string{names[pick.at]})
					if err != nil {
						b.Fatal(err)
					}
					if len(m) != 1 {
						b.Fatalf("got %d records", len(m))
					}
				}
			})
		}
	}
}
