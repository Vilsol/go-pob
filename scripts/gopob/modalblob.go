package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

const mbUsage = "usage: modal_blob.py <out.jsonl> <report.txt> <in1.jsonl> <in2.jsonl> ... " +
	"[--min-fraction=F]"

var mbTables = []string{
	"output", "outputMainHand", "outputOffHand", "enemyOutput",
	"minionOutput", "minionOutputMainHand", "minionOutputOffHand",
}

const (
	mbRelTol = 1e-9
	mbAbsTol = 1e-12
)

func mbRefuse(msg string) int {
	fmt.Fprintf(os.Stderr, "modal: %s\n", msg)
	return 1
}

type mbObs struct {
	absent bool
	v      float64
}

func mbClose(a, b float64) bool {
	if a == b {
		return true
	}
	d := math.Abs(a - b)
	if d <= mbAbsTol {
		return true
	}
	m := math.Max(math.Abs(a), math.Abs(b))
	return m > 0 && d/m <= mbRelTol
}

func mbSameObs(a, b mbObs) bool {
	if a.absent || b.absent {
		return a.absent && b.absent
	}
	return a.v == b.v
}

func mbVote(values []mbObs, n int, minFraction float64) (mbObs, bool, int, int) {
	unanimous := true
	for _, v := range values[1:] {
		if !mbSameObs(v, values[0]) {
			unanimous = false
			break
		}
	}
	if unanimous {
		return values[0], true, len(values), 1
	}

	type cluster struct {
		rep    mbObs
		values []float64
		n      int
	}
	var clusters []*cluster
	for _, v := range values {
		placed := false
		for _, c := range clusters {
			if v.absent {
				if c.rep.absent {
					c.n++
					placed = true
					break
				}
				continue
			}
			if !c.rep.absent && mbClose(c.rep.v, v.v) {
				c.values = append(c.values, v.v)
				c.n++
				placed = true
				break
			}
		}
		if !placed {
			c := &cluster{rep: v, n: 1}
			if !v.absent {
				c.values = append(c.values, v.v)
			}
			clusters = append(clusters, c)
		}
	}

	sort.SliceStable(clusters, func(i, j int) bool {
		if clusters[i].n != clusters[j].n {
			return clusters[i].n > clusters[j].n
		}
		return mbAbsentRank(clusters[i].rep) < mbAbsentRank(clusters[j].rep)
	})
	best := clusters[0]
	votes := best.n
	if float64(votes) <= minFraction*float64(n) {
		return mbObs{}, false, votes, len(clusters)
	}
	if best.rep.absent {
		return mbObs{absent: true}, true, votes, len(clusters)
	}
	counts := map[float64]int{}
	top := 0
	for _, v := range best.values {
		counts[v]++
		if counts[v] > top {
			top = counts[v]
		}
	}
	first := true
	var winner float64
	for v, c := range counts {
		if c == top && (first || v < winner) {
			winner, first = v, false
		}
	}
	return mbObs{v: winner}, true, votes, len(clusters)
}

func mbAbsentRank(o mbObs) int {
	if o.absent {
		return 0
	}
	return 1
}

type mbCounter struct {
	order []string
	count map[string]int
}

func newMBCounter() *mbCounter { return &mbCounter{count: map[string]int{}} }

func (c *mbCounter) add(k string) {
	if _, ok := c.count[k]; !ok {
		c.order = append(c.order, k)
	}
	c.count[k]++
}

func (c *mbCounter) len() int { return len(c.order) }

func (c *mbCounter) mostCommon(n int) [][2]any {
	keys := append([]string(nil), c.order...)
	sort.SliceStable(keys, func(i, j int) bool { return c.count[keys[i]] > c.count[keys[j]] })
	if n > 0 && len(keys) > n {
		keys = keys[:n]
	}
	out := make([][2]any, len(keys))
	for i, k := range keys {
		out[i] = [2]any{k, c.count[k]}
	}
	return out
}

