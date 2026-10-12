package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	dmSkillGlob     = "src/Data/Skills/*.lua"
	dmRefusalExit   = 3
	dmDefaultPoBRel = "PathOfBuilding"
)

func dmRefuse(msg string) {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	fmt.Fprintln(os.Stderr, "An instrument that cannot answer must refuse, never return empty.")
	os.Exit(dmRefusalExit)
}

type dmSourceMissingError struct{ msg string }

func (e *dmSourceMissingError) Error() string { return e.msg }

func dmMissing(format string, a ...any) error {
	return &dmSourceMissingError{fmt.Sprintf(format, a...)}
}

func dmExcName(err error) string {
	var sm *dmSourceMissingError
	if errorsAs(err, &sm) {
		return "SourceMissing"
	}
	return "LuaParseError"
}

func dmReadText(path, what string) string {
	body, err := os.ReadFile(path) //nolint:gosec // an operator-supplied checkout, as in the original
	if err != nil {
		dmRefuse(fmt.Sprintf("%s is not readable: %s", path, errnoText(err)))
	}
	if !utf8.Valid(body) {
		dmRefuse(fmt.Sprintf("%s is not valid UTF-8 (%s)", path, what))
	}
	s := string(body)
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

func dmEqual(a, b any) bool {
	ab, aIsBool := a.(bool)
	bb, bIsBool := b.(bool)
	if aIsBool || bIsBool {
		an, aok := dmAsNumber(a)
		bn, bok := dmAsNumber(b)
		if aIsBool && bIsBool {
			return ab == bb
		}
		if aIsBool && bok {
			return dmBoolNum(ab) == bn
		}
		if bIsBool && aok {
			return dmBoolNum(bb) == an
		}
		return false
	}
	if an, aok := dmAsNumber(a); aok {
		if bn, bok := dmAsNumber(b); bok {
			return an == bn
		}
		return false
	}
	switch at := a.(type) {
	case nil:
		return b == nil
	case string:
		bs, ok := b.(string)
		return ok && at == bs
	case []any:
		bl, ok := b.([]any)
		if !ok || len(at) != len(bl) {
			return false
		}
		for i := range at {
			if !dmEqual(at[i], bl[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		bm, ok := b.(map[string]any)
		if !ok || len(at) != len(bm) {
			return false
		}
		for k, v := range at {
			w, present := bm[k]
			if !present || !dmEqual(v, w) {
				return false
			}
		}
		return true
	}
	return false
}

func dmBoolNum(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func dmAsNumber(v any) (float64, bool) {
	switch t := v.(type) {
	case int:
		return float64(t), true
	case float64:
		return t, true
	}
	return 0, false
}

func dmJSONDumps(v any) string {
	var b strings.Builder
	dmEncode(v, &b)
	return b.String()
}

func dmEncode(v any, b *strings.Builder) {
	switch t := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if t {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case int:
		b.WriteString(strconv.Itoa(t))
	case float64:
		b.WriteString(pyFloat(t))
	case string:
		b.WriteString(dmJSONString(t))
	case []any:
		b.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				b.WriteString(", ")
			}
			dmEncode(e, b)
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(dmJSONString(k))
			b.WriteString(": ")
			dmEncode(t[k], b)
		}
		b.WriteByte('}')
	default:
		b.WriteString(dmJSONString(fmt.Sprintf("%v", v)))
	}
}

func dmJSONString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			switch {
			case r < 0x20:
				fmt.Fprintf(&b, `\u%04x`, r)
			case r < 0x7f:
				b.WriteRune(r)
			case r <= 0xffff:
				fmt.Fprintf(&b, `\u%04x`, r)
			default:
				r -= 0x10000
				fmt.Fprintf(&b, `\u%04x\u%04x`, 0xd800+(r>>10), 0xdc00+(r&0x3ff))
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func dmShort(v any) string {
	s := dmJSONDumps(v)
	r := []rune(s)
	if len(r) <= 90 {
		return s
	}
	return string(r[:87]) + "..."
}

func dmRetype(v any) any {
	switch t := v.(type) {
	case json.Number:
		if strings.ContainsAny(t.String(), ".eE") {
			f, err := t.Float64()
			if err != nil {
				return t.String()
			}
			return f
		}
		n, err := strconv.Atoi(t.String())
		if err != nil {
			f, ferr := t.Float64()
			if ferr != nil {
				return t.String()
			}
			return f
		}
		return n
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = dmRetype(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[k] = dmRetype(e)
		}
		return out
	}
	return v
}
