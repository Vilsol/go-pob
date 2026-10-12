package main

import (
	"fmt"
	"os"
	"strings"
)

func sbvRefuse(msg string) int {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	return 3
}

func sbvNum(v any) any {
	if v == nil {
		return nil
	}
	if b, ok := v.(bool); ok {
		if b {
			return 1.0
		}
		return 0.0
	}
	f, ok := oeFloat(v)
	if !ok {
		return sbvUnfloatable{}
	}
	return f
}

type sbvUnfloatable struct{}

func sbvStrList(v any) []any {
	a, _ := v.([]any)
	out := make([]any, 0, len(a))
	out = append(out, a...)
	return out
}

type sbvError struct{ msg string }

func (e *sbvError) Error() string { return e.msg }

func sbvNeed(d map[string]any, key, what string) (any, error) {
	v, ok := d[key]
	if !ok {
		return nil, &sbvError{fmt.Sprintf("%s tag has no %s", what, pyRepr(key))}
	}
	return v, nil
}

func sbvParts(t map[string]any, listKey, oneKey, what string, listByPresence bool) (any, error) {
	raw, ok := t[listKey]
	if ok && !listByPresence {
		ok = oeTruthy(raw)
	}
	if ok {
		a, _ := raw.([]any)
		out := make([]any, 0, len(a))
		for _, p := range a {
			n, ok := gemsInt(p)
			if !ok {
				return nil, &sbvError{fmt.Sprintf("%s tag has a non-numeric part", what)}
			}
			out = append(out, n)
		}
		return out, nil
	}
	one, err := sbvNeed(t, oneKey, what)
	if err != nil {
		return nil, err
	}
	n, ok := gemsInt(one)
	if !ok {
		return nil, &sbvError{fmt.Sprintf("%s tag has a non-numeric part", what)}
	}
	return n, nil
}

func sbvGet(t map[string]any, key string, def any) any {
	if v, ok := t[key]; ok {
		return v
	}
	return def
}

func sbvFloatOrZero(v any, trueIsOne bool) float64 {
	if b, ok := v.(bool); ok {
		if b && trueIsOne {
			return 1.0
		}
		return 0.0
	}
	if f, ok := oeFloat(v); ok {
		return f
	}
	return 0.0
}

func sbvNeedMod(m map[string]any, fields []string, what string) error {
	for _, f := range fields {
		if _, ok := m[f]; !ok {
			return &sbvError{fmt.Sprintf("a %s mod has no %s", what, pyRepr(f))}
		}
	}
	return nil
}

func sbvLoad(path, what string) (map[string]any, string) {
	m, reason := pyLoadJSONObject(path)
	if reason == "" {
		return m, ""
	}
	if strings.HasSuffix(reason, "is not a JSON object") {
		return nil, fmt.Sprintf("%s is not a JSON object", what)
	}
	return nil, strings.Replace(reason, path, fmt.Sprintf("%s (%s)", what, path), 1)
}

func tagTypeOrNil(t map[string]any) any {
	if v, ok := t["type"]; ok {
		return v
	}
	return nil
}

func sbvNoTables(v any) error {
	switch t := v.(type) {
	case map[string]any:
		return &sbvError{"a canonical value is a table; the dump wraps a value it cannot express"}
	case sbvUnfloatable:
		return &sbvError{"a canonical number is not a number; the dump wraps a value it cannot express"}
	case pyTuple:
		for _, x := range t {
			if err := sbvNoTables(x); err != nil {
				return err
			}
		}
	case []any:
		for _, x := range t {
			if err := sbvNoTables(x); err != nil {
				return err
			}
		}
	}
	return nil
}
