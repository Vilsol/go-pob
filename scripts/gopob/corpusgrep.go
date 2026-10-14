package main

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

const defaultVersionSource = "data/raw/main.go"

var (
	reDefaultVersion = regexp.MustCompile(`const\s+DefaultVersion\s*=\s*"([^"]+)"`)
	reUnsupportedRE  = regexp.MustCompile(`\(\?[=!<]|\\[1-9]`)
	reAnchored       = regexp.MustCompile(`[\^$]|\\[Az]`)
)

func cgRefuse(msg string) int {
	fmt.Fprintln(os.Stderr, "corpus-grep: "+msg)
	return root.RefusalCode
}

func corpusGrep(args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: corpus-grep takes no arguments, got %q\n", args[0])
		return 64
	}
	repo, err := root.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return root.RefusalCode
	}

	patterns, rc := cgPatterns()
	if rc != 0 {
		return rc
	}
	rxs := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		if os.Getenv("LUA") == "1" {
			out, rc := luaToRegex(pattern)
			if rc != 0 {
				return rc
			}
			pattern = out
			patterns[i] = out
		}
		if m := reUnsupportedRE.FindString(pattern); m != "" {
			return cgRefuse(fmt.Sprintf("pattern uses %s, which python allows and RE2 does not "+
				"(no lookaround, no backreferences). Rewrite it without, so this tool and its Go port "+
				"answer the same question.", pyRepr(m)))
		}
		rx, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "corpus-grep: bad pattern %s\n", pyRepr(pattern))
			return 1
		}
		rxs[i] = rx
	}

	corpus, rc := corpusDir(repo)
	if rc != 0 {
		return rc
	}
	files, err := filepath.Glob(filepath.Join(corpus, "*.xml"))
	if err != nil {
		return cgRefuse(fmt.Sprintf("%v", err))
	}
	sort.Strings(files)
	if len(files) == 0 {
		return cgRefuse("no builds under " + corpus)
	}

	scans := cgScan(files, rxs, cgLiterals(patterns))
	for _, sc := range scans {
		if sc.invalid {
			return cgRefuse(sc.path + " is not valid UTF-8")
		}
	}
	for i, pattern := range patterns {
		cgReport(corpus, pattern, len(files), scans, i)
	}
	return 0
}

func cgPatterns() ([]string, int) {
	pattern, file := os.Getenv("PATTERN"), os.Getenv("PATTERNS_FILE")
	if pattern != "" && file != "" {
		//go-only: PATTERNS_FILE has no counterpart in the original.
		return nil, cgRefuse("set PATTERN or PATTERNS_FILE, not both")
	}
	if file == "" {
		if pattern == "" {
			fmt.Fprintln(os.Stderr, "set PATTERN=<text|regex>")
			return nil, 2
		}
		return []string{pattern}, 0
	}
	b, err := os.ReadFile(file) //nolint:gosec // operator path
	if err != nil {
		//go-only: PATTERNS_FILE has no counterpart in the original.
		return nil, cgRefuse(fmt.Sprintf("cannot read PATTERNS_FILE=%s (%s)", file, errnoText(err)))
	}
	var out []string
	for _, ln := range strings.Split(string(b), "\n") {
		if ln = strings.TrimRight(ln, "\r"); strings.TrimSpace(ln) != "" {
			out = append(out, ln)
		}
	}
	if len(out) == 0 {
		//go-only: PATTERNS_FILE has no counterpart in the original.
		return nil, cgRefuse("PATTERNS_FILE=" + file + " has no patterns")
	}
	return out, 0
}

type cgFile struct {
	path    string
	skipped bool
	invalid bool
	naive   []bool
	counts  []int
}

func cgScan(files []string, rxs []*regexp.Regexp, lits []string) []cgFile {
	out := make([]cgFile, len(files))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for i := range jobs {
				out[i] = cgScanOne(files[i], rxs, lits)
			}
		})
	}
	for i := range files {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return out
}

