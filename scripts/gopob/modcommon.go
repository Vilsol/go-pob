package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type mcFlagBit struct {
	bit  int64
	name string
}

type mcVocab struct {
	mflag []mcFlagBit
	kw    []mcFlagBit
}

var mcGoAbsentKeywordFlags = map[string]bool{"Physical": true}

var mcMasks = []string{"Mask"}

func mcUseFlagTables(dump map[string]any) (*mcVocab, string) {
	for _, field := range []string{"modFlagBits", "keywordFlagBits"} {
		if !oeTruthy(dump[field]) {
			return nil, fmt.Sprintf("the dump carries no %s. Re-run the dumper -- decoding PoB's "+
				"numeric flags against this emitter's hardcoded 2.18.1 table silently mis-names "+
				"every keyword flag from Fire onward under the 3.20 pin.", pyRepr(field))
		}
	}
	build := func(raw any, prefix string, absent map[string]bool) []mcFlagBit {
		m, _ := raw.(map[string]any)
		out := make([]mcFlagBit, 0, len(m))
		for name, value := range m {
			skip := false
			for _, suffix := range mcMasks {
				if strings.HasSuffix(name, suffix) {
					skip = true
				}
			}
			if skip || absent[name] {
				continue
			}
			v, ok := gemsInt(value)
			if !ok {
				continue
			}
			out = append(out, mcFlagBit{bit: int64(v), name: prefix + name})
		}
		sort.Slice(out, func(a, b int) bool {
			if out[a].bit != out[b].bit {
				return out[a].bit < out[b].bit
			}
			return out[a].name < out[b].name
		})
		return out
	}
	return &mcVocab{
		mflag: build(dump["modFlagBits"], "MFlag", nil),
		kw:    build(dump["keywordFlagBits"], "KeywordFlag", mcGoAbsentKeywordFlags),
	}, ""
}

type mcUnrepresentableError struct{ reason string }

func (e *mcUnrepresentableError) Error() string { return e.reason }

func mcUnrep(format string, args ...any) error {
	return &mcUnrepresentableError{reason: fmt.Sprintf(format, args...)}
}

func mcHasTable(v any) bool {
	switch t := v.(type) {
	case map[string]any:
		return true
	case []any:
		for _, x := range t {
			if mcHasTable(x) {
				return true
			}
		}
	}
	return false
}

var mcModFields = []string{"value", "flags", "keywordFlags", "tags"}

func mcReqMod(m map[string]any) error {
	for _, f := range mcModFields {
		if _, ok := m[f]; !ok {
			return mcUnrep("mod has no %s", pyRepr(f))
		}
	}
	return nil
}

func mcExtraKeys(m map[string]any) []string {
	raw, ok := m["extraKeys"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, k := range raw {
		out = append(out, oeStr(k))
	}
	return out
}

func mcUnwrap(v any) (any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return v, nil
	}
	for _, key := range []string{"__number", "__string", "__bool"} {
		if x, ok := m[key]; ok {
			return x, nil
		}
	}
	if _, ok := m["__table"]; ok {
		out := map[string]any{}
		for k, x := range m {
			if k == "__table" {
				continue
			}
			u, err := mcUnwrap(x)
			if err != nil {
				return nil, err
			}
			out[k] = u
		}
		if len(out) > 0 && mcAllDigits(out) {
			arr := make([]any, 0, len(out))
			for i := 1; i <= len(out); i++ {
				x, ok := out[strconv.Itoa(i)]
				if !ok {
					return nil, mcUnrep("a table with all-digit keys is not the array 1..%d; "+
						"dkjson writes a contiguous run for a Lua array, so this dump has been "+
						"edited or truncated", len(out))
				}
				arr = append(arr, x)
			}
			return arr, nil
		}
		return out, nil
	}
	if _, ok := m["__other"]; ok {
		return nil, nil //nolint:nilnil // None is the dumper's value for an unrepresentable type
	}
	return v, nil
}

func mcAllDigits(m map[string]any) bool {
	for k := range m {
		if k == "" {
			return false
		}
		for _, r := range k {
			if !unicode.IsDigit(r) {
				return false
			}
		}
	}
	return true
}

func mcGoStr(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

func mcGoNum(v any) string {
	if n, ok := v.(json.Number); ok {
		return pyNumberRepr(n)
	}
	if f, ok := oeFloat(v); ok {
		if f == float64(int64(f)) {
			return strconv.FormatInt(int64(f), 10)
		}
		return pyFloat(f)
	}
	return oeStr(v)
}

func mcBits(value any, table []mcFlagBit) (string, bool) {
	n, ok := gemsInt(value)
	if !ok {
		return "", false
	}
	rest := int64(n)
	out := []string{}
	for _, row := range table {
		if rest&row.bit != 0 {
			out = append(out, "mod."+row.name)
			rest &^= row.bit
		}
	}
	if rest != 0 {
		return "", false
	}
	if len(out) == 0 {
		return "", false
	}
	return strings.Join(out, " | "), true
}

func mcRefuse(msg string) int {
	fmt.Fprintln(os.Stderr, "REFUSING: "+msg)
	return 1
}

var mcModType = map[string]string{
	"BASE": "mod.TypeBase", "INC": "mod.TypeIncrease", "MORE": "mod.TypeMore",
	"MAX": "mod.TypeMAX", "MIN": "mod.TypeMIN", "FLAG": "mod.TypeFlag", "LIST": "mod.TypeList",
	"OVERRIDE": "mod.TypeOverride", "DUMMY": "mod.TypeDummy",
	"CHANCE": "mod.TypeChance",
}

func mustNumber(s string) json.Number { return json.Number(s) }
