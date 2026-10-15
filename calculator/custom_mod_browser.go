package calculator

import (
	"regexp"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/data"
)

const ModBrowserEmpty = "No matching modifiers found"

var (
	modBrowserRangeRe  = regexp.MustCompile(`([+-]?)\((-?\d+\.?\d*)-(-?\d+\.?\d*)\)`)
	modBrowserNumberRe = regexp.MustCompile(`\d+\.?\d*`)
	modBrowserNotAlpha = regexp.MustCompile(`[^A-Za-z]+`)
	modBrowserWordRe   = regexp.MustCompile(`[A-Za-z0-9]+`)
)

//pob:block:src/Modules/ConfigModBrowser.lua:39-49
func modBrowserTemplate(modText string) string {
	return luaLower(modBrowserNumberRe.ReplaceAllString(modBrowserRangeRe.ReplaceAllString(modText, "${1}#"), "#"))
}

//pob:block:src/Modules/ConfigModBrowser.lua:213-221
func modBrowserSortKey(modText string) string {
	key := strings.ReplaceAll(modBrowserTemplate(modText), "#", " ")
	return strings.Trim(modBrowserNotAlpha.ReplaceAllString(key, " "), luaSpace)
}

//pob:block:src/Modules/ConfigModBrowser.lua:102-110
func modBrowserLess(a, b string) bool {
	if ak, bk := modBrowserSortKey(a), modBrowserSortKey(b); ak != bk {
		return ak < bk
	}
	if at, bt := modBrowserTemplate(a), modBrowserTemplate(b); at != bt {
		return at < bt
	}
	return a < b
}

//pob:block:src/Modules/ConfigModBrowser.lua:5-38
func modBrowserScore(modText, searchStr string, words []string) int {
	modLower := luaLower(modText)
	if strings.Contains(modLower, searchStr) {
		return 1
	}
	if len(words) > 1 {
		allFound := true
		for _, w := range words {
			if !strings.Contains(modLower, w) {
				allFound = false
				break
			}
		}
		if allFound {
			return 2
		}
	}
	if len(words) == 1 && len(searchStr) >= 3 {
		textWords := modBrowserWordRe.FindAllString(modLower, -1)
		for i := range textWords {
			for len1 := 2; len1 <= len(searchStr)-1; len1++ {
				part1, part2 := searchStr[:len1], searchStr[len1:]
				if strings.HasPrefix(textWords[i], part1) && i+1 < len(textWords) && strings.HasPrefix(textWords[i+1], part2) {
					return 3
				}
			}
		}
	}
	return 0
}

//pob:block:src/Modules/ConfigModBrowser.lua:113-150
func ModBrowserList(search string) []string {
	all := data.CustomModBrowserLines()
	searchStr := strings.Trim(luaLower(search), luaSpace)
	if searchStr == "" {
		return append([]string(nil), all...)
	}
	words := strings.FieldsFunc(searchStr, func(r rune) bool { return strings.ContainsRune(luaSpace, r) })
	type match struct {
		text string
		rank int
	}
	var matches []match
	for _, modText := range all {
		if rank := modBrowserScore(modText, searchStr, words); rank != 0 {
			matches = append(matches, match{modText, rank})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].rank != matches[j].rank {
			return matches[i].rank < matches[j].rank
		}
		return modBrowserLess(matches[i].text, matches[j].text)
	})
	if len(matches) == 0 {
		return []string{ModBrowserEmpty}
	}
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m.text
	}
	return out
}

//pob:block:src/Modules/ConfigModBrowser.lua:89-90
func ModBrowserSupported(line string) bool {
	if line == "" || line == ModBrowserEmpty {
		return false
	}
	entry := ParseMod(line, false)
	return entry != nil && entry.ModList != nil && entry.Extra == ""
}

//pob:block:src/Modules/ConfigModBrowser.lua:243-250
func ModBrowserSources(line string) []string {
	return data.CustomModBrowserSources(line)
}

