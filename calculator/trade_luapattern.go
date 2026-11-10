package calculator

import (
	"regexp"
	"strings"
	"sync"
)

var luaPatternCache sync.Map

// luaPattern compiles a Lua pattern built at runtime from the trade site's and the stat descriptions' text
// (TradeHelpers.lua builds them with gsub), so that its magic characters keep their Lua meaning. %b and %f are not
// supported.
func luaPattern(pat string) (*regexp.Regexp, bool) {
	if v, ok := luaPatternCache.Load(pat); ok {
		re, _ := v.(*regexp.Regexp)
		return re, re != nil
	}
	re, err := regexp.Compile(luaPatternRegexp(pat))
	if err != nil {
		re = nil
	}
	luaPatternCache.Store(pat, re)
	return re, re != nil
}

func luaClass(c byte) (string, bool) {
	switch c {
	case 'a':
		return `[A-Za-z]`, true
	case 'A':
		return `[^A-Za-z]`, true
	case 'd':
		return `[0-9]`, true
	case 'D':
		return `[^0-9]`, true
	case 'l':
		return `[a-z]`, true
	case 'L':
		return `[^a-z]`, true
	case 's':
		return `[\t\n\v\f\r ]`, true
	case 'S':
		return `[^\t\n\v\f\r ]`, true
	case 'u':
		return `[A-Z]`, true
	case 'U':
		return `[^A-Z]`, true
	case 'w':
		return `[0-9A-Za-z]`, true
	case 'W':
		return `[^0-9A-Za-z]`, true
	case 'x':
		return `[0-9A-Fa-f]`, true
	case 'X':
		return `[^0-9A-Fa-f]`, true
	case 'p':
		return `[[:punct:]]`, true
	case 'P':
		return `[^[:punct:]]`, true
	case 'c':
		return `[[:cntrl:]]`, true
	case 'C':
		return `[^[:cntrl:]]`, true
	}
	return "", false
}

func luaSetClass(c byte) string {
	switch c {
	case 'a':
		return `A-Za-z`
	case 'd':
		return `0-9`
	case 'l':
		return `a-z`
	case 's':
		return `\t\n\v\f\r `
	case 'u':
		return `A-Z`
	case 'w':
		return `0-9A-Za-z`
	case 'x':
		return `0-9A-Fa-f`
	case 'p':
		return `[:punct:]`
	case 'c':
		return `[:cntrl:]`
	}
	return regexp.QuoteMeta(string(c))
}

func luaPatternRegexp(pat string) string {
	var out strings.Builder
	out.WriteString(`(?s)`)
	i := 0
	if strings.HasPrefix(pat, "^") {
		out.WriteByte('^')
		i = 1
	}
	for i < len(pat) {
		c := pat[i]
		var item string
		switch {
		case c == '$' && i == len(pat)-1:
			out.WriteByte('$')
			i++
			continue
		case c == '(' || c == ')':
			out.WriteByte(c)
			i++
			continue
		case c == '%' && i+1 < len(pat):
			if cls, ok := luaClass(pat[i+1]); ok {
				item = cls
			} else {
				item = regexp.QuoteMeta(string(pat[i+1]))
			}
			i += 2
		case c == '[':
			j := i + 1
			var set strings.Builder
			set.WriteByte('[')
			if j < len(pat) && pat[j] == '^' {
				set.WriteByte('^')
				j++
			}
			first := true
			for j < len(pat) && (first || pat[j] != ']') {
				first = false
				switch {
				case pat[j] == '%' && j+1 < len(pat):
					set.WriteString(luaSetClass(pat[j+1]))
					j += 2
				case j+2 < len(pat) && pat[j+1] == '-' && pat[j+2] != ']':
					set.WriteString(regexp.QuoteMeta(string(pat[j])) + "-" + regexp.QuoteMeta(string(pat[j+2])))
					j += 3
				default:
					set.WriteString(regexp.QuoteMeta(string(pat[j])))
					j++
				}
			}
			set.WriteByte(']')
			item = set.String()
			i = j + 1
		case c == '.':
			item = `.`
			i++
		default:
			item = regexp.QuoteMeta(string(c))
			i++
		}
		if i < len(pat) {
			switch pat[i] {
			case '*', '+', '?':
				item += string(pat[i])
				i++
			case '-':
				item += `*?`
				i++
			}
		}
		out.WriteString(item)
	}
	return out.String()
}
