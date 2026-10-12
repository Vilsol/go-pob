package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

var benchModLines = []string{
	"10% increased Attack Speed",
	"+25% to Critical Strike Multiplier while Dual Wielding",
	"Minions have 10% increased Attack Speed",
	"Nearby Enemies have -5% to Cold Resistance",
	"Regenerate 2% of Life per second while Focused",
	"Socketed Gems are Supported by Level 20 Increased Area of Effect",
	"Vaal Glacial Hammer has a 50% chance to Freeze",
	"this line matches absolutely nothing in any pattern list",
}

func benchInitParser(tb testing.TB) {
	tb.Helper()

	if err := poe.InitializeAll(raw.DataContext(tb.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		tb.Fatal(err)
	}
}

// BenchmarkScanSpecialModList covers the single hottest scan() call site: the 1166-pattern anchored specialModList, walked once per parseMod.
func BenchmarkScanSpecialModList(b *testing.B) {
	benchInitParser(b)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, line := range benchModLines {
			scan(line, specialModListCompiled, false)
		}
	}
}

// BenchmarkScanPreSkillNameList covers the second hottest list, preSkillNameList.
func BenchmarkScanPreSkillNameList(b *testing.B) {
	benchInitParser(b)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, line := range benchModLines {
			scan(line, preSkillNameListCompiled, false)
		}
	}
}

// BenchmarkScanModNameList covers an unanchored list, where a pattern may match at any offset of the line.
func BenchmarkScanModNameList(b *testing.B) {
	benchInitParser(b)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, line := range benchModLines {
			scan(line, modNameListCompiled, true)
		}
	}
}

// BenchmarkScanAnchoredSpecialModList is the indexed counterpart of BenchmarkScanSpecialModList — the two together show what the prefilter removes.
func BenchmarkScanAnchoredSpecialModList(b *testing.B) {
	benchInitParser(b)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, line := range benchModLines {
			scanAnchored(line, specialModIndex)
		}
	}
}

// BenchmarkScanAnyModNameList is the indexed counterpart of BenchmarkScanModNameList.
func BenchmarkScanAnyModNameList(b *testing.B) {
	benchInitParser(b)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, line := range benchModLines {
			scanAny(line, modNameIndex)
		}
	}
}

// BenchmarkParseModUncached measures a full uncached mod-line parse, i.e.
func BenchmarkParseModUncached(b *testing.B) {
	benchInitParser(b)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, line := range benchModLines {
			parseMod(line, 1)
		}
	}
}
