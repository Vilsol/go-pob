package main

import (
	"fmt"
	"strconv"
	"strings"
)

type luaTable struct {
	arr  []any
	hash map[string]any
}

func newLuaTable() *luaTable { return &luaTable{hash: map[string]any{}} }

type luaFunc func(args []any) (any, error)

type luaEnv map[string]any

type luaTok struct {
	kind string
	text string
	num  float64
	line int
}

func luaLex(src string) ([]luaTok, error) {
	var toks []luaTok
	line := 1
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case strings.HasPrefix(src[i:], "--[["):
			end := strings.Index(src[i:], "]]")
			if end < 0 {
				return nil, fmt.Errorf("line %d: unterminated comment", line)
			}
			line += strings.Count(src[i:i+end], "\n")
			i += end + 2
		case strings.HasPrefix(src[i:], "--"):
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '"' || c == '\'':
			var b strings.Builder
			j := i + 1
			for ; j < len(src) && src[j] != c; j++ {
				if src[j] == '\\' && j+1 < len(src) {
					j++
					switch src[j] {
					case 'n':
						b.WriteByte('\n')
					case 't':
						b.WriteByte('\t')
					default:
						b.WriteByte(src[j])
					}
					continue
				}
				if src[j] == '\n' {
					return nil, fmt.Errorf("line %d: unterminated string", line)
				}
				b.WriteByte(src[j])
			}
			if j >= len(src) {
				return nil, fmt.Errorf("line %d: unterminated string", line)
			}
			toks = append(toks, luaTok{kind: "str", text: b.String(), line: line})
			i = j + 1
		case c >= '0' && c <= '9' || c == '.' && i+1 < len(src) && src[i+1] >= '0' && src[i+1] <= '9':
			j := i
			if strings.HasPrefix(src[i:], "0x") || strings.HasPrefix(src[i:], "0X") {
				j += 2
				for j < len(src) && strings.ContainsRune("0123456789abcdefABCDEF", rune(src[j])) {
					j++
				}
				v, err := strconv.ParseUint(src[i+2:j], 16, 64)
				if err != nil {
					return nil, fmt.Errorf("line %d: %w", line, err)
				}
				toks = append(toks, luaTok{kind: "num", text: src[i:j], num: float64(v), line: line})
				i = j
				continue
			}
			for j < len(src) && (src[j] >= '0' && src[j] <= '9' || src[j] == '.' || src[j] == 'e' || src[j] == 'E' ||
				(src[j] == '-' || src[j] == '+') && (src[j-1] == 'e' || src[j-1] == 'E')) {
				j++
			}
			v, err := strconv.ParseFloat(src[i:j], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
			toks = append(toks, luaTok{kind: "num", text: src[i:j], num: v, line: line})
			i = j
		case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z':
			j := i
			for j < len(src) && (src[j] == '_' || src[j] >= 'a' && src[j] <= 'z' || src[j] >= 'A' && src[j] <= 'Z' || src[j] >= '0' && src[j] <= '9') {
				j++
			}
			toks = append(toks, luaTok{kind: "name", text: src[i:j], line: line})
			i = j
		case strings.HasPrefix(src[i:], "..."):
			toks = append(toks, luaTok{kind: "op", text: "...", line: line})
			i += 3
		case strings.ContainsRune("{}[]()=,;.+-*/", rune(c)):
			toks = append(toks, luaTok{kind: "op", text: string(c), line: line})
			i++
		default:
			return nil, fmt.Errorf("line %d: unexpected %q", line, c)
		}
	}
	return append(toks, luaTok{kind: "eof", line: line}), nil
}

type luaParser struct {
	toks []luaTok
	pos  int
	env  luaEnv
}

func (p *luaParser) peek() luaTok { return p.toks[p.pos] }

func (p *luaParser) next() luaTok {
	t := p.toks[p.pos]
	if t.kind != "eof" {
		p.pos++
	}
	return t
}

func (p *luaParser) isOp(s string) bool { t := p.peek(); return t.kind == "op" && t.text == s }

func (p *luaParser) expect(s string) error {
	t := p.next()
	if t.kind != "op" || t.text != s {
		return fmt.Errorf("line %d: expected %q, found %q", t.line, s, t.text)
	}
	return nil
}

func (p *luaParser) expr() (any, error) {
	return p.binary(0)
}

var luaBinPrec = map[string]int{"+": 1, "-": 1, "*": 2, "/": 2}

