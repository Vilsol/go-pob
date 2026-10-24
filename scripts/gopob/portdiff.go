package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

const pdUsage = `Show OUR Go and the PINNED PoB's Lua for the same thing, side by side.

    task port-diff NAME=Condition:CanGainRage   # EVERY site on BOTH sides — the port-gap hunt
    task port-diff SYMBOL=enemyChaosDamage      # a name that appears in both trees
    task port-diff FUNC=CalcEnemyDamage         # a Go func, resolved through its //pob: citation
    task port-diff LUA=src/Modules/ConfigOptions.lua:1500-1520   # just show me that Lua
    task port-diff GREP='output%.Rage'          # FIND it in the Lua when you do not know the range`

var (
	pdMarker = regexp.MustCompile(`//pob(2)?:(block:)?(.+?)\s*$`)
	pdRange  = regexp.MustCompile(`^(src/[^\s:,]+?\.lua):(\d+)(?:-(\d+))?$`)
	pdGoFunc = regexp.MustCompile(`^func (?:\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\(`)
)

type pdEnv struct {
	root    string
	pob     string
	context int
	maxHits int
}

type pdRefusalError struct{ msg string }

func (e pdRefusalError) Error() string { return e.msg }

func pdRefuse(format string, a ...any) error { return pdRefusalError{fmt.Sprintf(format, a...)} }

func pdRel(e *pdEnv, p string) string {
	if r, err := filepath.Rel(e.root, p); err == nil && !strings.HasPrefix(r, "..") {
		return r
	}
	return p
}

func pdRead(path string) ([]string, error) {
	body, err := os.ReadFile(path) //nolint:gosec // an operator-supplied tree
	if err != nil {
		return nil, pdRefuse("%s is not readable: %s", path, errnoText(err))
	}
	if !utf8.Valid(body) {
		return nil, pdRefuse("%s is not valid UTF-8", path)
	}
	return strings.Split(string(body), "\n"), nil
}

func pdShow(e *pdEnv, path string, start, end int, label string) error {
	lines, err := pdRead(path)
	if err != nil {
		return err
	}
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	pad := 30 - len([]rune(label))
	if pad < 0 {
		pad = 0
	}
	fmt.Printf("\n──── %s: %s:%d-%d %s\n", label, pdRel(e, path), start, end,
		strings.Repeat("─", pad))
	if start > end {
		fmt.Println("  (empty range — the file is shorter than the citation says)")
		return nil
	}
	for n := start; n <= end; n++ {
		fmt.Printf("  %6d │ %s\n", n, lines[n-1])
	}
	return nil
}

func pdTrackedGoFiles(e *pdEnv) ([]string, error) {
	cmd := exec.CommandContext(context.Background(), "git", "ls-files", "-z", "--", "*.go")
	cmd.Dir = e.root
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.Error
		if errorsAs(err, &ee) {
			return nil, pdRefuse("cannot run git: %s", errnoText(err))
		}
		return nil, pdRefuse("git ls-files failed in %s", e.root)
	}
	var files []string
	for _, r := range strings.Split(string(out), "\x00") {
		if r != "" && !strings.HasSuffix(r, "_test.go") && !strings.Contains(r, "frontend/") {
			files = append(files, filepath.Join(e.root, r))
		}
	}
	return files, nil
}

func pdCitationAbove(lines []string, idx int) [][3]any {
	var out [][3]any
	for i := idx - 1; i >= 0 && strings.HasPrefix(strings.TrimLeft(lines[i], " \t"), "//"); i-- {
		m := pdMarker.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		for _, spec := range strings.Split(m[3], ",") {
			rm := pdRange.FindStringSubmatch(strings.TrimSpace(spec))
			if rm == nil {
				continue
			}
			s, _ := strconv.Atoi(rm[2])
			en := s
			if rm[3] != "" {
				en, _ = strconv.Atoi(rm[3])
			}
			path := rm[1]
			if m[1] != "" {
				path = "pob2:" + path
			}
			out = append(out, [3]any{path, s, en})
		}
	}
	return out
}

