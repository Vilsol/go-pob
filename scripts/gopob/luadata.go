package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type ldParseError struct{ msg string }

func (e *ldParseError) Error() string { return e.msg }

func ldFail(format string, a ...any) error { return &ldParseError{fmt.Sprintf(format, a...)} }

type ldExpr struct {
	text   string
	callee string
	args   []any
	isCall bool
}

func (e *ldExpr) pyReprSelf() string { return "Expr(" + pyRepr(e.text) + ")" }

type ldTable struct {
	array []any
	hash  map[any]any
	order []any
}

func newLDTable() *ldTable { return &ldTable{hash: map[any]any{}} }

func (t *ldTable) keys() []any { return t.order }

func (t *ldTable) set(key, val any) {
	if _, seen := t.hash[key]; !seen {
		t.order = append(t.order, key)
	}
	t.hash[key] = val
}

func (t *ldTable) get(key any) (any, bool) {
	v, ok := t.hash[key]
	return v, ok
}

func (t *ldTable) pyReprSelf() string {
	return fmt.Sprintf("Table(array=%s, hash=%s)", pyReprValue(t.array), t.reprDict())
}

func (t *ldTable) reprDict() string {
	parts := make([]string, len(t.order))
	for i, k := range t.order {
		parts[i] = pyReprValue(k) + ": " + pyReprValue(t.hash[k])
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

const ldWS = " \t\r\n"

type ldParser struct {
	s []rune
	i int
}

func newLDParser(text string, pos int) *ldParser { return &ldParser{s: []rune(text), i: pos} }

func (p *ldParser) startsWith(lit string, at int) bool {
	r := []rune(lit)
	if at < 0 || at+len(r) > len(p.s) {
		return false
	}
	for k, c := range r {
		if p.s[at+k] != c {
			return false
		}
	}
	return true
}

func (p *ldParser) find(lit string, from int) int {
	for i := from; i+len([]rune(lit)) <= len(p.s); i++ {
		if p.startsWith(lit, i) {
			return i
		}
	}
	return -1
}

func (p *ldParser) slice(lo, hi int) string {
	if lo < 0 {
		lo = 0
	}
	if hi > len(p.s) {
		hi = len(p.s)
	}
	if lo >= hi {
		return ""
	}
	return string(p.s[lo:hi])
}

func (p *ldParser) skip() {
	n := len(p.s)
	for p.i < n {
		c := p.s[p.i]
		switch {
		case strings.ContainsRune(ldWS, c):
			p.i++
		case p.startsWith("--", p.i):
			if p.startsWith("--[[", p.i) {
				end := p.find("]]", p.i+4)
				if end < 0 {
					p.i = n
				} else {
					p.i = end + 2
				}
			} else {
				nl := p.find("\n", p.i)
				if nl < 0 {
					p.i = n
				} else {
					p.i = nl + 1
				}
			}
		default:
			return
		}
	}
}

func (p *ldParser) peek() string {
	p.skip()
	if p.i < len(p.s) {
		return string(p.s[p.i])
	}
	return ""
}

func (p *ldParser) expect(ch string) error {
	if p.peek() != ch {
		return ldFail("expected %s at offset %d: %s", pyRepr(ch), p.i, pyRepr(p.slice(p.i, p.i+60)))
	}
	p.i++
	return nil
}

func (p *ldParser) value() (any, error) {
	c := p.peek()
	if c == "" {
		return nil, ldFail("unexpected end of input")
	}
	if c == "{" {
		return p.table()
	}
	if p.startsWith("function", p.i) && (p.i+8 >= len(p.s) ||
		!(ldIsAlnum(p.s[p.i+8]) || p.s[p.i+8] == '_')) {
		return p.function()
	}
	if c == "\"" || c == "'" {
		return p.stringLit()
	}
	if c == "[" && p.startsWith("[[", p.i) {
		end := p.find("]]", p.i+2)
		if end < 0 {
			return nil, ldFail("unterminated long string")
		}
		out := p.slice(p.i+2, end)
		p.i = end + 2
		return out, nil
	}
	return p.atom()
}

func (p *ldParser) stringLit() (any, error) {
	q := p.s[p.i]
	p.i++
	var out strings.Builder
	for {
		if p.i >= len(p.s) {
			return nil, ldFail("unterminated string")
		}
		c := p.s[p.i]
		switch {
		case c == '\\':
			if p.i+1 >= len(p.s) {
				return nil, ldFail("unterminated string")
			}
			switch p.s[p.i+1] {
			case 'n':
				out.WriteRune('\n')
			case 't':
				out.WriteRune('\t')
			case 'r':
				out.WriteRune('\r')
			default:
				out.WriteRune(p.s[p.i+1])
			}
			p.i += 2
		case c == q:
			p.i++
			return out.String(), nil
		default:
			out.WriteRune(c)
			p.i++
		}
	}
}

func (p *ldParser) function() (any, error) {
	start := p.i
	depth := 0
	n := len(p.s)
	for p.i < n {
		p.skip()
		if p.i >= n {
			break
		}
		c := p.s[p.i]
		if c == '"' || c == '\'' {
			if _, err := p.stringLit(); err != nil {
				return nil, err
			}
			continue
		}
		if !(unicode.IsLetter(c) || c == '_') {
			p.i++
			continue
		}
		j := p.i
		for j < n && (ldIsAlnum(p.s[j]) || p.s[j] == '_') {
			j++
		}
		word := string(p.s[p.i:j])
		p.i = j
		switch word {
		case "function", "if", "do":
			depth++
		case "end":
			depth--
			if depth == 0 {
				return &ldExpr{text: string(p.s[start:p.i])}, nil
			}
		}
	}
	return nil, ldFail("unterminated function at offset %d", start)
}

func (p *ldParser) atom() (any, error) {
	start := p.i
	n := len(p.s)
	depth := 0
	for p.i < n {
		c := p.s[p.i]
		switch {
		case c == '(' || c == '[':
			depth++
			p.i++
		case c == ')' || c == ']':
			if depth == 0 {
				goto done
			}
			depth--
			p.i++
		case (c == '"' || c == '\'') && depth > 0:
			if _, err := p.stringLit(); err != nil {
				return nil, err
			}
		case depth == 0 && (c == ',' || c == '}' || strings.ContainsRune(ldWS, c) ||
			p.startsWith("--", p.i)):
			goto done
		default:
			p.i++
		}
	}
done:
	text := strings.TrimSpace(p.slice(start, p.i))
	if text == "" {
		return nil, ldFail("empty atom at offset %d", start)
	}
	switch text {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "nil":
		return nil, nil //nolint:nilnil // a parsed Lua nil, not a missing result
	}
	if strings.ContainsAny(text, ".eE") {
		if f, err := ldPyFloat(text); err == nil {
			return f, nil
		}
	} else if v, err := ldPyInt(text); err == nil {
		return v, nil
	}
	e := &ldExpr{text: text}
	if strings.HasSuffix(text, ")") && strings.Contains(text, "(") {
		head := text[:strings.Index(text, "(")]
		if ldIsIdentifierish(head) {
			inner := newLDParser(text[strings.Index(text, "(")+1:len(text)-1], 0)
			args, err := inner.arglist()
			if err != nil {
				return nil, err
			}
			e.callee, e.args, e.isCall = head, args, true
		}
	}
	return e, nil
}

func (p *ldParser) arglist() ([]any, error) {
	out := []any{}
	for {
		p.skip()
		if p.i >= len(p.s) {
			return out, nil
		}
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
		p.skip()
		if p.i < len(p.s) && p.s[p.i] == ',' {
			p.i++
			continue
		}
		return out, nil
	}
}

func (p *ldParser) table() (any, error) {
	if err := p.expect("{"); err != nil {
		return nil, err
	}
	t := newLDTable()
	for {
		c := p.peek()
		if c == "" {
			return nil, ldFail("unterminated table")
		}
		if c == "}" {
			p.i++
			return t, nil
		}
		if c == "," {
			p.i++
			continue
		}
		var key any
		haveKey := false
		if c == "[" {
			save := p.i
			p.i++
			k, err := p.value()
			if err != nil {
				return nil, err
			}
			if p.peek() != "]" {
				p.i = save
			} else {
				p.i++
				if err := p.expect("="); err != nil {
					return nil, err
				}
				key, haveKey = k, true
			}
		} else {
			save := p.i
			j := p.i
			for j < len(p.s) && (ldIsAlnum(p.s[j]) || p.s[j] == '_') {
				j++
			}
			ident := string(p.s[p.i:j])
			if ident != "" {
				k := j
				for k < len(p.s) && strings.ContainsRune(ldWS, p.s[k]) {
					k++
				}
				if k < len(p.s) && p.s[k] == '=' && (k+1 >= len(p.s) || p.s[k+1] != '=') {
					key, haveKey = ident, true
					p.i = k + 1
				} else {
					p.i = save
				}
			}
		}
		val, err := p.value()
		if err != nil {
			return nil, err
		}
		if !haveKey {
			t.array = append(t.array, val)
			continue
		}
		if e, ok := key.(*ldExpr); ok {
			key = e.text
		}
		t.set(key, val)
	}
}

func ldPyInt(text string) (int, error) {
	n, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("not an int: %w", err)
	}
	return n, nil
}

func ldPyFloat(text string) (float64, error) {
	f, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("not a float: %w", err)
	}
	return f, nil
}

