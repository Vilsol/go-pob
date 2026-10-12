package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const insLogSampleDefault = 200

type insRow struct {
	slug, label, status, head string
	secs                      int
}

type insRun struct {
	repo, rundir, dumps string
	only, coverage      string
	timings             string
	logSample           string
	rows                []insRow
	headOf              map[string]string
	failures, notruns   []string
	invariants          []string
	invariantBad        bool
}

func insLines(b []byte) []string {
	s := string(b)
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}

func insReadLines(file string) []string {
	b, err := os.ReadFile(file) //nolint:gosec // a path this tool just wrote
	if err != nil {
		return nil
	}
	return insLines(b)
}

func insGrepFirst(rx *regexp.Regexp, file string) string {
	for _, ln := range insReadLines(file) {
		if m := rx.FindAllString(ln, -1); m != nil {
			return strings.Join(m, "\n")
		}
	}
	return ""
}

func insGrepAllLines(rx *regexp.Regexp, file string) []string {
	lines := insReadLines(file)
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		out = append(out, rx.FindAllString(ln, -1)...)
	}
	return out
}

func insGrepCountFixed(rx *regexp.Regexp, file string) int {
	n := 0
	for _, ln := range insReadLines(file) {
		if rx.MatchString(ln) {
			n++
		}
	}
	return n
}

func insGrepQ(rx *regexp.Regexp, file string) bool { return insGrepCountFixed(rx, file) > 0 }

func insMatchIn(rx *regexp.Regexp, s string) string {
	if m := rx.FindAllString(s, -1); m != nil {
		return strings.Join(m, "\n")
	}
	return ""
}

var insMultiSpace = regexp.MustCompile(` {2,}`)

func insSqueezeSpaces(s string) string { return insMultiSpace.ReplaceAllString(s, " ") }

func insNum(s string, n int) string {
	m := insDigits.FindAllString(s, -1)
	if n < 1 || n > len(m) {
		return ""
	}
	return m[n-1]
}

var insDigits = regexp.MustCompile(`[0-9]+`)

func insHasHeadline(s string) bool {
	return s != "" && !strings.HasPrefix(s, "(no headline matched")
}

func insIsDir(p string) bool {
	fi, err := os.Stat(p) //nolint:gosec // an operator-supplied path, as in the original
	return err == nil && fi.IsDir()
}

func insShim(repo, call string) (string, bool) {
	script := ". " + repo + "/scripts/gate_versions.sh"
	if call != "" {
		script += " && " + call
	}
	cmd := exec.Command("bash", "-c", script) //nolint:gosec,noctx // sourcing the shim is what the original does
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimRight(string(out), "\n"), true
}

var insLogRx = regexp.MustCompile(`^(?:\x1b\[[0-9;]*m)*(?:TRC|DBG|INF|WRN|ERR)(?:\x1b\[[0-9;]*m)*[[:space:]]`)

func insWithinCap(n int, capStr string) bool {
	if v, err := strconv.Atoi(capStr); err == nil {
		return n <= v
	}
	return strconv.Itoa(n) <= capStr
}

func (ir *insRun) insScrub(out []byte, logFile string) []byte {
	var clean, logs bytes.Buffer
	n := 0
	for _, ln := range insLines(out) {
		if insLogRx.MatchString(ln) {
			n++
			if insWithinCap(n, ir.logSample) {
				logs.WriteString(ln)
				logs.WriteByte('\n')
			}
			continue
		}
		clean.WriteString(ln)
		clean.WriteByte('\n')
	}
	if n > 0 {
		sampled := n
		if !insWithinCap(n, ir.logSample) {
			if v, err := strconv.Atoi(ir.logSample); err == nil {
				sampled = v
			}
		}
		fmt.Fprintf(&logs, "\n--- %d slog line(s) split off this run; %d sampled above.\n"+
			"--- The log statements are unchanged: re-run the command without\n"+
			"--- scripts/instruments.sh to see all of them.\n", n, sampled)
		_ = os.WriteFile(logFile, logs.Bytes(), 0o600)
	}
	return clean.Bytes()
}

func (ir *insRun) selected(slug string) bool {
	if ir.only == "" {
		return true
	}
	return strings.Contains(" "+ir.only+" ", " "+slug+" ")
}

