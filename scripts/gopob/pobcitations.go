package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	pcDefaultPoBRel       = "PathOfBuilding"
	pcManifest            = "scripts/port_manifest.txt"
	pcRefusalExit         = 3
	pcCitedFloor          = 16
	pcUnclassifiedCeiling = 0
	pcDriftMaxHits        = 12
)

var pcPortedPkgs = []string{"calculator", "moddb", "mod", "data", "pob", "utils", "builds"}

var pcClasses = []string{"PORT", "DATA", "GENERATED", "GO-NATIVE", "UNCLASSIFIED"}

var pcFuncLineGrandfather = map[string]bool{
	"calculator/calc_perform.go::calcDualWieldImpact":       true,
	"calculator/calc_perform.go::getTriggerDefaultCooldown": true,
	"calculator/cluster_jewels.go::findSocket":              true,
	"calculator/configs.go::enemyIsBossLevel":               true,
}

var (
	pcCiteRE        = regexp.MustCompile(`([A-Za-z][\w.\-]*(?:/[\w.\-]+)*\.lua):(\d+)(?:\s*-\s*(\d+))?`)
	pcMarkerRE      = regexp.MustCompile(`^\s*//pob:(.+?)\s*$`)
	pcEntryRE       = regexp.MustCompile(`^([\w.\-/]+\.lua)(?::(\d+)(?:-(\d+))?)?$`)
	pcLuaFunctionRE = regexp.MustCompile(`\bfunction\b`)
	pcBacktickRE    = regexp.MustCompile("`([^`]+)`")
	pcCodeishRE     = regexp.MustCompile(`[(){}\[\]=:.]`)
	pcIdentRE       = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]+`)
	pcTopFuncRE     = regexp.MustCompile(`(?m)^func(?:\s+\([^)]*\))?\s+(\w+)`)
	pcPackageRE     = regexp.MustCompile(`(?m)^package\s+\w+`)
)

var pcLuaKeywords = map[string]bool{
	"then": true, "else": true, "elseif": true, "local": true, "function": true,
	"return": true, "true": true, "false": true, "self": true, "and": true, "not": true,
	"nil": true, "end": true, "for": true, "while": true, "repeat": true, "until": true,
	"break": true, "pairs": true, "ipairs": true, "type": true, "table": true,
	"math": true, "string": true,
}

func pcRefuse(msg string) {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	os.Exit(pcRefusalExit)
}

func pcReadText(path, what string) string {
	body, err := os.ReadFile(path) //nolint:gosec // an operator-supplied path, as in the original
	if err != nil {
		pcRefuse(fmt.Sprintf("%s is not readable: %s", path, errnoText(err)))
	}
	if !utf8.Valid(body) {
		pcRefuse(fmt.Sprintf("%s is not valid UTF-8 (%s)", path, what))
	}
	return string(body)
}

func pcLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.SplitAfter(s, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func pcGit(dir string, args ...string) (string, bool) {
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // fixed argv, operator-supplied dir
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.Error
		if errors.As(err, &ee) {
			pcRefuse(fmt.Sprintf("cannot run git: %s", errnoText(err)))
		}
		return "", false
	}
	return string(out), true
}

func pcRepoRoot() string {
	out, ok := pcGit("", "rev-parse", "--show-toplevel")
	if !ok {
		pcRefuse("git rev-parse --show-toplevel failed; this must run inside the go-pob checkout")
	}
	return strings.TrimSpace(out)
}

func pcIndexPoB(pob string) map[string]int {
	src := pyPathJoin(pob, "src")
	fi, err := os.Stat(src) //nolint:gosec // an operator-supplied checkout path, as in the original
	if err != nil || !fi.IsDir() {
		pcRefuse(fmt.Sprintf("not a PoB checkout (no src/): %s", pob))
	}
	lines := map[string]int{}
	err = filepath.WalkDir(pob, func(p string, d os.DirEntry, err error) error { //nolint:gosec // an operator-supplied checkout path, as in the original
		if err != nil {
			return nil //nolint:nilerr // os.walk's onerror=None, ported deliberately
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".codegraph" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".lua") {
			return nil
		}
		body, rerr := os.ReadFile(p) //nolint:gosec // inside the operator-supplied checkout
		if rerr != nil {
			pcRefuse(fmt.Sprintf("%s is not readable: %s", p, errnoText(rerr)))
		}
		n := strings.Count(string(body), "\n")
		if len(body) > 0 && !strings.HasSuffix(string(body), "\n") {
			n++
		}
		rel, _ := filepath.Rel(pob, p)
		lines[rel] = n
		return nil
	})
	if err != nil {
		//go-only: os.walk yields nothing when the ROOT is unreadable; it does not raise.
		pcRefuse(fmt.Sprintf("cannot walk %s: %v", pob, err))
	}
	return lines
}

func pcCandidates(path string, lines map[string]int) []string {
	if _, ok := lines[path]; ok {
		return []string{path}
	}
	tail := "/" + path
	var out []string
	for p := range lines {
		if strings.HasSuffix(p, tail) {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

func pcDriftTokens(text string) []string {
	seen := map[string]bool{}
	for _, m := range pcBacktickRE.FindAllStringSubmatch(text, -1) {
		span := m[1]
		if !pcCodeishRE.MatchString(span) {
			continue
		}
		for _, t := range pcIdentRE.FindAllString(span, -1) {
			if !pcLuaKeywords[t] {
				seen[t] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func pcFindToken(pob, rel, tok string, start, end int) (bool, int, int) {
	body := pcLines(pcReadText(pyPathJoin(pob, rel), "cited Lua"))
	pat := regexp.MustCompile(`\b` + regexp.QuoteMeta(tok) + `\b`)
	var hits []int
	for i, l := range body {
		if pat.MatchString(l) {
			hits = append(hits, i+1)
		}
	}
	if len(hits) == 0 || len(hits) > pcDriftMaxHits {
		return true, 0, len(hits)
	}
	for _, h := range hits {
		if start <= h && h <= end {
			return true, 0, len(hits)
		}
	}
	best, bestD := hits[0], pcDist(hits[0], start, end)
	for _, h := range hits[1:] {
		if d := pcDist(h, start, end); d < bestD {
			best, bestD = h, d
		}
	}
	return false, best, len(hits)
}

func pcDist(h, start, end int) int {
	a, b := h-start, h-end
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	if a < b {
		return a
	}
	return b
}

func pcLuaRangeHasFunction(pob, rel string, start, end int) bool {
	for _, l := range pcSlice(pcLines(pcReadText(pyPathJoin(pob, rel), "cited Lua")), start, end) {
		s := strings.TrimSpace(l)
		if s != "" && !strings.HasPrefix(s, "--") && pcLuaFunctionRE.MatchString(s) {
			return true
		}
	}
	return false
}

func pcLuaHasCode(pob, rel string, start, end int) bool {
	for _, l := range pcSlice(pcLines(pcReadText(pyPathJoin(pob, rel), "cited Lua")), start, end) {
		s := strings.TrimSpace(l)
		if s != "" && !strings.HasPrefix(s, "--") {
			return true
		}
	}
	return false
}

func pcSlice(body []string, start, end int) []string {
	lo := start - 1
	if lo < 0 {
		lo = 0
	}
	if lo > len(body) {
		lo = len(body)
	}
	hi := end
	if hi < lo {
		hi = lo
	}
	if hi > len(body) {
		hi = len(body)
	}
	return body[lo:hi]
}

type pcLineText struct {
	line int
	text string
}

func pcBlankOut(src string) (string, []pcLineText, []pcLineText) {
	var code strings.Builder
	var comments, strs []pcLineText
	emit := func(text string) {
		for i := range len(text) {
			if text[i] == '\n' {
				code.WriteByte('\n')
			} else {
				code.WriteByte(' ')
			}
		}
	}
	i, n, line := 0, len(src), 1
	for i < n {
		switch {
		case strings.HasPrefix(src[i:], "//"):
			j := strings.IndexByte(src[i:], '\n')
			if j == -1 {
				j = n
			} else {
				j += i
			}
			comments = append(comments, pcLineText{line, src[i:j]})
			emit(src[i:j])
			i = j
		case strings.HasPrefix(src[i:], "/*"):
			j := strings.Index(src[i+2:], "*/")
			if j == -1 {
				j = n
			} else {
				j = i + 2 + j + 2
			}
			chunk := src[i:j]
			for k, l := range strings.Split(chunk, "\n") {
				comments = append(comments, pcLineText{line + k, l})
			}
			emit(chunk)
			line += strings.Count(chunk, "\n")
			i = j
		case src[i] == '"' || src[i] == '\'' || src[i] == '`':
			q, s, sline := src[i], i, line
			i++
			for i < n {
				switch {
				case src[i] == '\n':
					if q != '`' {
						goto done
					}
					line++
				case src[i] == '\\' && q != '`':
					i++
				case src[i] == q:
					i++
					goto done
				}
				i++
			}
		done:
			for k, l := range strings.Split(src[s:i], "\n") {
				strs = append(strs, pcLineText{sline + k, l})
			}
			emit(src[s:i])
		default:
			if src[i] == '\n' {
				line++
			}
			code.WriteByte(src[i])
			i++
		}
	}
	return code.String(), comments, strs
}