func cgScanOne(path string, rxs []*regexp.Regexp, lits []string) cgFile {
	sc := cgFile{path: path}
	b, err := os.ReadFile(path) //nolint:gosec // a path this tool globbed
	if err != nil {
		sc.skipped = true
		return sc
	}
	raw := string(b)
	if !utf8.ValidString(raw) {
		sc.invalid = true
		return sc
	}
	flat := html.UnescapeString(collapseSpaces(raw))
	sc.naive = make([]bool, len(rxs))
	sc.counts = make([]int, len(rxs))
	var lines []string
	var foldedRaw, foldedFlat string
	for i, rx := range rxs {
		if lits[i] != "" {
			if foldedRaw == "" {
				foldedRaw, foldedFlat = cgFold(raw), cgFold(flat)
			}
			sc.counts[i] = strings.Count(foldedFlat, lits[i])
			sc.naive[i] = strings.Contains(foldedRaw, lits[i])
			continue
		}
		sc.counts[i] = len(rx.FindAllString(flat, -1))
		if !reAnchored.MatchString(rx.String()) && !rx.MatchString(raw) {
			continue
		}
		if lines == nil {
			lines = strings.Split(raw, "\n")
		}
		for _, ln := range lines {
			if rx.MatchString(ln) {
				sc.naive[i] = true
				break
			}
		}
	}
	return sc
}

func cgLiterals(patterns []string) []string {
	out := make([]string, len(patterns))
	for i, p := range patterns {
		if regexp.QuoteMeta(p) == p && cgIsASCII(p) && !strings.Contains(p, "\n") {
			out[i] = cgFold(p)
		}
	}
	return out
}

func cgIsASCII(s string) bool {
	for i := range len(s) {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func cgFold(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\u017f':
			b.WriteByte('s')
		case r == '\u212a':
			b.WriteByte('k')
		case 'A' <= r && r <= 'Z':
			b.WriteByte(byte(r) + 'a' - 'A')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func cgReport(corpus, pattern string, nFiles int, scans []cgFile, i int) {
	type hit struct {
		name string
		n    int
	}
	var hits []hit
	total, naive := 0, 0
	for _, sc := range scans {
		if sc.skipped {
			continue
		}
		if sc.naive[i] {
			naive++
		}
		if n := sc.counts[i]; n > 0 {
			hits = append(hits, hit{strings.TrimSuffix(filepath.Base(sc.path), ".xml"), n})
			total += n
		}
	}

	fmt.Printf("CORPUS %s\n", corpus)
	fmt.Printf("PATTERN %s\n", pattern)
	fmt.Printf("MATCHED %d of %d builds, %d occurrence(s)\n", len(hits), nFiles, total)
	if len(hits) == 0 {
		fmt.Print("  ⚠️  0 matches means 0 in the INPUT XML — item text, gem names and config only.\n" +
			"      Mods from the PASSIVE TREE, from gem/skill data, or COMPOSED at parse time\n" +
			"      (ModParser's tails accumulate flags onto a base pattern) are INVISIBLE here.\n" +
			"      For those, size the population with `task corpus-predicate`, which walks the\n" +
			"      built environment, and do NOT report this 0 as 'nothing carries this'.\n")
	}
	if naive != len(hits) {
		fmt.Printf("  (a line-by-line grep would say %d — the difference is mod lines WRAPPED across\n"+
			"   newlines and &apos;-escaped text. That gap is why this script exists.)\n", naive)
	}
	show := hits
	if os.Getenv("NAMES") != "1" && len(show) > 15 {
		show = show[:15]
	}
	for _, h := range show {
		fmt.Printf("  %-52s %d\n", h.name, h.n)
	}
	if os.Getenv("NAMES") != "1" && len(hits) > len(show) {
		fmt.Printf("  … and %d more (NAMES=1 for all)\n", len(hits)-len(show))
	}
}

func corpusDir(repo string) (string, int) {
	if explicit := os.Getenv("CORPUS_DIR"); explicit != "" {
		if fi, err := os.Stat(explicit); err != nil || !fi.IsDir() { //nolint:gosec // operator path
			return "", cgRefuse("CORPUS_DIR=" + explicit + " is not a directory")
		}
		return explicit, 0
	}
	version := os.Getenv("POE_VERSION")
	if version == "" {
		var rc int
		version, rc = defaultVersion(repo)
		if rc != 0 {
			return "", rc
		}
	}
	path := filepath.Join(repo, "testdata", "corpus-"+version)
	//nolint:gosec // the repo root joined with a version string from POE_VERSION or data/raw
	if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
		return "", cgRefuse(fmt.Sprintf("no corpus for game-data version %s at %s. The corpus and "+
			"the version you are sizing for must match, so this refuses rather than measuring "+
			"another league.", pyRepr(version), path))
	}
	return path, 0
}

func defaultVersion(repo string) (string, int) {
	b, err := os.ReadFile(filepath.Join(repo, defaultVersionSource)) //nolint:gosec // a repo path
	if err != nil {
		return "", cgRefuse(fmt.Sprintf("cannot read %s (%s)", defaultVersionSource, errnoText(err)))
	}
	m := reDefaultVersion.FindStringSubmatch(string(b))
	if m == nil {
		return "", cgRefuse("no `const DefaultVersion` in " + defaultVersionSource +
			" — the default game-data version moved and this script cannot follow it. " +
			"Pass POE_VERSION explicitly.")
	}
	return m[1], 0
}

func gameDataVersion(repo string) (string, error) {
	if v := os.Getenv("POE_VERSION"); v != "" {
		return v, nil
	}
	b, err := os.ReadFile(filepath.Join(repo, defaultVersionSource)) //nolint:gosec // a repo path
	if err != nil {
		return "", fmt.Errorf("cannot read %s for DefaultVersion: %w", defaultVersionSource, err)
	}
	m := reDefaultVersion.FindStringSubmatch(string(b))
	if m == nil {
		return "", fmt.Errorf("no `const DefaultVersion` in %s", defaultVersionSource)
	}
	return m[1], nil
}

func collapseSpaces(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inRun := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !inRun {
				b.WriteByte(' ')
				inRun = true
			}
			continue
		}
		inRun = false
		b.WriteRune(r)
	}
	return b.String()
}

