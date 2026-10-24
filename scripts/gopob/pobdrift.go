package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	pbdUnchanged = "UNCHANGED"
	pbdMoved     = "MOVED"
	pbdChanged   = "CHANGED"
	pbdDeleted   = "DELETED"

	pbdIntactFraction = 0.85
)

var pbdVerdicts = []string{pbdUnchanged, pbdMoved, pbdChanged, pbdDeleted}

const pbdSpace = pySpaceClass

var (
	pbdFuncNameRE   = regexp.MustCompile(`\bfunction` + pbdSpace + `+([\w.:]+)`)
	pbdLocalFuncRE  = regexp.MustCompile(`\blocal` + pbdSpace + `+([\w]+)` + pbdSpace + `*=` + pbdSpace + `*function\b`)
	pbdSpecRE       = regexp.MustCompile(`[\w.\-/]+\.lua(?::\d+(?:-\d+)?)?`)
	pbdPunctSpaceRE = regexp.MustCompile(pbdSpace + `*([(){}\[\],;:=+\-*/%^#<>~.])` + pbdSpace + `*`)
	pbdRunsRE       = regexp.MustCompile(pbdSpace + `+`)
)

func pbdRefuse(msg string) {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	fmt.Fprintln(os.Stderr, "An instrument that cannot answer must refuse, never return empty.")
	os.Exit(pcRefusalExit)
}

func pbdReadText(path, what string) string {
	s := pcReadText(path, what)
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

func pbdStripInlineComment(s string) string {
	i := strings.Index(s, "--")
	for i != -1 {
		head := s[:i]
		if strings.Count(head, `"`)%2 == 0 && strings.Count(head, "'")%2 == 0 {
			return head
		}
		j := strings.Index(s[i+2:], "--")
		if j == -1 {
			break
		}
		i = i + 2 + j
	}
	return s
}

type pbdLine struct {
	n    int
	text string
}

func pbdNormaliseFile(text string) []pbdLine {
	var out []pbdLine
	inLong := false
	for i, raw := range strings.Split(text, "\n") {
		n := i + 1
		s := strings.TrimSpace(raw)
		if inLong {
			if strings.Contains(s, "]]") {
				inLong = false
				s = strings.TrimSpace(strings.SplitN(s, "]]", 2)[1])
			} else {
				continue
			}
		}
		if strings.HasPrefix(s, "--[[") || strings.HasPrefix(s, "--[=[") {
			if !strings.Contains(s[4:], "]]") {
				inLong = true
				continue
			}
			s = strings.TrimSpace(strings.SplitN(s, "]]", 2)[1])
		}
		if s == "" || strings.HasPrefix(s, "--") {
			continue
		}
		s = strings.TrimSpace(pbdStripInlineComment(s))
		if s == "" {
			continue
		}
		out = append(out, pbdLine{n, pbdPunctSpaceRE.ReplaceAllString(pbdRunsRE.ReplaceAllString(s, " "), "$1")})
	}
	return out
}

type pbdCheckout struct {
	path     string
	label    string
	stripSrc bool
	raws     map[string][]string
	rawSeen  map[string]bool
	norms    map[string][]pbdLine
	normSeen map[string]bool
}

func pbdRealpath(p string) string {
	p = pyExpandUser(p)
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = filepath.Clean(p)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

func pbdIsDir(p string) bool {
	st, err := os.Stat(p) //nolint:gosec // a checkout path the operator named
	return err == nil && st.IsDir()
}

func pbdNewCheckout(path, label string) *pbdCheckout {
	c := &pbdCheckout{
		path:  pbdRealpath(path),
		label: label,
		raws:  map[string][]string{}, rawSeen: map[string]bool{},
		norms: map[string][]pbdLine{}, normSeen: map[string]bool{},
	}
	if !pbdIsDir(c.path) {
		pbdRefuse(fmt.Sprintf("%s checkout does not exist: %s", label, path))
	}
	if !pbdIsDir(pyPathJoin(c.path, "src")) {
		if pbdIsDir(pyPathJoin(c.path, "Modules")) && pbdIsDir(pyPathJoin(c.path, "Classes")) {
			c.stripSrc = true
		} else {
			pbdRefuse(fmt.Sprintf("%s is not a PoB checkout: %s\n"+
				"  Expected either src/ (a source checkout) or Modules/ + Classes/ (an\n"+
				"  installed PoB).", label, c.path))
		}
	}
	return c
}

func (c *pbdCheckout) layoutNote() string {
	if c.stripSrc {
		return "  [installed layout: src/ stripped from every cited path]"
	}
	return ""
}

func (c *pbdCheckout) raw(rel string) []string {
	if !c.rawSeen[rel] {
		c.rawSeen[rel] = true
		sub := rel
		if c.stripSrc && strings.HasPrefix(rel, "src/") {
			sub = rel[4:]
		}
		full := pyPathJoin(c.path, sub)
		if _, err := os.Stat(full); err != nil {
			c.raws[rel] = nil
		} else {
			c.raws[rel] = strings.Split(pbdReadText(full, "cited Lua in the "+c.label), "\n")
		}
	}
	return c.raws[rel]
}

func (c *pbdCheckout) norm(rel string) []pbdLine {
	if !c.normSeen[rel] {
		c.normSeen[rel] = true
		if body := c.raw(rel); body != nil {
			c.norms[rel] = pbdNormaliseFile(strings.Join(body, "\n"))
		}
	}
	return c.norms[rel]
}

func pbdPySlice(xs []string, lo, hi int) []string {
	if lo < 0 {
		lo = 0
	}
	if hi > len(xs) {
		hi = len(xs)
	}
	if lo >= hi {
		return []string{}
	}
	return xs[lo:hi]
}

func (c *pbdCheckout) sliceRaw(rel string, start, end int) ([]string, bool) {
	body := c.raw(rel)
	if body == nil {
		return nil, false
	}
	if end < 0 {
		end = len(body)
	}
	return pbdPySlice(body, start-1, end), true
}

func (c *pbdCheckout) sliceNorm(rel string, start, end int) ([]pbdLine, bool) {
	nl := c.norm(rel)
	if c.raw(rel) == nil {
		return nil, false
	}
	out := []pbdLine{}
	for _, l := range nl {
		if start <= l.n && l.n <= end {
			out = append(out, l)
		}
	}
	return out, true
}

func (c *pbdCheckout) describe() string {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", c.path, "describe", "--tags", "--always", "--dirty") //nolint:gosec // a checkout the operator named
	out, _ := cmd.Output()
	if s := strings.TrimSpace(string(out)); s != "" {
		return s
	}
	return "(no git description)"
}

type pbdDrift struct {
	cite           *pcCite
	verdict        string
	note           string
	newStart       int
	newEnd         int
	diffSize       int
	similarity     float64
	candidates     int
	citedLines     int
	survivingLines int
}

func pbdNewDrift(c *pcCite) *pbdDrift {
	return &pbdDrift{cite: c, newStart: -1, newEnd: -1, similarity: 1.0, candidates: 1,
		citedLines: -1, survivingLines: -1}
}

func (d *pbdDrift) offset() int {
	if d.newStart < 0 {
		return 0
	}
	return d.newStart - d.cite.start
}

func (d *pbdDrift) newWhere() string {
	if d.newStart < 0 {
		return ""
	}
	if d.newStart == d.newEnd {
		return fmt.Sprintf("%s:%d", d.cite.path, d.newStart)
	}
	return fmt.Sprintf("%s:%d-%d", d.cite.path, d.newStart, d.newEnd)
}

func pbdFindRuns(hay, needle []string) []int {
	if len(needle) == 0 || len(needle) > len(hay) {
		return nil
	}
	var out []int
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i] != needle[0] {
			continue
		}
		match := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			out = append(out, i)
		}
	}
	return out
}

