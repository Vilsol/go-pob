package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

var reBuildID = regexp.MustCompile(`[0-9]{4,}-[\p{L}\p{N}_.-]+`)

func statOverlap(args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: stat-overlap takes no arguments, got %q\n", args[0])
		return 64
	}
	repo, err := root.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return root.RefusalCode
	}

	stats := strings.Fields(os.Getenv("STATS"))
	if len(stats) == 0 {
		fmt.Fprintln(os.Stderr, `REFUSING: set STATS="<stat> <stat> ..." (2 or more).`)
		return 2
	}
	if len(stats) < 2 {
		fmt.Fprintf(os.Stderr, "REFUSING: overlap needs at least 2 stats; got %d.\n", len(stats))
		return 2
	}

	subject := os.Getenv("SUBJECT")
	if subject != "" && !slices.Contains(stats, subject) {
		//go-only: SUBJECT= is a switch the deleted python original has no concept of, so no
		//go-only: fixture of the original can reach this and no original wording can match it.
		fmt.Fprintf(os.Stderr, "REFUSING: SUBJECT=%q is not in STATS; add it there too.\n", subject)
		return 2
	}

	ctx := context.Background()
	out := os.Getenv("OUT")
	if out == "" {
		sha, err := exec.CommandContext(ctx, "git", "-C", repo, "rev-parse", "--short", "HEAD").Output()
		if err != nil {
			sha = nil
		}
		out = "/tmp/stat-overlap-" + strings.TrimSpace(string(sha))
	}
	if err := os.MkdirAll(out, 0o750); err != nil { //nolint:gosec // $OUT, operator-supplied, as the original's `mkdir -p "$OUT"`
		fmt.Fprintf(os.Stderr, "REFUSING: cannot create %s: %s\n", out, errnoText(err))
		return 1
	}

	sets := map[string][]string{}
	var need []string
	for _, s := range stats {
		cache := filepath.Join(out, s+".txt")
		if fi, err := os.Stat(cache); err == nil && fi.Size() > 0 { //nolint:gosec // operator path
			b, readErr := os.ReadFile(cache) //nolint:gosec // ditto
			if readErr != nil {
				fmt.Fprintf(os.Stderr, "REFUSING: %s is not readable: %s\n", cache, errnoText(readErr))
				return 1
			}
			sets[s] = splitLinesNonEmpty(string(b))
			continue
		}
		need = append(need, s)
	}

	var extracted map[string][]string
	missingSection := map[string]bool{}
	if len(need) > 0 {
		ids, missing := runStatRatiosBatch(ctx, repo, need)
		extracted = ids
		for _, s := range missing {
			missingSection[s] = true
		}
	}

	for _, s := range stats {
		if _, done := sets[s]; done {
			continue
		}
		ids := extracted[s]
		cache := filepath.Join(out, s+".txt")
		//nolint:gosec // $OUT joined with a stat name from $STATS, both operator-supplied
		if err := os.WriteFile(cache, []byte(strings.Join(ids, "\n")+"\n"), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "REFUSING: cannot write %s: %s\n", cache, errnoText(err))
			return 1
		}
		if len(ids) == 0 {
			fmt.Fprintf(os.Stderr, "REFUSING: extracted 0 failing builds for '%s'. Either the name is wrong or it has no\n", s)
			fmt.Fprintf(os.Stderr, "          CLASS=wrong population — check with 'task corpus-vs-oracle STAT=%s CLASS=wrong'.\n", s)
			if missingSection[s] && len(missingSection) < len(need) {
				fmt.Fprintf(os.Stderr, "          (the batched run printed sections for the other stats but none for '%s')\n", s)
			}
			return 1
		}
		sets[s] = ids
	}

	fmt.Printf("sets cached in %s (CLASS=wrong)\n", out)
	fmt.Println()
	fmt.Println("=== failing-build set sizes ===")
	totals := soWorklistTotals(os.Getenv("WL"))
	for _, s := range stats {
		n := len(sets[s])
		switch {
		case totals == nil:
			fmt.Printf("  %-34s %5d builds\n", s, n)
		case totals[s] == 0:
			fmt.Printf("  %-34s %5d builds  ⚠️ absent from stats.tsv — a stat with zero miss is PERFECT and is omitted\n", s, n)
		default:
			starved := totals[s] - n
			flag := ""
			if starved > n {
				flag = "  ⚠️ MOST of this stat's residual is STARVED and invisible to every table below"
			}
			fmt.Printf("  %-34s %5d builds of %d disagreeing (%d starved)%s\n", s, n, totals[s], starved, flag)
		}
	}
	fmt.Println()
	fmt.Println("=== CONTAINMENT: % of ROW's failing builds that ALSO fail COL ===")
	fmt.Println("    a high row means ROW is plausibly DOWNSTREAM of COL; read down a column for shared inputs")
	fmt.Printf("  %-30s", "")
	for _, c := range stats {
		fmt.Printf(" %10.10s", c)
	}
	fmt.Println()
	for _, r := range stats {
		fmt.Printf("  %-30s", r)
		rn := len(sets[r])
		for _, c := range stats {
			both := intersectionSize(sets[r], sets[c])
			fmt.Printf("%10d%%", int(float64(both*100)/float64(rn)))
		}
		fmt.Println()
	}

	if subject != "" {
		printUnionCoverage(sets, stats, subject)
	}
	return 0
}