func (ir *insRun) runInstrument(slug, label string, needsDumps bool, argv []string) {
	if !ir.selected(slug) {
		return
	}
	clean := filepath.Join(ir.rundir, slug+".txt")
	logs := filepath.Join(ir.rundir, slug+".logs.txt")

	t0 := time.Now()
	var status, head string

	if needsDumps && !insIsDir(ir.dumps) {
		status = "NOT-RUN"
		head = "DUMPS=" + ir.dumps + " does not exist — 'mise run dump-corpus DUMPS=...'"
		ir.notruns = append(ir.notruns, slug+": "+head)
		_ = os.WriteFile(clean, []byte(head+"\n"), 0o600)
	} else {
		fmt.Fprintf(os.Stderr, "\033[2K\r  ... %s", slug)

		cmd := exec.Command(argv[0], argv[1:]...) //nolint:gosec,noctx // the chain's own table
		var merged bytes.Buffer
		cmd.Stdout = &merged
		cmd.Stderr = &merged
		runErr := cmd.Run()
		rc := 0
		if runErr != nil {
			rc = 1
			var ee *exec.ExitError
			if ok := asExitError(runErr, &ee); ok {
				rc = ee.ExitCode()
			}
		}
		_ = os.WriteFile(clean, ir.insScrub(merged.Bytes(), logs), 0o600)

		fmt.Fprint(os.Stderr, "\033[2K\r")
		if rc == 0 {
			status = "PASS"
		} else {
			status = "FAIL"
		}
		head = ir.headlineFor(slug, clean)
		if status == "FAIL" {
			ir.failures = append(ir.failures, slug+": "+ir.failDetail(slug, clean))
		}
	}

	ir.rows = append(ir.rows, insRow{slug, label, status, head, int(time.Since(t0).Seconds())})
	ir.headOf[slug] = head
}

var (
	insRxFailLine     = regexp.MustCompile(`^--- FAIL: `)
	insRxFailName     = regexp.MustCompile(`^--- FAIL: [A-Za-z_0-9]+`)
	insRxMatched      = regexp.MustCompile(`matched [0-9]+ / [0-9]+ +\([0-9.]+%\).*`)
	insRxGraded       = regexp.MustCompile(`[0-9]+ graded, [0-9]+ excluded`)
	insRxBuilt        = regexp.MustCompile(`[0-9]+/[0-9]+ built, [0-9]+ parse failure\(s\), [0-9]+ panic\(s\)`)
	insRxConfigTotal  = regexp.MustCompile(`TOTAL +[0-9]+ +[0-9]+ +[0-9]+`)
	insRxKeysetBoth   = regexp.MustCompile(`BOTH +[0-9]+ pairs`)
	insRxKeysetExtra  = regexp.MustCompile(`EXTRA +[0-9]+ pairs`)
	insRxKeysetAbsent = regexp.MustCompile(`ABSENT +[0-9]+ pairs`)
	insRxModDB        = regexp.MustCompile(`LEVEL 2: [A-Z ]*PER-MOD-NAME CENSUS \([0-9]+ builds, [0-9]+ mod names diverge\)`)
	insRxModDBRewrite = regexp.MustCompile(` CENSUS \(([0-9]*) builds, ([0-9]*) mod names diverge\)`)
	insRxCrossStage   = regexp.MustCompile(`CROSS-STAGE +[0-9]+ pairs`)
	insRxTagCensus    = regexp.MustCompile(`TAG CENSUS \([0-9]+ builds\)`)
	insRxTagsDiffer   = regexp.MustCompile(`TAGS-DIFFER`)
	insRxCalcsRung    = regexp.MustCompile(`FIRST DIVERGENT CALCS RUNG over [0-9]+/[0-9]+ side-car builds`)
	insRxCalcsBuckets = regexp.MustCompile(`CALCS BUCKETS[^=]*`)
	insRxLeech        = regexp.MustCompile(`LEECH PROBE \(graded [0-9]+/[0-9]+ builds vs dumps\)`)
	insRxAccuracy     = regexp.MustCompile(`GRAND TOTAL matched=[0-9]+ / [0-9]+ \([0-9.]+%\)`)
	insRxRunLine      = regexp.MustCompile(`^run[0-9]+ +[0-9]+ builds$`)
	insRxIdentical    = regexp.MustCompile(`^IDENTICAL: every build agrees`)
	insRxMaterial     = regexp.MustCompile(`MATERIAL \(rel > [0-9e.-]+\): [0-9]+ builds`)
	insRxMaterialN    = regexp.MustCompile(`[0-9]+ builds$`)
	insRxReassoc      = regexp.MustCompile(`FLOAT REASSOCIATION ONLY: [0-9]+ builds`)
	insRxReassocHead  = regexp.MustCompile(`^[A-Z ]*: [0-9]+`)
	insRxTrailingNum  = regexp.MustCompile(`[0-9]+$`)
	insRxCitations    = regexp.MustCompile(`(TOTAL|PORT) +[0-9]+ / [0-9]+ top-level funcs cited \([0-9.]+%\)`)
	insRxBroken       = regexp.MustCompile(`^ +[0-9]+ broken`)
	insRxOmissions    = regexp.MustCompile(`OMISSION REASONS: [0-9]+ checked, .*`)
	insRxNeverExec    = regexp.MustCompile(`never executed +[0-9]+ blocks \([0-9.]+%\)`)
	insRxInstrumented = regexp.MustCompile(`instrumented blocks +[0-9]+`)
	insRxFailDetail   = regexp.MustCompile(`^--- FAIL: |^ +[a-z_]+\.go:[0-9]+: |^FAIL: |^ *[0-9]+ marker\(s\) do not resolve`)
	insRxTrailSpace   = regexp.MustCompile(` *$`)
)

