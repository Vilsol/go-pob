package main

import (
	"math"
	"strconv"
	"strings"
	"unicode"
)

const (
	pySpaceChars    = `\t\n\v\f\r \x{1c}-\x{1f}\x{85}\p{Z}`
	pySpaceClass    = `[` + pySpaceChars + `]`
	pyNonSpaceClass = `[^` + pySpaceChars + `]`
)

func pyIsSpace(r rune) bool {
	return r == 0x1c || r == 0x1d || r == 0x1e || r == 0x1f || unicode.IsSpace(r)
}

func pyStrip(s string) string  { return strings.TrimFunc(s, pyIsSpace) }
func pyRStrip(s string) string { return strings.TrimRightFunc(s, pyIsSpace) }

func pySplitLines(s string) []string {
	out := []string{}
	cur := strings.Builder{}
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch r {
		case '\n', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
			out = append(out, cur.String())
			cur.Reset()
		case '\r':
			if i+1 < len(rs) && rs[i+1] == '\n' {
				i++
			}
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func pyUniversalLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if s == "" {
		return nil
	}
	parts := strings.SplitAfter(s, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func pyParseFloat(s string) (float64, bool) {
	t := pyStrip(s)
	if t == "" || strings.ContainsAny(t, "xX") {
		return 0, false
	}
	f, err := strconv.ParseFloat(t, 64)
	return f, err == nil
}

func pyFixed(v float64, prec int) string {
	switch {
	case math.IsNaN(v):
		return "nan"
	case math.IsInf(v, 1):
		return "inf"
	case math.IsInf(v, -1):
		return "-inf"
	}
	return strconv.FormatFloat(v, 'f', prec, 64)
}