func pdByFunc(e *pdEnv, name string) error {
	files, err := pdTrackedGoFiles(e)
	if err != nil {
		return err
	}
	type hit struct {
		file  string
		i     int
		lines []string
	}
	var hits []hit
	for _, f := range files {
		lines, err := pdRead(f)
		if err != nil {
			return err
		}
		for i, ln := range lines {
			if m := pdGoFunc.FindStringSubmatch(ln); m != nil && m[1] == name {
				hits = append(hits, hit{f, i, lines})
			}
		}
	}
	if len(hits) == 0 {
		return pdRefuse("no Go func named %s in any tracked non-test .go file.\n"+
			"Try SYMBOL=%s to grep both trees for the literal instead.", pyRepr(name), name)
	}
	for _, h := range hits {
		end, depth := h.i+1, 0
		last := h.i + 400
		if last > len(h.lines) {
			last = len(h.lines)
		}
		for j := h.i; j < last; j++ {
			depth += strings.Count(h.lines[j], "{") - strings.Count(h.lines[j], "}")
			end = j + 1
			if depth <= 0 && j > h.i {
				break
			}
		}
		if err := pdShow(e, h.file, h.i+1, end, "OURS"); err != nil {
			return err
		}
		cites := pdCitationAbove(h.lines, h.i)
		if len(cites) == 0 {
			fmt.Printf("\n  ⚠️ %s:%d carries NO //pob: citation, so there is nothing to show "+
				"beside it.\n     That is itself a finding. Use "+
				"SYMBOL= to search the Lua by name.\n", pdRel(e, h.file), h.i+1)
			continue
		}
		for _, c := range cites {
			pob, rel := e.pob, c[0].(string)
			if after, ok := strings.CutPrefix(rel, "pob2:"); ok {
				pob, rel = pcPoB2Dir(), after
			}
			lua := filepath.Join(pob, rel)
			if _, err := os.Stat(lua); err != nil {
				fmt.Printf("\n  ⚠️ cited %s does not exist under %s — the pin may be wrong "+
					"(run `task pob-pin`).\n", rel, pob)
				continue
			}
			if err := pdShow(e, lua, c[1].(int), c[2].(int), "PoB (cited)"); err != nil {
				return err
			}
		}
	}
	return nil
}

func pdByLua(e *pdEnv, spec string) error {
	rm := pdRange.FindStringSubmatch(strings.TrimSpace(spec))
	if rm == nil {
		return pdRefuse("LUA=%s is not `src/<path>.lua:<start>[-<end>]`", pyRepr(spec))
	}
	s, _ := strconv.Atoi(rm[2])
	en := s
	if rm[3] != "" {
		en, _ = strconv.Atoi(rm[3])
	}
	path := filepath.Join(e.pob, rm[1])
	if _, err := os.Stat(path); err != nil { //nolint:gosec // an operator-supplied checkout path
		return pdRefuse("%s does not exist — set POB_DIR, and check `task pob-pin`.", path)
	}
	return pdShow(e, path, s, en, "PoB")
}

func pdLuaFiles(e *pdEnv) []string {
	src := filepath.Join(e.pob, "src")
	fi, err := os.Stat(src)
	if err != nil || !fi.IsDir() {
		return nil
	}
	var out []string
	_ = filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error { //nolint:gosec // an operator-supplied checkout
		if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".lua") {
			out = append(out, p)
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return pdPathLess(out[i], out[j]) })
	return out
}

func pdPathLess(a, b string) bool {
	as := strings.Split(a, string(filepath.Separator))
	bs := strings.Split(b, string(filepath.Separator))
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] != bs[i] {
			return as[i] < bs[i]
		}
	}
	return len(as) < len(bs)
}

func pdGrepTree(files []string, needle string) ([][2]any, error) {
	var out [][2]any
	for _, f := range files {
		lines, err := pdRead(f)
		if err != nil {
			return nil, err
		}
		for i, ln := range lines {
			if strings.Contains(ln, needle) {
				out = append(out, [2]any{f, i + 1})
			}
		}
	}
	return out, nil
}

func pdDistinct(hits [][2]any) int {
	seen := map[string]bool{}
	for _, h := range hits {
		seen[h[0].(string)] = true
	}
	return len(seen)
}