func pbdFuncNames(texts []string) map[string]bool {
	names := map[string]bool{}
	for _, t := range texts {
		for _, m := range pbdFuncNameRE.FindAllStringSubmatch(t, -1) {
			names[m[1]] = true
		}
		for _, m := range pbdLocalFuncRE.FindAllStringSubmatch(t, -1) {
			names[m[1]] = true
		}
	}
	return names
}

func pbdAnchorIndex(candTexts, oldTexts []string, posHint int) (int, bool) {
	bestOff, bestHits, have := 0, 0, false
	for j, t := range oldTexts {
		var hits []int
		for i, c := range candTexts {
			if c == t {
				hits = append(hits, i)
			}
		}
		if len(hits) == 0 || len(hits) > 40 {
			continue
		}
		i := hits[0]
		best := pbdAbs((hits[0] - j) - posHint)
		for _, h := range hits[1:] {
			if d := pbdAbs((h - j) - posHint); d < best {
				best, i = d, h
			}
		}
		if !have || len(hits) < bestHits {
			bestOff, bestHits, have = i-j, len(hits), true
		}
	}
	return bestOff, have
}

func pbdAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func pbdDiffSize(oldTexts, newTexts []string) (int, float64) {
	sm := newPySeqMatcher(oldTexts, newTexts, false)
	changed := 0
	for _, op := range sm.getOpcodes() {
		if op.tag != "equal" {
			changed += max(op.i2-op.i1, op.j2-op.j1)
		}
	}
	return changed, sm.ratio()
}

func pbdTexts(ls []pbdLine) []string {
	out := make([]string, len(ls))
	for i, l := range ls {
		out[i] = l.text
	}
	return out
}

func pbdClassify(cite *pcCite, pin, cand *pbdCheckout) *pbdDrift {
	d := pbdNewDrift(cite)
	rel := cite.path

	if cand.raw(rel) == nil {
		d.verdict = pbdDeleted
		d.note = "the file is not in the candidate checkout"
		d.diffSize = 1000000
		d.similarity = 0.0
		if cite.start >= 0 {
			if ls, ok := pin.sliceNorm(rel, cite.start, cite.end); ok {
				d.citedLines = len(ls)
			} else {
				d.citedLines = 0
			}
			d.survivingLines = 0
		}
		return d
	}

	if cite.start < 0 {
		a, b := pin.raw(rel), cand.raw(rel)
		switch {
		case pbdEqualStrings(a, b):
			d.verdict, d.note = pbdUnchanged, "whole file, byte-identical"
		case pbdEqualStrings(pbdTexts(pin.norm(rel)), pbdTexts(cand.norm(rel))):
			d.verdict, d.note = pbdUnchanged, "whole file, whitespace/comment-only change"
		default:
			d.verdict = pbdChanged
			d.diffSize, d.similarity = pbdDiffSize(pbdTexts(pin.norm(rel)), pbdTexts(cand.norm(rel)))
			d.note = "whole file"
		}
		return d
	}

	oldRaw, oldOK := pin.sliceRaw(rel, cite.start, cite.end)
	newRaw, newOK := cand.sliceRaw(rel, cite.start, cite.end)
	oldPairs, _ := pin.sliceNorm(rel, cite.start, cite.end)
	oldTexts := pbdTexts(oldPairs)
	d.citedLines, d.survivingLines = len(oldTexts), len(oldTexts)

	if oldOK && newOK && pbdEqualStrings(oldRaw, newRaw) {
		d.verdict, d.note = pbdUnchanged, "byte-identical"
		d.newStart, d.newEnd = cite.start, cite.end
		return d
	}

	candPairs := cand.norm(rel)
	candTexts := pbdTexts(candPairs)

	if len(oldTexts) == 0 {
		same := newOK && pbdEqualStrings(pbdStripAll(oldRaw), pbdStripAll(newRaw))
		d.verdict = pbdChanged
		if same {
			d.verdict = pbdUnchanged
		}
		d.note = "range holds no Lua code; compared raw"
		if same {
			d.newStart, d.newEnd = cite.start, cite.end
		}
		return d
	}

	if runs := pbdFindRuns(candTexts, oldTexts); len(runs) > 0 {
		d.candidates = len(runs)
		best := runs[0]
		bd := pbdAbs(candPairs[runs[0]].n - cite.start)
		for _, i := range runs[1:] {
			if x := pbdAbs(candPairs[i].n - cite.start); x < bd {
				bd, best = x, i
			}
		}
		ns := candPairs[best].n
		ne := candPairs[best+len(oldTexts)-1].n
		ns -= oldPairs[0].n - cite.start
		ne += cite.end - oldPairs[len(oldPairs)-1].n
		d.newStart, d.newEnd = ns, ne
		if ns == cite.start && ne == cite.end {
			d.verdict, d.note = pbdUnchanged, "whitespace/comment-only change"
		} else {
			d.verdict = pbdMoved
			d.note = fmt.Sprintf("%+d line(s)", ns-cite.start)
			if len(runs) > 1 {
				d.note += fmt.Sprintf(", %d equally valid target(s) in the file", len(runs))
			}
		}
		return d
	}

	oldNames := pbdFuncNames(oldTexts)
	if len(oldNames) > 0 {
		present := false
		for n := range oldNames {
			short := n
			if i := strings.LastIndex(short, ":"); i >= 0 {
				short = short[i+1:]
			}
			if i := strings.LastIndex(short, "."); i >= 0 {
				short = short[i+1:]
			}
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(short) + `\b`)
			for _, t := range candTexts {
				if re.MatchString(t) {
					present = true
					break
				}
			}
			if present {
				break
			}
		}
		if !present {
			d.verdict = pbdDeleted
			d.note = "no definition of " + strings.Join(pbdSortedKeys(oldNames), ", ") + " survives in the file"
			d.diffSize, d.similarity = len(oldTexts), 0.0
			d.survivingLines = 0
			return d
		}
	}

	surviving := 0
	for _, t := range oldTexts {
		for _, c := range candTexts {
			if c == t {
				surviving++
				break
			}
		}
	}
	if surviving == 0 && len(oldNames) == 0 {
		d.verdict = pbdDeleted
		d.note = "not one line of the range survives in the file"
		d.diffSize, d.similarity = len(oldTexts), 0.0
		d.survivingLines = 0
		return d
	}

	d.verdict = pbdChanged
	d.survivingLines = surviving
	posHint := 0
	for _, l := range candPairs {
		if l.n < cite.start {
			posHint++
		}
	}
	hint, ok := pbdAnchorIndex(candTexts, oldTexts, posHint)
	if !ok {
		d.diffSize, d.similarity = len(oldTexts), 0.0
		d.note = "no anchor line survives; whole range treated as changed"
		return d
	}
	slack := max(20, len(oldTexts))
	lo := max(0, hint-slack/2)
	hi := min(len(candTexts), hint+len(oldTexts)+slack)
	d.diffSize, d.similarity = pbdDiffSize(oldTexts, pbdPySlice(candTexts, lo, hi))
	if lo < len(candPairs) {
		if hint < len(candPairs) {
			d.newStart = candPairs[max(0, hint)].n
			j := min(len(candPairs)-1, max(0, hint)+len(oldTexts)-1)
			d.newEnd = candPairs[j].n
		}
	}
	d.note = fmt.Sprintf("%d of %d line(s) survive", surviving, len(oldTexts))
	return d
}

