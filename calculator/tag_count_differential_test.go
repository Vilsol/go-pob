package calculator

//pob:src/Data/ModCache.lua:1

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func tdSplitTop(body string) []string {
	var out []string
	depth, start := 0, 0
	inStr := false
	for i := 0; i < len(body); i++ {
		c := body[i]
		if inStr {
			if c == '\\' {
				i++
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, body[start:i])
				start = i + 1
			}
		}
	}
	return append(out, body[start:])
}

func tdInner(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return "", false
	}
	return s[1 : len(s)-1], true
}

func tdPositional(entry string) (string, bool) {
	e := strings.TrimSpace(entry)
	if !strings.HasPrefix(e, "[") {
		return "", false
	}
	end := strings.Index(e, "]=")
	if end < 0 {
		return "", false
	}
	k := e[1:end]
	if k == "" {
		return "", false
	}
	for i := 0; i < len(k); i++ {
		if k[i] < '0' || k[i] > '9' {
			return "", false
		}
	}
	return e[end+2:], true
}

func tdValue(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return strings.ReplaceAll(v[1:len(v)-1], `\"`, `"`)
	}
	if body, ok := tdInner(v); ok {
		var items []string
		for _, e := range tdSplitTop(body) {
			if rhs, isPos := tdPositional(e); isPos {
				items = append(items, tdValue(rhs))
			} else {
				items = append(items, tdValue(e))
			}
		}
		return "{" + strings.Join(items, ",") + "}"
	}
	return v
}

func tdTag(lit string) canonTag {
	body, ok := tdInner(lit)
	if !ok {
		return canonTag{typ: "UNPARSED"}
	}
	typ := ""
	params := map[string]string{}
	for _, f := range tdSplitTop(body) {
		k, v, found := strings.Cut(strings.TrimSpace(f), "=")
		if !found {
			continue
		}
		if k == "type" {
			typ = tdValue(v)
			continue
		}
		params[k] = tdValue(v)
	}
	return canonPobTag(typ, params)
}

func tdModFlags(modLit string) (string, string) {
	body, ok := tdInner(modLit)
	if !ok {
		return "?", "?"
	}
	f, kf := "0", "0"
	for _, e := range tdSplitTop(body) {
		k, v, found := strings.Cut(strings.TrimSpace(e), "=")
		if !found {
			continue
		}
		switch k {
		case "flags":
			f = v
		case "keywordFlags":
			kf = v
		}
	}
	return f, kf
}

func tdParseRHS(rhs string) ([][]canonTag, bool) {
	modListLit := rhs[1 : len(rhs)-len(",nil}")]
	body, ok := tdInner(modListLit)
	if !ok {
		return nil, false
	}
	var mods [][]canonTag
	tdFlags = tdFlags[:0]
	for _, entry := range tdSplitTop(body) {
		modLit, isPos := tdPositional(entry)
		if !isPos {
			return nil, false
		}
		mBody, ok := tdInner(modLit)
		if !ok {
			return nil, false
		}
		f, kf := tdModFlags(modLit)
		tdFlags = append(tdFlags, [2]string{f, kf})
		tags := []canonTag{}
		for _, f := range tdSplitTop(mBody) {
			if tagLit, isPos := tdPositional(f); isPos {
				tags = append(tags, tdTag(tagLit))
			}
		}
		mods = append(mods, tags)
	}
	return mods, true
}

var tdFlags [][2]string