type pcFuncRef struct {
	line int
	name string
}

func pcGoTopLevelFuncs(code string) []pcFuncRef {
	ms := pcTopFuncRE.FindAllStringSubmatchIndex(code, -1)
	out := make([]pcFuncRef, 0, len(ms))
	for _, m := range ms {
		out = append(out, pcFuncRef{
			line: strings.Count(code[:m[0]], "\n") + 1,
			name: code[m[2]:m[3]],
		})
	}
	return out
}

type pcCite struct {
	gofile string
	goline int
	path   string
	start  int
	end    int
	formal bool
	block  bool
	status string
	detail string
	text   string
	drift  string
}

func (c *pcCite) where() string {
	if c.start < 0 {
		return c.path
	}
	if c.start == c.end {
		return fmt.Sprintf("%s:%d", c.path, c.start)
	}
	return fmt.Sprintf("%s:%d-%d", c.path, c.start, c.end)
}

func (c *pcCite) String() string {
	d := c.detail
	if d == "" {
		d = c.status
	}
	return fmt.Sprintf("%s:%d  %s  %s", c.gofile, c.goline, c.where(), d)
}

func pcGoFiles(repo string) []string {
	out, ok := pcGit(repo, "ls-files", "-z", "--cached", "--others", "--exclude-standard", "*.go")
	if !ok {
		pcRefuse(fmt.Sprintf("git ls-files failed in %s", repo))
	}
	seen := map[string]bool{}
	for _, f := range strings.Split(out, "\x00") {
		if f != "" && !strings.HasPrefix(f, "frontend/") && !strings.Contains(f, "/testdata/") {
			seen[f] = true
		}
	}
	files := make([]string, 0, len(seen))
	for f := range seen {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

func pcParseMarker(gofile string, line int, payload string) []*pcCite {
	block := false
	if strings.HasPrefix(payload, "block:") {
		block = true
		payload = payload[len("block:"):]
	}
	var cites []*pcCite
	for _, entry := range strings.Split(payload, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		m := pcEntryRE.FindStringSubmatch(entry)
		if m == nil {
			cites = append(cites, &pcCite{
				gofile: gofile, goline: line, path: entry, start: -1, end: -1, formal: true,
				status: "MALFORMED",
				detail: fmt.Sprintf("MALFORMED: %s is not `path.lua[:start[-end]]`", pyRepr(entry)),
			})
			continue
		}
		start, end := -1, -1
		if m[2] != "" {
			start, _ = strconv.Atoi(m[2])
			end = start
			if m[3] != "" {
				end, _ = strconv.Atoi(m[3])
			}
		}
		cites = append(cites, &pcCite{
			gofile: gofile, goline: line, path: m[1], start: start, end: end, formal: true,
		})
	}
	for _, c := range cites {
		c.block = block
	}
	return cites
}

func pcResolve(c *pcCite, lines map[string]int, pob string) {
	if c.status == "MALFORMED" {
		return
	}
	cands := pcCandidates(c.path, lines)
	if len(cands) == 0 {
		c.status = "NOFILE"
		c.detail = fmt.Sprintf("NOFILE: no %s in the pinned PoB", c.path)
		return
	}
	if _, exact := lines[c.path]; c.formal && !exact {
		c.status = "PATH"
		c.detail = fmt.Sprintf("PATH: write it from the PoB root, i.e. %s", cands[0])
		return
	}
	if c.start < 0 {
		c.status = "OK"
		c.path = cands[0]
		return
	}
	var fits []string
	for _, p := range cands {
		if 1 <= c.start && c.start <= c.end && c.end <= lines[p] {
			fits = append(fits, p)
		}
	}
	if len(fits) == 0 {
		c.status = "RANGE"
		c.detail = fmt.Sprintf("RANGE: %d-%d outside %s (%d lines)",
			c.start, c.end, cands[0], lines[cands[0]])
		return
	}
	if len(fits) > 1 {
		head := fits
		if len(head) > 3 {
			head = head[:3]
		}
		c.status = "AMBIG"
		c.detail = fmt.Sprintf("AMBIG: %s matches %d files (%s)",
			c.path, len(fits), strings.Join(head, ", "))
		return
	}
	c.path = fits[0]
	if !pcLuaHasCode(pob, c.path, c.start, c.end) {
		c.status = "EMPTY"
		c.detail = fmt.Sprintf("EMPTY: %s is entirely blank or comment", c.where())
		return
	}
	c.status = "OK"
}

type pcScan struct {
	formal     []*pcCite
	informal   []*pcCite
	fileMarker map[string]bool
	funcs      map[string][]pcFuncRef
	cited      map[string]map[int]string
	attached   map[string]map[int][]int
	order      []string
}

func pcCollect(repo, pob string) *pcScan {
	lines := pcIndexPoB(pob)
	sc := &pcScan{
		fileMarker: map[string]bool{},
		funcs:      map[string][]pcFuncRef{},
		cited:      map[string]map[int]string{},
		attached:   map[string]map[int][]int{},
	}
	for _, rel := range pcGoFiles(repo) {
		sc.order = append(sc.order, rel)
		src := pcReadText(pyPathJoin(repo, rel), "Go source")
		code, comments, strs := pcBlankOut(src)
		byLine := map[int]string{}
		for _, c := range comments {
			byLine[c.line] = c.text
		}
		markerLines := map[int]bool{}

		for _, ct := range comments {
			if m := pcMarkerRE.FindStringSubmatch(ct.text); m != nil {
				sc.formal = append(sc.formal, pcParseMarker(rel, ct.line, m[1])...)
				markerLines[ct.line] = true
				continue
			}
			sc.informal = append(sc.informal, pcSweep(rel, ct)...)
		}
		for _, st := range strs {
			sc.informal = append(sc.informal, pcSweep(rel, st)...)
		}

		pkgLine := 1
		if m := pcPackageRE.FindStringIndex(code); m != nil {
			pkgLine = strings.Count(code[:m[0]], "\n") + 1
		}
		for l := range markerLines {
			if l < pkgLine {
				sc.fileMarker[rel] = true
				break
			}
		}

		sc.funcs[rel] = pcGoTopLevelFuncs(code)
		sc.cited[rel] = map[int]string{}
		sc.attached[rel] = map[int][]int{}
		for _, fn := range sc.funcs[rel] {
			k := fn.line - 1
			var hits []int
			for {
				t, ok := byLine[k]
				if !ok || !strings.HasPrefix(strings.TrimLeft(t, " \t\r\n\v\f"), "//") {
					break
				}
				if markerLines[k] {
					hits = append(hits, k)
				}
				k--
			}
			if len(hits) > 0 {
				sc.cited[rel][fn.line] = fn.name
				sc.attached[rel][fn.line] = hits
			}
		}
	}

	for _, c := range append(append([]*pcCite{}, sc.formal...), sc.informal...) {
		pcResolve(c, lines, pob)
		if (c.status == "OK" || c.status == "EMPTY") && c.start >= 0 && c.text != "" {
			var bestTok string
			var bestNear, bestHits int
			have := false
			for _, tok := range pcDriftTokens(c.text) {
				inside, near, hits := pcFindToken(pob, c.path, tok, c.start, c.end)
				if !inside && (!have || hits < bestHits) {
					bestTok, bestNear, bestHits, have = tok, near, hits, true
				}
			}
			if have {
				c.drift = fmt.Sprintf("quotes `%s` (%d hit(s) in the file), nearest at :%d",
					bestTok, bestHits, bestNear)
			}
		}
	}
	return sc
}

func pcSweep(rel string, lt pcLineText) []*pcCite {
	ms := pcCiteRE.FindAllStringSubmatch(lt.text, -1)
	out := make([]*pcCite, 0, len(ms))
	for _, cm := range ms {
		a, _ := strconv.Atoi(cm[2])
		b := a
		if cm[3] != "" {
			b, _ = strconv.Atoi(cm[3])
		}
		out = append(out, &pcCite{
			gofile: rel, goline: lt.line, path: cm[1], start: a, end: b, text: lt.text,
		})
	}
	return out
}

type pcFLFail struct{ key, detail string }

func pcFuncLineFailures(pob string, sc *pcScan) []pcFLFail {
	bySite := map[string][]*pcCite{}
	for _, c := range sc.formal {
		k := fmt.Sprintf("%s\x00%d", c.gofile, c.goline)
		bySite[k] = append(bySite[k], c)
	}
	var out []pcFLFail
	rels := make([]string, 0, len(sc.attached))
	for rel := range sc.attached {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		names := map[int]string{}
		for _, fn := range sc.funcs[rel] {
			names[fn.line] = fn.name
		}
		flines := make([]int, 0, len(sc.attached[rel]))
		for fl := range sc.attached[rel] {
			flines = append(flines, fl)
		}
		sort.Ints(flines)
		for _, fline := range flines {
			var cs []*pcCite
			for _, ml := range sc.attached[rel][fline] {
				cs = append(cs, bySite[fmt.Sprintf("%s\x00%d", rel, ml)]...)
			}
			if len(cs) == 0 {
				continue
			}
			anyBlock := false
			for _, c := range cs {
				if c.block {
					anyBlock = true
				}
			}
			if anyBlock {
				continue
			}
			var ranged []*pcCite
			for _, c := range cs {
				if c.status == "OK" && c.start >= 0 {
					ranged = append(ranged, c)
				}
			}
			if len(ranged) == 0 {
				continue
			}
			ok := false
			for _, c := range ranged {
				if pcLuaRangeHasFunction(pob, c.path, c.start, c.end) {
					ok = true
					break
				}
			}
			if ok {
				continue
			}
			name, has := names[fline]
			if !has {
				if n, ok2 := sc.cited[rel][fline]; ok2 {
					name = n
				} else {
					name = "?"
				}
			}
			wheres := make([]string, len(ranged))
			for i, c := range ranged {
				wheres[i] = c.where()
			}
			out = append(out, pcFLFail{
				key: fmt.Sprintf("%s::%s", rel, name),
				detail: fmt.Sprintf("%s:%d  func %s cites %s, which holds no Lua `function` line",
					rel, fline, name, strings.Join(wheres, ", ")),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].key != out[j].key {
			return out[i].key < out[j].key
		}
		return out[i].detail < out[j].detail
	})
	return out
}

type pcEntry struct {
	cls     string
	payload string
	line    int
}

type pcBad struct {
	line int
	code string
	msg  string
}

type pcFail struct{ code, where, msg string }

func pcSplitN(s string, n int) []string {
	var out []string
	i := 0
	for len(out) < n {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' ||
			s[i] == '\v' || s[i] == '\f') {
			i++
		}
		if i >= len(s) {
			return out
		}
		j := i
		for j < len(s) && !(s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r' ||
			s[j] == '\v' || s[j] == '\f') {
			j++
		}
		out = append(out, s[i:j])
		i = j
	}
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' ||
		s[i] == '\v' || s[i] == '\f') {
		i++
	}
	if i < len(s) {
		out = append(out, s[i:])
	}
	return out
}

func pcParseManifest(text string) (map[string]pcEntry, []pcBad) {
	entries := map[string]pcEntry{}
	var bad []pcBad
	for n, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := pcSplitN(line, 2)
		if len(parts) < 2 {
			bad = append(bad, pcBad{n + 1, "MANIFEST-SYNTAX",
				fmt.Sprintf("%s is not `<go file>  <CLASS>  <payload>`", pyRepr(line))})
			continue
		}
		gofile, cls := parts[0], parts[1]
		payload := ""
		if len(parts) > 2 {
			payload = strings.TrimSpace(parts[2])
		}
		known := false
		for _, c := range pcClasses {
			if c == cls {
				known = true
			}
		}
		if !known {
			bad = append(bad, pcBad{n + 1, "MANIFEST-CLASS",
				fmt.Sprintf("%s declares %s; the classes are %s",
					gofile, pyRepr(cls), strings.Join(pcClasses, "/"))})
			continue
		}
		if _, dup := entries[gofile]; dup {
			bad = append(bad, pcBad{n + 1, "MANIFEST-DUP",
				fmt.Sprintf("%s is listed twice", gofile)})
			continue
		}
		entries[gofile] = pcEntry{cls, payload, n + 1}
	}
	return entries, bad
}