func pbdEqualStrings(a, b []string) bool {
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

func pbdStripAll(xs []string) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = strings.TrimSpace(x)
	}
	return out
}

func pbdSortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func pbdRewriteLine(text string, pairs [][2]string) (string, int) {
	pending := append([][2]string(nil), pairs...)
	var out strings.Builder
	last := 0
	for _, loc := range pbdSpecRE.FindAllStringIndex(text, -1) {
		if len(pending) > 0 && text[loc[0]:loc[1]] == pending[0][0] {
			out.WriteString(text[last:loc[0]])
			out.WriteString(pending[0][1])
			pending = pending[1:]
			last = loc[1]
		}
	}
	out.WriteString(text[last:])
	return out.String(), len(pairs) - len(pending)
}

func pbdApplyFixes(repo string, drifts []*pbdDrift, write bool) int {
	byFile := map[string]map[int][]*pbdDrift{}
	skipped := 0
	for _, d := range drifts {
		if d.verdict != pbdMoved {
			continue
		}
		if d.candidates > 1 {
			skipped++
			continue
		}
		if byFile[d.cite.gofile] == nil {
			byFile[d.cite.gofile] = map[int][]*pbdDrift{}
		}
		byFile[d.cite.gofile][d.cite.goline] = append(byFile[d.cite.gofile][d.cite.goline], d)
	}

	touched := 0
	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, gofile := range files {
		full := pyPathJoin(repo, gofile)
		before := pbdReadText(full, "a Go file carrying markers")
		lines := strings.Split(before, "\n")
		golines := make([]int, 0, len(byFile[gofile]))
		for gl := range byFile[gofile] {
			golines = append(golines, gl)
		}
		sort.Ints(golines)
		for _, goline := range golines {
			ds := append([]*pbdDrift(nil), byFile[gofile][goline]...)
			sort.SliceStable(ds, func(i, j int) bool { return ds[i].cite.start < ds[j].cite.start })
			pairs := make([][2]string, len(ds))
			for i, d := range ds {
				pairs[i] = [2]string{d.cite.where(), d.newWhere()}
			}
			newLine, n := pbdRewriteLine(lines[goline-1], pairs)
			if n != len(ds) {
				fmt.Printf("  SKIP %s:%d  %d of %d spec(s) not found verbatim on the line; left alone\n",
					gofile, goline, len(ds)-n, len(ds))
				continue
			}
			lines[goline-1] = newLine
			touched += len(ds)
		}
		after := strings.Join(lines, "\n")
		if after == before {
			continue
		}
		for _, ln := range pyUnifiedDiff(pySplitKeepEnds(before), pySplitKeepEnds(after),
			"a/"+gofile, "b/"+gofile, "", "", 3, "\n") {
			fmt.Print(ln)
		}
		if write {
			if err := os.WriteFile(full, []byte(after), 0o644); err != nil { //nolint:gosec // matching the original's mode
				pbdRefuse(fmt.Sprintf("%s is not writable: %s\n"+
					"  Some markers may already have been rewritten; `git diff` before rerunning.",
					full, errnoText(err)))
			}
		}
	}

	verb := "WOULD REWRITE"
	tail := "  (--dry-run: nothing was written)"
	if write {
		verb, tail = "REWROTE", ""
	}
	fmt.Printf("\n%s %d marker range(s) across %d file(s)%s\n", verb, touched, len(byFile), tail)
	if skipped > 0 {
		fmt.Printf("%d MOVED range(s) had several equally valid targets and were NOT rewritten; "+
			"fix those by hand.\n", skipped)
	}
	return 0
}

