package importproxy

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const jsSpace = `\t\n\v\f\r \x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}`

func isJSSpace(r rune) bool {
	switch {
	case r == '\t', r == '\n', r == '\v', r == '\f', r == '\r', r == ' ', r == 0xa0, r == 0x1680,
		r >= 0x2000 && r <= 0x200a, r == 0x2028, r == 0x2029, r == 0x202f, r == 0x205f, r == 0x3000, r == 0xfeff:
		return true
	}
	return false
}

func jsTrim(s string) string { return strings.TrimFunc(s, isJSSpace) }

func decodeUTF8(b []byte, keepBOM bool) string {
	if !keepBOM && len(b) >= 3 && b[0] == 0xef && b[1] == 0xbb && b[2] == 0xbf {
		b = b[3:]
	}
	if utf8.Valid(b) {
		return string(b)
	}
	var out strings.Builder
	needed, seen, cp := 0, 0, rune(0)
	lower, upper := byte(0x80), byte(0xbf)
	for i := 0; i < len(b); i++ {
		c := b[i]
		if needed == 0 {
			switch {
			case c <= 0x7f:
				out.WriteByte(c)
			case c >= 0xc2 && c <= 0xdf:
				needed, cp = 1, rune(c&0x1f)
			case c >= 0xe0 && c <= 0xef:
				if c == 0xe0 {
					lower = 0xa0
				} else if c == 0xed {
					upper = 0x9f
				}
				needed, cp = 2, rune(c&0xf)
			case c >= 0xf0 && c <= 0xf4:
				if c == 0xf0 {
					lower = 0x90
				} else if c == 0xf4 {
					upper = 0x8f
				}
				needed, cp = 3, rune(c&0x7)
			default:
				out.WriteRune(utf8.RuneError)
			}
			continue
		}
		if c < lower || c > upper {
			needed, seen, cp = 0, 0, 0
			lower, upper = 0x80, 0xbf
			out.WriteRune(utf8.RuneError)
			i--
			continue
		}
		lower, upper = 0x80, 0xbf
		cp = cp<<6 | rune(c&0x3f)
		seen++
		if seen == needed {
			out.WriteRune(cp)
			needed, seen, cp = 0, 0, 0
		}
	}
	if needed != 0 {
		out.WriteRune(utf8.RuneError)
	}
	return out.String()
}

func latin1(s string) string {
	if utf8.ValidString(s) && !strings.ContainsFunc(s, func(r rune) bool { return r >= 0x80 }) {
		return s
	}
	rs := make([]rune, len(s))
	for i := 0; i < len(s); i++ {
		rs[i] = rune(s[i])
	}
	return string(rs)
}

func unhex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

func percentDecode(s string) []byte {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			h, ok1 := unhex(s[i+1])
			l, ok2 := unhex(s[i+2])
			if ok1 && ok2 {
				out = append(out, h<<4|l)
				i += 2
				continue
			}
		}
		out = append(out, s[i])
	}
	return out
}

type searchParams [][2]string

func parseSearchParams(query string) searchParams {
	var ps searchParams
	for _, pair := range strings.Split(query, "&") {
		if pair == "" {
			continue
		}
		k, v, _ := strings.Cut(pair, "=")
		ps = append(ps, [2]string{
			decodeUTF8(percentDecode(strings.ReplaceAll(k, "+", " ")), true),
			decodeUTF8(percentDecode(strings.ReplaceAll(v, "+", " ")), true),
		})
	}
	return ps
}

func (ps searchParams) get(name string) (string, bool) {
	for _, p := range ps {
		if p[0] == name {
			return p[1], true
		}
	}
	return "", false
}

func (ps searchParams) value(name string) string {
	v, _ := ps.get(name)
	return v
}

func encodeURIComponent(s string) string {
	const hexDigits = "0123456789ABCDEF"
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.IndexByte("-_.!~*'()", c) >= 0 {
			out.WriteByte(c)
			continue
		}
		out.WriteByte('%')
		out.WriteByte(hexDigits[c>>4])
		out.WriteByte(hexDigits[c&0xf])
	}
	return out.String()
}

func decodeURIComponent(s string) (string, bool) {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			out = append(out, s[i])
			continue
		}
		if i+2 >= len(s) {
			return "", false
		}
		h, ok1 := unhex(s[i+1])
		l, ok2 := unhex(s[i+2])
		if !ok1 || !ok2 {
			return "", false
		}
		out = append(out, h<<4|l)
		i += 2
	}
	if !utf8.Valid(out) {
		return "", false
	}
	return string(out), true
}

var (
	jsDecimal = regexp.MustCompile(`^[+-]?(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?$`)
	jsRadix   = regexp.MustCompile(`^0([xXoObB])([0-9a-fA-F]+)$`)
)

func jsNumber(s string) float64 {
	s = jsTrim(s)
	switch s {
	case "":
		return 0
	case "Infinity", "+Infinity":
		return math.Inf(1)
	case "-Infinity":
		return math.Inf(-1)
	}
	if jsDecimal.MatchString(s) {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
	if m := jsRadix.FindStringSubmatch(s); m != nil {
		base := map[byte]int{'x': 16, 'X': 16, 'o': 8, 'O': 8, 'b': 2, 'B': 2}[m[1][0]]
		if n, err := strconv.ParseUint(m[2], base, 64); err == nil {
			return float64(n)
		}
	}
	return math.NaN()
}

func jsNumberString(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case f == 0:
		return "0"
	}
	if a := math.Abs(f); a >= 1e21 || a < 1e-6 {
		mant, exp, _ := strings.Cut(strconv.FormatFloat(f, 'e', -1, 64), "e")
		sign, digits := exp[:1], strings.TrimLeft(exp[1:], "0")
		return mant + "e" + sign + digits
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