func pcLoadManifest(repo string) (map[string]pcEntry, []pcBad) {
	return pcParseManifest(pcReadText(pyPathJoin(repo, pcManifest), "the port manifest"))
}

func pcCheckManifest(repo, pob string, lines map[string]int, entries map[string]pcEntry,
	present map[string]bool) ([]pcFail, int) {
	var fails []pcFail
	fail := func(code, where, msg string) { fails = append(fails, pcFail{code, where, msg}) }

	if present == nil {
		present = map[string]bool{}
		for _, f := range pcGoFiles(repo) {
			if !strings.HasSuffix(f, "_test.go") {
				present[f] = true
			}
		}
	}
	var missing, dangling []string
	for f := range present {
		if _, ok := entries[f]; !ok {
			missing = append(missing, f)
		}
	}
	for f := range entries {
		if !present[f] {
			dangling = append(dangling, f)
		}
	}
	sort.Strings(missing)
	sort.Strings(dangling)
	for _, gofile := range missing {
		fail("MANIFEST-MISSING", gofile,
			fmt.Sprintf("not classified in %s. A new file is UNCLASSIFIED by default and there is "+
				"no GO-NATIVE fallback: add it with evidence, or as UNCLASSIFIED with the question "+
				"you could not answer.", pcManifest))
	}
	for _, gofile := range dangling {
		e := entries[gofile]
		fail("MANIFEST-DANGLING", fmt.Sprintf("%s:%d", pcManifest, e.line),
			fmt.Sprintf("%s is classified %s but no such non-test .go file is tracked", gofile, e.cls))
	}

	unclassified := 0
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, gofile := range keys {
		e := entries[gofile]
		where := fmt.Sprintf("%s:%d", pcManifest, e.line)
		switch {
		case e.cls == "PORT" || e.cls == "DATA":
			if e.payload == "" {
				fail("MANIFEST-NO-COUNTERPART", where,
					fmt.Sprintf("%s is %s but names no Lua file", gofile, e.cls))
				continue
			}
			for _, p := range strings.Split(e.payload, ",") {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if !strings.HasPrefix(p, "src/") {
					fail("MANIFEST-PATH", where,
						fmt.Sprintf("%s cites %s; the convention is a path from the PoB root, "+
							"i.e. src/...", gofile, pyRepr(p)))
				} else if len(pcCandidates(p, lines)) == 0 {
					fail("MANIFEST-NOFILE", where,
						fmt.Sprintf("%s cites %s, which is not in the pinned PoB (%s)", gofile, p, pob))
				}
			}
		case e.cls == "GENERATED":
			switch {
			case e.payload == "":
				fail("MANIFEST-NO-EMITTER", where,
					fmt.Sprintf("%s is GENERATED but names no emitter", gofile))
			case strings.HasPrefix(e.payload, "[unverifiable:"):
				rest := strings.TrimSpace(strings.TrimRight(e.payload[len("[unverifiable:"):], "]"))
				if rest == "" {
					fail("MANIFEST-EMPTY-WHY", where,
						fmt.Sprintf("%s opts out with an empty [unverifiable:]", gofile))
				}
			default:
				if _, err := os.Stat(pyPathJoin(repo, e.payload)); err != nil {
					fail("MANIFEST-NO-EMITTER", where,
						fmt.Sprintf("%s names emitter %s, which is not in this repo. Either it "+
							"moved, or the file is no longer generated.", gofile, e.payload))
				}
			}
		default:
			if e.payload == "" {
				fail("MANIFEST-NO-REASON", where,
					fmt.Sprintf("%s is %s with no reason. A bare class assertion is the prose "+
						"claim this file exists to replace.", gofile, e.cls))
			}
			if e.cls == "UNCLASSIFIED" {
				unclassified++
			}
		}
	}
	if unclassified > pcUnclassifiedCeiling {
		fail("UNCLASSIFIED-CEILING", pcManifest,
			fmt.Sprintf("%d UNCLASSIFIED file(s), above the ceiling of %d. The ceiling only "+
				"moves DOWN.", unclassified, pcUnclassifiedCeiling))
	}
	return fails, unclassified
}

