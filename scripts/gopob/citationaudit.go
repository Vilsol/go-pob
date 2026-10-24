package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

var (
	reAuditMarker = regexp.MustCompile(`//pob(2)?:(block:)?(.+?)\s*$`)
	reAuditRange  = regexp.MustCompile(`^(src/[^\s:,]+?\.lua):([1-9][0-9]*)(?:-([1-9][0-9]*))?$`)
	reCloser      = regexp.MustCompile(`^\s*(end\b|end\)|end,|end;|until\b)`)
	reComment     = regexp.MustCompile(`^\s*--`)
	reDangling    = regexp.MustCompile(`(?:\b(?:and|or|not)|\.\.|[-+*/%^<>=])$`)
	reFuncWord    = regexp.MustCompile(`\bfunction\b`)
)

func codeOnly(line string) string {
	for i := strings.Index(line, "--"); i != -1; {
		head := line[:i]
		if strings.Count(head, `"`)%2 == 0 && strings.Count(head, "'")%2 == 0 {
			return strings.TrimRight(head, " \t\r\n\v\f")
		}
		next := strings.Index(line[i+2:], "--")
		if next == -1 {
			break
		}
		i += 2 + next
	}
	return strings.TrimRight(line, " \t\r\n\v\f")
}

func danglingTail(lines []string, end int) string {
	if end < 1 || end > len(lines) {
		return ""
	}
	return reDangling.FindString(strings.TrimSpace(codeOnly(lines[end-1])))
}

type auditCtx struct {
	pob   string
	cache map[string][]string
	poe2  *auditCtx
}

// pin is the checkout a marker resolves against: //pob2: on the PoE2 pin, //pob: on POB_DIR.
func (a *auditCtx) pin(m []string) *auditCtx {
	if m[1] == "" {
		return a
	}
	if a.poe2 == nil {
		a.poe2 = &auditCtx{pob: pcPoB2Dir(), cache: map[string][]string{}}
	}
	return a.poe2
}

func (a *auditCtx) luaLines(path string) []string {
	if v, ok := a.cache[path]; ok {
		return v
	}
	full := filepath.Join(a.pob, path)
	if _, err := os.Stat(full); err != nil { //nolint:gosec // POB_DIR joined with a citation path
		a.cache[path] = nil
		return nil
	}
	text, rc := readTextRefusing3(full)
	if rc != 0 {
		os.Exit(rc)
	}
	a.cache[path] = strings.Split(text, "\n")
	return a.cache[path]
}

func citationAudit(args []string) int {
	repo, err := root.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return root.RefusalCode
	}
	pob := os.Getenv("POB_DIR")
	if pob == "" {
		pob = expandUser("~/PathOfBuilding")
	}
	a := &auditCtx{pob: pob, cache: map[string][]string{}}
	if os.Getenv("SWEEP") == "1" {
		return auditSweep(a, repo)
	}
	gofile := os.Getenv("FILE")
	if gofile == "" && len(args) > 0 {
		gofile = args[0]
	}
	if gofile == "" {
		fmt.Fprintln(os.Stderr, "set FILE=<go file>, or SWEEP=1 to check every PORT file")
		return 2
	}
	ctxRaw := os.Getenv("CONTEXT")
	if ctxRaw == "" {
		ctxRaw = "0"
	}
	if !reNonNegI.MatchString(ctxRaw) {
		fmt.Fprintf(os.Stderr, "REFUSING: CONTEXT must be a non-negative integer, not %s\n", ctxRaw)
		return root.RefusalCode
	}
	ctx, _ := strconv.Atoi(ctxRaw)
	return auditOne(a, gofile, ctx)
}

