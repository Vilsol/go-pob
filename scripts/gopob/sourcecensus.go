package main

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	scMinTreeSources = 5
	scTolerance      = 0.005
)

const scUsage = "usage: source_census.py --our-dump F --stat S [--stat S]... [--dumps D] " +
	"[--blob B] [--min-support N] [--top N] [--granular] [--fail-class wrong|miss|extra] " +
	"[--direction both|high|low] [--self-check]"

func scExit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func scDie(format string, args ...any) {
	scExit("REFUSING: " + fmt.Sprintf(format, args...))
}

type scArgs struct {
	ourDump    string
	dumps      string
	blob       string
	stat       []string
	minSupport int
	top        int
	granular   bool
	failClass  string
	direction  string
	selfCheck  bool
}

func scParseArgs(argv []string) scArgs {
	a := scArgs{
		dumps:      pyExpandUser("~/go-pob-corpus/dumps-3.29"),
		blob:       "testdata/oracle-3.29/output-3.29.jsonl.gz",
		minSupport: 8,
		top:        25,
		failClass:  "wrong",
		direction:  "both",
	}
	valued := map[string]bool{
		"--our-dump": true, "--dumps": true, "--blob": true, "--stat": true,
		"--min-support": true, "--top": true, "--fail-class": true, "--direction": true,
	}
	choices := map[string][]string{
		"--fail-class": {"wrong", "miss", "extra"},
		"--direction":  {"both", "high", "low"},
	}
	for i := 0; i < len(argv); {
		x := argv[i]
		switch {
		case valued[x]:
			if i+1 >= len(argv) {
				scExit(fmt.Sprintf("source-census: %s needs a value\n%s", x, scUsage))
			}
			v := argv[i+1]
			if cs, ok := choices[x]; ok {
				found := false
				for _, c := range cs {
					if c == v {
						found = true
					}
				}
				if !found {
					scExit(fmt.Sprintf("source-census: %s must be one of %s, not %s\n%s",
						x, strings.Join(cs, ", "), pyRepr(v), scUsage))
				}
			}
			switch x {
			case "--our-dump":
				a.ourDump = v
			case "--dumps":
				a.dumps = v
			case "--blob":
				a.blob = v
			case "--stat":
				a.stat = append(a.stat, v)
			case "--min-support":
				a.minSupport = scArgInt(x, v)
			case "--top":
				a.top = scArgInt(x, v)
			case "--fail-class":
				a.failClass = v
			case "--direction":
				a.direction = v
			}
			i += 2
		case x == "--granular":
			a.granular = true
			i++
		case x == "--self-check":
			a.selfCheck = true
			i++
		default:
			scExit(fmt.Sprintf("source-census: unknown argument %s\n%s", pyRepr(x), scUsage))
		}
	}
	if a.ourDump == "" {
		scExit(fmt.Sprintf("source-census: --our-dump is required\n%s", scUsage))
	}
	if len(a.stat) == 0 {
		scExit(fmt.Sprintf("source-census: at least one --stat is required\n%s", scUsage))
	}
	return a
}

func scArgInt(flag, v string) int {
	n, err := pbdPyInt(v)
	if err != nil {
		scExit(fmt.Sprintf("source-census: %s needs an integer, not %s\n%s", flag, pyRepr(v), scUsage))
	}
	return n
}

func scTextLines(path string, gz bool) []string {
	f, err := os.Open(path) //nolint:gosec // an operator-supplied path, as in the original
	if err != nil {
		scDie("%s is not readable: %s", path, errnoText(err))
	}
	defer func() { _ = f.Close() }()

	var raw []byte
	if gz {
		zr, err := gzip.NewReader(f)
		if err != nil {
			if errors.Is(err, gzip.ErrHeader) || errors.Is(err, io.EOF) ||
				errors.Is(err, io.ErrUnexpectedEOF) {
				scDie("%s is not a gzip file", path)
			}
			scDie("%s is not readable: %s", path, errnoText(err))
		}
		defer func() { _ = zr.Close() }()
		raw, err = io.ReadAll(zr)
		if err != nil {
			if errors.Is(err, gzip.ErrHeader) {
				scDie("%s is not a gzip file", path)
			}
			scDie("%s is truncated, or is not text", path)
		}
	} else {
		raw, err = io.ReadAll(f)
		if err != nil {
			scDie("%s is not readable: %s", path, errnoText(err))
		}
	}
	if !utf8.Valid(raw) {
		scDie("%s is truncated, or is not text", path)
	}
	return pyUniversalLines(string(raw))
}