var luaClasses = map[string]string{
	"%d": `\d`, "%a": "[A-Za-z]", "%s": `\s`, "%w": `\w`, "%p": `[^\w\s]`,
	"%u": "[A-Z]", "%l": "[a-z]", "%%": "%",
}

var luaEscapes = map[string]string{
	"%.": `\.`, "%+": `\+`, "%-": `\-`, "%(": `\(`, "%)": `\)`, "%[": `\[`, "%]": `\]`,
}

func luaToRegex(pat string) (string, int) {
	var out []string
	r := []rune(pat)
	for i := 0; i < len(r); {
		two := ""
		if i+1 < len(r) {
			two = string(r[i : i+2])
		}
		switch {
		case luaClasses[two] != "":
			out = append(out, luaClasses[two])
			i += 2
		case luaEscapes[two] != "":
			out = append(out, luaEscapes[two])
			i += 2
		case r[i] == '%':
			fmt.Fprintf(os.Stderr, "corpus-grep: %s is a Lua class this script cannot translate "+
				"faithfully. Write the regex yourself rather than let it guess.\n", pyRepr(two))
			return "", 1
		case r[i] == '-' && len(out) > 0:
			prev := out[len(out)-1]
			if strings.HasPrefix(prev, "[") || strings.HasPrefix(prev, `\`) {
				out = append(out, "*?")
			} else {
				out = append(out, `\-`)
			}
			i++
		case strings.ContainsRune("{}\\", r[i]):
			out = append(out, `\`+string(r[i]))
			i++
		default:
			out = append(out, string(r[i]))
			i++
		}
	}
	return strings.Join(out, ""), 0
}
