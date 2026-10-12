package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const daUsage = "usage: dump-alone <pob-dir> <repo-root> <builds-dir> <committed-dumps-dir> <out-dir> <list-file>"

func daOutputDiff(committed, alone map[string]any) ([]string, int) {
	union := map[string]bool{}
	for k := range committed {
		union[k] = true
	}
	for k := range alone {
		union[k] = true
	}
	var diff []string
	for k := range union {
		a, okA := committed[k]
		b, okB := alone[k]
		if okA != okB || !reflect.DeepEqual(a, b) {
			diff = append(diff, k)
		}
	}
	sort.Strings(diff)
	return diff, len(union)
}

func daReadOutput(path string) (map[string]any, error) {
	b, err := os.ReadFile(path) //nolint:gosec // an operator-supplied dump path
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var d struct {
		Output map[string]any `json:"output"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if d.Output == nil {
		return nil, fmt.Errorf("%s has no output table", path)
	}
	return d.Output, nil
}

func daReadList(path string) ([]string, error) {
	f, err := os.Open(path) //nolint:gosec // an operator-supplied list
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		l = strings.TrimSuffix(strings.TrimSuffix(filepath.Base(l), ".xml"), ".json")
		out = append(out, l)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return out, nil
}

//go-only: the helper every dump-alone refusal goes through; the tool has no original.

func daRefuse(code int, format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "dump-alone: "+format+"\n", args...)
	return code
}

func dumpAlone(argv []string) int {
	if len(argv) != 6 {
		//go-only: dump-alone is Go-native; there is no original to share the usage text with.
		return daRefuse(2, "%s", daUsage)
	}
	pob, here, builds, committed, out, listFile := argv[0], argv[1], argv[2], argv[3], argv[4], argv[5]
	for _, d := range []string{filepath.Join(pob, "src"), builds, committed} {
		if fi, err := os.Stat(d); err != nil || !fi.IsDir() { //nolint:gosec // an operator-supplied path
			//go-only: dump-alone is Go-native; a missing input directory refuses rather than dumping nothing.
			return daRefuse(2, "no directory %s", d)
		}
	}
	luajit := os.Getenv("LUAJIT")
	if luajit == "" {
		//go-only: dump-alone is Go-native; an unset interpreter would silently fall back to stock.
		return daRefuse(2, "LUAJIT is unset; point it at .luajit-det/bin/luajit")
	}
	if !strings.Contains(luajit, "luajit-det") {
		fmt.Fprintf(os.Stderr, "dump-alone: ⚠️ LUAJIT=%s is not the deterministic build; a per-process "+
			"pairs() draw will read as a leak\n", luajit)
	}
	if dpNonEmpty(out) {
		//go-only: dump-alone is Go-native; the non-destructive rule is shared with dump-shard.
		return daRefuse(3, "%s exists and is NOT EMPTY — refusing to overwrite another run's output", out)
	}
	names, err := daReadList(listFile)
	if err != nil {
		//go-only: dump-alone is Go-native; an unreadable list must not read as an empty one.
		return daRefuse(4, "cannot read the build list %s: %v", listFile, err)
	}
	if len(names) == 0 {
		//go-only: dump-alone is Go-native; an empty list would report a clean floor having looked at nothing.
		return daRefuse(4, "the build list %s names no builds", listFile)
	}
	for _, n := range names {
		for _, p := range []string{filepath.Join(builds, n+".xml"), filepath.Join(committed, n+".json")} {
			if _, err := os.Stat(p); err != nil { //nolint:gosec // an operator-supplied path
				//go-only: dump-alone is Go-native; an incomparable build refuses the whole run.
				return daRefuse(4, "%s does not exist", p)
			}
		}
	}
	if err := os.MkdirAll(out, 0o750); err != nil { //nolint:gosec // an operator-supplied output path
		//go-only: dump-alone is Go-native.
		return daRefuse(3, "cannot create %s: %v", out, err)
	}
	jobs := 4
	if v, err := strconv.Atoi(os.Getenv("JOBS")); err == nil && v > 0 {
		jobs = v
	}
	machine := ""
	if o, err := exec.Command("uname", "-m").Output(); err == nil { //nolint:gosec,noctx // as dump-shard
		machine = strings.TrimRight(string(o), "\n")
	}
	dumper := pyPathJoin(here, "oracle", "dump_build.lua")

	type row struct {
		name  string
		diff  []string
		total int
		err   error
	}
	rows := make([]row, len(names))
	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs)
	for i, n := range names {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			rows[i] = row{name: n}
			abs, _ := filepath.Abs(out)
			in := filepath.Join(abs, n, "in")
			dump := filepath.Join(abs, n, "dump")
			if err := os.MkdirAll(in, 0o750); err != nil {
				rows[i].err = err
				return
			}
			xml, _ := filepath.Abs(filepath.Join(builds, n+".xml"))
			if err := os.Symlink(xml, filepath.Join(in, n+".xml")); err != nil {
				rows[i].err = err
				return
			}
			script := fmt.Sprintf("ip link set lo up; cd '%s/src' && exec '%s' '%s' '%s' '%s'",
				pob, luajit, dumper, in, dump)
			cmd := exec.Command("setarch", machine, "-R", "unshare", "-rn", "--", "sh", "-c", script) //nolint:gosec,noctx // as dump-shard
			if log, err := cmd.CombinedOutput(); err != nil {
				_ = os.WriteFile(filepath.Join(abs, n, "dump.log"), log, 0o600) //nolint:gosec // inside the operator-supplied output dir
			}
			a, err := daReadOutput(filepath.Join(dump, n+".json"))
			if err != nil {
				rows[i].err = err
				return
			}
			c, err := daReadOutput(filepath.Join(committed, n+".json"))
			if err != nil {
				rows[i].err = err
				return
			}
			rows[i].diff, rows[i].total = daOutputDiff(c, a)
		}()
	}
	wg.Wait()

	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	fmt.Println("build\tdiffering\ttotal\tkeys")
	differ, failed := 0, 0
	for _, r := range rows {
		if r.err != nil {
			failed++
			fmt.Printf("%s\tERROR\t0\t%v\n", r.name, r.err)
			continue
		}
		if len(r.diff) > 0 {
			differ++
		}
		fmt.Printf("%s\t%d\t%d\t%s\n", r.name, len(r.diff), r.total, strings.Join(r.diff, ","))
	}
	fmt.Fprintf(os.Stderr, "dump-alone: %d build(s): %d reproduce, %d differ, %d failed\n",
		len(rows), len(rows)-differ-failed, differ, failed)
	if failed > 0 {
		return 5
	}
	return 0
}