func scOurDump(path string) ([]string, map[string]map[string]float64, map[string]string) {
	order := []string{}
	vals := map[string]map[string]float64{}
	meta := map[string]string{}
	var cur map[string]float64
	for _, line := range scTextLines(path, strings.HasSuffix(path, ".gz")) {
		kind, rest, _ := strings.Cut(line, "\t")
		rest = strings.TrimRight(rest, "\n")
		switch kind {
		case "S":
			key, val, _ := strings.Cut(rest, "\t")
			if cur == nil {
				scDie("%s has an S record before any B record; the dump is malformed", path)
			}
			f, ok := pyParseFloat(val)
			if !ok {
				scDie("%s: stat %s has a non-numeric value %s", path, pyRepr(key), pyRepr(val))
			}
			cur[key] = f
		case "B":
			build, _, _ := strings.Cut(rest, "\t")
			if _, seen := vals[build]; !seen {
				vals[build] = map[string]float64{}
				order = append(order, build)
			}
			cur = vals[build]
		case "V":
			k, v, _ := strings.Cut(rest, "\t")
			meta[k] = v
		}
	}
	if len(vals) == 0 {
		scDie("%s contains no build records. Re-run `task corpus-dump OUT=...`.", path)
	}
	return order, vals, meta
}

func scBlob(path string) map[string]map[string]any {
	recs := map[string]map[string]any{}
	for n, line := range scTextLines(path, true) {
		if pyStrip(line) == "" {
			continue
		}
		dec := json.NewDecoder(strings.NewReader(line))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			scDie("%s line %d is not valid JSON", path, n+1)
		}
		d, ok := v.(map[string]any)
		if !ok {
			scDie("%s line %d is not a JSON object", path, n+1)
		}
		bv, ok := d["build"]
		if !ok {
			scDie("%s line %d has no 'build' field", path, n+1)
		}
		b, ok := bv.(string)
		if !ok {
			scDie("%s line %d has a non-string 'build'", path, n+1)
		}
		recs[b] = d
	}
	if len(recs) == 0 {
		scDie("%s is empty — the oracle blob is LFS; is it checked out?", path)
	}
	return recs
}

func scNum(v any) (float64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	f, err := n.Float64()
	return f, err == nil
}

func scTreeSources(rec map[string]any, build string) float64 {
	v, present := rec["treeSources"]
	if !present {
		return 0
	}
	f, ok := scNum(v)
	if !ok {
		scDie("%s: treeSources is %s, which is not a number", build, pyReprValue(v))
	}
	return f
}

func scOutputOf(rec map[string]any, build string) map[string]any {
	v, present := rec["output"]
	if !present {
		return map[string]any{}
	}
	o, ok := v.(map[string]any)
	if !ok {
		scDie("%s: 'output' is %s, which is not a table", build, pyReprValue(v))
	}
	return o
}

func scGradedValue(table map[string]any, stat, what string) float64 {
	v, present := table[stat]
	if !present {
		return 0.0
	}
	f, ok := scNum(v)
	if !ok {
		scDie("%s: %s is %s, which is not a number", what, pyRepr(stat), pyReprValue(v))
	}
	return f
}

func scClassify(exp, got float64, hasExp, hasGot bool) string {
	if !hasExp && !hasGot {
		return ""
	}
	if !hasExp {
		return "extra"
	}
	if math.Abs(got-exp) < 1e-4 || (exp != 0 && math.Abs((got-exp)/exp) < scTolerance) {
		return "ok"
	}
	if exp != 0 && got == 0 {
		return "miss"
	}
	return "wrong"
}