func pcShow(pob, spec string) int {
	lines := pcIndexPoB(pob)
	cs := pcParseMarker("-", 0, spec)
	if len(cs) == 0 {
		pcRefuse(fmt.Sprintf("--show got no citation to resolve in %s", pyRepr(spec)))
	}
	c := cs[0]
	c.formal = false
	pcResolve(c, lines, pob)
	if c.status != "OK" && c.status != "EMPTY" {
		fmt.Println(c.detail)
		return 1
	}
	body := pcLines(pcReadText(pyPathJoin(pob, c.path), "cited Lua"))
	if c.start < 0 {
		fmt.Printf("%s (%d lines)\n", c.path, lines[c.path])
		return 0
	}
	hi := c.end
	if len(body) < hi {
		hi = len(body)
	}
	for i := c.start; i <= hi; i++ {
		fmt.Printf("%5d\t%s\n", i, strings.TrimRight(body[i-1], "\n"))
	}
	return 0
}

func pcDedup(cs []*pcCite) []*pcCite {
	sorted := append([]*pcCite{}, cs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].gofile != sorted[j].gofile {
			return sorted[i].gofile < sorted[j].gofile
		}
		return sorted[i].goline < sorted[j].goline
	})
	var out []*pcCite
	seen := map[string]bool{}
	for _, c := range sorted {
		key := fmt.Sprintf("%s\x00%d\x00%d\x00%s\x00%s", c.path, c.start, c.end, c.status, c.drift)
		if !seen[key] {
			seen[key] = true
			out = append(out, c)
		}
	}
	return out
}

