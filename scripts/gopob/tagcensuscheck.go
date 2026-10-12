package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	tcMinTreeSources   = 5
	tcExpectedExcluded = 184
	tcNoneType         = "\x00None"
)

func splitTop(s string, sep byte) []string {
	var out []string
	depth, start := 0, 0
	for i := range len(s) {
		switch ch := s[i]; {
		case ch == '{' || ch == '[':
			depth++
		case ch == '}' || ch == ']':
			depth--
		case ch == sep && depth == 0:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

type tcMod struct {
	Source *string  `json:"source"`
	Type   *string  `json:"type"`
	Flags  *string  `json:"flags"`
	Kw     *string  `json:"kw"`
	Fmt    *string  `json:"fmt"`
	Tags   []string `json:"tags"`
}

func tagSegment(name string, m tcMod) (string, bool) {
	typ, flags, kw := "", "-", "-"
	if m.Type != nil {
		typ = *m.Type
	}
	if m.Flags != nil {
		flags = *m.Flags
	}
	if m.Kw != nil {
		kw = *m.Kw
	}
	marker := fmt.Sprintf(" = %s|%s|%s|%s|", name, typ, flags, kw)
	f := ""
	if m.Fmt != nil {
		f = *m.Fmt
	}
	i := strings.LastIndex(f, marker)
	if i < 0 {
		return "", false
	}
	return f[i+len(marker):], true
}

func tagTypes(seg string, ok bool) []string {
	if !ok || seg == "-" || seg == "" {
		return nil
	}
	var out []string
	for _, raw := range splitTop(seg, ',') {
		typ := tcNoneType
		for _, kv := range splitTop(raw, '/') {
			k, v, found := strings.Cut(kv, "=")
			if k == "type" {
				if found {
					typ = v
				} else {
					typ = ""
				}
				break
			}
		}
		out = append(out, typ)
	}
	return out
}

func normalizeTreeSource(src string) (string, bool) {
	if !strings.HasPrefix(src, "Tree") {
		return "", false
	}
	rest := strings.TrimLeft(src[len("Tree"):], ":")
	for i := range len(rest) {
		if rest[i] == ' ' || rest[i] == '(' {
			rest = rest[:i]
			break
		}
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", false
	}
	n := 0
	for i := range len(rest) {
		if rest[i] < '0' || rest[i] > '9' {
			return "", false
		}
		n = n*10 + int(rest[i]-'0')
		if n >= 65536 {
			return "", false
		}
	}
	return "Tree:" + rest, true
}

func orderedObject(raw json.RawMessage) ([]string, []json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	t, err := dec.Token()
	if err != nil {
		return nil, nil, fmt.Errorf("decoding object: %w", err)
	}
	if d, ok := t.(json.Delim); !ok || d != '{' {
		return nil, nil, fmt.Errorf("expected an object, got %v", t)
	}
	var keys []string
	var vals []json.RawMessage
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, nil, fmt.Errorf("decoding key: %w", err)
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return nil, nil, fmt.Errorf("decoding value: %w", err)
		}
		keys = append(keys, fmt.Sprint(kt))
		vals = append(vals, v)
	}
	return keys, vals, nil
}

func tagCensusCheck(args []string) int {
	var dumps string
	if len(args) > 0 {
		dumps = args[0]
	} else {
		version, err := gameDataVersion(".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
			return 1
		}
		dumps = expandUser("~/go-pob-corpus/dumps-" + version)
	}
	files, _ := filepath.Glob(filepath.Join(dumps, "*.json"))
	sort.Strings(files)
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no dumps in %s\n", dumps)
		return 1
	}

	type actorTotals struct{ mods, tags int }
	totals := map[string]*actorTotals{"PLAYER": {}, "ENEMY": {}}
	unparsed := map[string]int{}
	typeDisagree := map[string]int{}
	histOrder := []string{}
	hist := map[string]int{}
	excluded, included := 0, 0

	for _, f := range files {
		b, err := os.ReadFile(f) //nolint:gosec // a path this tool globbed
		if err != nil {
			fmt.Fprintf(os.Stderr, "REFUSING: %s is not readable: %s\n", f, errnoText(err))
			return 1
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(b, &doc); err != nil {
			fmt.Fprintf(os.Stderr, "REFUSING: %s is not valid JSON\n", f)
			return 1
		}
		tree := map[string]bool{}
		if raw, ok := doc["modDB"]; ok && string(raw) != "null" {
			_, vals, err := orderedObject(raw)
			if err != nil {
				fmt.Fprintf(os.Stderr, "REFUSING: %s is not valid JSON\n", f)
				return 1
			}
			for _, v := range vals {
				var lst []tcMod
				if json.Unmarshal(v, &lst) != nil {
					continue
				}
				for _, m := range lst {
					src := ""
					if m.Source != nil {
						src = *m.Source
					}
					if n, ok := normalizeTreeSource(src); ok {
						tree[n] = true
					}
				}
			}
		}
		if len(tree) < tcMinTreeSources {
			excluded++
			continue
		}
		included++

		for _, ak := range [][2]string{{"PLAYER", "modDB"}, {"ENEMY", "enemyModDB"}} {
			actor, key := ak[0], ak[1]
			raw, ok := doc[key]
			if !ok || string(raw) == "null" {
				continue
			}
			names, vals, err := orderedObject(raw)
			if err != nil {
				fmt.Fprintf(os.Stderr, "REFUSING: %s is not valid JSON\n", f)
				return 1
			}
			for i, name := range names {
				var lst []tcMod
				if json.Unmarshal(vals[i], &lst) != nil {
					continue
				}
				for _, m := range lst {
					totals[actor].mods++
					seg, found := tagSegment(name, m)
					if !found {
						unparsed[actor]++
						continue
					}
					types := tagTypes(seg, true)
					totals[actor].tags += len(types)
					if !tcEqual(types, m.Tags) {
						typeDisagree[actor]++
					}
					if actor == "PLAYER" {
						for _, t := range types {
							if _, seen := hist[t]; !seen {
								histOrder = append(histOrder, t)
							}
							hist[t]++
						}
					}
				}
			}
		}
	}

	fmt.Printf("dumps            %s\n", dumps)
	fmt.Printf("files            %d\n", len(files))
	fmt.Printf("excluded         %d (want %d)\n", excluded, tcExpectedExcluded)
	fmt.Printf("included         %d\n", included)
	for _, actor := range []string{"PLAYER", "ENEMY"} {
		fmt.Printf("%-8s dumped mods %9d   dumped tags %9d   UNPARSED %d   fmt-vs-tags disagreements %d\n",
			actor, totals[actor].mods, totals[actor].tags, unparsed[actor], typeDisagree[actor])
	}
	fmt.Println("\nPLAYER tag types over included builds:")
	order := append([]string{}, histOrder...)
	sort.SliceStable(order, func(i, j int) bool { return hist[order[i]] > hist[order[j]] })
	for _, t := range order {
		label := t
		if t == tcNoneType {
			label = "None"
		}
		fmt.Printf("  %9d  %s\n", hist[t], label)
	}

	if excluded != tcExpectedExcluded {
		fmt.Fprintln(os.Stderr, "exclusion count moved: the dump set changed")
		return 1
	}
	if unparsed["PLAYER"]+unparsed["ENEMY"]+typeDisagree["PLAYER"]+typeDisagree["ENEMY"] != 0 {
		fmt.Fprintln(os.Stderr, "extraction is not clean")
		return 1
	}
	return 0
}

func tcEqual(types, tags []string) bool {
	if len(types) != len(tags) {
		return false
	}
	for i := range types {
		if types[i] != tags[i] {
			return false
		}
	}
	return true
}