func (p *luaParser) binary(minPrec int) (any, error) {
	left, err := p.unary()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		prec, ok := luaBinPrec[t.text]
		if t.kind != "op" || !ok || prec <= minPrec {
			return left, nil
		}
		p.next()
		right, err := p.binary(prec)
		if err != nil {
			return nil, err
		}
		a, aok := left.(float64)
		b, bok := right.(float64)
		if !aok || !bok {
			return nil, fmt.Errorf("line %d: arithmetic on a non-number", t.line)
		}
		switch t.text {
		case "+":
			left = a + b
		case "-":
			left = a - b
		case "*":
			left = a * b
		case "/":
			left = a / b
		}
	}
}

func (p *luaParser) unary() (any, error) {
	if p.isOp("-") {
		t := p.next()
		v, err := p.binary(2)
		if err != nil {
			return nil, err
		}
		f, ok := v.(float64)
		if !ok {
			return nil, fmt.Errorf("line %d: negation of a non-number", t.line)
		}
		return -f, nil
	}
	return p.primary()
}

func (p *luaParser) primary() (any, error) {
	t := p.next()
	switch t.kind {
	case "num":
		return t.num, nil
	case "str":
		return t.text, nil
	case "op":
		switch t.text {
		case "{":
			return p.table()
		case "(":
			v, err := p.expr()
			if err != nil {
				return nil, err
			}
			return v, p.expect(")")
		}
	case "name":
		switch t.text {
		case "nil":
			return nil, nil //nolint:nilnil // Lua's nil is a value
		case "true":
			return true, nil
		case "false":
			return false, nil
		}
		v, ok := p.env[t.text]
		if !ok {
			return nil, fmt.Errorf("line %d: %s is not defined", t.line, t.text)
		}
		for p.isOp(".") {
			p.next()
			field := p.next()
			tbl, ok := v.(map[string]any)
			if field.kind != "name" || !ok {
				return nil, fmt.Errorf("line %d: cannot index %s", field.line, t.text)
			}
			if v, ok = tbl[field.text]; !ok {
				return nil, fmt.Errorf("line %d: %s.%s is not defined", field.line, t.text, field.text)
			}
		}
		if p.isOp("(") {
			fn, ok := v.(luaFunc)
			if !ok {
				return nil, fmt.Errorf("line %d: %s is not a function", t.line, t.text)
			}
			p.next()
			var args []any
			for !p.isOp(")") {
				a, err := p.expr()
				if err != nil {
					return nil, err
				}
				args = append(args, a)
				if !p.isOp(")") {
					if err := p.expect(","); err != nil {
						return nil, err
					}
				}
			}
			p.next()
			return fn(args)
		}
		return v, nil
	}
	return nil, fmt.Errorf("line %d: unexpected %q", t.line, t.text)
}

func (p *luaParser) table() (any, error) {
	tbl := newLuaTable()
	for !p.isOp("}") {
		switch {
		case p.isOp("["):
			p.next()
			k, err := p.expr()
			if err != nil {
				return nil, err
			}
			ks, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("line %d: only string keys are modelled", p.peek().line)
			}
			if err := p.expect("]"); err != nil {
				return nil, err
			}
			if err := p.expect("="); err != nil {
				return nil, err
			}
			v, err := p.expr()
			if err != nil {
				return nil, err
			}
			tbl.set(ks, v)
		case p.peek().kind == "name" && p.toks[p.pos+1].kind == "op" && p.toks[p.pos+1].text == "=":
			k := p.next().text
			p.next()
			v, err := p.expr()
			if err != nil {
				return nil, err
			}
			tbl.set(k, v)
		default:
			v, err := p.expr()
			if err != nil {
				return nil, err
			}
			tbl.arr = append(tbl.arr, v)
		}
		if p.isOp(",") || p.isOp(";") {
			p.next()
		} else if !p.isOp("}") {
			t := p.peek()
			return nil, fmt.Errorf("line %d: expected , or }, found %q", t.line, t.text)
		}
	}
	p.next()
	for len(tbl.arr) > 0 && tbl.arr[len(tbl.arr)-1] == nil {
		tbl.arr = tbl.arr[:len(tbl.arr)-1]
	}
	return tbl, nil
}

func (t *luaTable) set(k string, v any) {
	if v == nil {
		delete(t.hash, k)
		return
	}
	t.hash[k] = v
}

func luaNumStr(f float64) string {
	return strconv.FormatFloat(f, 'g', 14, 64)
}
