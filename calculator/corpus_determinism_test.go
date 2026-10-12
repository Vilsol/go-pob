package calculator

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

type runDumpMeta struct {
	dataVersion string
	corpusDir   string
}

const determinismRuns = 8

const determinismJobs = 1

type runRecord struct {
	build  string
	status string
	output map[string]float64
}

// TestCorpusIsDeterministic is the DETERMINISM GATE.
func TestCorpusIsDeterministic(t *testing.T) {
	requireOperatorRun(t, "cross-process determinism, 110.8s; runs at integrate")
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}

	bin := os.Args[0]
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("cannot re-exec the test binary (%s): %s", bin, err)
	}

	runs := determinismRuns
	if v := os.Getenv("RUNS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 1 {
			runs = parsed
		}
	}

	jobs := determinismJobs
	if v := os.Getenv("JOBS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			jobs = parsed
		}
	}
	if jobs > runs {
		jobs = runs
	}

	dir := t.TempDir()
	loaded := make([]labelledRun, runs)
	errs := make([]error, runs)

	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs)
	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			path := filepath.Join(dir, fmt.Sprintf("run-%d.txt.gz", i+1))

			cmd := exec.CommandContext(t.Context(), bin, "-test.run=^TestCorpusDump$")
			cmd.Env = append(os.Environ(), "DUMP="+path)
			if out, err := cmd.CombinedOutput(); err != nil {
				errs[i] = fmt.Errorf("run %d failed: %w\n%s", i+1, err, out)
				return
			}

			recs, _, err := readRunDump(path)
			if err != nil {
				errs[i] = fmt.Errorf("run %d: %w", i+1, err)
				return
			}
			loaded[i] = newLabelledRun(fmt.Sprintf("run%d", i+1), recs)
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	deltas := diffRuns(loaded)
	printRunDiff("CORPUS DETERMINISM", loaded, deltas)

	unstable := materialDeltas(deltas)
	if len(unstable) == 0 {
		return
	}

	names := make([]string, 0, len(unstable))
	for _, d := range unstable {
		names = append(names, d.build)
	}
	t.Errorf("%d of %d builds are NONDETERMINISTIC across %d processes: %s",
		len(unstable), len(loaded[0].order), len(loaded), strings.Join(names, " "))
}

