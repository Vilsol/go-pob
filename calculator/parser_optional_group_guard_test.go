package calculator

import (
	"os"
	"strings"
	"testing"
)

// TestNoOptionalCapturingGroupsInParserPatterns refuses an OPTIONAL CAPTURING group anywhere in mod_parser.go.
func TestNoOptionalCapturingGroupsInParserPatterns(t *testing.T) {
	t.Parallel()

	const path = "mod_parser.go"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	src := blankGoComments(string(raw))

	var bad []string
	for i := 0; i+1 < len(src); i++ {
		if src[i] != ')' || src[i+1] != '?' {
			continue
		}
		if i+2 < len(src) && (src[i+2] == ':' || src[i+2] == '?') {
			continue
		}
		open := matchingOpenParen(src, i)
		if open < 0 {
			continue
		}
		if open+1 < len(src) && src[open+1] == '?' {
			continue
		}
		if isEscaped(src, open) || isEscaped(src, i) {
			continue
		}
		line := 1 + strings.Count(src[:open], "\n")
		frag := src[open:min(i+2, len(src))]
		if len(frag) > 60 {
			frag = frag[:60] + "…"
		}
		bad = append(bad, "\t"+path+":"+itoa(line)+"  "+frag)
	}

	if len(bad) > 0 {
		t.Fatalf("OPTIONAL CAPTURING GROUP — this PANICS every ParseMod call, it does not "+
			"mis-parse.\nGo reports -1 for a non-participating group and scanState.result slices "+
			"s.line[-1:-1].\nUse a NON-capturing group `(?: … )?` instead:\n%s",
			strings.Join(bad, "\n"))
	}
}

func matchingOpenParen(src string, closeIdx int) int {
	depth := 0
	for i := closeIdx; i >= 0; i-- {
		switch src[i] {
		case ')':
			if !isEscaped(src, i) {
				depth++
			}
		case '(':
			if !isEscaped(src, i) {
				depth--
				if depth == 0 {
					return i
				}
			}
		}
	}
	return -1
}

func isEscaped(src string, i int) bool {
	n := 0
	for j := i - 1; j >= 0 && src[j] == '\\'; j-- {
		n++
	}
	return n%2 == 1
}

func blankGoComments(src string) string {
	out := []byte(src)
	const (
		code = iota
		lineComment
		blockComment
		inStr
		inRaw
		inChar
	)
	state := code
	for i := 0; i < len(out); i++ {
		c := out[i]
		switch state {
		case code:
			switch {
			case c == '/' && i+1 < len(out) && out[i+1] == '/':
				state, out[i], out[i+1] = lineComment, ' ', ' '
				i++
			case c == '/' && i+1 < len(out) && out[i+1] == '*':
				state, out[i], out[i+1] = blockComment, ' ', ' '
				i++
			case c == '"':
				state = inStr
			case c == '`':
				state = inRaw
			case c == '\'':
				state = inChar
			}
		case lineComment:
			if c == '\n' {
				state = code
			} else {
				out[i] = ' '
			}
		case blockComment:
			if c == '*' && i+1 < len(out) && out[i+1] == '/' {
				out[i], out[i+1] = ' ', ' '
				i++
				state = code
			} else if c != '\n' {
				out[i] = ' '
			}
		case inStr:
			if c == '\\' {
				i++
			} else if c == '"' {
				state = code
			}
		case inRaw:
			if c == '`' {
				state = code
			}
		case inChar:
			if c == '\\' {
				i++
			} else if c == '\'' {
				state = code
			}
		}
	}
	return string(out)
}