func (ir *insRun) headlineFor(slug, f string) string { //nolint:gocyclo // one arm per instrument, as in the original
	h := ""
	switch slug {
	case "unit":
		nfail := insGrepCountFixed(insRxFailLine, f)
		if nfail > 0 {
			names := map[string]bool{}
			for _, m := range insGrepAllLines(insRxFailName, f) {
				names[strings.TrimPrefix(m, "--- FAIL: ")] = true
			}
			h = fmt.Sprintf("%d failing test(s): %s", nfail, strings.Join(insSortedKeys(names), " "))
		} else {
			h = "all packages ok"
		}
	case "oracle":
		h = insGrepFirst(insRxMatched, f)
		if h != "" {
			h = h + "; " + insGrepFirst(insRxGraded, f)
		}
	case "corpus":
		h = insGrepFirst(insRxBuilt, f)
	case "config":
		h = insGrepFirst(insRxConfigTotal, filepath.Join(ir.rundir, "config.txt"))
		h = strings.Replace(insSqueezeSpaces(h), "TOTAL", "DIFFER+ABSENT+EXTRA:", 1)
	case "keyset":
		h = insSqueezeSpaces(insGrepFirst(insRxKeysetBoth, f))
		h += "; " + insSqueezeSpaces(insGrepFirst(insRxKeysetExtra, f))
		h += "; " + insSqueezeSpaces(insGrepFirst(insRxKeysetAbsent, f))
	case "moddb":
		matches := insGrepAllLines(insRxModDB, f)
		parts := make([]string, 0, len(matches))
		for _, m := range matches {
			m = strings.Replace(m, "LEVEL 2: PER-MOD-NAME", "PLAYER", 1)
			m = strings.Replace(m, "LEVEL 2: ENEMY PER-MOD-NAME", "ENEMY", 1)
			m = insRxModDBRewrite.ReplaceAllString(m, " $2 names diverge over $1 builds")
			parts = append(parts, m)
		}
		h = strings.Join(parts, "; ")
	case "modorder":
		h = insSqueezeSpaces(insGrepFirst(insRxCrossStage, f))
		h += "  (use this, NOT the raw ORDER count)"
	case "tags":
		h = insTagsHeadline(f)
	case "calcsdamage":
		h = insGrepFirst(insRxCalcsRung, f)
		h = insOr(h, "(no census header — see the file)") + "  [REPORT-ONLY, never ratchet]"
	case "calcsbuckets":
		h = insRxTrailSpace.ReplaceAllString(insGrepFirst(insRxCalcsBuckets, f), "")
		h = insOr(h, "(no bucket header — see the file)") + "  [REPORT-ONLY, never ratchet]"
	case "leech":
		h = insGrepFirst(insRxLeech, f)
	case "accuracy":
		h = insGrepFirst(insRxAccuracy, f)
	case "determinism":
		procs := insGrepCountFixed(insRxRunLine, f)
		if insGrepQ(insRxIdentical, f) {
			h = fmt.Sprintf("IDENTICAL across %d processes", procs)
		} else {
			mat := insMatchIn(insRxMaterialN, insGrepFirst(insRxMaterial, f))
			noise := insMatchIn(insRxTrailingNum, insMatchIn(insRxReassocHead, insGrepFirst(insRxReassoc, f)))
			h = fmt.Sprintf("MATERIAL %s across %d processes (+ %s float-reassociation-only, not a failure)",
				insOr(mat, "?"), procs, insOr(noise, "0"))
		}
	case "citations":
		h = insGrepFirst(insRxCitations, f)
		broken := insMatchIn(insDigits, insGrepFirst(insRxBroken, f))
		if broken != "" && broken != "0" {
			h += "; " + broken + " BROKEN //pob: marker(s)"
		}
	case "omissions":
		h = insGrepFirst(insRxOmissions, f)
	case "coverage":
		h = insSqueezeSpaces(insGrepFirst(insRxNeverExec, f))
		h = insOr(h, "see the file") + " of " + insMatchIn(insDigits, insGrepFirst(insRxInstrumented, f))
		h += "  [REPORT-ONLY, deliberately not ratcheted]"
	}
	if h == "" {
		h = fmt.Sprintf("(no headline matched — read %s/%s.txt)", ir.rundir, slug)
	}
	return h
}

