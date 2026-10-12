package calculator

import (
	"regexp"
	"regexp/syntax"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"
)

type lazyRegexp struct {
	src  string
	once sync.Once
	re   *regexp.Regexp
}

func lazyRegex(src string) *lazyRegexp {
	return &lazyRegexp{src: src}
}

func (l *lazyRegexp) compiled() *regexp.Regexp {
	l.once.Do(func() { l.re = regexp.MustCompile(l.src) })
	return l.re
}

func (l *lazyRegexp) String() string {
	return l.src
}

func (l *lazyRegexp) FindAllStringSubmatchIndex(s string, n int) [][]int {
	return l.compiled().FindAllStringSubmatchIndex(s, n)
}

func (l *lazyRegexp) MatchString(s string) bool {
	return l.compiled().MatchString(s)
}

const prefixKeyLen = 4

type patternEntry[T any] struct {
	pattern  string
	compiled CompiledList[T]
	need     string
}

func (e *patternEntry[T]) tryOn(state *scanState[T]) {
	if e.need != "" && !strings.Contains(state.lineLower, e.need) {
		return
	}
	state.consider(e.pattern, e.compiled)
}

type anchoredIndex[T any] struct {
	byPrefix map[string][]patternEntry[T]
	always   []patternEntry[T]
}

var (
	specialModIndex   *anchoredIndex[interface{}]
	preFlagIndex      *anchoredIndex[modNameListType]
	preSkillNameIndex *anchoredIndex[modNameListType]
)

func newAnchoredIndex[T any](patternList map[string]CompiledList[T]) *anchoredIndex[T] {
	idx := &anchoredIndex[T]{byPrefix: make(map[string][]patternEntry[T], len(patternList))}

	for pattern, compiled := range patternList {
		prefix, anchored, need := requiredPrefix(compiled.Regex.String())
		entry := patternEntry[T]{pattern: pattern, compiled: compiled, need: need}

		if anchored && len(prefix) >= prefixKeyLen {
			key := prefix[:prefixKeyLen]
			idx.byPrefix[key] = append(idx.byPrefix[key], entry)
		} else {
			idx.always = append(idx.always, entry)
		}
	}

	return idx
}

func scanAnchored[T any](line string, idx *anchoredIndex[T]) (*T, string, []string) {
	state := newScanState[T](line)

	for i := range idx.always {
		idx.always[i].tryOn(&state)
	}

	if len(state.lineLower) >= prefixKeyLen {
		entries := idx.byPrefix[state.lineLower[:prefixKeyLen]]
		for i := range entries {
			entries[i].tryOn(&state)
		}
	}

	return state.result()
}

type anyOffsetIndex[T any] struct {
	byPrefix map[string][]patternEntry[T]
	always   []patternEntry[T]
}

var (
	formListIndex    *anyOffsetIndex[string]
	modNameIndex     *anyOffsetIndex[modNameListType]
	modTagIndex      *anyOffsetIndex[modNameListType]
	modFlagIndex     *anyOffsetIndex[modNameListType]
	suffixTypesIndex *anyOffsetIndex[string]
	skillNameIndex   *anyOffsetIndex[modNameListType]
)

func newAnyOffsetIndex[T any](patternList map[string]CompiledList[T]) *anyOffsetIndex[T] {
	idx := &anyOffsetIndex[T]{byPrefix: make(map[string][]patternEntry[T], len(patternList))}

	for pattern, compiled := range patternList {
		prefix, _, need := requiredPrefix(compiled.Regex.String())
		entry := patternEntry[T]{pattern: pattern, compiled: compiled, need: need}

		if len(prefix) >= prefixKeyLen {
			key := prefix[:prefixKeyLen]
			idx.byPrefix[key] = append(idx.byPrefix[key], entry)
		} else {
			idx.always = append(idx.always, entry)
		}
	}

	return idx
}

