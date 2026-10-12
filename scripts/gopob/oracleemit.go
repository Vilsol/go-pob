package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	oeUsageExit   = 2
	oeRefusalExit = 3
)

func oeRefuse(msg string) int { return oeRefuseWith(oeRefusalExit, msg) }

func oeRefuseWith(code int, msg string) int {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	return code
}

func oeLoadStdin(what string) (map[string]any, int) {
	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, oeRefuse(fmt.Sprintf("%s is not readable: %s", what, errnoText(err)))
	}
	return oeDecodeDump(body, what)
}

func oeUsage(line string) int {
	fmt.Fprintln(os.Stderr, line)
	return oeUsageExit
}

func oeLoadDump(path string) (map[string]any, int) {
	m, _, code := oeLoadDumpBytes(path)
	return m, code
}

func oeLoadDumpBytes(path string) (map[string]any, []byte, int) {
	body, err := os.ReadFile(path) //nolint:gosec // the dump path is the caller's argument
	if err != nil {
		return nil, nil, oeRefuse(fmt.Sprintf("%s is not readable: %s", path, errnoText(err)))
	}
	m, code := oeDecodeDump(body, path)
	return m, body, code
}

func oeDecodeDump(body []byte, what string) (map[string]any, int) {
	if !utf8.Valid(body) {
		return nil, oeRefuse(fmt.Sprintf("%s is not valid JSON", what))
	}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, oeRefuse(fmt.Sprintf("%s is not valid JSON", what))
	}
	if dec.More() {
		return nil, oeRefuse(fmt.Sprintf("%s is not valid JSON", what))
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, oeRefuse(fmt.Sprintf("%s is not a JSON object", what))
	}
	return m, 0
}

func oeWriteGo(path, body string) int {
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil { //nolint:gosec // matching the original's mode
		return oeRefuse(fmt.Sprintf("%s is not writable: %s", path, errnoText(err)))
	}
	return 0
}

func oeGoStr(s string) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return `""`
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func oeGoNum(v any) string {
	f, ok := oeFloat(v)
	if !ok {
		return oeStr(v)
	}
	if f == math.Trunc(f) && !math.IsInf(f, 0) {
		return strconv.FormatInt(int64(f), 10)
	}
	return pyFloat(f)
}

func oeGoFloat(v any) string {
	f, ok := oeFloat(v)
	if !ok {
		return oeStr(v)
	}
	return strings.TrimSuffix(pyFloat(f), ".0")
}

func oeFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case float64:
		return t, true
	case int:
		return float64(t), true
	}
	return 0, false
}

func oeGoBool(v any) string {
	switch t := v.(type) {
	case bool:
		if t {
			return "true"
		}
		return "false"
	case nil:
		return "none"
	case json.Number:
		return t.String()
	case string:
		return strings.ToLower(t)
	}
	return strings.ToLower(fmt.Sprintf("%v", v))
}

func oeStr(v any) string {
	switch t := v.(type) {
	case nil:
		return "None"
	case bool:
		if t {
			return "True"
		}
		return "False"
	case string:
		return t
	case json.Number:
		return t.String()
	}
	return fmt.Sprintf("%v", v)
}

func oeIntStr(v any) string {
	if n, ok := v.(json.Number); ok {
		if i, err := n.Int64(); err == nil {
			return strconv.FormatInt(i, 10)
		}
		if f, err := n.Float64(); err == nil {
			return strconv.FormatInt(int64(f), 10)
		}
	}
	return oeStr(v)
}

func oeSortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func oeObject(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func oeArray(v any) []any {
	a, _ := v.([]any)
	return a
}

func oeTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case map[string]any:
		return len(t) > 0
	case json.Number:
		f, err := t.Float64()
		return err == nil && f != 0
	}
	return true
}