func insTagsHeadline(f string) string {
	actor := ""
	var parts []string
	for _, ln := range insReadLines(f) {
		if insRxTagCensus.MatchString(ln) {
			fields := strings.Fields(ln)
			for i, w := range fields {
				if w == "TAG" {
					if i == 0 {
						actor = ln
					} else {
						actor = fields[i-1]
					}
				}
			}
		}
		if insRxTagsDiffer.MatchString(ln) {
			s := strings.TrimLeft(ln, " ")
			s = insSqueezeSpaces(s)
			parts = append(parts, actor+" "+s)
		}
	}
	return strings.Join(parts, "; ")
}

func (ir *insRun) failDetail(slug, f string) string {
	var got []string
	for _, ln := range insReadLines(f) {
		if insRxFailDetail.MatchString(ln) {
			got = append(got, insSqueezeSpaces(insRxTrailSpace.ReplaceAllString(ln, "")))
			if len(got) == 3 {
				break
			}
		}
	}
	d := strings.Join(got, " | ")
	return insOr(d, "exited non-zero") + "  ->  " + filepath.Join(ir.rundir, slug+".txt")
}

func (ir *insRun) invariant(name, ok, detail string) {
	switch ok {
	case "SKIP":
		ir.invariants = append(ir.invariants, "SKIP  "+name+" — "+detail)
	case "1":
		ir.invariants = append(ir.invariants, "OK    "+name+" — "+detail)
	default:
		ir.invariants = append(ir.invariants, "BAD   "+name+" — "+detail)
		ir.invariantBad = true
	}
}

var insRxExcluded = regexp.MustCompile(`[0-9]+ excluded`)

func (ir *insRun) checkInvariants() {
	ohead, chead := ir.headOf["oracle"], ir.headOf["corpus"]

	ir.invariant("keyset BOTH+ABSENT == oracle denominator", "SKIP",
		"TestKeySetCensus is not in the tree; restoring it restores this invariant")

	if !insHasHeadline(ohead) {
		ir.invariant("oracle reports an excluded-build count", "SKIP",
			"oracle did not run, or its scoreboard shape changed and headline_for matched nothing")
	} else {
		excl := insMatchIn(insDigits, insGrepFirst(insRxExcluded, filepath.Join(ir.rundir, "oracle.txt")))
		if excl != "" {
			ir.invariant("oracle reports an excluded-build count", "1",
				excl+" excluded (<5 tree sources) — compare against the previous run, not against a literal")
		} else {
			ir.invariant("oracle reports an excluded-build count", "0",
				"no excluded count in the oracle output — the scoreboard shape changed")
		}
	}

	if !insHasHeadline(chead) {
		ir.invariant("every corpus build calculates", "SKIP",
			"corpus did not run, or its scoreboard shape changed and headline_for matched nothing")
	} else {
		b, t := insNum(chead, 1), insNum(chead, 2)
		switch {
		case b == "" || t == "":
			ir.invariant("every corpus build calculates", "0",
				"could not read two counts out of '"+chead+"' — REFUSING rather than guessing")
		case b == t:
			ir.invariant("every corpus build calculates", "1", b+"/"+t+" built, 0 panics")
		default:
			ir.invariant("every corpus build calculates", "0", b+"/"+t+" built — some build panicked")
		}
	}
}

func insMMSS(secs int) string { return fmt.Sprintf("%dm%02ds", secs/60, secs%60) }

