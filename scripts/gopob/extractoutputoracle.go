package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

const eooUsage = "usage: extract_output_oracle.py <dumpsDir> <out.jsonl>"

var eooOptionalTables = []string{
	"outputMainHand",
	"outputOffHand",
	"enemyOutput",
	"minionOutput",
	"minionOutputMainHand",
	"minionOutputOffHand",
}

func eooRefuse(msg string) int {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	return 1
}

func extractOutputOracle(args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, eooUsage)
		return 1
	}
	dumpsDir, outPath := args[0], args[1]

	entries, err := os.ReadDir(dumpsDir)
	if err != nil {
		return eooRefuse(fmt.Sprintf("%s is not a readable directory: %s", dumpsDir, errnoText(err)))
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return eooRefuse(fmt.Sprintf("%s holds no .json dumps", dumpsDir))
	}

	out, err := os.Create(outPath) //nolint:gosec // an operator-supplied path, as in the original
	if err != nil {
		return eooRefuse(fmt.Sprintf("cannot write %s: %s", outPath, errnoText(err)))
	}
	defer out.Close() //nolint:errcheck // the flush below is what can fail meaningfully
	w := bufio.NewWriter(out)

	written := 0
	for _, name := range names {
		dump, code := eooLoadDump(pyPathJoin(dumpsDir, name))
		if code != 0 {
			return code
		}
		rec := map[string]any{"build": strings.TrimSuffix(name, ".json") + ".xml"}
		if eooTruthy(dump["error"]) {
			rec["error"] = dump["error"]
			rec["treeSources"] = 0
			rec["output"] = map[string]any{}
		} else {
			rec["treeSources"] = eooTreeSources(dump)
			rec["output"] = eooNumeric(dump["output"])
			for _, key := range eooOptionalTables {
				if v, ok := dump[key]; ok && v != nil {
					rec[key] = eooNumeric(v)
				}
			}
		}
		var b strings.Builder
		pyDumps(rec, &b)
		if _, err := w.WriteString(b.String() + "\n"); err != nil {
			return eooRefuse(fmt.Sprintf("cannot write %s: %s", outPath, errnoText(err)))
		}
		written++
	}
	if err := w.Flush(); err != nil {
		return eooRefuse(fmt.Sprintf("cannot write %s: %s", outPath, errnoText(err)))
	}
	fmt.Fprintf(os.Stderr, "oracle: wrote %d records to %s\n", written, outPath)
	return 0
}

func eooLoadDump(path string) (map[string]any, int) {
	m, reason := pyLoadJSONObject(path)
	if reason != "" {
		return nil, eooRefuse(reason)
	}
	return m, 0
}

func eooTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case json.Number:
		f, err := t.Float64()
		return err == nil && f != 0
	case []any:
		return len(t) != 0
	case map[string]any:
		return len(t) != 0
	}
	return true
}

func eooNumeric(table any) map[string]any {
	m, ok := table.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if n, ok := v.(json.Number); ok {
			out[k] = n
		}
	}
	return out
}

func eooTreeSources(dump map[string]any) int {
	modDB, ok := dump["modDB"].(map[string]any)
	if !ok {
		return 0
	}
	ids := map[string]struct{}{}
	for _, v := range modDB {
		mods, ok := v.([]any)
		if !ok {
			continue
		}
		for _, mv := range mods {
			m, ok := mv.(map[string]any)
			if !ok {
				continue
			}
			src, ok := m["source"].(string)
			if !ok || !strings.HasPrefix(src, "Tree") {
				continue
			}
			rest := strings.TrimLeft(src[len("Tree"):], ":")
			cut := -1
			for _, sep := range []byte{' ', '('} {
				if i := strings.IndexByte(rest, sep); i >= 0 && (cut < 0 || i < cut) {
					cut = i
				}
			}
			if cut >= 0 {
				rest = rest[:cut]
			}
			rest = strings.TrimSpace(rest)
			if !eooASCIIDigits(rest) {
				continue
			}
			if n, err := strconv.Atoi(rest); err == nil && n < 65536 {
				ids[rest] = struct{}{}
			}
		}
	}
	return len(ids)
}

func eooASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
