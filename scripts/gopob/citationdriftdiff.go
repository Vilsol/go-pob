package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

const cddLow = 0.45

var (
	cddBoringRE  = regexp.MustCompile(`^\s*(end\)?,?|else|do|then|\}|--.*|)$`)
	cddCommentRE = regexp.MustCompile(`--.*$`)
	cddSpaceRE   = regexp.MustCompile(`\s+`)
)

type cddEnv struct {
	repo, pin, old string
	pinCache       map[string][]string
	oldCache       map[string][]string
	normCache      map[string]string
	countCache     map[string]map[string]int
}

func (e *cddEnv) norm(line string) string {
	if v, ok := e.normCache[line]; ok {
		return v
	}
	v := strings.TrimSpace(cddSpaceRE.ReplaceAllString(cddCommentRE.ReplaceAllString(line, ""), " "))
	e.normCache[line] = v
	return v
}

func (e *cddEnv) oldSource(path string) []string {
	if v, ok := e.oldCache[path]; ok {
		return v
	}
	var out []string
	if fi, err := os.Stat(e.old); err == nil && fi.IsDir() {
		full := pyPathJoin(e.old, path)
		if _, err := os.Stat(full); err == nil { //nolint:gosec // an operator-supplied tree
			out = strings.Split(pcReadText(full, "the OLD tree"), "\n")
		}
	} else {
		//nolint:gosec // OLD and the cited path are the operator's own, passed as one argv element
		cmd := exec.CommandContext(context.Background(), "git", "show", e.old+":"+path)
		cmd.Dir = e.pin
		b, err := cmd.Output()
		if err != nil {
			var ee *exec.Error
			if errorsAs(err, &ee) {
				pcRefuse(fmt.Sprintf("cannot run git: %s", errnoText(err)))
			}
		} else {
			if !utf8.Valid(b) {
				pcRefuse(fmt.Sprintf("%s:%s is not valid UTF-8", e.old, path))
			}
			out = strings.Split(string(b), "\n")
		}
	}
	e.oldCache[path] = out
	return out
}

func (e *cddEnv) pinSource(path string) []string {
	if v, ok := e.pinCache[path]; ok {
		return v
	}
	var out []string
	full := pyPathJoin(e.pin, path)
	if _, err := os.Stat(full); err == nil { //nolint:gosec // an operator-supplied checkout
		out = strings.Split(pcReadText(full, "the pinned PoB"), "\n")
	}
	e.pinCache[path] = out
	return out
}

func (e *cddEnv) normCounts(path string, lines []string) map[string]int {
	if v, ok := e.countCache[path]; ok {
		return v
	}
	m := map[string]int{}
	for _, l := range lines {
		m[e.norm(l)]++
	}
	e.countCache[path] = m
	return m
}

func (e *cddEnv) anchorLine(block []string, counts map[string]int) string {
	best := ""
	haveBest := false
	for _, line := range block {
		n := e.norm(line)
		if len([]rune(n)) < 12 || cddBoringRE.MatchString(line) {
			continue
		}
		if counts[n] != 1 {
			continue
		}
		if !haveBest || len([]rune(n)) > len([]rune(best)) {
			best, haveBest = n, true
		}
	}
	return best
}

func (e *cddEnv) findWindow(anchor string, blockLen int, pinLines []string, offset int) (int, []string, bool) {
	var hits []int
	for i, line := range pinLines {
		if e.norm(line) == anchor {
			hits = append(hits, i)
		}
	}
	if len(hits) != 1 {
		return 0, nil, false
	}
	start := hits[0] - offset
	if start < 0 {
		start = 0
	}
	return start, cddSlice(pinLines, start, start+blockLen), true
}

func cddSlice(xs []string, lo, hi int) []string {
	if lo > len(xs) {
		lo = len(xs)
	}
	if hi > len(xs) {
		hi = len(xs)
	}
	if hi < lo {
		hi = lo
	}
	return xs[lo:hi]
}