func pySplitKeepEnds(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.SplitAfter(s, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func pbdLoadCites(repo string, pin *pbdCheckout) []*pcCite {
	sc := pcCollect(repo, pin.path)
	var broken, formal []*pcCite
	for _, c := range sc.formal {
		if c.pob2 {
			continue
		}
		formal = append(formal, c)
		if c.status != "OK" {
			broken = append(broken, c)
		}
	}
	sc.formal = formal
	if len(broken) > 0 {
		pbdRefuse(fmt.Sprintf("%d of %d marker(s) do not resolve against the PIN %s.\n"+
			"  The pin is wrong, or the tree is. Fix `mise run citations` first -- every\n"+
			"  verdict below would be measured against the wrong baseline.\n  e.g. %s",
			len(broken), len(sc.formal), pin.path, broken[0].String()))
	}
	if len(sc.formal) == 0 {
		pbdRefuse("no //pob: markers found in " + repo)
	}
	return sc.formal
}

func pbdCap[T any](rows []T, top int) []T {
	if top <= 0 || top >= len(rows) {
		return rows
	}
	return rows[:top]
}

type pbdRatio struct {
	r float64
	d *pbdDrift
}

func pbdReportPerCitation(weighted []*pbdDrift, top int) {
	buckets := []struct {
		edge  float64
		label string
	}{
		{1.00, "100%   intact"}, {pbdIntactFraction, ">= 90%  update"},
		{0.75, ">= 75%  heavy edit"}, {0.50, ">= 50%  half rewritten"},
		{0.0001, "<  50%  rewrite"}, {-1.0, "0%     gone"},
	}
	ratios := make([]pbdRatio, len(weighted))
	for i, d := range weighted {
		ratios[i] = pbdRatio{float64(d.survivingLines) / float64(d.citedLines), d}
	}
	sort.SliceStable(ratios, func(i, j int) bool { return ratios[i].r < ratios[j].r })
	fmt.Println()
	fmt.Println("=== PER-CITATION SURVIVAL  (one row = one site a human must handle) ===")
	haveLo := false
	lo := 0.0
	for _, b := range buckets {
		var sel []pbdRatio
		for _, rd := range ratios {
			if rd.r >= b.edge && (!haveLo || rd.r < lo) {
				sel = append(sel, rd)
			}
		}
		lo, haveLo = b.edge, true
		if len(sel) > 0 {
			cited := 0
			for _, rd := range sel {
				cited += rd.d.citedLines
			}
			fmt.Printf("  %-22s %4d citation(s)  %6d cited line(s)\n", b.label, len(sel), cited)
		}
	}
	var need []pbdRatio
	for _, rd := range ratios {
		if rd.r < pbdIntactFraction {
			need = append(need, rd)
		}
	}
	needCited := 0
	for _, rd := range need {
		needCited += rd.d.citedLines
	}
	intactPct := 100.0 * float64(len(weighted)-len(need)) / float64(len(weighted))
	fmt.Println("  ---")
	fmt.Printf("  RE-PORT SURFACE: %d of %d citation(s) survive below %d%%  (%d cited line(s))\n",
		len(need), len(weighted), int(pbdIntactFraction*100), needCited)
	fmt.Printf("  %-22s %5.1f%% of citations survive >= %d%% of their own lines\n",
		"GO/NO-GO", intactPct, int(pbdIntactFraction*100))
	verdict := "RE-PORT -- too many individual sites diverged. A different budget; do not bump."
	if intactPct >= 90 {
		verdict = "BOUNDED -- work the sites listed below."
	}
	fmt.Printf("  VERDICT: %s\n", verdict)
	shown := pbdCap(need, top)
	for _, rd := range shown {
		fmt.Printf("      %5.1f%%  %-40s %s:%d-%d\n", 100*rd.r,
			fmt.Sprintf("%s:%d", rd.d.cite.gofile, rd.d.cite.goline),
			rd.d.cite.path, rd.d.cite.start, rd.d.cite.end)
	}
	if len(need) > len(shown) {
		fmt.Printf("      ... %d more (--top N, or --top 0 for all)\n", len(need)-len(shown))
	}
}

type pbdTally struct {
	order []string
	count map[string]int
	size  map[string]int
}

func newPBDTally() *pbdTally {
	return &pbdTally{count: map[string]int{}, size: map[string]int{}}
}

func (t *pbdTally) add(key string, sz int) {
	if _, ok := t.count[key]; !ok {
		t.order = append(t.order, key)
	}
	t.count[key]++
	t.size[key] += sz
}

func (t *pbdTally) ranked() []string {
	keys := append([]string(nil), t.order...)
	sort.SliceStable(keys, func(i, j int) bool { return t.size[keys[i]] > t.size[keys[j]] })
	return keys
}

func pbdSite(d *pbdDrift) string { return fmt.Sprintf("%s:%d", d.cite.gofile, d.cite.goline) }

func pbdReport(pin, cand *pbdCheckout, drifts []*pbdDrift, portFiles map[string]bool,
	top int, details bool) int {
	ranged := 0
	for _, d := range drifts {
		if d.cite.start >= 0 {
			ranged++
		}
	}
	counts := map[string][]*pbdDrift{}
	for _, v := range pbdVerdicts {
		for _, d := range drifts {
			if d.verdict == v {
				counts[v] = append(counts[v], d)
			}
		}
	}
	n := len(drifts)

	fmt.Printf("PIN        %s  %s%s\n", pin.path, pin.describe(), pin.layoutNote())
	fmt.Printf("CANDIDATE  %s  %s%s\n", cand.path, cand.describe(), cand.layoutNote())
	if pin.path == cand.path {
		fmt.Println("           IDENTITY RUN -- pin and candidate are the same checkout. Every verdict")
		fmt.Println("           must be UNCHANGED; anything else means this tool is broken.")
	}
	fmt.Println()

	luaSet := map[string]bool{}
	goSet := map[string]bool{}
	var luaOrder []string
	for _, d := range drifts {
		if !luaSet[d.cite.path] {
			luaSet[d.cite.path] = true
			luaOrder = append(luaOrder, d.cite.path)
		}
		goSet[d.cite.gofile] = true
	}
	var missing []string
	for _, f := range luaOrder {
		if cand.raw(f) == nil {
			missing = append(missing, f)
		}
	}
	sort.Strings(missing)
	inPort := 0
	for _, d := range drifts {
		if portFiles[d.cite.gofile] {
			inPort++
		}
	}
	fmt.Println("=== DENOMINATOR ===")
	fmt.Printf("  %d citation(s) -- %d ranged, %d whole-file -- across %d Go file(s), citing %d Lua file(s)\n",
		n, ranged, n-ranged, len(goSet), len(luaSet))
	fmt.Printf("  %d of them sit in PORT files (scripts/port_manifest.txt)\n", inPort)
	fmt.Printf("  %d cited Lua file(s) absent from the candidate\n", len(missing))
	for _, f := range pbdCap(missing, 10) {
		fmt.Printf("      %s\n", f)
	}

	fmt.Println()
	fmt.Println("=== VERDICTS ===")
	exact := 0
	for _, d := range counts[pbdUnchanged] {
		if strings.HasSuffix(d.note, "byte-identical") {
			exact++
		}
	}
	for _, v := range pbdVerdicts {
		extra := ""
		if v == pbdUnchanged {
			extra = fmt.Sprintf("  (%d byte-identical, %d whitespace/comment-only)",
				exact, len(counts[v])-exact)
		}
		fmt.Printf("  %-10s %5d   %5.1f%%%s\n", v, len(counts[v]),
			100.0*float64(len(counts[v]))/float64(n), extra)
	}
	ok := len(counts[pbdUnchanged]) + len(counts[pbdMoved])
	fmt.Printf("  %-10s %5d   %5.1f%%   (sites to open; NOT the go/no-go -- see PER-CITATION below)\n",
		"OK-or-MOVED", ok, 100.0*float64(ok)/float64(n))

	var weighted []*pbdDrift
	for _, d := range drifts {
		if d.citedLines > 0 {
			weighted = append(weighted, d)
		}
	}
	skipped := n - len(weighted)
	if len(weighted) > 0 {
		tot, surv := 0, 0
		for _, d := range weighted {
			tot += d.citedLines
			surv += d.survivingLines
		}
		fmt.Printf("  %-10s %5d / %d cited Lua line(s) survive = %5.1f%%   <- SIZE-INDEPENDENT\n",
			"by LINES", surv, tot, 100.0*float64(surv)/float64(tot))
		pbdReportPerCitation(weighted, top)
		fmt.Printf("             over %d ranged citation(s); %d whole-file marker(s) have no line "+
			"count and are excluded\n", len(weighted), skipped)
	}

	if len(counts[pbdMoved]) > 0 {
		fmt.Println()
		fmt.Println("=== MOVED  (mechanical; `--fix` rewrites these) ===")
		rows := append([]*pbdDrift(nil), counts[pbdMoved]...)
		sort.SliceStable(rows, func(i, j int) bool {
			ai, aj := pbdAbs(rows[i].offset()), pbdAbs(rows[j].offset())
			if ai != aj {
				return ai < aj
			}
			return rows[i].cite.gofile < rows[j].cite.gofile
		})
		for _, d := range pbdCap(rows, top) {
			spec := d.newWhere()
			if i := strings.LastIndex(spec, ":"); i >= 0 {
				spec = spec[i+1:]
			}
			fmt.Printf("  %-46s %-42s -> %-14s %s\n", pbdSite(d), d.cite.where(), spec, d.note)
		}
		if top > 0 && len(counts[pbdMoved]) > top {
			fmt.Printf("  ... %d more (--top N)\n", len(counts[pbdMoved])-top)
		}
	}

	if len(counts[pbdDeleted]) > 0 {
		fmt.Println()
		fmt.Println("=== DELETED  (our port may now be dead code -- read each one) ===")
		rows := append([]*pbdDrift(nil), counts[pbdDeleted]...)
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].diffSize > rows[j].diffSize })
		for _, d := range pbdCap(rows, top) {
			fmt.Printf("  %-46s %-42s %s\n", pbdSite(d), d.cite.where(), d.note)
		}
		if top > 0 && len(counts[pbdDeleted]) > top {
			fmt.Printf("  ... %d more (--top N)\n", len(counts[pbdDeleted])-top)
		}
	}

	if len(counts[pbdChanged]) > 0 {
		fmt.Println()
		fmt.Println("=== CHANGED  (THE UPGRADE WORKLIST, ranked by diff size) ===")
		fmt.Printf("  %6s %6s  %-40s %-40s %s\n", "lines", "sim", "Go site", "Lua range", "note")
		rows := append([]*pbdDrift(nil), counts[pbdChanged]...)
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].diffSize != rows[j].diffSize {
				return rows[i].diffSize > rows[j].diffSize
			}
			return rows[i].cite.gofile < rows[j].cite.gofile
		})
		for _, d := range pbdCap(rows, top) {
			fmt.Printf("  %6d %5.0f%%  %-40s %-40s %s\n", d.diffSize, 100*d.similarity,
				pbdSite(d), d.cite.where(), d.note)
		}
		if top > 0 && len(counts[pbdChanged]) > top {
			fmt.Printf("  ... %d more (--top N)\n", len(counts[pbdChanged])-top)
		}

		fmt.Println()
		fmt.Println("  by Lua file:")
		perLua := newPBDTally()
		for _, d := range counts[pbdChanged] {
			perLua.add(d.cite.path, d.diffSize)
		}
		for _, f := range pbdCap(perLua.ranked(), top) {
			fmt.Printf("    %5d range(s) %7d changed line(s)  %s\n", perLua.count[f], perLua.size[f], f)
		}

		fmt.Println()
		fmt.Println("  by Go file:")
		perGo := newPBDTally()
		for _, d := range counts[pbdChanged] {
			perGo.add(d.cite.gofile, d.diffSize)
		}
		for _, f := range pbdCap(perGo.ranked(), top) {
			fmt.Printf("    %5d range(s) %7d changed line(s)  %s\n", perGo.count[f], perGo.size[f], f)
		}
	}

	if details {
		fmt.Println()
		scope := fmt.Sprintf("top %d", top)
		if top <= 0 {
			scope = "all"
		}
		fmt.Printf("=== CHANGED, in full (%s) ===\n", scope)
		rows := append([]*pbdDrift(nil), counts[pbdChanged]...)
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].diffSize > rows[j].diffSize })
		for _, d := range pbdCap(rows, top) {
			fmt.Println()
			fmt.Printf("--- %s  %s  (%d changed line(s), %.0f%% similar)\n",
				pbdSite(d), d.cite.where(), d.diffSize, 100*d.similarity)
			var a, b []string
			if d.cite.start < 0 {
				a = pbdRStripAll(pin.raw(d.cite.path))
				b = pbdRStripAll(cand.raw(d.cite.path))
			} else {
				s, _ := pin.sliceRaw(d.cite.path, d.cite.start, d.cite.end)
				a = pbdRStripAll(s)
				lo, hi := d.cite.start, d.cite.end
				if d.newStart > 0 {
					lo, hi = d.newStart, d.newEnd
				}
				t, _ := cand.sliceRaw(d.cite.path, lo, hi)
				b = pbdRStripAll(t)
			}
			for _, ln := range pyUnifiedDiff(a, b, "pin", "candidate", "", "", 1, "") {
				fmt.Printf("    %s\n", ln)
			}
		}
	}

	fmt.Println()
	if len(counts[pbdChanged]) > 0 || len(counts[pbdDeleted]) > 0 {
		fmt.Printf("%d range(s) need human work; %d are mechanical.\n",
			len(counts[pbdChanged])+len(counts[pbdDeleted]), len(counts[pbdMoved]))
	} else {
		fmt.Println("Nothing needs human work.")
	}
	return 0
}