func (ir *insRun) secsCell(s int) string {
	if ir.timings == "0" {
		return "-"
	}
	return strconv.Itoa(s)
}

func (ir *insRun) report(started time.Time) int {
	fmt.Println()
	fmt.Printf("=== go-pob INSTRUMENTS — %s ===\n", insGitLine())
	fmt.Printf("run dir   %s\n", ir.rundir)
	missing := ""
	if !insIsDir(ir.dumps) {
		missing = "   <- MISSING"
	}
	fmt.Printf("dumps     %s%s\n", ir.dumps, missing)
	if ir.coverage == "" {
		fmt.Println("coverage  skipped (COVERAGE=1 to include, ~4 min)")
	}
	fmt.Println()

	fmt.Printf("  %-38s %-8s %6s  %s\n", "INSTRUMENT", "STATUS", "secs", "HEADLINE")
	for _, r := range ir.rows {
		fmt.Printf("  %-38s %-8s %6s  %s\n", r.label, r.status, ir.secsCell(r.secs), r.head)
	}

	fmt.Println()
	fmt.Println("--- FAILURES ---")
	if len(ir.failures) == 0 && len(ir.notruns) == 0 {
		fmt.Println("  none")
	} else {
		for _, l := range ir.failures {
			fmt.Println("  FAIL     " + l)
		}
		if len(ir.notruns) > 0 {
			for _, l := range ir.notruns {
				fmt.Println("  NOT-RUN  " + l)
			}
			fmt.Println("  A NOT-RUN instrument answered nothing. It is not a pass and it is not a skip.")
		}
	}

	fmt.Println()
	fmt.Println("--- CROSS-INSTRUMENT INVARIANTS ---")
	for _, l := range ir.invariants {
		fmt.Println("  " + l)
	}
	if ir.invariantBad {
		fmt.Println("  A BAD invariant is worse than any single number moving: two instruments that disagree about")
		fmt.Println("  the population were never comparable, so every delta measured across them is void. Resolve it")
		fmt.Println("  before reading anything else above.")
	}

	fmt.Println()
	if ir.timings == "0" {
		fmt.Println("total -")
	} else {
		fmt.Println("total " + insMMSS(int(time.Since(started).Seconds())))
	}
	fmt.Println()
	fmt.Println("full output (reports scrubbed of slog lines; a sample of those is in *.logs.txt beside them):")
	for _, r := range ir.rows {
		fmt.Printf("  %s\n", filepath.Join(ir.rundir, r.slug+".txt"))
	}

	if len(ir.failures) > 0 || len(ir.notruns) > 0 || ir.invariantBad {
		return 1
	}
	return 0
}

func insGitLine() string {
	cmd := exec.Command("git", "log", "--oneline", "-1") //nolint:gosec,noctx // as in the original
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(out), "\n")
}

var insRxDefaultVersion = regexp.MustCompile(`^const DefaultVersion = "(.*)"$`)