func pobCitations(args []string) int {
	pob := os.Getenv("POB_DIR")
	if pob == "" {
		home, ok := pyPathHome()
		if !ok {
			//go-only: Path.home() RAISES RuntimeError; unreachable without both an
			pcRefuse("cannot resolve the home directory to find the pinned PoB")
		}
		pob = pyPathJoin(home, pcDefaultPoBRel)
	}
	if len(args) > 0 && args[0] == "--selftest" {
		return pcSelftest(pob)
	}
	if len(args) > 0 && args[0] == "--show" {
		if len(args) < 2 {
			pcRefuse("--show needs a citation, e.g. --show src/Modules/CalcPerform.lua:2962-2970")
		}
		return pcShow(pob, args[1])
	}

	repo := pcRepoRoot()
	reportOnly := true
	switch os.Getenv("REPORT") {
	case "", "0", "false":
		reportOnly = false
	}
	sc := pcCollect(repo, pob)
	lines := pcIndexPoB(pob)
	entries, manifestBad := pcLoadManifest(repo)
	manifestFails, nUnclassified := pcCheckManifest(repo, pob, lines, entries, nil)
	for _, b := range manifestBad {
		manifestFails = append(manifestFails, pcFail{
			b.code, fmt.Sprintf("%s:%d", pcManifest, b.line), b.msg})
	}

	nCited := 0
	for _, v := range sc.cited {
		nCited += len(v)
	}
	var broken []*pcCite
	for _, c := range sc.formal {
		if c.status != "OK" {
			broken = append(broken, c)
		}
	}

	fmt.Printf("PoB checkout: %s\n", pob)
	fmt.Println()
	fmt.Println("=== MARKERS  //pob:<path>[:range] ===")
	fmt.Printf("  %d citation(s) on %d symbol(s), %d whole-file marker(s), across %d file(s)\n",
		len(sc.formal), nCited, len(sc.fileMarker), pcDistinctFiles(sc.formal))
	sort.SliceStable(broken, func(i, j int) bool {
		if broken[i].gofile != broken[j].gofile {
			return broken[i].gofile < broken[j].gofile
		}
		return broken[i].goline < broken[j].goline
	})
	for _, c := range broken {
		fmt.Printf("  BROKEN %s\n", c)
	}
	fmt.Printf("  %d broken\n", len(broken))

	fmt.Println()
	fmt.Println("=== SWEEP of informal citations (Foo.lua:123 in a comment or a provenance string) ===")
	kinds := map[string]int{}
	for _, c := range sc.informal {
		kinds[c.status]++
	}
	targets := map[string]bool{}
	for _, c := range sc.informal {
		targets[fmt.Sprintf("%s\x00%d\x00%d", c.path, c.start, c.end)] = true
	}
	fmt.Printf("  %d occurrence(s), %d distinct target(s), in %d file(s)\n",
		len(sc.informal), len(targets), pcDistinctFiles(sc.informal))
	kks := make([]string, 0, len(kinds))
	for k := range kinds {
		kks = append(kks, k)
	}
	sort.Strings(kks)
	parts := make([]string, len(kks))
	for i, k := range kks {
		parts[i] = fmt.Sprintf("%s=%d", k, kinds[k])
	}
	fmt.Println("  " + strings.Join(parts, "  "))

	var wrong, drifted, soft []*pcCite
	for _, c := range sc.informal {
		switch c.status {
		case "NOFILE", "RANGE", "MALFORMED":
			wrong = append(wrong, c)
		}
		if c.drift != "" {
			drifted = append(drifted, c)
		}
		if (c.status == "EMPTY" || c.status == "PATH" || c.status == "AMBIG") && c.drift == "" {
			soft = append(soft, c)
		}
	}
	for _, c := range pcDedup(wrong) {
		fmt.Printf("  WRONG    %s\n", c)
	}
	fmt.Printf("  -> %d occurrence(s) name a file or line that does not exist (%d distinct)\n",
		len(wrong), len(pcDedup(wrong)))
	for _, c := range pcDedup(drifted) {
		fmt.Printf("  DRIFT    %s:%d  %s  %s\n", c.gofile, c.goline, c.where(), c.drift)
	}
	fmt.Printf("  -> %d occurrence(s) quote Lua that is not in the range they cite (%d distinct)\n",
		len(drifted), len(pcDedup(drifted)))
	for _, c := range pcDedup(soft) {
		fmt.Printf("  ADVISORY %s\n", c)
	}
	fmt.Printf("  -> %d occurrence(s) cite a comment-only range or a short path (%d distinct).\n",
		len(soft), len(pcDedup(soft)))
	fmt.Println("     EMPTY HAS KNOWN FALSE POSITIVES: a citation may deliberately point at a Lua")
	fmt.Println("     COMMENT (`gem_lookup_test.go` cites CalcSetup.lua:924, which IS the comment it")
	fmt.Println("     quotes). Read before believing. It is reported because the same shape is also")
	fmt.Println("     what an off-by-one looks like.")

	fmt.Println()
	fmt.Println("=== FUNCTION-LINE CHECK  (a func's marker must cite a Lua `function` line) ===")
	fl := pcFuncLineFailures(pob, sc)
	grandfather := pcFuncLineGrandfather
	if path := os.Getenv("POB_CITATIONS_GRANDFATHER"); path != "" {
		b, err := os.ReadFile(path) //nolint:gosec // a test fixture path
		if err != nil {
			//go-only: the grandfather override is a difftest fixture hook; the original had none.
			pcRefuse(fmt.Sprintf("cannot read POB_CITATIONS_GRANDFATHER=%s: %v", path, err))
		}
		grandfather = map[string]bool{}
		for _, k := range strings.Fields(string(b)) {
			grandfather[k] = true
		}
	}
	var flNew []pcFLFail
	flKeys := map[string]bool{}
	for _, f := range fl {
		flKeys[f.key] = true
		if !grandfather[f.key] {
			flNew = append(flNew, f)
		}
	}
	var flStale []string
	for k := range grandfather {
		if !flKeys[k] {
			flStale = append(flStale, k)
		}
	}
	sort.Strings(flStale)
	for _, f := range flNew {
		fmt.Printf("  MID-BODY %s\n", f.detail)
	}
	fmt.Printf("  -> %d func-attached marker(s) cite a range with no `function` line; %d "+
		"grandfathered, %d NEW\n", len(fl), len(grandfather), len(flNew))
	for _, k := range flStale {
		fmt.Printf("  STALE-GRANDFATHER %s no longer needs grandfathering -- delete it from "+
			"FUNC_LINE_GRANDFATHER\n", k)
	}
	fmt.Println("     A Go func that genuinely ports an INLINE Lua block writes `//pob:block:`,")
	fmt.Println("     which is an explicit claim rather than a silent exemption.")

	fmt.Println()
	fmt.Printf("=== CLASSIFICATION  (%s) ===\n", pcManifest)
	byClass := map[string][]string{}
	for rel, e := range entries {
		byClass[e.cls] = append(byClass[e.cls], rel)
	}
	for _, cls := range pcClasses {
		nf := 0
		for _, r := range byClass[cls] {
			nf += len(sc.funcs[r])
		}
		fmt.Printf("  %-13s %3d file(s)  %4d top-level func(s)\n", cls, len(byClass[cls]), nf)
	}
	for _, f := range manifestFails {
		fmt.Printf("  %-24s %s\n%s%s\n", f.code, f.where, strings.Repeat(" ", 27), f.msg)
	}
	fmt.Printf("  -> %d classification failure(s)\n", len(manifestFails))

	fmt.Println()
	fmt.Println("=== COVERAGE ===")
	type covRow struct {
		gap, n, c int
		rel       string
	}
	var rows []covRow
	totF, totC, rawF, rawC := 0, 0, 0, 0
	for _, rel := range sc.order {
		fns := sc.funcs[rel]
		if strings.HasSuffix(rel, "_test.go") || len(fns) == 0 {
			continue
		}
		n := len(fns)
		c := len(sc.cited[rel])
		top := strings.SplitN(rel, "/", 2)[0]
		for _, p := range pcPortedPkgs {
			if top == p {
				rawF += n
				rawC += c
				break
			}
		}
		if entries[rel].cls != "PORT" {
			continue
		}
		totF += n
		totC += c
		rows = append(rows, covRow{n - c, n, c, rel})
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.gap != b.gap {
			return a.gap > b.gap
		}
		if a.n != b.n {
			return a.n > b.n
		}
		if a.c != b.c {
			return a.c > b.c
		}
		return a.rel > b.rel
	})
	fmt.Printf("  %6s %6s  %s\n", "funcs", "cited", "PORT file")
	head := rows
	if len(head) > 20 {
		head = head[:20]
	}
	for _, r := range head {
		fmt.Printf("  %6d %6d  %s\n", r.n, r.c, r.rel)
	}
	fmt.Printf("  ... %d more PORT file(s)\n", max(0, len(rows)-20))
	fmt.Printf("  PORT   %d / %d top-level funcs cited (%.1f%%) across %d file(s)  <- the number "+
		"that means something\n", totC, totF, pcPct(totC, totF), len(rows))
	fmt.Printf("  RAW    %d / %d over all of %s (%.1f%%)  <- for continuity only; ~%d of those "+
		"funcs are go-pob's own code with nothing to cite\n",
		rawC, rawF, strings.Join(pcPortedPkgs, ", "), pcPct(rawC, rawF), rawF-totF)
	if len(sc.fileMarker) > 0 {
		marked := make([]string, 0, len(sc.fileMarker))
		for r := range sc.fileMarker {
			marked = append(marked, r)
		}
		sort.Strings(marked)
		fmt.Printf("  %d file(s) carry a WHOLE-FILE marker, credited 0 funcs above (see the note "+
			"in the source):\n", len(marked))
		for _, rel := range marked {
			cls := "?"
			if e, ok := entries[rel]; ok {
				cls = e.cls
			}
			flag := ""
			if cls == "PORT" {
				flag = "  <- PORT file: cite its FUNCTIONS instead"
			}
			fmt.Printf("    [%s] %s%s\n", cls, rel, flag)
		}
	}

	fmt.Println()
	fail := 0
	if len(broken) > 0 {
		fmt.Printf("FAIL: %d marker(s) do not resolve against %s\n", len(broken), pob)
		fail = 1
	}
	if nCited < pcCitedFloor {
		fmt.Printf("FAIL: %d cited symbol(s), below the floor of %d\n", nCited, pcCitedFloor)
		fail = 1
	}
	if len(manifestFails) > 0 {
		fmt.Printf("FAIL: %d classification failure(s) in %s\n", len(manifestFails), pcManifest)
		fail = 1
	}
	if len(flNew) > 0 {
		fmt.Printf("FAIL: %d func-attached marker(s) cite a range with no Lua `function` line. "+
			"Re-read each with `awk 'NR>=<start> && NR<=<end> {printf \"%%d\\t%%s\\n\", NR, $0}' "+
			"<file>` — NEVER `sed -n`; if the Go func really ports an inline block, "+
			"write `//pob:block:`.\n", len(flNew))
		fail = 1
	}
	if len(flStale) > 0 {
		fmt.Printf("FAIL: %d grandfathered entr(ies) are no longer needed. The list may only "+
			"SHRINK -- delete them.\n", len(flStale))
		fail = 1
	}
	if fail == 0 {
		fmt.Printf("OK: every marker resolves and cites a Lua `function` line; %d cited symbol(s) "+
			">= floor %d; %d file(s) classified, %d UNCLASSIFIED (ceiling %d)\n",
			nCited, pcCitedFloor, len(entries), nUnclassified, pcUnclassifiedCeiling)
	}
	if reportOnly && fail != 0 {
		fmt.Println("REPORT=1: exiting 0 anyway")
	}
	if reportOnly {
		return 0
	}
	return fail
}

