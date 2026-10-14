package calculator

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

func initParserForTest(tb testing.TB) {
	tb.Helper()

	if err := poe.InitializeAll(raw.DataContext(tb.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		tb.Fatal(err)
	}
}

func modLinesForTest(tb testing.TB) []string {
	tb.Helper()

	file, err := os.ReadFile("../testdata/many-mods.txt")
	testza.AssertNoError(tb, err)

	lines := strings.Split(string(file), "\n")
	testza.AssertGreater(tb, len(lines), 1000)

	out := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		out = append(out, line, line+" ")
	}

	return out
}

func assertScanEqual[T any](t *testing.T, line string, patternList map[string]CompiledList[T], idx *anchoredIndex[T]) {
	t.Helper()

	wantValue, wantLine, wantCaps := scan(line, patternList, false)
	gotValue, gotLine, gotCaps := scanAnchored(line, idx)

	testza.AssertEqual(t, wantValue == nil, gotValue == nil, "match presence for %q", line)
	testza.AssertEqual(t, wantLine, gotLine, "remainder for %q", line)
	testza.AssertEqual(t, wantCaps, gotCaps, "captures for %q", line)

	if wantValue != nil && gotValue != nil {
		testza.AssertEqual(t, fmt.Sprintf("%+v", *wantValue), fmt.Sprintf("%+v", *gotValue), "value for %q", line)
	}
}

// TestAnchoredIndexMatchesFullScan: The anchored index must be a pure prefilter.
func TestAnchoredIndexMatchesFullScan(t *testing.T) {
	t.Parallel()
	initParserForTest(t)

	for _, line := range modLinesForTest(t) {
		assertScanEqual(t, line, preFlagListCompiled, preFlagIndex)
		assertScanEqual(t, line, preSkillNameListCompiled, preSkillNameIndex)
	}
}

// TestAnchoredIndexSpecialModListMatchesFullScan: specialModList holds bare interface{} values.
func TestAnchoredIndexSpecialModListMatchesFullScan(t *testing.T) {
	t.Parallel()
	initParserForTest(t)

	for _, line := range modLinesForTest(t) {
		wantValue, wantLine, wantCaps := scan(line, specialModListCompiled, false)
		gotValue, gotLine, gotCaps := scanAnchored(line, specialModIndex)

		testza.AssertEqual(t, wantValue == nil, gotValue == nil, "match presence for %q", line)
		testza.AssertEqual(t, wantLine, gotLine, "remainder for %q", line)
		testza.AssertEqual(t, wantCaps, gotCaps, "captures for %q", line)
	}
}

func assertScanAnyEqual[T any](t *testing.T, line string, patternList map[string]CompiledList[T], idx *anyOffsetIndex[T]) {
	t.Helper()

	wantValue, wantLine, wantCaps := scan(line, patternList, false)
	gotValue, gotLine, gotCaps := scanAny(line, idx)

	testza.AssertEqual(t, wantValue == nil, gotValue == nil, "match presence for %q", line)
	testza.AssertEqual(t, wantLine, gotLine, "remainder for %q", line)
	testza.AssertEqual(t, wantCaps, gotCaps, "captures for %q", line)

	if wantValue != nil && gotValue != nil {
		testza.AssertEqual(t, fmt.Sprintf("%+v", *wantValue), fmt.Sprintf("%+v", *gotValue), "value for %q", line)
	}
}

// TestAnyOffsetIndexMatchesFullScan: The any-offset index must be a pure prefilter too.
func TestAnyOffsetIndexMatchesFullScan(t *testing.T) {
	t.Parallel()
	initParserForTest(t)

	for _, line := range modLinesForTest(t) {
		assertScanAnyEqual(t, line, formListCompiled, formListIndex)
		assertScanAnyEqual(t, line, modNameListCompiled, modNameIndex)
		assertScanAnyEqual(t, line, modTagListCompiled, modTagIndex)
		assertScanAnyEqual(t, line, modFlagListCompiled, modFlagIndex)
		assertScanAnyEqual(t, line, suffixTypesCompiled, suffixTypesIndex)
		assertScanAnyEqual(t, line, skillNameListCompiled, skillNameIndex)
	}
}

func TestRequiredPrefix(t *testing.T) {
	tests := []struct {
		source   string
		prefix   string
		anchored bool
	}{
		{`^attack speed$`, "attack speed", true},
		{`attack speed`, "attack speed", false},
		{`^vaal glacial hammer has ?a? `, "vaal glacial hammer has", true},
		{`^(\d+)% increased damage$`, "", true},
		{`^adds (\d+) to (\d+) fire damage$`, "adds ", true},
		{`^ab{0,2}c`, "a", true},
		{`^ab+c`, "ab", true},
		{`^a(?:bc|bd)`, "ab", true},
		{`^a(?:bc|xd)`, "a", true},
		{`(?i)^abc`, "", true},
		{`^a.c`, "a", true},
		{`^a[bc]d`, "a", true},
		{`^a*bc`, "", true},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			prefix, anchored, _ := requiredPrefix(test.source)
			testza.AssertEqual(t, test.prefix, prefix, "prefix")
			testza.AssertEqual(t, test.anchored, anchored, "anchored")
		})
	}
}

func compiledPatternSources[T any](out map[string]*lazyRegexp, list map[string]CompiledList[T]) {
	for k, v := range list {
		out[k] = v.Regex
	}
}