func pdCGSymbols(rootDir, name string) [][3]any {
	if fi, err := os.Stat(filepath.Join(rootDir, ".codegraph")); err != nil || !fi.IsDir() {
		return nil
	}
	//nolint:gosec // the symbol is the user's own query, passed as one argv element
	cmd := exec.CommandContext(context.Background(), "codegraph", "query", name,
		"--json", "-l", "20", "-p", rootDir)
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return nil
	}
	var rows []map[string]any
	if json.Unmarshal(out, &rows) != nil {
		return nil
	}
	var res [][3]any
	for _, r := range rows {
		n, _ := r["node"].(map[string]any)
		if n == nil {
			continue
		}
		if s, _ := n["name"].(string); s != name {
			continue
		}
		p, _ := n["filePath"].(string)
		s, sok := pdJSONInt(n["startLine"])
		e2, eok := pdJSONInt(n["endLine"])
		if p != "" && sok && eok && e2 >= s {
			res = append(res, [3]any{filepath.Join(rootDir, p), s, e2})
		}
	}
	return res
}

func pdJSONInt(v any) (int, bool) {
	f, ok := v.(float64)
	if !ok || f != float64(int(f)) {
		return 0, false
	}
	return int(f), true
}

func pdVerifyRange(e *pdEnv, path string, start int, name string) string {
	lines, err := pdRead(path)
	if err != nil {
		return ""
	}
	if !(1 <= start && start <= len(lines)) || !strings.Contains(lines[start-1], name) {
		return fmt.Sprintf("  ⚠️ STALE INDEX: %s:%d does not contain %s. The range came\n"+
			"     from CodeGraph and the CONTENT from disk, so what you see below is real but\n"+
			"     MISPLACED — do not cite it. `task pob-pin` reindexes the pinned checkout\n"+
			"     automatically; this repo's index is watched by a daemon.",
			pdRel(e, path), start, pyRepr(name))
	}
	return ""
}

func pdBySymbol(e *pdEnv, sym string) error {
	goFiles, err := pdTrackedGoFiles(e)
	if err != nil {
		return err
	}
	ours, err := pdGrepTree(goFiles, sym)
	if err != nil {
		return err
	}
	luaFiles := pdLuaFiles(e)
	if len(luaFiles) == 0 {
		return pdRefuse("no Lua under %s/src — set POB_DIR, and check `task pob-pin`.", e.pob)
	}
	theirs, err := pdGrepTree(luaFiles, sym)
	if err != nil {
		return err
	}

	fmt.Printf("=== port-diff SYMBOL=%s ===\n", pyRepr(sym))
	fmt.Printf("  ours   %d hit(s) across %d tracked Go file(s)\n", len(ours), pdDistinct(ours))
	fmt.Printf("  PoB    %d hit(s) across %d Lua file(s)  (%s)\n", len(theirs), pdDistinct(theirs), e.pob)

	shown := 0
	for _, side := range [][2]string{{"OURS", e.root}, {"PoB", e.pob}} {
		for _, s := range pdCGSymbols(side[1], sym) {
			if warn := pdVerifyRange(e, s[0].(string), s[1].(int), sym); warn != "" {
				fmt.Println("\n" + warn)
			}
			if err := pdShow(e, s[0].(string), s[1].(int), s[2].(int),
				side[0]+" (whole symbol)"); err != nil {
				return err
			}
			shown++
		}
	}
	if shown > 0 {
		fmt.Println("\n  ↑ whole symbol(s) via CodeGraph. Literal text hits follow; a name can be BOTH a\n" +
			"    symbol and a string, and on this project it usually is.")
	}

	if len(ours) == 0 && len(theirs) == 0 {
		return pdRefuse("%s appears on NEITHER side. That is far more likely to be a typo or a case\n"+
			"difference than a real absence — CalcActiveSkill.lua writes `skillData.CritChance`\n"+
			"with a capital C among 21 lower-camel neighbours, and a case-sensitive query there\n"+
			"reports 0 on the one row with a real population. Check the spelling, then believe it.",
			pyRepr(sym))
	}

	for _, side := range []struct {
		label string
		hits  [][2]any
	}{{"OURS", ours}, {"PoB", theirs}} {
		if len(side.hits) == 0 {
			tail := "We may have invented it."
			if side.label == "OURS" {
				tail = "We may not have ported it."
			}
			fmt.Printf("\n──── %s: NO HITS. %s\n", side.label, tail)
			continue
		}
		n := len(side.hits)
		if n > e.maxHits {
			n = e.maxHits
		}
		for _, h := range side.hits[:n] {
			if err := pdShow(e, h[0].(string), h[1].(int)-e.context, h[1].(int)+e.context,
				side.label); err != nil {
				return err
			}
		}
		if len(side.hits) > e.maxHits {
			fmt.Printf("\n  … %d more %s hit(s) NOT shown (MAX_HITS=%d — raise it, or narrow the "+
				"symbol).\n", len(side.hits)-e.maxHits, side.label, e.maxHits)
		}
	}
	return nil
}

