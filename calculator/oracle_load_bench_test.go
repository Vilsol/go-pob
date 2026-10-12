package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/internal/oraclebin"
)

func oracleBenchPath(tb testing.TB) string {
	tb.Helper()
	path, err := oracleBlobForVersion(raw.LatestVersion)
	if err != nil {
		tb.Fatal(err)
	}
	return path
}

// BenchmarkOracleDecodeJSONL measures the UNCACHED JSONL decode: what one call site used to pay.
func BenchmarkOracleDecodeJSONL(b *testing.B) {
	path := oracleBenchPath(b)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m, err := oraclebin.DecodeJSONL(path)
		if err != nil {
			b.Fatal(err)
		}
		if len(m) == 0 {
			b.Fatal("empty")
		}
	}
}

// BenchmarkOracleLoadMemoized measures the CACHED path: what the 2nd..51st call site pays.
func BenchmarkOracleLoadMemoized(b *testing.B) {
	path := oracleBenchPath(b)
	if _, err := loadOutputOracle(path); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m, err := loadOutputOracle(path)
		if err != nil {
			b.Fatal(err)
		}
		if len(m) == 0 {
			b.Fatal("empty")
		}
	}
}

// BenchmarkOracleDecodeBin measures the COLUMNAR side-car decode: the cost of the ONE load the memo cannot remove.
func BenchmarkOracleDecodeBin(b *testing.B) {
	path := oracleBenchPath(b)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m, err := oraclebin.Load(path)
		if err != nil {
			b.Fatal(err)
		}
		if len(m) == 0 {
			b.Fatal("empty")
		}
	}
}

// BenchmarkOracleDecodeBinNoDigest isolates the decode from the content check, so the price of the digest is a number rather than an assumption.
func BenchmarkOracleDecodeBinNoDigest(b *testing.B) {
	blob, err := os.ReadFile(oraclebin.BinPath(oracleBenchPath(b)))
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m, _, err := oraclebin.Decode(blob)
		if err != nil {
			b.Fatal(err)
		}
		if len(m) == 0 {
			b.Fatal("empty")
		}
	}
}