func modalBlob(args []string) int {
	minFraction := 0.5
	minFractionRaw := "0.5"
	var pos []string
	for _, a := range args {
		if strings.HasPrefix(a, "--min-fraction=") {
			raw := strings.SplitN(a, "=", 2)[1]
			f, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return mbRefuse(fmt.Sprintf("--min-fraction must be a number, not %s", pyRepr(raw)))
			}
			minFraction, minFractionRaw = f, ""
		} else {
			pos = append(pos, a)
		}
	}
	if len(pos) < 5 {
		fmt.Fprintln(os.Stderr, mbUsage)
		return 1
	}
	outPath, reportPath := pos[0], pos[1]
	inputs := pos[2:]
	n := len(inputs)
	if n < 3 {
		return mbRefuse(fmt.Sprintf("modal_blob: need at least 3 runs to have a mode; got %d", n))
	}

	readers := make([]*bufio.Reader, n)
	for i, path := range inputs {
		fh, err := os.Open(path) //nolint:gosec // an operator-supplied path, as in the original
		if err != nil {
			return mbRefuse(fmt.Sprintf("%s is not readable: %s", path, errnoText(err)))
		}
		defer fh.Close() //nolint:errcheck // read-only
		readers[i] = bufio.NewReaderSize(fh, 1<<20)
	}

	out, err := os.Create(outPath) //nolint:gosec // an operator-supplied path
	if err != nil {
		return mbRefuse(fmt.Sprintf("cannot write %s: %s", outPath, errnoText(err)))
	}
	defer out.Close() //nolint:errcheck // the flush below is what can fail meaningfully
	ow := bufio.NewWriterSize(out, 1<<20)

	st := &mbState{
		n:            n,
		runDiff:      make([]int, n),
		droppedByKey: newMBCounter(), droppedByBuild: newMBCounter(),
		splitByKey: newMBCounter(), splitByBuild: newMBCounter(),
		splitHist: map[int]int{},
	}

	for {
		lines := make([]string, n)
		anyLine, allLines := false, true
		for i, r := range readers {
			line, err := r.ReadString('\n')
			if err != nil && line == "" {
				lines[i] = ""
			} else {
				lines[i] = line
			}
			if strings.TrimSpace(lines[i]) != "" {
				anyLine = true
			} else {
				allLines = false
			}
		}
		if !anyLine {
			break
		}
		if !allLines {
			return mbRefuse("runs have different record COUNTS — inputs are not comparable")
		}
		present := make([]map[string]any, n)
		for i, line := range lines {
			dec := json.NewDecoder(strings.NewReader(line))
			dec.UseNumber()
			var v any
			if err := dec.Decode(&v); err != nil {
				return mbRefuse(fmt.Sprintf("%s: line %d is not valid JSON", inputs[i], st.nbuilds+1))
			}
			m, ok := v.(map[string]any)
			if !ok {
				return mbRefuse(fmt.Sprintf("%s: line %d has no build name", inputs[i], st.nbuilds+1))
			}
			if _, ok := m["build"]; !ok {
				return mbRefuse(fmt.Sprintf("%s: line %d has no build name", inputs[i], st.nbuilds+1))
			}
			present[i] = m
		}
		nameSet := map[string]bool{}
		for _, p := range present {
			if s, ok := p["build"].(string); ok {
				nameSet[s] = true
			} else {
				nameSet[pyReprValue(p["build"])] = true
			}
		}
		names := make([]string, 0, len(nameSet))
		for s := range nameSet {
			names = append(names, s)
		}
		sort.Strings(names)
		if len(names) != 1 {
			return mbRefuse(fmt.Sprintf("runs desynchronised at %s — inputs are not comparable",
				pyReprValue(names)))
		}
		build := names[0]
		st.nbuilds++
		if rc := st.emit(ow, present, build, minFraction); rc != 0 {
			return rc
		}
	}
	if err := ow.Flush(); err != nil {
		return mbRefuse(fmt.Sprintf("cannot write %s: %s", outPath, errnoText(err)))
	}

	if rc := st.writeReport(reportPath, inputs, minFraction, minFractionRaw); rc != 0 {
		return rc
	}
	fmt.Fprintf(os.Stderr, "modal: %d builds, %d pairs, %d disagreeing, %d dropped -> %s\n",
		st.nbuilds, st.totalKeys, st.splitKeys, st.droppedKeys, outPath)
	return 0
}

type mbTreeRow struct {
	build string
	hist  string
}

