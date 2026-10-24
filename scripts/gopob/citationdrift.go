package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

var (
	reMarker    = regexp.MustCompile(`//pob:(?:block:)?(.+?)\s*$`)
	reLuaRange  = regexp.MustCompile(`^(src/[^\s:,]+?\.lua):([1-9][0-9]*)(?:-([1-9][0-9]*))?$`)
	reStrongTok = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_.]{6,}`)
)

func normWS(s string) string { return strings.Join(strings.Fields(s), " ") }

type driftRow struct {
	verdict, goref, luaref string
	where                  int
	block                  []string
}

type driftEnv struct {
	pob, old string
	oldCache map[string][]string
	newCache map[string][]string
}

func citationDrift(args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: citation-drift takes no arguments, got %q\n", args[0])
		return 64
	}
	repo, err := root.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return root.RefusalCode
	}
	pob := expandUser(pobPinDir())
	old := os.Getenv("OLD")
	if old == "" {
		old = "v2.18.1"
	}
	flags := os.Getenv("FLAGS")

	env := &driftEnv{pob: pob, old: old,
		oldCache: map[string][]string{}, newCache: map[string][]string{}}

	ctx := context.Background()
	targets, rc := driftTargets(repo, flags)
	if rc != 0 {
		return rc
	}

	seen := map[string]bool{}
	var rows []driftRow
	for _, tg := range targets {
		m := reLuaRange.FindStringSubmatch(tg[1])
		if m == nil {
			continue
		}
		path := m[1]
		start, _ := strconv.Atoi(m[2])
		end := start
		if m[3] != "" {
			end, _ = strconv.Atoi(m[3])
		}
		key := tg[0] + "\x00" + tg[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		verdict, where, block, rc := env.classify(ctx, path, start, end)
		if rc != 0 {
			return rc
		}
		rows = append(rows, driftRow{verdict, tg[0], tg[1], where, block})
	}

	order := map[string]int{"CHANGED": 0, "GONE": 1, "MOVED": 2, "SAME": 3}
	ord := func(v string) int {
		if n, ok := order[v]; ok {
			return n
		}
		return 9
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if oa, ob := ord(a.verdict), ord(b.verdict); oa != ob {
			return oa < ob
		}
		if a.goref != b.goref {
			return a.goref < b.goref
		}
		return a.luaref < b.luaref
	})

	counts := map[string]int{}
	for _, r := range rows {
		counts[r.verdict]++
		if r.verdict == "SAME" {
			continue
		}
		loc := ""
		if r.where != 0 {
			loc = fmt.Sprintf(" -> pin:%d", r.where)
		}
		fmt.Printf("%-8s %-44s %s%s\n", r.verdict, r.goref, r.luaref, loc)
		if (r.verdict == "CHANGED" || r.verdict == "GONE") && len(r.block) > 0 {
			fmt.Printf("         old: %s\n", truncRunes(normWS(r.block[0]), 96))
		}
	}
	fmt.Println()
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if oa, ob := ord(keys[i]), ord(keys[j]); oa != ob {
			return oa < ob
		}
		return keys[i] < keys[j]
	})
	for _, k := range keys {
		fmt.Printf("  %4d  %s\n", counts[k], k)
	}
	fmt.Printf("  %4d  total citations examined against %s\n", len(rows), old)
	fmt.Println("\n⚠️ MOVED is reliable; CHANGED and GONE are HYPOTHESES from a string match. Read both")
	fmt.Println("   versions before touching code. Re-pointing a CHANGED marker without re-porting the")
	fmt.Println("   Go silences the only signal that the function drifted.")
	return 0
}

func driftTargets(repo, flags string) ([][2]string, int) {
	var targets [][2]string
	if flags != "" {
		if _, err := os.Stat(flags); err == nil { //nolint:gosec // an operator-supplied path
			src, rc := readTextRefusing(flags)
			if rc != 0 {
				return nil, rc
			}
			for _, line := range strings.Split(src, "\n") {
				if f := strings.Fields(line); len(f) >= 2 {
					targets = append(targets, [2]string{f[0], f[1]})
				}
			}
			return targets, 0
		}
	}
	err := filepath.WalkDir(repo, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		dir := filepath.Dir(p)
		if strings.Contains(dir, "/.git") || strings.Contains(dir, "/frontend") {
			return nil
		}
		b, readErr := os.ReadFile(p) //nolint:gosec // a path under the resolved repo root
		if readErr != nil {
			return fmt.Errorf("reading %s: %w", p, readErr)
		}
		if !utf8.Valid(b) {
			return fmt.Errorf("%s is not valid UTF-8", p)
		}
		rel, _ := filepath.Rel(repo, p)
		for i, line := range strings.Split(string(b), "\n") {
			m := reMarker.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			for _, part := range strings.Split(m[1], ",") {
				part = strings.TrimSpace(part)
				if reLuaRange.MatchString(part) {
					targets = append(targets, [2]string{fmt.Sprintf("%s:%d", rel, i+1), part})
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
		return nil, 1
	}
	return targets, 0
}

func (e *driftEnv) oldFile(ctx context.Context, path string) ([]string, int) {
	if v, ok := e.oldCache[path]; ok {
		return v, 0
	}
	//nolint:gosec // POB_DIR and a citation path, both operator-supplied
	out, err := exec.CommandContext(ctx, "git", "-C", e.pob, "show", e.old+":"+path).Output()
	switch {
	case err == nil:
		e.oldCache[path] = strings.Split(string(out), "\n")
	case isExitError(err):
		e.oldCache[path] = nil
	default:
		msg := errnoText(err)
		if errors.Is(err, exec.ErrNotFound) {
			msg = "No such file or directory"
		}
		fmt.Fprintf(os.Stderr, "REFUSING: cannot run git: %s\n", msg)
		return nil, 1
	}
	return e.oldCache[path], 0
}

func isExitError(err error) bool {
	var ee *exec.ExitError
	return errors.As(err, &ee)
}

func (e *driftEnv) newFile(path string) ([]string, int) {
	if v, ok := e.newCache[path]; ok {
		return v, 0
	}
	full := filepath.Join(e.pob, path)
	if _, err := os.Stat(full); err != nil { //nolint:gosec // POB_DIR joined with a citation path
		e.newCache[path] = nil
		return nil, 0
	}
	text, rc := readTextRefusing(full)
	if rc != 0 {
		return nil, rc
	}
	e.newCache[path] = strings.Split(text, "\n")
	return e.newCache[path], 0
}

func (e *driftEnv) classify(ctx context.Context, path string, start, end int) (string, int, []string, int) {
	o, rc := e.oldFile(ctx, path)
	if rc != 0 {
		return "", 0, nil, rc
	}
	if o == nil {
		return "NOFILE-OLD", 0, nil, 0
	}
	n, rc := e.newFile(path)
	if rc != 0 {
		return "", 0, nil, rc
	}
	if n == nil {
		return "NOFILE-NEW", 0, nil, 0
	}
	var block []string
	for i := start - 1; i < end && i < len(o); i++ {
		s := normWS(o[i])
		if s != "" && !strings.HasPrefix(s, "--") {
			block = append(block, o[i])
		}
	}
	if len(block) == 0 {
		return "EMPTY-OLD", 0, nil, 0
	}
	anchor := block[0]
	for _, l := range block[1:] {
		if len(normWS(l)) > len(normWS(anchor)) {
			anchor = l
		}
	}
	na := normWS(anchor)
	for i, l := range n {
		if normWS(l) == na {
			if i+1 != start {
				return "MOVED", i + 1, block, 0
			}
			return "SAME", i + 1, block, 0
		}
	}
	var toks []string
	for _, t := range reStrongTok.FindAllString(na, -1) {
		if strings.Contains(t, ".") || len(t) > 10 {
			toks = append(toks, t)
		}
	}
	for _, t := range toks[:min(4, len(toks))] {
		for i, l := range n {
			if strings.Contains(l, t) {
				return "CHANGED", i + 1, block, 0
			}
		}
	}
	return "GONE", 0, block, 0
}