func auditSweep(a *auditCtx, repo string) int {
	manSrc, rc := readTextRefusing3(filepath.Join(repo, "scripts/port_manifest.txt"))
	if rc != 0 {
		return rc
	}
	var man []string
	for _, ln := range strings.Split(manSrc, "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" && !strings.HasPrefix(ln, "#") && strings.Contains(ln, " PORT") {
			man = append(man, strings.Fields(ln)[0])
		}
	}
	total, flagged := 0, 0
	for _, rel := range man {
		full := filepath.Join(repo, rel)
		if _, err := os.Stat(full); err != nil { //nolint:gosec // a manifest-declared repo path
			continue
		}
		src, rc := readTextRefusing3(full)
		if rc != 0 {
			return rc
		}
		for i, gl := range strings.Split(src, "\n") {
			m := reAuditMarker.FindStringSubmatch(gl)
			if m == nil {
				continue
			}
			for _, spec := range strings.Split(m[3], ",") {
				rm := reAuditRange.FindStringSubmatch(strings.TrimSpace(spec))
				if rm == nil {
					continue
				}
				path := rm[1]
				start, _ := strconv.Atoi(rm[2])
				end := start
				if rm[3] != "" {
					end, _ = strconv.Atoi(rm[3])
				}
				lines := a.pin(m).luaLines(path)
				if lines == nil || start < 1 || start > len(lines) {
					continue
				}
				if m[1] != "" {
					path = "pob2:" + path
				}
				total++
				first := lines[start-1]
				why := ""
				switch {
				case strings.TrimSpace(first) == "":
					why = "first line is BLANK (sed hides this; the block starts below)"
				case reCloser.MatchString(first):
					why = "first line is a CLOSER (cites the previous block's tail)"
				case reComment.MatchString(first) && start == end:
					why = "entire range is one COMMENT line"
				}
				if why != "" {
					flagged++
					fmt.Printf("  FLAG %s:%d  %s:%d-%d  %s\n       %6d | [%s]\n",
						rel, i+1, path, start, end, why, start, first)
				}
				if op := danglingTail(lines, end); op != "" {
					flagged++
					after := ""
					if end < len(lines) {
						after = lines[end]
					}
					fmt.Printf("  FLAG %s:%d  %s:%d-%d  last line ends MID-STATEMENT on `%s` "+
						"(the expression continues on :%d)\n       %6d | [%s]\n       %6d | [%s]\n",
						rel, i+1, path, start, end, op, end+1, end, lines[end-1], end+1, after)
				}
			}
		}
	}
	fmt.Printf("citation shapes: %d range(s) across %d PORT file(s), %d flagged (ceiling 0)\n",
		total, len(man), flagged)
	if flagged > 0 {
		fmt.Println("A flagged range is almost certainly WRONG. Thirty such citations were found by hand\n" +
			"on 2026-08-03, every one of which passed `mise run citations`. Re-read it with\n" +
			"`mise run task -- citations-audit FILE=<the go file>` and fix the range.")
		return 1
	}
	return 0
}

func auditOne(a *auditCtx, gofile string, ctx int) int {
	src, rc := readTextRefusing3(gofile)
	if rc != 0 {
		return rc
	}
	flagged, seen := 0, 0
	for i, gl := range strings.Split(src, "\n") {
		m := reAuditMarker.FindStringSubmatch(gl)
		if m == nil {
			continue
		}
		for _, spec := range strings.Split(m[3], ",") {
			spec = strings.TrimSpace(spec)
			rm := reAuditRange.FindStringSubmatch(spec)
			if rm == nil {
				fmt.Printf("  %s:%d  UNPARSED %s\n", gofile, i+1, pyRepr(spec))
				continue
			}
			path := rm[1]
			start, _ := strconv.Atoi(rm[2])
			end := start
			if rm[3] != "" {
				end, _ = strconv.Atoi(rm[3])
			}
			lines := a.pin(m).luaLines(path)
			seen++
			blockPrefix := ""
			if m[1] != "" {
				blockPrefix = "pob2:"
			}
			if m[2] != "" {
				blockPrefix += "block:"
			}
			fmt.Printf("\n%s:%d  %s%s:%d-%d\n", gofile, i+1, blockPrefix, path, start, end)
			if lines == nil {
				fmt.Printf("    !! file not found under %s\n", a.pin(m).pob)
				flagged++
				continue
			}
			show := func(n int, mark string) {
				if n >= 1 && n <= len(lines) {
					fmt.Printf("  %s %6d | [%s]\n", mark, n, lines[n-1])
				}
			}
			for n := max(1, start-ctx); n < start; n++ {
				show(n, "-")
			}
			show(start, ">")
			if end != start {
				if end-start > 1 {
					fmt.Printf("    %6s | ... %d line(s) ...\n", "", end-start-1)
				}
				show(end, "<")
			}
			for n := end + 1; n < end+1+ctx; n++ {
				show(n, "-")
			}

			first := ""
			if start >= 1 && start <= len(lines) {
				first = lines[start-1]
			}
			_ = reFuncWord
			var notes []string
			if reCloser.MatchString(first) {
				notes = append(notes, "FIRST LINE IS A CLOSER — this range almost certainly starts one "+
					"line too early and cites the previous block's tail")
			}
			if reComment.MatchString(first) && start == end {
				notes = append(notes, "THE ENTIRE RANGE IS ONE COMMENT LINE — it cites nothing executable")
			}
			if strings.TrimSpace(first) == "" {
				notes = append(notes, "FIRST LINE IS BLANK")
			}
			if op := danglingTail(lines, end); op != "" {
				notes = append(notes, fmt.Sprintf("LAST LINE ENDS MID-STATEMENT on `%s` — the "+
					"expression continues onto :%d, which this range excludes", op, end+1))
			}
			for _, n := range notes {
				fmt.Printf("    !! %s\n", n)
			}
			if len(notes) > 0 {
				flagged++
			}
		}
	}
	fmt.Printf("\n%d range(s) checked, %d flagged.\n", seen, flagged)
	if flagged == 0 {
		fmt.Println("No suspicious shapes. That is NOT proof the ranges are right — only reading the " +
			"Lua proves that. It rules out the shape that produced most of the 30 known cases.")
	}
	return 0
}

func readTextRefusing3(path string) (string, int) {
	s, rc := readTextRefusing(path)
	if rc != 0 {
		return "", root.RefusalCode
	}
	return s, 0
}