func pbdRStripAll(xs []string) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = strings.TrimRightFunc(x, pbdIsPySpace)
	}
	return out
}

func pbdIsPySpace(r rune) bool { return pyIsSpace(r) }

const pbdUsage = `usage: pob_drift.py [--candidate DIR] [--pin DIR] [--top N]
                    [--details] [--fix] [--dry-run] [--selftest]

Classify every //pob: citation against a CANDIDATE PoB checkout: the upgrade worklist.

  --candidate DIR  the PoB checkout to measure the pin's citations against (or CANDIDATE=)
  --pin DIR        the pinned checkout the markers were written against (or POB_DIR=)
  --top N          rows per section; 0 or less means ALL (or TOP=, default 30)
  --details        print the diff of each CHANGED range (or DETAILS=1)
  --fix            rewrite MOVED markers' line numbers. Opt-in, and never combined with the
                   report -- this run prints a diff and nothing else (or FIX=1)
  --dry-run        with --fix: print the diff, write nothing (or DRYRUN=1)
  --selftest       the synthetic control
  -h, --help       this text
`

type pbdArgs struct {
	candidate, pin  string
	top             int
	details, fix    bool
	dryRun, selfTst bool
}

func pbdParseArgs(argv []string) pbdArgs {
	home, _ := pyPathHome()
	a := pbdArgs{
		candidate: os.Getenv("CANDIDATE"),
		pin:       pyPathJoin(home, pcDefaultPoBRel),
		details:   os.Getenv("DETAILS") == "1",
		fix:       os.Getenv("FIX") == "1",
		dryRun:    os.Getenv("DRYRUN") == "1",
	}
	if v := os.Getenv("POB_DIR"); v != "" {
		a.pin = v
	}
	topRaw := os.Getenv("TOP")
	if topRaw == "" {
		topRaw = "30"
	}
	for i := 0; i < len(argv); i++ {
		x, val, hasVal := argv[i], "", false
		if strings.HasPrefix(x, "--") && strings.Contains(x, "=") {
			parts := strings.SplitN(x, "=", 2)
			x, val, hasVal = parts[0], parts[1], true
		}
		switch x {
		case "--candidate", "--pin", "--top":
			if !hasVal {
				if i+1 >= len(argv) {
					pbdRefuse(fmt.Sprintf("%s needs a value\n\n%s", x, pbdUsage))
				}
				i++
				val = argv[i]
			}
			switch x {
			case "--candidate":
				a.candidate = val
			case "--pin":
				a.pin = val
			case "--top":
				topRaw = val
			}
		case "--details", "--fix", "--dry-run", "--selftest":
			if hasVal {
				pbdRefuse(fmt.Sprintf("%s takes no value\n\n%s", x, pbdUsage))
			}
			switch x {
			case "--details":
				a.details = true
			case "--fix":
				a.fix = true
			case "--dry-run":
				a.dryRun = true
			case "--selftest":
				a.selfTst = true
			}
		case "-h", "--help":
			fmt.Print(pbdUsage)
			os.Exit(0)
		default:
			pbdRefuse(fmt.Sprintf("unknown argument %s\n\n%s", pyRepr(argv[i]), pbdUsage))
		}
	}
	n, err := pbdPyInt(topRaw)
	if err != nil {
		pbdRefuse(fmt.Sprintf("--top wants an integer, not %s (TOP= sets it too)", pyRepr(topRaw)))
	}
	a.top = n
	return a
}