func instrumentsMain(argv []string) int { //nolint:gocyclo // a linear chain, as in the original
	_ = argv
	started := time.Now()

	repo := ""
	{
		cmd := exec.Command("git", "rev-parse", "--show-toplevel") //nolint:gosec,noctx // as in the original
		cmd.Stderr = os.Stderr
		if out, err := cmd.Output(); err == nil {
			repo = strings.TrimRight(string(out), "\n")
		}
	}
	if repo != "" {
		if err := os.Chdir(repo); err != nil {
			fmt.Fprintf(os.Stderr, "REFUSING: could not enter the repository root (%s).\n", repo)
			return 2
		}
	}

	if _, ok := insShim(repo, ""); !ok {
		fmt.Fprintln(os.Stderr,
			"REFUSING: scripts/gate_versions.sh could not be sourced; the gate-version table is unreadable.")
		return 3
	}

	sha := ""
	{
		cmd := exec.Command("git", "rev-parse", "--short", "HEAD") //nolint:gosec,noctx // as in the original
		cmd.Stderr = os.Stderr
		if out, err := cmd.Output(); err == nil {
			sha = strings.TrimRight(string(out), "\n")
		}
	}
	{
		cmd := exec.Command("git", "status", "--porcelain") //nolint:gosec,noctx // as in the original
		cmd.Stderr = os.Stderr
		if out, err := cmd.Output(); err == nil && strings.TrimRight(string(out), "\n") != "" {
			sha += "-dirty"
		}
	}

	ir := &insRun{
		only:      os.Getenv("ONLY"),
		coverage:  os.Getenv("COVERAGE"),
		headOf:    map[string]string{},
		logSample: insEnvOr("LOG_SAMPLE", strconv.Itoa(insLogSampleDefault)),
		timings:   insEnvOr("INSTRUMENT_TIMINGS", "1"),
		repo:      repo,
	}
	ir.rundir = insEnvOr("RUNDIR", filepath.Join(repo, ".instruments", sha))

	gateVersion := os.Getenv("POE_VERSION")
	if gateVersion == "" {
		var hits []string
		for _, ln := range insReadLines("data/raw/main.go") {
			if m := insRxDefaultVersion.FindStringSubmatch(ln); m != nil {
				hits = append(hits, m[1])
			}
		}
		gateVersion = strings.Join(hits, "\n")
	}
	if gateVersion == "" {
		fmt.Fprintln(os.Stderr,
			"REFUSING: could not read DefaultVersion from data/raw/main.go — the dump path would be a guess.")
		return 3
	}

	home := os.Getenv("HOME")
	ir.dumps = os.Getenv("DUMPS")
	if ir.dumps == "" {
		ir.dumps = home + "/go-pob-corpus/dumps-" + gateVersion
	}
	if strings.HasPrefix(ir.dumps, "~/") {
		ir.dumps = home + "/" + strings.TrimPrefix(ir.dumps, "~/")
	}

	if err := os.MkdirAll(ir.rundir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: could not create the run directory (%s).\n", ir.rundir)
		return 2
	}

	chainTests, ok := insShim(repo, "gv_all_tests")
	if !ok {
		return 3
	}
	chainTests += "|TestModDBCensus|TestModOrderCensus|TestTagCensus|TestConfigCensus"
	chainTests += "|TestCorpusIsDeterministic"

	const goBin = "go"

	ir.runInstrument("unit", "OPERATOR=true CORPUS=true go test ./... (rest)", false,
		[]string{"env", "OPERATOR=true", "CORPUS=true", goBin, "test", "./...", "-timeout", "60m",
			"-skip", chainTests})

	authRow, ok := insShim(repo, "authoritative_row")
	if !ok {
		return 3
	}
	authOracleTest := tckField(authRow, 2)
	authCorpusTest := tckField(authRow, 3)

	ir.runInstrument("oracle", authOracleTest+" (AUTHORITATIVE)", false,
		[]string{"env", "CORPUS=true", goBin, "test", "./calculator/", "-run", authOracleTest, "-v", "-timeout", "30m"})

	ir.runInstrument("corpus", authCorpusTest+" (crash gate)", false,
		[]string{"env", "CORPUS=true", goBin, "test", "./calculator/", "-run", authCorpusTest + "$", "-v", "-timeout", "30m"})

	for _, c := range []struct{ slug, test string }{
		{"moddb", "TestModDBCensus"},
		{"modorder", "TestModOrderCensus"},
		{"tags", "TestTagCensus"},
		{"config", "TestConfigCensus"},
	} {
		ir.runInstrument(c.slug, c.test, true,
			[]string{"env", "OPERATOR=true", "CORPUS=true", goBin, "test", "./calculator/",
				"-run", c.test, "-v", "-timeout", "30m"})
	}

	ir.runInstrument("determinism", "TestCorpusIsDeterministic", false,
		[]string{"env", "OPERATOR=true", "CORPUS=true", goBin, "test", "./calculator/", "-count=1",
			"-run", "TestCorpusIsDeterministic", "-v", "-timeout", "30m"})

	ir.runInstrument("citations", "task citations", false, []string{"task", "citations"})
	ir.runInstrument("omissions", "task omissions", false, []string{"task", "omissions"})
	ir.runInstrument("taskrefs", "task task-test-refs", false, []string{"task", "task-test-refs"})
	ir.runInstrument("cachekey", "task test-cache-key", false, []string{"task", "test-cache-key"})

	if ir.coverage != "" {
		ir.runInstrument("coverage", "task coverage-census (opt-in)", false, []string{"task", "coverage-census"})
	}

	ir.checkInvariants()
	return ir.report(started)
}

func insEnvOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func insOr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func insSortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func asExitError(err error, out **exec.ExitError) bool {
	if ee, ok := err.(*exec.ExitError); ok { //nolint:errorlint // the concrete type is the point
		*out = ee
		return true
	}
	return false
}