type mbState struct {
	n              int
	nbuilds        int
	totalKeys      int
	splitKeys      int
	droppedKeys    int
	tableDisagree  int
	runDiff        []int
	droppedByKey   *mbCounter
	droppedByBuild *mbCounter
	splitByKey     *mbCounter
	splitByBuild   *mbCounter
	splitHist      map[int]int
	treeDisagree   []mbTreeRow
}

func mbNum(v any) (float64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	f, err := n.Float64()
	return f, err == nil
}

func (st *mbState) emit(ow *bufio.Writer, present []map[string]any, build string, minFraction float64) int {
	n := st.n
	rec := map[string]any{"build": build}

	tsVals := make([]mbObs, n)
	tsToks := make([]string, n)
	for i, p := range present {
		v, ok := p["treeSources"]
		if !ok {
			v = json.Number("0")
		}
		f, isNum := mbNum(v)
		tsVals[i] = mbObs{v: f}
		tsToks[i] = pyReprValue(v)
		if !isNum {
			return mbRefuse(fmt.Sprintf("%s: treeSources is not a number", build))
		}
	}
	ts, _, _, tsClusters := mbVote(tsVals, n, 0.0)
	if tsClusters > 1 {
		st.treeDisagree = append(st.treeDisagree, mbTreeRow{build, mbHist(tsVals, tsToks)})
	}
	if ts.absent {
		rec["treeSources"] = 0
	} else {
		rec["treeSources"] = int(ts.v)
	}

	for _, p := range present {
		if e, ok := p["error"]; ok && eooTruthy(e) {
			rec["error"] = e
			rec["output"] = map[string]float64{}
			var b strings.Builder
			pyDumps(rec, &b)
			ow.WriteString(b.String() + "\n") //nolint:errcheck // flushed and checked by the caller
			return 0
		}
	}

	for _, table := range mbTables {
		have := make([]map[string]any, n)
		presentCount := 0
		for i, p := range present {
			if v, ok := p[table]; ok && v != nil {
				if m, isMap := v.(map[string]any); isMap {
					have[i] = m
					presentCount++
				}
			}
		}
		if presentCount == 0 {
			continue
		}
		if presentCount != n {
			st.tableDisagree++
			if float64(presentCount) <= minFraction*float64(n) {
				continue
			}
		}
		keySet := map[string]bool{}
		for _, h := range have {
			for k := range h {
				keySet[k] = true
			}
		}
		keys := make([]string, 0, len(keySet))
		for k := range keySet {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		merged := map[string]float64{}
		for _, k := range keys {
			st.totalKeys++
			vals := make([]mbObs, n)
			for i, h := range have {
				raw, ok := h[k]
				if h == nil || !ok {
					vals[i] = mbObs{absent: true}
					continue
				}
				f, isNum := mbNum(raw)
				if !isNum {
					return mbRefuse(fmt.Sprintf("%s.%s.%s is not a number", build, table, k))
				}
				vals[i] = mbObs{v: f}
			}
			winner, hasWinner, votes, clusters := mbVote(vals, n, minFraction)
			if hasWinner {
				for ri, v := range vals {
					same := (v.absent && winner.absent) ||
						(!v.absent && !winner.absent && mbClose(v.v, winner.v))
					if !same {
						st.runDiff[ri]++
					}
				}
			}
			if clusters > 1 {
				st.splitKeys++
				st.splitByBuild.add(build)
				st.splitByKey.add(table + ":" + k)
				st.splitHist[votes]++
			}
			if !hasWinner || winner.absent {
				if !hasWinner {
					st.droppedKeys++
					st.droppedByKey.add(table + ":" + k)
					st.droppedByBuild.add(build)
				}
				continue
			}
			merged[k] = winner.v
		}
		rec[table] = merged
	}

	var b strings.Builder
	pyDumps(rec, &b)
	ow.WriteString(b.String() + "\n") //nolint:errcheck // flushed and checked by the caller
	return 0
}

func mbHist(vals []mbObs, toks []string) string {
	counts := map[float64]int{}
	tok := map[float64]string{}
	var order []float64
	for i, v := range vals {
		if v.absent {
			continue
		}
		if _, seen := counts[v.v]; !seen {
			order = append(order, v.v)
			tok[v.v] = toks[i]
		}
		counts[v.v]++
	}
	sort.Float64s(order)
	parts := make([]string, len(order))
	for i, v := range order {
		parts[i] = fmt.Sprintf("(%s, %d)", tok[v], counts[v])
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (st *mbState) writeReport(path string, inputs []string, minFraction float64, _ string) int {
	rep, err := os.Create(path) //nolint:gosec // an operator-supplied path
	if err != nil {
		return mbRefuse(fmt.Sprintf("cannot write %s: %s", path, errnoText(err)))
	}
	defer rep.Close() //nolint:errcheck // the flush below is what can fail meaningfully
	w := bufio.NewWriter(rep)
	n := st.n

	fmt.Fprintf(w, "MODAL ORACLE REPORT\n")
	fmt.Fprintf(w, "runs                 %d\n", n)
	for _, p := range inputs {
		fmt.Fprintf(w, "  %s\n", p)
	}
	fmt.Fprintf(w, "min-fraction         %s (strict majority when 0.5)\n", pyFloat(minFraction))
	fmt.Fprintf(w, "builds               %d\n", st.nbuilds)
	fmt.Fprintf(w, "(build,key) pairs    %d  seen in at least one run\n", st.totalKeys)
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "DISAGREEMENT — the term a single-process blob samples blind\n")
	fmt.Fprintf(w, "  keys where runs disagreed   %d  (%.4f%% of pairs)\n",
		st.splitKeys, 100.0*float64(st.splitKeys)/float64(max(st.totalKeys, 1)))
	fmt.Fprintf(w, "  builds carrying any of them %d\n", st.splitByBuild.len())
	fmt.Fprintf(w, "  distinct table:key          %d\n", st.splitByKey.len())
	fmt.Fprintf(w, "  winning-vote histogram      %s\n", mbHistPairs(st.splitHist))
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "PER-RUN DISTANCE FROM THE MODE — what ONE process's blob would have graded wrongly\n")
	sum, lo, hi := 0, st.runDiff[0], st.runDiff[0]
	for i, d := range st.runDiff {
		fmt.Fprintf(w, "  run %-2d  %6d pairs differ from the mode   %s\n", i+1, d, inputs[i])
		sum += d
		lo, hi = min(lo, d), max(hi, d)
	}
	fmt.Fprintf(w, "  mean %.1f, min %d, max %d\n", float64(sum)/float64(n), lo, hi)
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "DROPPED — no cluster reached the threshold, so the blob does not grade them\n")
	fmt.Fprintf(w, "  pairs   %d\n", st.droppedKeys)
	fmt.Fprintf(w, "  builds  %d\n", st.droppedByBuild.len())
	fmt.Fprintf(w, "  keys    %d\n", st.droppedByKey.len())
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "  top dropped keys:\n")
	mbTop(w, st.droppedByKey, 30)
	fmt.Fprintf(w, "  top dropped builds:\n")
	mbTop(w, st.droppedByBuild, 20)
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "  top disagreeing keys:\n")
	mbTop(w, st.splitByKey, 30)
	fmt.Fprintf(w, "  top disagreeing builds:\n")
	mbTop(w, st.splitByBuild, 25)
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "STRUCTURAL\n")
	fmt.Fprintf(w, "  builds whose treeSources differed between runs   %d\n", len(st.treeDisagree))
	rows := st.treeDisagree
	if len(rows) > 20 {
		rows = rows[:20]
	}
	for _, r := range rows {
		fmt.Fprintf(w, "    %s  %s\n", r.build, r.hist)
	}
	fmt.Fprintf(w, "  (build,table) present in some runs and not others %d\n", st.tableDisagree)
	if err := w.Flush(); err != nil {
		return mbRefuse(fmt.Sprintf("cannot write %s: %s", path, errnoText(err)))
	}
	return 0
}

func mbTop(w *bufio.Writer, c *mbCounter, n int) {
	for _, kv := range c.mostCommon(n) {
		fmt.Fprintf(w, "    %6d  %s\n", kv[1], kv[0])
	}
}

func mbHistPairs(h map[int]int) string {
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("(%d, %d)", k, h[k])
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