func pbdPyInt(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimFunc(s, pbdIsPySpace))
	if err != nil {
		return 0, fmt.Errorf("not an integer: %w", err)
	}
	return n, nil
}

func pobDrift(argv []string) int {
	args := pbdParseArgs(argv)
	if args.selfTst {
		return pbdSelftest()
	}
	if args.candidate == "" {
		pbdRefuse(fmt.Sprintf("no candidate checkout. Pass --candidate <dir> (or CANDIDATE=<dir>).\n"+
			"  The pin is %s -- ~/PoB/PoE1 is a NEWER DECOY, not the pin.", args.pin))
	}

	pin := pbdNewCheckout(args.pin, "pin")
	cand := pbdNewCheckout(args.candidate, "candidate")
	repo := pcRepoRoot()
	cites := pbdLoadCites(repo, pin)

	luaFiles := map[string]bool{}
	for _, c := range cites {
		luaFiles[c.path] = true
	}
	resolvable := 0
	for f := range luaFiles {
		if cand.raw(f) != nil {
			resolvable++
		}
	}
	if resolvable == 0 {
		pbdRefuse(fmt.Sprintf("not one of the %d cited Lua files exists in the candidate %s.\n"+
			"  Either it is not a PoB checkout of a comparable layout, or the paths moved\n"+
			"  wholesale -- a run against it would report %d DELETED and mean nothing.",
			len(luaFiles), cand.path, len(luaFiles)))
	}

	drifts := make([]*pbdDrift, len(cites))
	for i, c := range cites {
		drifts[i] = pbdClassify(c, pin, cand)
	}

	portFiles := map[string]bool{}
	for _, ln := range strings.Split(pbdReadText(pyPathJoin(repo, pcManifest), "the port manifest"), "\n") {
		ln = strings.TrimSpace(ln)
		if f := strings.Fields(ln); ln != "" && !strings.HasPrefix(ln, "#") && len(f) > 1 && f[1] == "PORT" {
			portFiles[f[0]] = true
		}
	}

	if args.fix {
		fmt.Printf("PIN        %s\nCANDIDATE  %s\n\n", pin.path, cand.path)
		return pbdApplyFixes(repo, drifts, !args.dryRun)
	}
	return pbdReport(pin, cand, drifts, portFiles, args.top, args.details)
}

