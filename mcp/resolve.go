package mcp

import (
	"slices"
	"strings"
	"unicode"
)

// norm lowercases and keeps letters, digits and single spaces, so "Kaom's Heart" and "kaoms heart" meet.
func norm(s string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(r)
		case r == '\'' || r == '’':
		default:
			space = true
		}
	}
	return b.String()
}

type matchRank int

const (
	rankExact matchRank = iota
	rankPrefix
	rankWord
	rankText
	rankFuzzy
	rankNone
)

func (r matchRank) label() string {
	if r == rankExact {
		return "exact"
	}
	return "fuzzy"
}

// rankMatch ranks a candidate against a query: exact name > prefix > every word in the name > every word in the
// text > a near spelling of the name.
func rankMatch(query, name, text string) matchRank {
	q, n := norm(query), norm(name)
	if q == "" {
		return rankWord
	}
	switch {
	case q == n:
		return rankExact
	case strings.HasPrefix(n, q):
		return rankPrefix
	}
	words := strings.Fields(q)
	if containsWords(n, words) {
		return rankWord
	}
	if text != "" && containsWords(norm(text), words) {
		return rankText
	}
	if nearSpelling(q, n) {
		return rankFuzzy
	}
	return rankNone
}

func containsWords(hay string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(hay, w) {
			return false
		}
	}
	return true
}

func nearSpelling(q, n string) bool {
	d := levenshtein(q, n)
	return d <= max(1, len(q)/5)
}

func levenshtein(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	cur := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		cur[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(br)]
}

// suggest returns up to n names closest to query, for did_you_mean.
func suggest(query string, names []string, n int) []string {
	q := norm(query)
	type scored struct {
		name string
		d    int
	}
	seen := map[string]bool{}
	var list []scored
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		nn := norm(name)
		d := levenshtein(q, nn)
		if strings.Contains(nn, q) || (q != "" && strings.Contains(q, nn)) {
			d = min(d, 1+abs(len(nn)-len(q))/4)
		} else if containsWords(nn, strings.Fields(q)) {
			d = min(d, 2)
		}
		list = append(list, scored{name, d})
	}
	slices.SortStableFunc(list, func(a, b scored) int {
		if a.d != b.d {
			return a.d - b.d
		}
		return strings.Compare(a.name, b.name)
	})
	out := make([]string, 0, n)
	for _, s := range list {
		if len(out) == n || s.d > max(3, len(q)/2) {
			break
		}
		out = append(out, s.name)
	}
	return out
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// resolveName picks the best candidate by rank. ok is false with no hits (not found) or with several differently named
// hits at the best rank short of exact (ambiguous); hits holds every candidate at that rank.
func resolveName[T any](query string, list []T, name func(T) string, text func(T) string) (T, bool, []T) {
	var picked T
	var hits []T
	best := rankNone
	for _, c := range list {
		t := ""
		if text != nil {
			t = text(c)
		}
		r := rankMatch(query, name(c), t)
		switch {
		case r < best:
			best = r
			hits = append(hits[:0], c)
		case r == best && r != rankNone:
			hits = append(hits, c)
		}
	}
	if len(hits) == 0 {
		return picked, false, nil
	}
	if best == rankExact {
		return hits[0], true, hits
	}
	first := norm(name(hits[0]))
	for _, h := range hits[1:] {
		if norm(name(h)) != first {
			return picked, false, hits
		}
	}
	return hits[0], true, hits
}