func everyParserPattern(tb testing.TB) []*lazyRegexp {
	tb.Helper()
	initParserForTest(tb)
	all := make([]*lazyRegexp, 0, 8192)
	for _, add := range []func(map[string]*lazyRegexp){
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, formListCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, modNameListCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, modFlagListCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, preFlagListCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, modTagListCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, suffixTypesCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, dmgTypesCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, penTypesCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, regenTypesCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, degenTypesCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, costTypesCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, baseCostTypesCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, flagTypesCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, specialModListCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, skillNameListCompiled) },
		func(m map[string]*lazyRegexp) { compiledPatternSources(m, preSkillNameListCompiled) },
	} {
		m := map[string]*lazyRegexp{}
		add(m)
		for _, re := range m {
			all = append(all, re)
		}
	}
	return all
}

// TestRequiredPrefixFastPathMatchesParse: the no-metacharacter shortcut must give what parsing the pattern gives, on every pattern the parser holds.
func TestRequiredPrefixFastPathMatchesParse(t *testing.T) {
	fast := 0
	patterns := everyParserPattern(t)
	for _, re := range patterns {
		src := re.String()
		gotPrefix, gotAnchored, gotNeed := requiredPrefix(src)
		wantPrefix, wantAnchored, wantNeed := requiredPrefixParsed(src)
		if gotPrefix != wantPrefix || gotAnchored != wantAnchored || gotNeed != wantNeed {
			t.Errorf("%q: shortcut (%q, %v, %q), parsed (%q, %v, %q)", src, gotPrefix, gotAnchored, gotNeed, wantPrefix, wantAnchored, wantNeed)
		}
		if body, _ := strings.CutPrefix(src, "^"); !strings.ContainsAny(body, regexMeta) {
			fast++
		}
	}
	if fast < 1000 || fast == len(patterns) {
		t.Fatalf("REFUSING: %d of %d patterns take the shortcut; both paths must be exercised", fast, len(patterns))
	}
}

func TestRequiredLiteral(t *testing.T) {
	for _, test := range []struct{ source, need string }{
		{`^(\d+)% chance to gain elusive on kill$`, "% chance to gain elusive on kill"},
		{`^y?o?u?r? ?chance to block is unlucky$`, "chance to block is unlucky"},
		{`^[ct][ar][si][tg]g?e?r?s? level (\d+) (.+) when equipped$`, " when equipped"},
		{`^a(?:bc|bd)e`, "a"},
		{`(abc)+x`, "abc"},
		{`(abcd){0,2}x`, "x"},
		{`(?i)abc`, ""},
		{`a|b`, ""},
		{`plain text`, "plain text"},
	} {
		if _, _, need := requiredPrefix(test.source); need != test.need {
			t.Errorf("%q: need %q, want %q", test.source, need, test.need)
		}
	}
}

// TestRequiredLiteralIsInEveryMatch: on many-mods.txt, every pattern a scan would try on a line and skips for lacking its literal does not match that line.
func TestRequiredLiteralIsInEveryMatch(t *testing.T) {
	t.Parallel()
	initParserForTest(t)
	ParseMod("+10 to maximum Life", false)
	checked := 0
	check := func(line string, entries []patternEntry[any]) {
		for _, e := range entries {
			if e.need != "" && !strings.Contains(line, e.need) {
				checked++
				if e.compiled.Regex.MatchString(line) {
					t.Errorf("%q matches %q, which lacks %q", e.compiled.Regex.String(), line, e.need)
				}
			}
		}
	}
	anchored := func(line string, always []patternEntry[any], byPrefix map[string][]patternEntry[any]) {
		check(line, always)
		if len(line) >= prefixKeyLen {
			check(line, byPrefix[line[:prefixKeyLen]])
		}
	}
	anyOffset := func(line string, always []patternEntry[any], byPrefix map[string][]patternEntry[any]) {
		check(line, always)
		for s := 0; s+prefixKeyLen <= len(line); s++ {
			check(line, byPrefix[line[s:s+prefixKeyLen]])
		}
	}
	for _, raw := range modLinesForTest(t) {
		line := toLowerASCII(raw)
		anchored(line, specialModIndex.always, specialModIndex.byPrefix)
		anchored(line, asAny(preFlagIndex.always), asAnyMap(preFlagIndex.byPrefix))
		anyOffset(line, asAny(formListIndex.always), asAnyMap(formListIndex.byPrefix))
		anyOffset(line, asAny(modNameIndex.always), asAnyMap(modNameIndex.byPrefix))
		anyOffset(line, asAny(modTagIndex.always), asAnyMap(modTagIndex.byPrefix))
	}
	if checked < 100000 {
		t.Fatalf("vacuity: %d skipped tries checked", checked)
	}
}

func asAny[T any](entries []patternEntry[T]) []patternEntry[any] {
	out := make([]patternEntry[any], len(entries))
	for i, e := range entries {
		out[i] = patternEntry[any]{pattern: e.pattern, compiled: CompiledList[any]{Regex: e.compiled.Regex}, need: e.need}
	}
	return out
}

func asAnyMap[T any](m map[string][]patternEntry[T]) map[string][]patternEntry[any] {
	out := make(map[string][]patternEntry[any], len(m))
	for k, v := range m {
		out[k] = asAny(v)
	}
	return out
}

// TestEveryParserPatternCompiles: the patterns compile on first use now, so a bad one no longer fails data loading.
func TestEveryParserPatternCompiles(t *testing.T) {
	patterns := everyParserPattern(t)
	if len(patterns) < 5000 {
		t.Fatalf("REFUSING: only %d patterns", len(patterns))
	}
	for _, re := range patterns {
		if _, err := regexp.Compile(re.String()); err != nil {
			t.Errorf("%q: %v", re.String(), err)
			continue
		}
		if got := re.compiled().String(); got != re.String() {
			t.Errorf("compiled %q from %q", got, re.String())
		}
	}
}