const pbdSample = "-- Fake.lua\n" +
	"local m = { }\n" +
	"\n" +
	"-- adds two numbers\n" +
	"function m.add(a, b)\n" +
	"\tlocal total = a + b\n" +
	"\treturn total\n" +
	"end\n" +
	"\n" +
	"function m.mul(a, b)\n" +
	"\treturn a * b\n" +
	"end\n" +
	"\n" +
	"function m.doomed(a)\n" +
	"\treturn a\n" +
	"end\n" +
	"\n" +
	"for i = 1, 10 do\n" +
	"\tprint(i)\n" +
	"end\n"

func pbdWrite(root, rel, text string) {
	full := pyPathJoin(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		//go-only: Path.mkdir/write_text/tempfile raise OSError in this tool's OWN
		pbdRefuse("selftest sandbox: " + errnoText(err))
	}
	if err := os.WriteFile(full, []byte(text), 0o644); err != nil { //nolint:gosec // a sandbox under TMPDIR
		//go-only: Path.mkdir/write_text/tempfile raise OSError in this tool's OWN
		pbdRefuse("selftest sandbox: " + errnoText(err))
	}
}

func pbdIndexOf(xs []string, want string) int {
	for i, x := range xs {
		if x == want {
			return i
		}
	}
	return -1
}