func tdRender(tags []canonTag) string {
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = t.String()
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func TestTagCountDifferential(t *testing.T) {
	pobRoot := os.Getenv("POB_DIR")
	if pobRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("NOT-RUN: no $HOME: %v", err)
		}
		pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
	}
	raw, err := os.ReadFile(filepath.Join(pobRoot, "src", "Data", "ModCache.lua"))
	if err != nil {
		t.Skipf("NOT-RUN: %v", err)
	}

	only := os.Getenv("TDONLY")
	var whole, unparsedRHS, countDiffers, comparableLines, cntDiff, setDiff int
	type row struct{ line, pob, ours string }
	var cntRows, setRows []row

	for _, ln := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(ln, `c["`) {
			continue
		}
		end := strings.Index(ln, `"]=`)
		if end < 0 {
			continue
		}
		key := strings.ReplaceAll(ln[3:end], `\"`, `"`)
		rhs := ln[end+3:]
		if !strings.HasPrefix(rhs, "{{[1]=") || !strings.HasSuffix(rhs, ",nil}") {
			continue
		}
		whole++
		pmods, ok := tdParseRHS(rhs)
		if !ok {
			unparsedRHS++
			continue
		}
		e := ParseMod(key, false)
		if e == nil || len(e.ModList) != len(pmods) || e.Extra != "" {
			countDiffers++
			continue
		}
		comparableLines++
		if only != "" && !strings.Contains(strings.ToLower(key), strings.ToLower(only)) {
			continue
		}
		countD, setD := false, false
		var pobAll, ourAll []string
		for i := range pmods {
			var ot []canonTag
			for _, tg := range e.ModList[i].Tags() {
				ot = append(ot, canonOurTag(tg))
			}
			if len(ot) != len(pmods[i]) {
				countD = true
			}
			if tdRender(ot) != tdRender(pmods[i]) {
				setD = true
			}
			pf, pkf := tdFlags[i][0], tdFlags[i][1]
			of := strconv.Itoa(int(e.ModList[i].Flags()))
			okf := strconv.Itoa(int(e.ModList[i].KeywordFlags()))
			pobAll = append(pobAll, tdRender(pmods[i])+"f"+pf+"k"+pkf)
			ourAll = append(ourAll, tdRender(ot)+"f"+of+"k"+okf)
			if pf != of || pkf != okf {
				setD = true
			}
		}
		if countD {
			cntDiff++
			cntRows = append(cntRows, row{key, strings.Join(pobAll, ""), strings.Join(ourAll, "")})
		}
		if setD {
			setDiff++
			setRows = append(setRows, row{key, strings.Join(pobAll, ""), strings.Join(ourAll, "")})
		}
	}

	t.Logf("PoB parses WHOLE with mods    %6d", whole)
	t.Logf("  rhs unparsed by the probe   %6d", unparsedRHS)
	t.Logf("  mod COUNT or leftover diff  %6d", countDiffers)
	t.Logf("  comparableLines                  %6d", comparableLines)
	t.Logf("  DIFFERING TAG COUNT         %6d", cntDiff)
	t.Logf("  DIFFERING TAG SET           %6d", setDiff)

	if comparableLines < 5000 {
		t.Fatalf("REFUSING: only %d comparableLines lines — the probe is broken, not the tree", comparableLines)
	}
	ctl := ParseMod("1% increased Minion Damage per Summoned Phantasm", false)
	t.Logf("CONTROL per Summoned Phantasm: mods=%d extra=%q", len(ctl.ModList), ctl.Extra)

	dump := func(label string, rows []row) {
		clusters := map[string][]string{}
		for _, r := range rows {
			clusters["pob="+r.pob+" ours="+r.ours] = append(clusters["pob="+r.pob+" ours="+r.ours], r.line)
		}
		keys := make([]string, 0, len(clusters))
		for k := range clusters {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if len(clusters[keys[i]]) != len(clusters[keys[j]]) {
				return len(clusters[keys[i]]) > len(clusters[keys[j]])
			}
			return keys[i] < keys[j]
		})
		t.Logf("=== %s: %d clusters ===", label, len(keys))
		for _, k := range keys {
			t.Logf("%3d  %s   e.g. %q", len(clusters[k]), k, clusters[k][0])
		}
	}
	dump("COUNT", cntRows)
	if os.Getenv("TDSET") != "" {
		dump("SET", setRows)
	}
}