func scSourceClass(src string, granular bool) string {
	src = pyStrip(src)
	if !granular {
		for _, prefix := range []string{"Item:", "Tree:"} {
			if strings.HasPrefix(src, prefix) {
				return strings.TrimRight(prefix, ":")
			}
		}
	}
	if paren := strings.Index(src, " ("); paren > 0 {
		return src[:paren]
	}
	return src
}

func scSourceKeys(dumpdir, build, stat string, granular bool) (int, map[string]bool) {
	p := pdtJoin(dumpdir, strings.TrimSuffix(build, ".xml")+".json")
	d, reason := pyLoadJSONObject(p)
	if reason != "" {
		scDie("%s", reason)
	}
	ssv, present := d["statSources"]
	if !present {
		return 0, map[string]bool{}
	}
	ss, ok := ssv.(map[string]any)
	if !ok {
		scDie("%s: 'statSources' is not a table", p)
	}
	onev, present := ss[stat]
	if !present {
		return 0, map[string]bool{}
	}
	one, ok := onev.(map[string]any)
	if !ok {
		scDie("%s: statSources[%s] is not a table", p, pyRepr(stat))
	}
	n := 0
	seen := map[string]bool{}
	for modtype, entries := range one {
		if modtype == "value" {
			continue
		}
		list, ok := entries.([]any)
		if !ok {
			continue
		}
		for _, e := range list {
			em, ok := e.(map[string]any)
			if !ok {
				scDie("%s: a statSources[%s][%s] entry is not a table", p, pyRepr(stat), pyRepr(modtype))
			}
			srcv, present := em["source"]
			src := "?"
			if present {
				s, ok := srcv.(string)
				if !ok {
					scDie("%s: a statSources[%s][%s] entry has a non-string 'source'",
						p, pyRepr(stat), pyRepr(modtype))
				}
				src = s
			}
			n++
			seen[scSourceClass(src, granular)+"|"+modtype] = true
		}
	}
	return n, seen
}

func scPad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return strings.Repeat(" ", w-len(s)) + s
}

type scRow struct {
	norm, raw, pf, pp float64
	nf, np            int
	cls               string
}

