package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type verdictRec struct {
	Parsed *bool    `json:"parsed"`
	Extra  *string  `json:"extra"`
	Names  []string `json:"names"`
	Line   *string  `json:"line"`
}

type caseRec struct {
	Name *string `json:"name"`
	Raw  *string `json:"raw"`
}

type triageRow struct {
	weight  int
	line    string
	names   []string
	missing []string
}

var reAllDigits = regexp.MustCompile(`^[0-9]+$`)

func triageDrops(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "REFUSING: usage: triage_drops.py <verdict.json> <cases.json> [top]")
		return 1
	}
	verdictPath, casesPath := args[0], args[1]
	top := 60
	if len(args) > 2 {
		if !reAllDigits.MatchString(args[2]) {
			fmt.Fprintf(os.Stderr, "REFUSING: top must be a non-negative integer, not %s\n", args[2])
			return 1
		}
		top, _ = strconv.Atoi(args[2])
	}

	verdictRaw, rc := readFile(verdictPath)
	if rc != 0 {
		return rc
	}
	var verdicts []verdictRec
	if rc := decodeJSON(verdictPath, verdictRaw, &verdicts); rc != 0 {
		return rc
	}
	if len(verdicts) == 0 {
		fmt.Fprintln(os.Stderr, "REFUSING: verdict file is empty")
		return 1
	}
	if !anyCarriesNames(verdictRaw) {
		fmt.Fprintln(os.Stderr, "REFUSING: no record carries `names`; regenerate with the current "+
			"oracle/mod_parse.lua or every line will look BLOCKED")
		return 1
	}

	casesRaw, rc := readFile(casesPath)
	if rc != 0 {
		return rc
	}
	var cases []caseRec
	if rc := decodeJSON(casesPath, casesRaw, &cases); rc != 0 {
		return rc
	}

	weight := map[string]int{}
	for i, c := range cases {
		if c.Name == nil {
			fmt.Fprintf(os.Stderr, "REFUSING: %s record %d has no `name`\n", casesPath, i)
			return 1
		}
		parts := strings.SplitN(*c.Name, "|", 3)
		if len(parts) < 2 {
			continue
		}
		if c.Raw == nil {
			fmt.Fprintf(os.Stderr, "REFUSING: %s record %d has no `raw`\n", casesPath, i)
			return 1
		}
		lines := strings.Split(strings.TrimSpace(*c.Raw), "\n")
		n, ok := pyInt(parts[1])
		if !ok {
			continue
		}
		weight[lines[len(lines)-1]] = n
	}

	cache := map[string]bool{}
	var ready, blocked []triageRow
	for i, v := range verdicts {
		if v.Parsed == nil || !*v.Parsed {
			continue
		}
		if v.Extra != nil && strings.TrimSpace(*v.Extra) != "" {
			continue
		}
		if len(v.Names) == 0 {
			continue
		}
		if v.Line == nil {
			fmt.Fprintf(os.Stderr, "REFUSING: %s record %d has no `line`\n", verdictPath, i)
			return 1
		}
		var missing []string
		for _, n := range v.Names {
			if _, seen := cache[n]; !seen {
				cache[n] = hasConsumer(n)
			}
			if !cache[n] {
				missing = append(missing, n)
			}
		}
		row := triageRow{weight: weight[*v.Line], line: *v.Line, names: v.Names, missing: missing}
		if len(missing) > 0 {
			blocked = append(blocked, row)
		} else {
			ready = append(ready, row)
		}
	}

	sort.SliceStable(ready, func(i, j int) bool { return ready[i].weight > ready[j].weight })
	sort.SliceStable(blocked, func(i, j int) bool { return blocked[i].weight > blocked[j].weight })

	fmt.Printf("READY   %4d lines, %6d build-instances -- every mod name already has a consumer\n",
		len(ready), sumWeights(ready))
	fmt.Printf("BLOCKED %4d lines, %6d build-instances -- needs a consumer first (grep is an "+
		"upper bound on READY, see header)\n", len(blocked), sumWeights(blocked))
	fmt.Println()
	fmt.Printf("=== READY, top %d ===\n", top)
	for _, r := range head(ready, top) {
		fmt.Printf("%6d  %s  %s\n", r.weight, padTrunc(strings.Join(r.names, ","), 44),
			truncRunes(r.line, 78))
	}
	fmt.Println()
	fmt.Println("=== BLOCKED, top 20 (each needs its consumer written) ===")
	for _, r := range head(blocked, 20) {
		fmt.Printf("%6d  missing=%s  %s\n", r.weight, padTrunc(strings.Join(r.missing, ","), 40),
			truncRunes(r.line, 70))
	}
	return 0
}

func sumWeights(rows []triageRow) int {
	t := 0
	for _, r := range rows {
		t += r.weight
	}
	return t
}

func head(rows []triageRow, n int) []triageRow {
	if n > len(rows) {
		n = len(rows)
	}
	return rows[:n]
}

func anyCarriesNames(b []byte) bool {
	var raw []map[string]json.RawMessage
	if json.Unmarshal(b, &raw) != nil {
		return false
	}
	for _, r := range raw {
		if _, ok := r["names"]; ok {
			return true
		}
	}
	return false
}

func readFile(path string) ([]byte, int) {
	b, err := os.ReadFile(path) //nolint:gosec // an operator-supplied path
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: %s is not readable: %s\n", path, errnoText(err))
		return nil, 1
	}
	return b, 0
}

func decodeJSON(path string, b []byte, into any) int {
	if err := json.Unmarshal(b, into); err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: %s is not valid JSON\n", path)
		return 1
	}
	return 0
}

func pyInt(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	sign := ""
	if s[0] == '+' || s[0] == '-' {
		sign, s = string(s[0]), s[1:]
	}
	if s == "" || strings.HasPrefix(s, "_") || strings.HasSuffix(s, "_") ||
		strings.Contains(s, "__") {
		return 0, false
	}
	s = strings.ReplaceAll(s, "_", "")
	if !reAllDigits.MatchString(s) {
		return 0, false
	}
	n, err := strconv.Atoi(sign + s)
	return n, err == nil
}

func truncRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}

func padTrunc(s string, n int) string {
	return fmt.Sprintf("%-*s", n, truncRunes(s, n))
}

func hasConsumer(name string) bool {
	needle := []byte(`"` + name + `"`)
	found := false
	for _, dir := range []string{"calculator/", "data/", "moddb/"} {
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || found {
				return nil //nolint:nilerr // grep skips what it cannot read, and so does this
			}
			if d.IsDir() || !strings.HasSuffix(p, ".go") {
				return nil
			}
			if strings.HasSuffix(p, "_test.go") || strings.HasSuffix(p, "calculator/mod_parser.go") {
				return nil
			}
			b, readErr := os.ReadFile(p) //nolint:gosec // a repo-relative source path
			if readErr == nil && bytes.Contains(b, needle) {
				found = true
			}
			return nil
		})
	}
	return found
}