func scanAny[T any](line string, idx *anyOffsetIndex[T]) (*T, string, []string) {
	state := newScanState[T](line)

	for i := range idx.always {
		idx.always[i].tryOn(&state)
	}

	var keyStore [64]string
	keys := keyStore[:0]

	for s := 0; s+prefixKeyLen <= len(state.lineLower); s++ {
		key := state.lineLower[s : s+prefixKeyLen]

		entries := idx.byPrefix[key]
		if len(entries) == 0 || slices.Contains(keys, key) {
			continue
		}
		keys = append(keys, key)

		for i := range entries {
			entries[i].tryOn(&state)
		}
	}

	return state.result()
}

const regexMeta = `\.+*?()|[]{}^$`

func requiredPrefix(source string) (string, bool, string) {
	if body, anchored := strings.CutPrefix(source, "^"); !strings.ContainsAny(body, regexMeta) && utf8.ValidString(body) {
		return body, anchored, body
	}
	return requiredPrefixParsed(source)
}

func requiredPrefixParsed(source string) (string, bool, string) {
	parsed, err := syntax.Parse(source, syntax.Perl)
	if err != nil {
		return "", false, ""
	}

	parsed = parsed.Simplify()
	prefix, _ := literalPrefix(parsed)

	return prefix, anchoredAtStart(parsed), requiredLiteral(parsed)
}

func requiredLiteral(re *syntax.Regexp) string {
	switch re.Op {
	case syntax.OpLiteral:
		if re.Flags&syntax.FoldCase != 0 {
			return ""
		}
		return string(re.Rune)
	case syntax.OpCapture, syntax.OpPlus:
		return requiredLiteral(re.Sub[0])
	case syntax.OpRepeat:
		if re.Min >= 1 {
			return requiredLiteral(re.Sub[0])
		}
		return ""
	case syntax.OpConcat:
		best := ""
		for _, sub := range re.Sub {
			if lit := requiredLiteral(sub); len(lit) > len(best) {
				best = lit
			}
		}
		return best
	default:
		return ""
	}
}

func literalPrefix(re *syntax.Regexp) (string, bool) {
	switch re.Op {
	case syntax.OpEmptyMatch, syntax.OpBeginLine, syntax.OpBeginText,
		syntax.OpEndLine, syntax.OpEndText, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return "", true
	case syntax.OpLiteral:
		if re.Flags&syntax.FoldCase != 0 {
			return "", false
		}
		return string(re.Rune), true
	case syntax.OpCapture:
		return literalPrefix(re.Sub[0])
	case syntax.OpConcat:
		var out strings.Builder
		for _, sub := range re.Sub {
			prefix, exact := literalPrefix(sub)
			out.WriteString(prefix)
			if !exact {
				return out.String(), false
			}
		}
		return out.String(), true
	case syntax.OpAlternate:
		if len(re.Sub) == 0 {
			return "", false
		}
		common, _ := literalPrefix(re.Sub[0])
		for _, sub := range re.Sub[1:] {
			prefix, _ := literalPrefix(sub)
			common = common[:commonPrefixLen(common, prefix)]
			if common == "" {
				break
			}
		}
		return common, false
	case syntax.OpPlus:
		prefix, _ := literalPrefix(re.Sub[0])
		return prefix, false
	case syntax.OpRepeat:
		if re.Min >= 1 {
			prefix, _ := literalPrefix(re.Sub[0])
			return prefix, false
		}
		return "", false
	default:
		return "", false
	}
}

func commonPrefixLen(a, b string) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func anchoredAtStart(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpBeginText:
		return true
	case syntax.OpCapture, syntax.OpPlus:
		return anchoredAtStart(re.Sub[0])
	case syntax.OpConcat:
		return len(re.Sub) > 0 && anchoredAtStart(re.Sub[0])
	case syntax.OpAlternate:
		for _, sub := range re.Sub {
			if !anchoredAtStart(sub) {
				return false
			}
		}
		return len(re.Sub) > 0
	default:
		return false
	}
}