func sourceCensus(argv []string) int {
	args := scParseArgs(argv)

	order, ours, meta := scOurDump(args.ourDump)
	blob := scBlob(args.blob)

	dumpdir := pyPathStr(args.dumps)
	if fi, err := os.Stat(dumpdir); err != nil || !fi.IsDir() {
		scDie("%s is not a directory. The PoB dump set carries statSources; without it this "+
			"script has nothing to census.", dumpdir)
	}

	present := map[string]bool{}
	if ents, err := os.ReadDir(dumpdir); err == nil {
		for _, e := range ents {
			if strings.HasSuffix(e.Name(), ".json") {
				present[strings.TrimSuffix(e.Name(), ".json")+".xml"] = true
			}
		}
	}
	graded := []string{}
	for _, b := range order {
		if rec, ok := blob[b]; ok && scTreeSources(rec, b) >= scMinTreeSources {
			graded = append(graded, b)
		}
	}
	if len(graded) == 0 {
		dv := meta["dataVersion"]
		if dv == "" {
			dv = "UNVERSIONED"
		}
		scDie("no build is present in BOTH the run dump and the oracle blob with treeSources >= "+
			"%d. Are the run dump and the blob the same game version? (run dump says %s)",
			scMinTreeSources, dv)
	}
	covered := []string{}
	for _, b := range graded {
		if present[b] {
			covered = append(covered, b)
		}
	}
	if float64(len(covered)) < 0.9*float64(len(graded)) {
		scDie("%s covers only %d of %d graded builds. That is a STALE OR WRONG DUMP SET — repoint "+
			"the symlink at the current one and re-run. Censusing a subset produces percentages "+
			"that look entirely reasonable and are not about this corpus.",
			dumpdir, len(covered), len(graded))
	}

	dv := meta["dataVersion"]
	if dv == "" {
		dv = "UNVERSIONED"
	}
	cd := meta["corpusDir"]
	if cd == "" {
		cd = "?"
	}
	fmt.Printf("run dump   %s  (%s, %s)\n", args.ourDump, dv, cd)
	fmt.Printf("blob       %s\n", args.blob)
	fmt.Printf("PoB dumps  %s  covering %d of %d graded builds\n", dumpdir, len(covered), len(graded))
	fmt.Printf("tolerance  %s  ·  treeSources >= %d  ·  failing class = %s\n\n",
		pyFloat(scTolerance), scMinTreeSources, args.failClass)

	for _, stat := range args.stat {
		anyHas := false
		for _, b := range covered {
			if _, ok := scOutputOf(blob[b], b)[stat]; ok {
				anyHas = true
				break
			}
		}
		if !anyHas {
			lim := min(len(covered), 200)
			elsewhereSet := map[string]bool{}
			for _, b := range covered[:lim] {
				for t, v := range blob[b] {
					if m, ok := v.(map[string]any); ok {
						if _, has := m[stat]; has {
							elsewhereSet[t] = true
						}
					}
				}
			}
			elsewhere := make([]string, 0, len(elsewhereSet))
			for t := range elsewhereSet {
				elsewhere = append(elsewhere, t)
			}
			sort.Strings(elsewhere)
			extra := ""
			if len(elsewhere) > 0 {
				extra = fmt.Sprintf("It IS present in: %s. ", strings.Join(elsewhere, ", "))
			}
			scDie("%s does not appear in the blob's `output` table on ANY of %d graded builds, "+
				"and this census reads `output` ONLY.\n%sCensusing it here would classify every "+
				"build as `extra`, which is a confident wrong answer rather than an empty one. "+
				"Use the per-actor instruments (`task corpus-vs-oracle`, `task diag`) for "+
				"non-player tables.", pyRepr(stat), len(covered), extra)
		}

		type classed struct {
			build, class string
		}
		classes := []classed{}
		counts := map[string]int{}
		for _, b := range covered {
			expMap, gotMap := scOutputOf(blob[b], b), ours[b]
			_, hasExp := expMap[stat]
			_, hasGot := gotMap[stat]
			c := scClassify(scGradedValue(expMap, stat, b), gotMap[stat], hasExp, hasGot)
			if c != "" {
				classes = append(classes, classed{b, c})
				counts[c]++
			}
		}

		if args.selfCheck {
			parts := make([]string, 0, 4)
			for _, k := range []string{"ok", "wrong", "miss", "extra"} {
				parts = append(parts, fmt.Sprintf("%s %d", k, counts[k]))
			}
			fmt.Printf("%s: %s\n", stat, strings.Join(parts, " · "))
			continue
		}

		fails, passes := []string{}, []string{}
		for _, c := range classes {
			switch c.class {
			case args.failClass:
				fails = append(fails, c.build)
			case "ok":
				passes = append(passes, c.build)
			}
		}
		fmt.Printf("=== %s ===\n", stat)
		fmt.Printf("ok %d · wrong %d · miss %d · extra %d\n",
			counts["ok"], counts["wrong"], counts["miss"], counts["extra"])
		if len(fails) == 0 || len(passes) == 0 {
			fmt.Printf("  (cannot census: %d failing and %d passing builds — an enrichment needs "+
				"both sides)\n\n", len(fails), len(passes))
			continue
		}

		pobValue := func(b string) float64 { return scGradedValue(scOutputOf(blob[b], b), stat, b) }
		hi, lo := []string{}, []string{}
		ratios := []float64{}
		for _, b := range fails {
			pv := pobValue(b)
			switch {
			case ours[b][stat] > pv:
				hi = append(hi, b)
			case ours[b][stat] < pv:
				lo = append(lo, b)
			}
			if pv != 0 {
				ratios = append(ratios, ours[b][stat]/pv)
			}
		}
		sort.Float64s(ratios)
		med := math.NaN()
		if len(ratios) > 0 {
			med = ratios[len(ratios)/2]
		}
		fmt.Printf("  direction: ours HIGH on %d · ours LOW on %d · median ratio %s\n",
			len(hi), len(lo), pyFixed(med, 4))
		switch args.direction {
		case "high":
			fails = hi
		case "low":
			fails = lo
		}
		fmt.Printf("  censusing the %s side: %d failing builds\n",
			strings.ToUpper(args.direction), len(fails))
		if len(fails) == 0 {
			fmt.Print("  (that side is EMPTY — a measurement, not a failure)\n\n")
			continue
		}

		rowsF, rowsP := []int{}, []int{}
		carryF, carryP := map[string]int{}, map[string]int{}
		for _, g := range []struct {
			group []string
			rows  *[]int
			carry map[string]int
		}{{fails, &rowsF, carryF}, {passes, &rowsP, carryP}} {
			for _, b := range g.group {
				n, seen := scSourceKeys(dumpdir, b, stat, args.granular)
				*g.rows = append(*g.rows, n)
				for k := range seen {
					g.carry[k]++
				}
			}
		}
		sum := func(xs []int) float64 {
			t := 0
			for _, x := range xs {
				t += x
			}
			return float64(t)
		}
		meanF := sum(rowsF) / float64(len(rowsF))
		meanP := sum(rowsP) / float64(len(rowsP))
		if meanP == 0 {
			fmt.Printf("  (cannot census: the %d passing builds carry NO attributed source rows\n"+
				"   for %s, so the complexity baseline is undefined. PoB attributes nothing\n"+
				"   here, which is itself the finding — see the module docstring)\n\n",
				len(passes), stat)
			continue
		}
		baseline := meanF / meanP
		fmt.Printf("  source rows: FAIL mean %s · PASS mean %s · BASELINE %sx  <- every raw ratio "+
			"below must beat THIS, not 1.0\n\n", pyFixed(meanF, 2), pyFixed(meanP, 2),
			pyFixed(baseline, 2))

		union := map[string]bool{}
		for k := range carryF {
			union[k] = true
		}
		for k := range carryP {
			union[k] = true
		}
		scored := []scRow{}
		for cls := range union {
			nf, np := carryF[cls], carryP[cls]
			if nf < args.minSupport {
				continue
			}
			pf := float64(nf) / float64(len(fails))
			pp := float64(np) / float64(len(passes))
			raw := math.Inf(1)
			if pp != 0 {
				raw = pf / pp
			}
			norm := math.Inf(1)
			if baseline != 0 {
				norm = raw / baseline
			}
			scored = append(scored, scRow{norm, raw, pf, pp, nf, np, cls})
		}
		sort.Slice(scored, func(i, j int) bool {
			a, b := scored[i], scored[j]
			if a.norm != b.norm {
				return a.norm > b.norm
			}
			if a.nf != b.nf {
				return a.nf > b.nf
			}
			return a.cls < b.cls
		})

		fmt.Printf("  %s %s %s %s %s %s   CLASS\n", scPad("NORM", 7), scPad("raw", 7),
			scPad("%fail", 7), scPad("%pass", 7), scPad("nf", 5), scPad("np", 5))
		for _, r := range scored[:min(len(scored), args.top)] {
			n := "INF"
			if !math.IsInf(r.norm, 1) {
				n = pyFixed(r.norm, 2)
			}
			raw := "INF"
			if !math.IsInf(r.raw, 1) {
				raw = pyFixed(r.raw, 2)
			}
			fmt.Printf("  %s %s %s%% %s%% %s %s   %s\n", scPad(n, 7), scPad(raw, 7),
				scPad(pyFixed(r.pf*100, 1), 6), scPad(pyFixed(r.pp*100, 1), 6),
				scPad(strconv.Itoa(r.nf), 5), scPad(strconv.Itoa(r.np), 5), r.cls)
		}
		if len(scored) == 0 {
			fmt.Printf("  (no class reaches --min-support %d failing builds)\n", args.minSupport)
		}
		fmt.Print("\n  READ IT: NORM > 1 means enriched BEYOND the complexity confound. NORM near or\n" +
			"  below 1 is the confound itself, however large `raw` looks. NORM = INF is a class\n" +
			"  carried by failing builds and by NO passing build — the cleanest signal here.\n" +
			"  A flat table is EVIDENCE OF INVENTION: this can only find a source PoB HAS.\n\n")
	}
	return 0
}