func pdByGrep(e *pdEnv, pat string) error {
	luaFiles := pdLuaFiles(e)
	if len(luaFiles) == 0 {
		return pdRefuse("no Lua under %s/src — set POB_DIR, and check `task pob-pin`.", e.pob)
	}
	rx, err := regexp.Compile(pat)
	if err != nil {
		return pdRefuse("GREP=%s is not a valid regex", pyRepr(pat))
	}
	var hits [][2]any
	for _, f := range luaFiles {
		lines, rerr := pdRead(f)
		if rerr != nil {
			return rerr
		}
		for i, ln := range lines {
			if rx.MatchString(ln) {
				hits = append(hits, [2]any{f, i + 1})
			}
		}
	}
	fmt.Printf("=== port-diff GREP=%s over %d Lua file(s) under %s/src ===\n",
		pyRepr(pat), len(luaFiles), e.pob)
	if len(hits) == 0 {
		return pdRefuse("%s matches nothing. Check the case (CalcActiveSkill.lua writes\n"+
			"`skillData.CritChance` with a capital C among 21 lower-camel neighbours) and\n"+
			"remember this is a REGEX — `output.Rage` matches any char where you meant a dot.",
			pyRepr(pat))
	}
	fmt.Printf("  %d hit(s) across %d file(s)\n", len(hits), pdDistinct(hits))
	n := len(hits)
	if n > e.maxHits {
		n = e.maxHits
	}
	for _, h := range hits[:n] {
		if err := pdShow(e, h[0].(string), h[1].(int)-e.context, h[1].(int)+e.context, "PoB"); err != nil {
			return err
		}
	}
	if len(hits) > e.maxHits {
		fmt.Printf("\n  … %d more hit(s) NOT shown (MAX_HITS=%d).\n", len(hits)-e.maxHits, e.maxHits)
	}
	return nil
}

func pdByName(e *pdEnv, name string) error {
	goFiles, err := pdTrackedGoFiles(e)
	if err != nil {
		return err
	}
	ours, err := pdGrepTree(goFiles, name)
	if err != nil {
		return err
	}
	luaFiles := pdLuaFiles(e)
	if len(luaFiles) == 0 {
		return pdRefuse("no Lua under %s/src — set POB_DIR, and check `task pob-pin`.", e.pob)
	}
	theirs, err := pdGrepTree(luaFiles, name)
	if err != nil {
		return err
	}

	fmt.Printf("=== port-diff NAME=%s ===\n", pyRepr(name))
	fmt.Printf("  PoB   %4d site(s) across %d file(s)\n", len(theirs), pdDistinct(theirs))
	fmt.Printf("  ours  %4d site(s) across %d file(s)\n", len(ours), pdDistinct(ours))
	switch gap := len(theirs) - len(ours); {
	case gap > 0:
		fmt.Printf("  ⚠️ PoB has %d MORE site(s). That is the port-gap signal — but a count is not a\n"+
			"     defect: one Go row can port several Lua rows, and vice versa. Read the lines.\n", gap)
	case gap < 0:
		fmt.Printf("  ⚠️ WE have %d more site(s). We have implemented things PoB does not have\n"+
			"     before; an over-applied filter and a starved producer both make numbers wrong.\n", -gap)
	}

	if len(ours) == 0 && len(theirs) == 0 {
		return pdRefuse("%s appears on NEITHER side — far more likely a typo or a case difference than\n"+
			"a real absence. ⚠️ AND A NAME BUILT BY CONCATENATION HAS NO LITERAL: "+
			"`Multiplier:BlueSocketInHelmet` is assembled at item.go:976, so grepping the whole\n"+
			"name finds nothing while the code produces it. Try a distinctive FRAGMENT.", pyRepr(name))
	}

	for _, side := range []struct {
		label string
		hits  [][2]any
		base  string
	}{{"PoB", theirs, e.pob}, {"OURS", ours, e.root}} {
		fmt.Printf("\n──── %s ────\n", side.label)
		if len(side.hits) == 0 {
			fmt.Println("  (none)")
			continue
		}
		per := map[string][][2]any{}
		for _, h := range side.hits {
			r, _ := filepath.Rel(side.base, h[0].(string))
			per[r] = append(per[r], h)
		}
		keys := make([]string, 0, len(per))
		for k := range per {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if len(per[keys[i]]) != len(per[keys[j]]) {
				return len(per[keys[i]]) > len(per[keys[j]])
			}
			return keys[i] < keys[j]
		})
		for _, path := range keys {
			fmt.Printf("  %3d  %s\n", len(per[path]), path)
		}
		fmt.Println()
		for _, path := range keys {
			n := len(per[path])
			if n > e.maxHits {
				n = e.maxHits
			}
			for _, h := range per[path][:n] {
				text := ""
				if lines, rerr := pdRead(h[0].(string)); rerr == nil && h[1].(int)-1 < len(lines) {
					text = strings.TrimSpace(lines[h[1].(int)-1])
				}
				if r := []rune(text); len(r) > 100 {
					text = string(r[:100])
				}
				fmt.Printf("    %s:%d: %s\n", path, h[1].(int), text)
			}
			if len(per[path]) > e.maxHits {
				fmt.Printf("    … %d more in %s (MAX_HITS=%d)\n",
					len(per[path])-e.maxHits, path, e.maxHits)
			}
		}
	}
	return nil
}