func pbdCapture(f func()) string {
	tmp, err := os.CreateTemp("", "pobdrift-cap-")
	if err != nil {
		//go-only: Path.mkdir/write_text/tempfile raise OSError in this tool's OWN
		pbdRefuse("selftest capture: " + errnoText(err))
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	saved := os.Stdout
	os.Stdout = tmp
	f()
	os.Stdout = saved
	_ = tmp.Close()
	body, err := os.ReadFile(tmp.Name())
	if err != nil {
		//go-only: Path.mkdir/write_text/tempfile raise OSError in this tool's OWN
		pbdRefuse("selftest capture: " + errnoText(err))
	}
	return string(body)
}

type pbdCase struct {
	name    string
	cite    *pcCite
	want    string
	wantOff int
}

func pbdSelftest() int {
	root, err := os.MkdirTemp("", "pobdrift-")
	if err != nil {
		//go-only: Path.mkdir/write_text/tempfile raise OSError in this tool's OWN
		pbdRefuse("selftest sandbox: " + errnoText(err))
	}
	defer func() { _ = os.RemoveAll(root) }()
	old, nw := pyPathJoin(root, "old"), pyPathJoin(root, "new")
	rel, other := "src/Modules/Fake.lua", "src/Modules/Gone.lua"

	pbdWrite(old, rel, pbdSample)
	pbdWrite(old, other, "function g.x()\n\treturn 1\nend\n")

	body := strings.Split(pbdSample, "\n")
	moved := append(make([]string, 10), body...)
	moved[10+3] = "-- sums its two arguments"
	moved[10+5] = "    local  total  =  a + b"
	moved = append(moved[:10+6], append([]string{""}, moved[10+6:]...)...)
	moved[pbdIndexOf(moved, "\treturn a * b")] = "\treturn a * b * 2"
	j := pbdIndexOf(moved, "function m.doomed(a)")
	moved = append(moved[:j], moved[j+4:]...)
	pbdWrite(nw, rel, strings.Join(moved, "\n"))

	pin, cand := pbdNewCheckout(old, "pin"), pbdNewCheckout(nw, "candidate")

	cite := func(start, end int, path string) *pcCite {
		if path == "" {
			path = rel
		}
		return &pcCite{gofile: "x.go", goline: 1, path: path, start: start, end: end,
			formal: true, status: "OK"}
	}

	cases := []pbdCase{
		{"blank lines inserted above a function", cite(5, 8, ""), pbdMoved, 10},
		{"reindent + reworded comment + inserted blank inside", cite(4, 8, ""), pbdMoved, 10},
		{"a statement changed", cite(10, 12, ""), pbdChanged, -1},
		{"a function deleted", cite(14, 16, ""), pbdDeleted, -1},
		{"the whole file gone", cite(1, 3, other), pbdDeleted, -1},
		{"an untouched block below every edit", cite(18, 20, ""), pbdMoved, 7},
		{"a whole-file marker on a changed file", cite(-1, -1, ""), pbdChanged, -1},
	}
	bad := 0
	mark := func(ok bool) string {
		if ok {
			return "ok"
		}
		return "BROKEN"
	}
	for _, cs := range cases {
		d := pbdClassify(cs.cite, pin, cand)
		ok := d.verdict == cs.want && (cs.wantOff < 0 || d.offset() == cs.wantOff)
		if !ok {
			bad++
		}
		off := "-"
		if d.newStart > 0 {
			off = strconv.Itoa(d.offset())
		}
		fmt.Printf("  %-7s expect %-10s got %-10s off=%-4s  %s\n",
			mark(ok), cs.want, d.verdict, off, cs.name)
	}

	for _, cs := range cases {
		if cs.cite.path == other {
			continue
		}
		d := pbdClassify(cs.cite, pin, pin)
		ok := d.verdict == pbdUnchanged && strings.HasSuffix(d.note, "byte-identical")
		if !ok {
			bad++
		}
		fmt.Printf("  %-7s expect %-10s got %-10s off=%-4s  identity: %s\n",
			mark(ok), pbdUnchanged, d.verdict, "-", cs.name)
	}

	check := func(ok bool, tail string, want, got string) {
		if !ok {
			bad++
		}
		fmt.Printf("  %-7s expect %-10s got %-10s off=-     %s\n", mark(ok), want, got, tail)
	}

	pbdWrite(nw, "src/Modules/C.lua", "-- a totally different comment\nreturn 1 + 1\n")
	pbdWrite(old, "src/Modules/C.lua", "-- the original comment\nreturn 1 + 1\n")
	d := pbdClassify(cite(1, 2, "src/Modules/C.lua"), pbdNewCheckout(old, "pin"), pbdNewCheckout(nw, "candidate"))
	check(d.verdict == pbdUnchanged, "a comment-only edit", pbdUnchanged, d.verdict)

	pbdWrite(old, "src/Modules/W.lua", "function f(a)\n\treturn a\nend\n")
	pbdWrite(nw, "src/Modules/W.lua", "function  f( a )\n    return   a\nend\n")
	d = pbdClassify(cite(1, 3, "src/Modules/W.lua"), pbdNewCheckout(old, "pin"), pbdNewCheckout(nw, "candidate"))
	check(d.verdict == pbdUnchanged && d.note == "whitespace/comment-only change",
		"a reindent in place", pbdUnchanged, d.verdict)

	pbdWrite(old, "src/Modules/R.lua", "function m.oldName(a)\n\treturn a + 1\nend\n")
	pbdWrite(nw, "src/Modules/R.lua", "function m.brandNew(a)\n\treturn a + 1\nend\n")
	d = pbdClassify(cite(1, 3, "src/Modules/R.lua"), pbdNewCheckout(old, "pin"), pbdNewCheckout(nw, "candidate"))
	check(d.verdict == pbdDeleted, "a renamed function", pbdDeleted, d.verdict)

	for _, tc := range []struct {
		nm        string
		c         *pcCite
		pin, cand *pbdCheckout
	}{
		{"a deleted function", cite(14, 16, ""), pin, cand},
		{"a renamed function", cite(1, 3, "src/Modules/R.lua"),
			pbdNewCheckout(old, "pin"), pbdNewCheckout(nw, "candidate")},
		{"a file absent from the candidate", cite(1, 3, other), pin, cand},
	} {
		dd := pbdClassify(tc.c, tc.pin, tc.cand)
		check(dd.verdict == pbdDeleted && dd.citedLines == 3 && dd.survivingLines == 0,
			"DELETED counts 0 of 3 line(s): "+tc.nm, "0/3",
			fmt.Sprintf("%s/%s", pbdNoneInt(dd.survivingLines), pbdNoneInt(dd.citedLines)))
	}

	var pinP []string
	for i := 1; i <= 20; i++ {
		pinP = append(pinP, fmt.Sprintf("-- filler comment %d", i))
	}
	pinP = append(pinP, "local anchorAlpha = 1", "local anchorBeta = 2")
	pbdWrite(old, "src/Modules/P.lua", strings.Join(pinP, "\n")+"\n")
	var candP []string
	for i := 1; i <= 16; i++ {
		candP = append(candP, fmt.Sprintf("-- comment %d", i))
	}
	candP = append(candP, "local pad0 = 0", "local anchorAlpha = 1", "local pad1 = 1", "local anchorBeta = 2")
	for i := 0; i < 17; i++ {
		candP = append(candP, fmt.Sprintf("local mid%d = %d", i, i))
	}
	candP = append(candP, "local anchorAlpha = 1", "local pad9 = 9", "local anchorBeta = 2")
	pbdWrite(nw, "src/Modules/P.lua", strings.Join(candP, "\n")+"\n")
	d = pbdClassify(cite(21, 22, "src/Modules/P.lua"), pbdNewCheckout(old, "pin"), pbdNewCheckout(nw, "candidate"))
	check(d.verdict == pbdChanged && d.newStart == 18, "CHANGED anchors nearest where it sat",
		"at 18", "at "+pbdNoneInt(d.newStart))

	text := "//pob:src/Modules/Fake.lua:5-8, src/Modules/Fake.lua:10-12"
	got, nfix := pbdRewriteLine(text, [][2]string{
		{"src/Modules/Fake.lua:5-8", "src/Modules/Fake.lua:15-18"},
		{"src/Modules/Fake.lua:10-12", "src/Modules/Fake.lua:20-22"}})
	want := "//pob:src/Modules/Fake.lua:15-18, src/Modules/Fake.lua:20-22"
	okFix := got == want && nfix == 2
	shown := got
	if okFix {
		shown = "rewritten"
	}
	check(okFix, "--fix rewrites both entries of one marker", "rewritten", shown)

	caseDrifts := func() []*pbdDrift {
		out := make([]*pbdDrift, len(cases))
		for i, cs := range cases {
			out[i] = pbdClassify(cs.cite, pin, cand)
		}
		return out
	}

	raisedWith := ""
	noRaise := true
	func() {
		defer func() {
			if r := recover(); r != nil {
				noRaise = false
				raisedWith = fmt.Sprintf("%v", r)
			}
		}()
		_ = pbdCapture(func() { pbdReport(pin, cand, caseDrifts(), map[string]bool{}, 30, true) })
	}()
	if raisedWith != "" {
		fmt.Printf("       %s\n", raisedWith)
	}
	raised := "no raise"
	if !noRaise {
		raised = "raised"
	}
	check(noRaise, "report(--details) over every verdict class", "no raise", raised)

	buf := pbdCapture(func() { pbdReport(pin, cand, caseDrifts(), map[string]bool{}, 0, true) })
	nch := 0
	for _, dd := range caseDrifts() {
		if dd.verdict == pbdChanged {
			nch++
		}
	}
	gotRows := strings.Count(buf, "\n--- ")
	check(gotRows == nch && !strings.Contains(buf, "more (--top"),
		"--top 0 lists every row, in every section",
		fmt.Sprintf("%d row(s)", nch), fmt.Sprintf("%d row(s)", gotRows))

	total := len(cases) + len(cases) - 1 + 10
	fmt.Printf("\nSELFTEST: %d of %d checks behave as declared\n", total-bad, total)
	if bad > 0 {
		return 1
	}
	return 0
}

func pbdNoneInt(v int) string {
	if v < 0 {
		return "None"
	}
	return strconv.Itoa(v)
}