func ModBrowserListFuzzy(search string) []string {
	out := ModBrowserList(search)
	searchStr := strings.Trim(luaLower(search), luaSpace)
	if searchStr == "" {
		return out
	}
	if len(out) == 1 && out[0] == ModBrowserEmpty {
		out = nil
	}
	words := strings.FieldsFunc(searchStr, func(r rune) bool { return strings.ContainsRune(luaSpace, r) })
	type match struct {
		text  string
		score int
	}
	var matches []match
	var buf [5][]int
	for _, modText := range data.CustomModBrowserLines() {
		if modBrowserScore(modText, searchStr, words) != 0 {
			continue
		}
		lower, score := luaLower(modText), 0
		if n := len(lower); cap(buf[0]) < n {
			for k := range buf {
				buf[k] = make([]int, n)
			}
		}
		for _, w := range words {
			s, ok := modBrowserFuzzyWord(lower, w, &buf)
			if !ok {
				score = -1
				break
			}
			score += s
		}
		if score >= 0 {
			matches = append(matches, match{modText, score})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return modBrowserLess(matches[i].text, matches[j].text)
	})
	for _, m := range matches {
		out = append(out, m.text)
	}
	if len(out) == 0 {
		return []string{ModBrowserEmpty}
	}
	return out
}

const (
	fuzzyMatch       = 16
	fuzzyBoundary    = 8
	fuzzyConsecutive = 8
	fuzzyGapOpen     = 3
	fuzzyGapExtend   = 0
	fuzzyMinPerByte  = 10
	fuzzyDropped     = 16
	fuzzyNone        = -1 << 30
)

func modBrowserFuzzyWord(text, w string, buf *[5][]int) (int, bool) {
	missing := 0
	for i := range len(w) {
		if strings.IndexByte(text, w[i]) < 0 {
			missing++
		}
	}
	drop := len(w) >= 5
	if missing > 1 || missing == 1 && !drop {
		return 0, false
	}
	n := len(text)
	d0Before, d0, d1, c0, c1 := buf[0][:n], buf[1][:n], buf[2][:n], buf[3][:n], buf[4][:n]
	for j := range n {
		d0Before[j], d0[j], d1[j] = fuzzyNone, fuzzyNone, fuzzyNone
		if text[j] == w[0] && modBrowserWordStart(text, j) {
			d0[j] = fuzzyMatch + fuzzyBoundary
		}
	}
	for i := 1; i < len(w); i++ {
		g0, g1, gs := fuzzyNone, fuzzyNone, fuzzyNone
		for j := range n {
			c0[j], c1[j] = fuzzyNone, fuzzyNone
			g0 = fuzzyGap(g0, d0, j)
			g1 = fuzzyGap(g1, d1, j)
			gs = fuzzyGap(gs, d0Before, j)
			if text[j] != w[i] {
				continue
			}
			score := fuzzyMatch
			if modBrowserWordStart(text, j) {
				score += fuzzyBoundary
			}
			if b := fuzzyFrom(g0, d0, j); b != fuzzyNone {
				c0[j] = b + score
			}
			if !drop {
				continue
			}
			b := max(fuzzyFrom(g1, d1, j), fuzzyFrom(gs, d0Before, j))
			if i == 1 && score == fuzzyMatch+fuzzyBoundary {
				b = max(b, 0)
			}
			if b != fuzzyNone {
				c1[j] = b + score
			}
		}
		copy(d0Before, d0)
		copy(d0, c0)
		copy(d1, c1)
	}
	best, bestDropped := fuzzyNone, fuzzyNone
	for j := range n {
		best = max(best, d0[j])
		bestDropped = max(bestDropped, d1[j], d0Before[j])
	}
	if best >= fuzzyMinPerByte*len(w) {
		return best, true
	}
	if drop && bestDropped >= fuzzyMinPerByte*(len(w)-1) {
		return bestDropped - fuzzyDropped, true
	}
	return 0, false
}

func fuzzyGap(gap int, prev []int, j int) int {
	if gap != fuzzyNone {
		gap -= fuzzyGapExtend
	}
	if j >= 2 && prev[j-2] != fuzzyNone {
		gap = max(gap, prev[j-2]-fuzzyGapOpen)
	}
	return gap
}

func fuzzyFrom(gap int, prev []int, j int) int {
	if j >= 1 && prev[j-1] != fuzzyNone {
		return max(gap, prev[j-1]+fuzzyConsecutive)
	}
	return gap
}

func modBrowserWordStart(text string, j int) bool {
	if j == 0 {
		return true
	}
	b := text[j-1]
	return !(b >= 'a' && b <= 'z' || b >= '0' && b <= '9')
}