func (e *cddEnv) bestWindow(block, pinLines []string) (float64, int, []string, bool) {
	var a []string
	for _, x := range block {
		if n := e.norm(x); n != "" {
			a = append(a, n)
		}
	}
	if len(a) == 0 {
		return 0, 0, nil, false
	}
	span := len(block)
	bestRatio, bestStart := 0.0, -1
	limit := len(pinLines) - span
	if limit < 1 {
		limit = 1
	}
	for start := 0; start < limit; start += 3 {
		var b []string
		for _, x := range cddSlice(pinLines, start, start+span) {
			if n := e.norm(x); n != "" {
				b = append(b, n)
			}
		}
		if len(b) == 0 {
			continue
		}
		if r := newPySeqMatcher(a, b, true).quickRatio(); r > bestRatio {
			bestRatio, bestStart = r, start
		}
	}
	if bestStart < 0 {
		return 0, 0, nil, false
	}
	window := cddSlice(pinLines, bestStart, bestStart+span)
	var b []string
	for _, x := range window {
		if n := e.norm(x); n != "" {
			b = append(b, n)
		}
	}
	return newPySeqMatcher(a, b, true).ratio(), bestStart, window, true
}

type cddMarker struct {
	gofile     string
	goline     int
	lua        string
	start, end int
}

func (e *cddEnv) markers() []cddMarker {
	var out []cddMarker
	for _, gofile := range pcGoFiles(e.repo) {
		src := strings.Split(pcReadText(pyPathJoin(e.repo, gofile), "Go source"), "\n")
		for i, line := range src {
			m := pcMarkerRE.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			payload := strings.TrimPrefix(m[1], "block:")
			for _, entry := range strings.Split(payload, ",") {
				en := pcEntryRE.FindStringSubmatch(strings.TrimSpace(entry))
				if en == nil || en[2] == "" {
					continue
				}
				s, _ := strconv.Atoi(en[2])
				fin := s
				if en[3] != "" {
					fin, _ = strconv.Atoi(en[3])
				}
				out = append(out, cddMarker{gofile, i + 1, en[1], s, fin})
			}
		}
	}
	return out
}

func (e *cddEnv) goSymbol(gofile string, goline int) string {
	src := strings.Split(pcReadText(pyPathJoin(e.repo, gofile), "Go source"), "\n")
	for _, line := range cddSlice(src, goline-1, goline+40) {
		if strings.HasPrefix(line, "func ") || strings.HasPrefix(line, "type ") ||
			strings.HasPrefix(line, "var ") {
			s := strings.TrimSpace(strings.SplitN(line, "{", 2)[0])
			return cddCut(s, 110)
		}
	}
	return ""
}