func ldIsAlnum(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

func ldIsIdentifierish(head string) bool {
	stripped := strings.NewReplacer(".", "", ":", "").Replace(head)
	return ldIsIdentifier(stripped) || ldIsIdentifier(head)
}

func ldIsIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !(unicode.IsLetter(r) || r == '_') {
				return false
			}
			continue
		}
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return false
		}
	}
	return true
}

func ldParseValue(text string, pos int) (any, int, error) {
	p := newLDParser(text, pos)
	v, err := p.value()
	if err != nil {
		return nil, 0, err
	}
	return v, p.i, nil
}

type ldAssignment struct {
	key   any
	table *ldTable
	line  int
}

func ldScanAssignments(text, prefix string) ([]ldAssignment, error) {
	runes := []rune(text)
	scan := &ldParser{s: runes}
	needle := "\n" + prefix + "["
	var anchors []int
	for i := scan.find(needle, 0); i >= 0; i = scan.find(needle, i+1) {
		anchors = append(anchors, i+1)
	}

	var out []ldAssignment
	for n, start := range anchors {
		p := &ldParser{s: runes, i: start + len([]rune(prefix)) + 1}
		key, err := p.value()
		if err != nil {
			return nil, err
		}
		if p.peek() != "]" {
			continue
		}
		p.i++
		if p.peek() != "=" {
			continue
		}
		p.i++
		line := ldCountNewlines(runes, start) + 1
		val, err := p.value()
		if err != nil {
			return nil, err
		}
		tbl, ok := val.(*ldTable)
		if !ok {
			return nil, ldFail("%s[%s] is not a table (line %d)", prefix, pyReprValue(key), line)
		}
		if n+1 < len(anchors) && p.i > anchors[n+1] {
			return nil, ldFail("%s[%s] (line %d) parsed past the next entry at line %d",
				prefix, pyReprValue(key), line, ldCountNewlines(runes, anchors[n+1])+1)
		}
		out = append(out, ldAssignment{key: key, table: tbl, line: line})
	}
	return out, nil
}

func ldCountNewlines(runes []rune, upto int) int {
	if upto > len(runes) {
		upto = len(runes)
	}
	n := 0
	for i := 0; i < upto; i++ {
		if runes[i] == '\n' {
			n++
		}
	}
	return n
}

func ldParseReturnTable(text string) (*ldTable, error) {
	runes := []rune(text)
	scan := &ldParser{s: runes}
	i := scan.find("\nreturn", 0)
	if i < 0 && !strings.HasPrefix(text, "return") {
		return nil, ldFail("no top-level `return` found")
	}
	from := i
	if from < 0 {
		from = 0
	}
	brace := scan.find("{", from)
	if brace < 0 {
		return nil, ldFail("no top-level `return` found")
	}
	val, _, err := ldParseValue(text, brace)
	if err != nil {
		return nil, err
	}
	tbl, ok := val.(*ldTable)
	if !ok {
		return nil, ldFail("`return` is not a table")
	}
	return tbl, nil
}

func ldLineOf(text, keyLiteral string) int {
	runes := []rune(text)
	scan := &ldParser{s: runes}
	i := scan.find(keyLiteral, 0)
	if i < 0 {
		return 0
	}
	return ldCountNewlines(runes, i) + 1
}