func printUnionCoverage(sets map[string][]string, stats []string, subject string) {
	union := map[string]bool{}
	others := 0
	for _, s := range stats {
		if s == subject {
			continue
		}
		others++
		for _, id := range sets[s] {
			union[id] = true
		}
	}

	subjectSet := sets[subject]
	both := 0
	for _, id := range subjectSet {
		if union[id] {
			both++
		}
	}

	fmt.Println()
	fmt.Printf("=== UNION COVERAGE of %s by the other %d stat(s) ===\n", subject, others)
	if len(subjectSet) == 0 {
		fmt.Printf("  %s has no failing builds — nothing to explain\n", subject)
		return
	}
	fmt.Printf("  union of the other sets            %5d builds\n", len(union))
	fmt.Printf("  %-34s %5d failing, %d (%.1f%%) ALSO fail at least one of them\n",
		subject, len(subjectSet), both, float64(both*100)/float64(len(subjectSet)))
	fmt.Printf("  %-34s %5d failing with EVERY listed stat right — this is the unexplained residual\n",
		"", len(subjectSet)-both)
	fmt.Println("  ⚠️ as complete as STATS: a stat left out is one this reports as explaining nothing")
}

func runStatRatios(ctx context.Context, repo, stat string) []string {
	cmd := exec.CommandContext(ctx, "go", "test", "./calculator/", "-run", "TestStatRatios",
		"-v", "-count=1")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "OPERATOR=true", "CORPUS=true", "STAT="+stat,
		"CLASS=wrong", "PRECISION=4", "MEMBERS=4000", "LIMIT=4000")
	out, _ := cmd.CombinedOutput()
	seen := map[string]bool{}
	var ids []string
	for _, m := range reBuildID.FindAllString(string(out), -1) {
		if !seen[m] {
			seen[m] = true
			ids = append(ids, m)
		}
	}
	sort.Strings(ids)
	return ids
}

func splitLinesNonEmpty(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func intersectionSize(a, b []string) int {
	in := make(map[string]bool, len(b))
	for _, s := range b {
		in[s] = true
	}
	n := 0
	for _, s := range a {
		if in[s] {
			n++
		}
	}
	return n
}

func soWorklistTotals(wl string) map[string]int {
	if wl == "" {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(wl, "stats.tsv")) //nolint:gosec // operator-supplied path
	if err != nil {
		fmt.Fprintf(os.Stderr, "note: WL=%s has no readable stats.tsv (%s); set sizes will be CLASS=wrong only\n",
			wl, errnoText(err))
		return nil
	}
	out := map[string]int{}
	for i, line := range strings.Split(string(b), "\n") {
		if i == 0 || line == "" {
			continue
		}
		c := strings.Split(line, "\t")
		if len(c) < 3 {
			continue
		}
		seen, err1 := strconv.Atoi(c[1])
		ok, err2 := strconv.Atoi(c[2])
		if err1 != nil || err2 != nil {
			continue
		}
		if seen-ok > 0 {
			out[c[0]] = seen - ok
		}
	}
	return out
}

var soSectionHead = regexp.MustCompile(`^=== STAT RATIOS: (\S+) \(table `)

func splitBatchedStatRatios(out string, stats []string) (map[string][]string, []string) {
	want := make(map[string]bool, len(stats))
	for _, s := range stats {
		want[s] = true
	}
	perStat := map[string]map[string]bool{}
	preamble := map[string]bool{}

	cur := ""
	for _, line := range strings.Split(out, "\n") {
		if m := soSectionHead.FindStringSubmatch(line); m != nil && want[m[1]] {
			cur = m[1]
			if perStat[cur] == nil {
				perStat[cur] = map[string]bool{}
			}
		}
		into := preamble
		if cur != "" {
			into = perStat[cur]
		}
		for _, id := range reBuildID.FindAllString(line, -1) {
			into[id] = true
		}
	}

	sets := map[string][]string{}
	var missing []string
	for _, s := range stats {
		if perStat[s] == nil {
			missing = append(missing, s)
			continue
		}
		ids := make([]string, 0, len(perStat[s])+len(preamble))
		for id := range perStat[s] {
			ids = append(ids, id)
		}
		for id := range preamble {
			if !perStat[s][id] {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		sets[s] = ids
	}
	return sets, missing
}

func runStatRatiosBatch(ctx context.Context, repo string, stats []string) (map[string][]string, []string) {
	cmd := exec.CommandContext(ctx, "go", "test", "./calculator/", "-run", "TestStatRatios",
		"-v", "-count=1")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "OPERATOR=true", "CORPUS=true", "STAT="+strings.Join(stats, " "),
		"CLASS=wrong", "PRECISION=4", "MEMBERS=4000", "LIMIT=4000")
	out, _ := cmd.CombinedOutput()
	return splitBatchedStatRatios(string(out), stats)
}