func pdIntEnv(name, def string) (int, error) {
	raw := os.Getenv(name)
	if raw == "" {
		raw = def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, pdRefuse("%s=%s is not an integer", name, pyRepr(raw))
	}
	return n, nil
}

func portDiff(_ []string) int {
	if err := pdMain(); err != nil {
		var r pdRefusalError
		if errorsAs(err, &r) {
			fmt.Fprintf(os.Stderr, "REFUSING: %s\n", r.msg)
			return 3
		}
		fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
		return 3
	}
	return 0
}

func pdMain() error {
	r, err := root.Resolve()
	if err != nil {
		return pdRefuse("%v", err)
	}
	pob := pobPinDir()
	e := &pdEnv{root: r, pob: pob}
	if e.context, err = pdIntEnv("CONTEXT", "6"); err != nil {
		return err
	}
	if e.maxHits, err = pdIntEnv("MAX_HITS", "8"); err != nil {
		return err
	}

	sym, fn, lua := os.Getenv("SYMBOL"), os.Getenv("FUNC"), os.Getenv("LUA")
	grep, name := os.Getenv("GREP"), os.Getenv("NAME")
	// A value written like a //pob2: marker payload ("pob2:src/...") reads the PoE2 pin.
	for _, v := range []*string{&sym, &lua, &grep, &name} {
		if after, ok := strings.CutPrefix(*v, "pob2:"); ok {
			*v, pob = after, pcPoB2Dir()
			e.pob = pob
		}
	}
	given := 0
	for _, v := range []string{sym, fn, lua, grep, name} {
		if v != "" {
			given++
		}
	}
	if given != 1 {
		fmt.Fprintln(os.Stderr, pdUsage)
		fmt.Fprintln(os.Stderr, "\nREFUSING: pass exactly one of NAME=, SYMBOL=, FUNC=, LUA=, GREP=")
		os.Exit(3)
	}
	if fi, serr := os.Stat(pob); serr != nil || !fi.IsDir() { //nolint:gosec // the operator-supplied checkout path
		return pdRefuse("pinned PoB not found at %s. Set POB_DIR.\n"+
			"⚠️ ~/PoB/PoE1 is a NEWER DECOY — never point this at it.", pob)
	}
	switch {
	case name != "":
		return pdByName(e, name)
	case grep != "":
		return pdByGrep(e, grep)
	case lua != "":
		return pdByLua(e, lua)
	case fn != "":
		return pdByFunc(e, fn)
	}
	return pdBySymbol(e, sym)
}