func pcDistinctFiles(cs []*pcCite) int {
	seen := map[string]bool{}
	for _, c := range cs {
		seen[c.gofile] = true
	}
	return len(seen)
}

func pcPct(a, b int) float64 {
	if b == 0 {
		return 0.0
	}
	return 100.0 * float64(a) / float64(b)
}

func pcSelftest(pob string) int {
	repo := pcRepoRoot()
	lines := pcIndexPoB(pob)
	bad := 0
	report := func(name string, ok bool, want, got string) {
		verdict := "BROKEN"
		if ok {
			verdict = "ok"
		}
		w := want
		if w == "" {
			w = "(pass)"
		}
		g := got
		if g == "" {
			g = "-"
		}
		fmt.Printf("  %-8s expect %-24s got %-24s %s\n", verdict, w, g, name)
		if !ok {
			bad++
		}
	}

	present := map[string]bool{"a/x.go": true, "a/y.go": true}
	cases := []struct {
		name, text string
		files      map[string]bool
		want       string
	}{
		{"a file nobody classified", "a/x.go PORT src/Classes/ModStore.lua",
			map[string]bool{"a/x.go": true, "a/NEW.go": true}, "MANIFEST-MISSING"},
		{"an entry whose Go file was deleted",
			"a/x.go PORT src/Classes/ModStore.lua\na/gone.go GO-NATIVE r",
			map[string]bool{"a/x.go": true}, "MANIFEST-DANGLING"},
		{"a PORT naming Lua that is not in the pinned PoB",
			"a/x.go PORT src/Classes/NoSuchFile.lua",
			map[string]bool{"a/x.go": true}, "MANIFEST-NOFILE"},
		{"a PORT naming a bare basename instead of a src/ path", "a/x.go PORT ModStore.lua",
			map[string]bool{"a/x.go": true}, "MANIFEST-PATH"},
		{"a PORT naming nothing at all", "a/x.go PORT",
			map[string]bool{"a/x.go": true}, "MANIFEST-NO-COUNTERPART"},
		{"a DATA entry naming Lua that is gone", "a/x.go DATA src/Data/NoSuchTable.lua",
			map[string]bool{"a/x.go": true}, "MANIFEST-NOFILE"},
		{"a GENERATED entry naming an emitter that is gone", "a/x.go GENERATED oracle/no_such_emit.py",
			map[string]bool{"a/x.go": true}, "MANIFEST-NO-EMITTER"},
		{"a GENERATED entry opting out with an empty why", "a/x.go GENERATED [unverifiable:]",
			map[string]bool{"a/x.go": true}, "MANIFEST-EMPTY-WHY"},
		{"a GO-NATIVE entry with no reason", "a/x.go GO-NATIVE",
			map[string]bool{"a/x.go": true}, "MANIFEST-NO-REASON"},
		{"an UNCLASSIFIED entry over the ceiling",
			"a/x.go UNCLASSIFIED is this a port of PassiveTree geometry?",
			map[string]bool{"a/x.go": true}, "UNCLASSIFIED-CEILING"},
		{"a well-formed manifest",
			"a/x.go PORT src/Classes/ModStore.lua\na/y.go GO-NATIVE because", present, ""},
	}
	for _, tc := range cases {
		entries, parseBad := pcParseManifest(tc.text)
		fails, _ := pcCheckManifest(repo, pob, lines, entries, tc.files)
		codes := map[string]bool{}
		for _, f := range fails {
			codes[f.code] = true
		}
		for _, b := range parseBad {
			codes[b.code] = true
		}
		got := pcJoinSorted(codes)
		report(tc.name, got == tc.want, tc.want, got)
	}

	for _, tc := range []struct{ name, text, want string }{
		{"a line with no class at all", "a/x.go", "MANIFEST-SYNTAX"},
		{"an invented class", "a/x.go PORTED src/Classes/ModStore.lua", "MANIFEST-CLASS"},
		{"the same file listed twice",
			"a/x.go PORT src/Classes/ModStore.lua\na/x.go GO-NATIVE r", "MANIFEST-DUP"},
	} {
		_, parseBad := pcParseManifest(tc.text)
		codes := map[string]bool{}
		for _, b := range parseBad {
			codes[b.code] = true
		}
		got := pcJoinSorted(codes)
		report(tc.name, got == tc.want, tc.want, got)
	}

	for _, tc := range []struct {
		name, rel string
		a, b      int
		want      bool
	}{
		{"a range starting INSIDE a function body", "src/Classes/ModStore.lua", 458, 459, false},
		{"a range that opens on the `function` line", "src/Classes/ModStore.lua", 246, 250, true},
		{"a range holding only the comment above a function",
			"src/Modules/CalcBreakdown.lua", 15, 16, false},
		{"a range holding the whole function", "src/Modules/CalcBreakdown.lua", 17, 20, true},
	} {
		got := pcLuaRangeHasFunction(pob, tc.rel, tc.a, tc.b)
		report(fmt.Sprintf("%s (%s:%d-%d)", tc.name, tc.rel, tc.a, tc.b),
			got == tc.want, pyBool(tc.want), pyBool(got))
	}

	for _, tc := range []struct {
		name, payload string
		want          bool
	}{
		{"a plain marker is subject to the check", "src/Classes/ModStore.lua:458-459", false},
		{"a block: marker is exempt", "block:src/Classes/ModStore.lua:458-459", true},
		{"block: still resolves every entry",
			"block:src/Classes/ModStore.lua:458-459,src/Classes/ModStore.lua:60-68", true},
	} {
		cs := pcParseMarker("-", 0, tc.payload)
		got := len(cs) > 0
		for _, c := range cs {
			if !c.block {
				got = false
			}
		}
		report(fmt.Sprintf("%s (%s)", tc.name, pyRepr(tc.payload)),
			got == tc.want, pyBool(tc.want), pyBool(got))
	}

	total := len(cases) + 3 + 4 + 3
	fmt.Printf("\nSELFTEST: %d of %d checks behave as declared\n", total-bad, total)
	if bad > 0 {
		return 1
	}
	return 0
}

func pcJoinSorted(set map[string]bool) string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

func pyBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}
