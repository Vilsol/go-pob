package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func pyNumberRepr(n json.Number) string {
	s := n.String()
	if !strings.ContainsAny(s, ".eE") {
		if s == "-0" {
			return "0"
		}
		return s
	}
	f, err := n.Float64()
	if err != nil {
		return s
	}
	return pyFloat(f)
}

func pyRepr(s string) string {
	quote := byte('\'')
	if strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		quote = '"'
	}
	var b strings.Builder
	b.WriteByte(quote)
	for _, r := range s {
		switch {
		case r == rune(quote):
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case unicode.IsPrint(r):
			b.WriteRune(r)
		case r < 0x100:
			fmt.Fprintf(&b, `\x%02x`, r)
		case r < 0x10000:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			fmt.Fprintf(&b, `\U%08x`, r)
		}
	}
	b.WriteByte(quote)
	return b.String()
}

type pyTuple []any

type pyReprSelfer interface{ pyReprSelf() string }

func pyReprValue(v any) string {
	switch t := v.(type) {
	case nil:
		return "None"
	case pyReprSelfer:
		return t.pyReprSelf()
	case int:
		return strconv.Itoa(t)
	case float64:
		return pyFloat(t)
	case bool:
		if t {
			return "True"
		}
		return "False"
	case string:
		return pyRepr(t)
	case json.Number:
		return pyNumberRepr(t)
	case []any:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = pyReprValue(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case []string:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = pyRepr(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case pyTuple:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = pyReprValue(e)
		}
		if len(t) == 1 {
			return "(" + parts[0] + ",)"
		}
		return "(" + strings.Join(parts, ", ") + ")"
	case map[string]any:
		return fmt.Sprintf("<dict len %d>", len(t))
	}
	return fmt.Sprintf("%v", v)
}

func pyTypeName(v any) string {
	switch t := v.(type) {
	case nil:
		return "NoneType"
	case bool:
		return "bool"
	case string:
		return "str"
	case json.Number:
		if strings.ContainsAny(t.String(), ".eE") {
			return "float"
		}
		return "int"
	case []any:
		return "list"
	case map[string]any:
		return "dict"
	}
	return "object"
}

func errorsAs(err error, target any) bool { return errors.As(err, target) }