func cddCut(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func citationDriftDiff(args []string) int {
	repo, err := root.Resolve()
	if err != nil {
		pcRefuse(fmt.Sprintf("%v", err))
	}
	pin := os.Getenv("POB_DIR")
	if pin == "" {
		home, ok := pyPathHome()
		if !ok {
			//go-only: Path.home() RAISES RuntimeError; unreachable without both an
			pcRefuse("cannot resolve the home directory to find the pinned PoB")
		}
		pin = pyPathJoin(home, "PathOfBuilding")
	}
	old := os.Getenv("OLD")
	if old == "" {
		old = "v2.18.1"
	}
	e := &cddEnv{repo: repo, pin: pin, old: old,
		pinCache: map[string][]string{}, oldCache: map[string][]string{},
		normCache: map[string]string{}, countCache: map[string]map[string]int{}}

	if len(args) > 0 && args[0] == "--selftest" {
		return cddSelftest(e)
	}
	only := os.Getenv("FILE")
	if only == "" && len(args) > 0 {
		only = args[0]
	}

	seen, changed, noanchor, fuzzed := 0, 0, 0, 0
	for _, mk := range e.markers() {
		if only != "" && !strings.Contains(mk.gofile, only) {
			continue
		}
		oldLines := e.oldSource(mk.lua)
		pinLines := e.pinSource(mk.lua)
		if oldLines == nil || pinLines == nil {
			continue
		}
		if mk.start > len(oldLines) {
			continue
		}
		block := cddSlice(oldLines, mk.start-1, mk.end)
		nonEmpty := false
		for _, x := range block {
			if e.norm(x) != "" {
				nonEmpty = true
				break
			}
		}
		if !nonEmpty {
			continue
		}
		seen++

		how := ""
		var pinStart int
		var window []string
		anchor := e.anchorLine(block, e.normCounts(mk.lua, oldLines))
		found := false
		if anchor != "" {
			offset := 0
			for i, x := range block {
				if e.norm(x) == anchor {
					offset = i
					break
				}
			}
			pinStart, window, found = e.findWindow(anchor, len(block), pinLines, offset)
		}
		if !found {
			ratio, fs, fw, ok := e.bestWindow(block, pinLines)
			if !ok || ratio < cddLow {
				noanchor++
				score := ""
				if ok {
					score = fmt.Sprintf(", best match only %.2f", ratio)
				}
				fmt.Printf("NO-ANCHOR %s:%d  %s:%d-%d  (nothing in the pin resembles this "+
					"block%s — possibly DELETED)\n", mk.gofile, mk.goline, mk.lua, mk.start,
					mk.end, score)
				continue
			}
			how = fmt.Sprintf("  [FUZZY %.2f]", ratio)
			pinStart, window = fs, fw
			fuzzed++
		}

		a := make([]string, len(block))
		for i, x := range block {
			a[i] = e.norm(x)
		}
		b := make([]string, len(window))
		for i, x := range window {
			b[i] = e.norm(x)
		}
		if cddEqual(a, b) {
			continue
		}
		changed++
		fmt.Printf("\n=== %s:%d  %s:%d-%d -> pin:%d%s\n",
			mk.gofile, mk.goline, mk.lua, mk.start, mk.end, pinStart+1, how)
		if sym := e.goSymbol(mk.gofile, mk.goline); sym != "" {
			fmt.Printf("    go: %s\n", sym)
		}
		for _, d := range pyUnifiedDiff(a, b,
			fmt.Sprintf("%s:%d", e.old, mk.start), fmt.Sprintf("pin:%d", pinStart+1),
			"", "", 1, "") {
			if strings.HasPrefix(d, "---") || strings.HasPrefix(d, "+++") {
				continue
			}
			fmt.Println("   " + cddCut(d, 150))
		}
	}

	fmt.Printf("\n  %4d citations with a non-empty old block\n", seen)
	fmt.Printf("  %4d CHANGED — the upstream delta is printed above\n", changed)
	fmt.Printf("  %4d of those located by SIMILARITY, not an exact anchor — check the ratio\n", fuzzed)
	fmt.Printf("  %4d NO-ANCHOR — located nothing; read by hand, do NOT read as unchanged\n", noanchor)
	fmt.Print("\n⚠️ A DIFF IS NOT A DEFECT. It says the Lua moved on; whether OUR GO caught up is a\n" +
		"   separate read. Several of these were re-ported here years after the marker was\n" +
		"   written and are already current.\n")
	return 0
}

func cddEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func cddSelftest(e *cddEnv) int {
	fail := func(msg string) int {
		fmt.Fprintln(os.Stderr, msg)
		return 1
	}
	old := []string{"local function f(x)", "  return x + 1", "end"}
	if got := e.anchorLine(old, e.normCounts("<selftest-old>", old)); got != "local function f(x)" {
		return fail(fmt.Sprintf("anchor_line: got %s", pyRepr(got)))
	}
	boring := []string{"end", "end"}
	if got := e.anchorLine(boring, e.normCounts("<selftest-boring>", boring)); got != "" {
		return fail("boilerplate must not anchor")
	}
	pin := []string{"-- new comment", "local function f(x)", "  return x + 2", "end"}
	_, w, ok := e.findWindow("local function f(x)", 3, pin, 0)
	if !ok {
		return fail("find_window found nothing")
	}
	got := make([]string, len(w))
	for i, x := range w {
		got[i] = e.norm(x)
	}
	if !cddEqual(got, []string{"local function f(x)", "return x + 2", "end"}) {
		return fail(fmt.Sprintf("find_window window: %v", got))
	}
	if _, _, ok := e.findWindow("nowhere", 3, pin, 0); ok {
		return fail("a missing anchor must refuse")
	}
	fmt.Println("selftest OK")
	return 0
}
