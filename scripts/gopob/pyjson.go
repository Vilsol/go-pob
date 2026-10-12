package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

func pyLoadJSONObject(path string) (map[string]any, string) {
	raw, err := os.ReadFile(path) //nolint:gosec // an operator-supplied path, as in the originals
	if err != nil {
		return nil, fmt.Sprintf("%s is not readable: %s", path, errnoText(err))
	}
	return pyLoadJSONObjectBytes(raw, path)
}

func pyLoadJSONObjectBytes(raw []byte, path string) (map[string]any, string) {
	if !utf8.Valid(raw) {
		return nil, fmt.Sprintf("%s is not valid JSON", path)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Sprintf("%s is not valid JSON", path)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Sprintf("%s is not valid JSON", path)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Sprintf("%s is not a JSON object", path)
	}
	return m, ""
}

func pyDumps(v any, b *strings.Builder) {
	switch t := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if t {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case string:
		b.WriteString(pyJSONString(t))
	case int:
		b.WriteString(strconv.Itoa(t))
	case float64:
		b.WriteString(pyFloat(t))
	case map[string]float64:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(pyJSONString(k))
			b.WriteByte(':')
			b.WriteString(pyFloat(t[k]))
		}
		b.WriteByte('}')
	case json.Number:
		b.WriteString(pyNumberRepr(t))
	case []any:
		b.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			pyDumps(e, b)
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
				b.WriteByte(',')
			}
			b.WriteString(pyJSONString(k))
			b.WriteByte(':')
			pyDumps(t[k], b)
		}
		b.WriteByte('}')
	default:
		fmt.Fprintf(b, "%q", fmt.Sprint(t))
	}
}

func pyDumpsIndent(v any, indent, level int, b *strings.Builder) {
	pad := func(n int) { b.WriteString(strings.Repeat(" ", indent*n)) }
	switch t := v.(type) {
	case []any:
		if len(t) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteString("[\n")
		for i, e := range t {
			if i > 0 {
				b.WriteString(",\n")
			}
			pad(level + 1)
			pyDumpsIndent(e, indent, level+1, b)
		}
		b.WriteByte('\n')
		pad(level)
		b.WriteByte(']')
	case map[string]any:
		if len(t) == 0 {
			b.WriteString("{}")
			return
		}
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString("{\n")
		for i, k := range keys {
			if i > 0 {
				b.WriteString(",\n")
			}
			pad(level + 1)
			b.WriteString(pyJSONString(k))
			b.WriteString(": ")
			pyDumpsIndent(t[k], indent, level+1, b)
		}
		b.WriteByte('\n')
		pad(level)
		b.WriteByte('}')
	default:
		pyDumps(v, b)
	}
}

func pyObjectKeyOrder(body []byte, field string) ([]string, bool) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	t, err := dec.Token()
	if err != nil {
		return nil, false
	}
	if d, ok := t.(json.Delim); !ok || d != '{' {
		return nil, false
	}
	skip := func() bool {
		var v any
		return dec.Decode(&v) == nil
	}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, false
		}
		if key, _ := kt.(string); key != field {
			if !skip() {
				return nil, false
			}
			continue
		}
		nt, err := dec.Token()
		if err != nil {
			return nil, false
		}
		if d, ok := nt.(json.Delim); !ok || d != '{' {
			return nil, false
		}
		order := []string{}
		for dec.More() {
			k2, err := dec.Token()
			if err != nil {
				return nil, false
			}
			s, _ := k2.(string)
			order = append(order, s)
			if !skip() {
				return nil, false
			}
		}
		return order, true
	}
	return nil, false
}