// TestCorpusDump writes this process's per-build corpus output to DUMP=<path>, gzipped when the path ends in .gz.
func TestCorpusDump(t *testing.T) {
	path := os.Getenv("DUMP")
	if path == "" {
		t.SkipNow()
	}

	corpusDir, names := corpusNamesForLoadedData(t)
	meta := runDumpMeta{dataVersion: raw.LatestVersion, corpusDir: corpusDir}
	recs := evaluateCorpusRun(corpusDir, names)
	if err := writeRunDump(path, meta, recs); err != nil {
		t.Fatal(err)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	fmt.Printf("wrote %d build records (%s, game data %s) to %s\n",
		len(recs), corpusDir, raw.LatestVersion, abs)
}

// TestCorpusDiff compares two dumps written by TestCorpusDump and reports which builds moved and which stats moved in them.
func TestCorpusDiff(t *testing.T) {
	a, b := os.Getenv("DIFF_A"), os.Getenv("DIFF_B")
	if a == "" || b == "" {
		t.SkipNow()
	}

	recsA, metaA, err := readRunDump(a)
	if err != nil {
		t.Fatal(err)
	}
	recsB, metaB, err := readRunDump(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkDumpsComparable(a, metaA, b, metaB); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("both dumps: %s corpus %s\n", metaA.dataVersion, metaA.corpusDir)

	runs := []labelledRun{
		newLabelledRun(dumpLabel(a), recsA),
		newLabelledRun(dumpLabel(b), recsB),
	}
	printRunDiff("CORPUS RUN DIFF", runs, diffRuns(runs))
}

func dumpLabel(path string) string {
	name := filepath.Base(path)
	name = strings.TrimSuffix(name, ".gz")
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func checkDumpsComparable(pathA string, a runDumpMeta, pathB string, b runDumpMeta) error {
	describe := func(path string, m runDumpMeta) string {
		if m.dataVersion == "" {
			return fmt.Sprintf("%s: UNVERSIONED (written before dumps recorded their version)", path)
		}
		return fmt.Sprintf("%s: game data %s, corpus %s", path, m.dataVersion, m.corpusDir)
	}
	if a.dataVersion == "" || b.dataVersion == "" || a.dataVersion != b.dataVersion {
		return fmt.Errorf("REFUSING to diff dumps that are not from the same game-data version:\n"+
			"  %s\n  %s\n"+
			"Diffing across versions reports EVERY build as moved — a structured, confident, false\n"+
			"answer. Re-dump both sides with the same POE_VERSION.",
			describe(pathA, a), describe(pathB, b))
	}
	return nil
}

func evaluateCorpusRun(corpusDir string, names []string) []runRecord {
	recs := make([]runRecord, len(names))

	runConcurrently(len(names), evalWorkers(), func(i int, cache *EnvironmentCache) {
		recs[i].build = names[i]

		file, err := os.ReadFile(filepath.Join(corpusDir, names[i]))
		if err != nil {
			recs[i].status = "read-error"
			return
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			recs[i].status = "parse-error"
			return
		}

		var env *Environment
		func() {
			defer func() {
				if r := recover(); r != nil {
					recs[i].status = fmt.Sprintf("panic: %v", r)
				}
			}()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		if env == nil {
			if recs[i].status == "" {
				recs[i].status = "nil-env"
			}
			return
		}

		out := make(map[string]float64, len(env.Player.Output)+256)
		for k, v := range env.Player.Output {
			out[k] = v
		}
		for table, m := range env.Player.OutputTable {
			for k, v := range m {
				out[string(table)+"."+k] = v
			}
		}
		recs[i].output = out
	})

	return recs
}

func writeRunDump(path string, meta runDumpMeta, recs []runRecord) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	var closers []io.Closer
	var sink io.Writer = f
	if strings.HasSuffix(path, ".gz") {
		gz, gzErr := gzip.NewWriterLevel(f, gzip.BestSpeed)
		if gzErr != nil {
			return fmt.Errorf("gzip %s: %w", path, gzErr)
		}
		closers = append(closers, gz)
		sink = gz
	}

	w := bufio.NewWriterSize(sink, 1<<20)
	fmt.Fprintf(w, "V\tdataVersion\t%s\n", meta.dataVersion)
	fmt.Fprintf(w, "V\tcorpusDir\t%s\n", meta.corpusDir)
	keys := make([]string, 0, 2048)
	line := make([]byte, 0, 256)
	for _, rec := range recs {
		fmt.Fprintf(w, "B\t%s\t%s\n", rec.build, rec.status)

		keys = keys[:0]
		for k := range rec.output {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			line = append(line[:0], "S\t"...)
			line = append(line, k...)
			line = append(line, '\t')
			line = strconv.AppendFloat(line, rec.output[k], 'g', 17, 64)
			line = append(line, '\n')
			_, _ = w.Write(line)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	for _, c := range closers {
		if err := c.Close(); err != nil {
			return fmt.Errorf("close %s: %w", path, err)
		}
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	return nil
}

func readRunDump(path string) ([]runRecord, runDumpMeta, error) {
	var meta runDumpMeta

	f, err := os.Open(path)
	if err != nil {
		return nil, meta, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var src io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, gzErr := gzip.NewReader(f)
		if gzErr != nil {
			return nil, meta, fmt.Errorf("%s: not gzip: %w", path, gzErr)
		}
		defer gz.Close()
		src = gz
	}

	recs := make([]runRecord, 0, 2048)
	cur := -1

	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "V\t"):
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) != 3 {
				return nil, meta, fmt.Errorf("%s: malformed header line %q", path, line)
			}
			switch parts[1] {
			case "dataVersion":
				meta.dataVersion = parts[2]
			case "corpusDir":
				meta.corpusDir = parts[2]
			default:
			}
		case strings.HasPrefix(line, "B\t"):
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) != 3 {
				return nil, meta, fmt.Errorf("%s: malformed build line %q", path, line)
			}
			recs = append(recs, runRecord{build: parts[1], status: parts[2], output: make(map[string]float64, 1024)})
			cur = len(recs) - 1
		case strings.HasPrefix(line, "S\t"):
			if cur < 0 {
				return nil, meta, fmt.Errorf("%s: stat line before any build line", path)
			}
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) != 3 {
				return nil, meta, fmt.Errorf("%s: malformed stat line %q", path, line)
			}
			v, parseErr := strconv.ParseFloat(parts[2], 64)
			if parseErr != nil {
				return nil, meta, fmt.Errorf("%s: %q: %w", path, line, parseErr)
			}
			recs[cur].output[parts[1]] = v
		default:
			return nil, meta, fmt.Errorf("%s: unrecognised line %q", path, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, meta, fmt.Errorf("read %s: %w", path, err)
	}
	if len(recs) == 0 {
		return nil, meta, fmt.Errorf("%s: empty dump", path)
	}
	return recs, meta, nil
}

type labelledRun struct {
	label   string
	order   []string
	byBuild map[string]*runRecord
}

func newLabelledRun(label string, recs []runRecord) labelledRun {
	run := labelledRun{
		label:   label,
		order:   make([]string, 0, len(recs)),
		byBuild: make(map[string]*runRecord, len(recs)),
	}
	for i := range recs {
		run.order = append(run.order, recs[i].build)
		run.byBuild[recs[i].build] = &recs[i]
	}
	return run
}

type statValue struct {
	value   float64
	present bool
}

type statDelta struct {
	stat     string
	values   []statValue
	material bool
}

type buildDelta struct {
	build      string
	statuses   []string
	statusDiff bool
	stats      []statDelta
	material   bool
}

const runDiffMaterialRel = 1e-9

func sameFloat(a, b float64) bool {
	return a == b || (math.IsNaN(a) && math.IsNaN(b))
}

func materialValues(values []statValue) bool {
	ref := values[0]
	for _, v := range values[1:] {
		switch {
		case v.present != ref.present:
			return true
		case !v.present:
			continue
		case sameFloat(ref.value, v.value):
			continue
		case math.IsNaN(ref.value) != math.IsNaN(v.value):
			return true
		case math.IsInf(ref.value, 0) || math.IsInf(v.value, 0):
			return true
		}

		scale := math.Max(math.Abs(ref.value), math.Abs(v.value))
		if scale == 0 || math.Abs(ref.value-v.value)/scale > runDiffMaterialRel {
			return true
		}
	}
	return false
}

func diffRuns(runs []labelledRun) []buildDelta {
	if len(runs) < 2 {
		return nil
	}

	names := append([]string(nil), runs[0].order...)
	seen := make(map[string]bool, len(names))
	for _, n := range names {
		seen[n] = true
	}
	for _, run := range runs[1:] {
		for _, n := range run.order {
			if !seen[n] {
				seen[n] = true
				names = append(names, n)
			}
		}
	}

	deltas := make([]buildDelta, 0, 16)
	for _, name := range names {
		recs := make([]*runRecord, len(runs))
		for i, run := range runs {
			recs[i] = run.byBuild[name]
		}

		delta := buildDelta{build: name, statuses: make([]string, len(runs))}
		for i, rec := range recs {
			if rec == nil {
				delta.statuses[i] = "<absent>"
				continue
			}
			delta.statuses[i] = rec.status
			if delta.statuses[i] == "" {
				delta.statuses[i] = "ok"
			}
		}
		for i := 1; i < len(delta.statuses); i++ {
			if delta.statuses[i] != delta.statuses[0] {
				delta.statusDiff = true
			}
		}

		stats := make([]string, 0, 1024)
		statSeen := make(map[string]bool, 1024)
		for _, rec := range recs {
			if rec == nil {
				continue
			}
			for k := range rec.output {
				if !statSeen[k] {
					statSeen[k] = true
					stats = append(stats, k)
				}
			}
		}
		sort.Strings(stats)

		for _, stat := range stats {
			values := make([]statValue, len(recs))
			for i, rec := range recs {
				if rec == nil {
					continue
				}
				v, ok := rec.output[stat]
				values[i] = statValue{value: v, present: ok}
			}

			differs := false
			for i := 1; i < len(values); i++ {
				if values[i].present != values[0].present || !sameFloat(values[i].value, values[0].value) {
					differs = true
					break
				}
			}
			if differs {
				material := materialValues(values)
				delta.material = delta.material || material
				delta.stats = append(delta.stats, statDelta{stat: stat, values: values, material: material})
			}
		}
		delta.material = delta.material || delta.statusDiff

		if delta.statusDiff || len(delta.stats) > 0 {
			deltas = append(deltas, delta)
		}
	}

	return deltas
}

func (d buildDelta) materialStats() []statDelta {
	out := make([]statDelta, 0, len(d.stats))
	for _, s := range d.stats {
		if s.material {
			out = append(out, s)
		}
	}
	return out
}

func materialDeltas(deltas []buildDelta) []buildDelta {
	out := make([]buildDelta, 0, len(deltas))
	for _, d := range deltas {
		if d.material {
			out = append(out, d)
		}
	}
	return out
}

func renderValues(runs []labelledRun, values []statValue) string {
	render := func(v statValue) string {
		if !v.present {
			return "<absent>"
		}
		return strconv.FormatFloat(v.value, 'g', 17, 64)
	}

	if len(values) <= 4 {
		parts := make([]string, len(values))
		for i, v := range values {
			parts[i] = runs[i].label + "=" + render(v)
		}
		return strings.Join(parts, "  ")
	}

	order := make([]string, 0, 4)
	count := map[string]int{}
	first := map[string]string{}
	for i, v := range values {
		key := render(v)
		if _, seen := count[key]; !seen {
			order = append(order, key)
			first[key] = runs[i].label
		}
		count[key]++
	}

	parts := make([]string, len(order))
	for i, key := range order {
		parts[i] = fmt.Sprintf("%s x%d (%s)", key, count[key], first[key])
	}
	return strings.Join(parts, "  ")
}

func summarise(names []string) string {
	if len(names) == 0 {
		return ""
	}
	if len(names) > runDiffMaxBuilds {
		return fmt.Sprintf(" [%s ... +%d]", strings.Join(names[:runDiffMaxBuilds], " "), len(names)-runDiffMaxBuilds)
	}
	return fmt.Sprintf(" [%s]", strings.Join(names, " "))
}

const (
	runDiffMaxBuilds = 20
	runDiffMaxStats  = 12
)

func printRunDiff(title string, runs []labelledRun, deltas []buildDelta) {
	labels := make([]string, len(runs))
	for i, run := range runs {
		labels[i] = run.label
	}

	fmt.Printf("\n=== %s: %s ===\n", title, strings.Join(labels, " vs "))
	for _, run := range runs {
		fmt.Printf("%-12s %d builds\n", run.label, len(run.order))
	}

	if len(deltas) == 0 {
		fmt.Printf("IDENTICAL: every build agrees on every stat\n")
		return
	}

	material := make([]buildDelta, 0, len(deltas))
	noise := make([]string, 0, len(deltas))
	for _, d := range deltas {
		if d.material {
			material = append(material, d)
			continue
		}
		noise = append(noise, d.build)
	}

	fmt.Printf("MATERIAL (rel > %g): %d builds\n", runDiffMaterialRel, len(material))
	fmt.Printf("FLOAT REASSOCIATION ONLY: %d builds%s\n", len(noise), summarise(noise))

	if len(material) == 0 {
		return
	}
	deltas = material

	statBuilds := map[string]int{}
	for _, d := range deltas {
		for _, s := range d.stats {
			if s.material {
				statBuilds[s.stat]++
			}
		}
	}
	ranked := make([]string, 0, len(statBuilds))
	for stat := range statBuilds {
		ranked = append(ranked, stat)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if statBuilds[ranked[i]] != statBuilds[ranked[j]] {
			return statBuilds[ranked[i]] > statBuilds[ranked[j]]
		}
		return ranked[i] < ranked[j]
	})

	fmt.Printf("\nBUILDS (%d):\n", len(deltas))
	for _, d := range deltas {
		status := ""
		if d.statusDiff {
			status = fmt.Sprintf("  status %s", strings.Join(d.statuses, " | "))
		}
		fmt.Printf("  %-44s %5d stats differ%s\n", d.build, len(d.materialStats()), status)
	}

	fmt.Printf("\nSTATS (%d distinct, ranked by builds affected):\n", len(ranked))
	for i, stat := range ranked {
		if i >= runDiffMaxStats*4 {
			fmt.Printf("  ... %d more\n", len(ranked)-i)
			break
		}
		fmt.Printf("  %-52s %5d builds\n", stat, statBuilds[stat])
	}

	detailCap := runDiffMaxBuilds
	if os.Getenv("DETAIL") == "all" {
		detailCap = len(deltas)
	}
	fmt.Printf("\nDETAIL (%d of %d builds; DETAIL=all for every one):\n",
		min(detailCap, len(deltas)), len(deltas))
	for i, d := range deltas {
		if i >= detailCap {
			omitted := make([]string, 0, len(deltas)-i)
			for _, rest := range deltas[i:] {
				omitted = append(omitted, rest.build)
			}
			fmt.Printf("  ... %d more builds TRUNCATED (they are in BUILDS above): %s\n",
				len(omitted), strings.Join(omitted, " "))
			break
		}
		fmt.Printf("  %s\n", d.build)
		if d.statusDiff {
			for j, run := range runs {
				fmt.Printf("      status %-10s %s\n", run.label, d.statuses[j])
			}
		}
		stats := d.materialStats()
		for j, s := range stats {
			if j >= runDiffMaxStats {
				omitted := make([]string, 0, len(stats)-j)
				for _, rest := range stats[j:] {
					omitted = append(omitted, rest.stat)
				}
				fmt.Printf("      ... %d more stats TRUNCATED: %s\n",
					len(omitted), strings.Join(omitted, " "))
				break
			}
			fmt.Printf("      %-44s %s\n", s.stat, renderValues(runs, s.values))
		}
	}
}
