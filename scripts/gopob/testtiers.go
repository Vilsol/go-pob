package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

const (
	tiListFile    = "scripts/gopob/operator_slow_tests.txt"
	tiSlowSeconds = 10.0
)

var reTIName = regexp.MustCompile(`^(Test|Fuzz)\w*$`)

type tiEntry struct {
	pkg     string
	name    string
	seconds float64
}

type tiEvent struct {
	Action     string
	Package    string
	ImportPath string
	Test       string
	Elapsed    float64
	Output     string
}

func tiRefuse(msg string) int {
	fmt.Fprintln(os.Stderr, "REFUSING: test-tiers: "+msg)
	return root.RefusalCode
}

func tiReadList(path string) ([]tiEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("slow list: %w", err)
	}
	defer f.Close()
	var out []tiEntry
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 || !reTIName.MatchString(fields[1]) {
			return nil, fmt.Errorf("%s:%d: want `<package> <TestName> <seconds>`, got %q", path, n, line)
		}
		s, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, n, err)
		}
		out = append(out, tiEntry{fields[0], fields[1], s})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("slow list: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s lists no tests", path)
	}
	return out, nil
}

func tiNamesRegex(entries []tiEntry) string {
	seen := map[string]bool{}
	var names []string
	for _, e := range entries {
		if !seen[e.name] {
			seen[e.name] = true
			names = append(names, e.name)
		}
	}
	sort.Strings(names)
	return "^(" + strings.Join(names, "|") + ")$"
}

func tiScan(r io.Reader, w io.Writer) (map[[2]string]float64, int, error) {
	times := map[[2]string]float64{}
	outputs := map[[2]string][]string{}
	failed := 0
	dec := json.NewDecoder(r)
	for {
		var ev tiEvent
		if err := dec.Decode(&ev); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return times, failed, fmt.Errorf("go test -json: %w", err)
		}
		pkg := atShort(ev.Package)
		top := ev.Test
		if i := strings.IndexByte(top, '/'); i >= 0 {
			top = top[:i]
		}
		key := [2]string{pkg, top}
		switch ev.Action {
		case "build-output":
			fmt.Fprint(w, ev.Output)
		case "output":
			outputs[key] = append(outputs[key], ev.Output)
		case "pass", "fail", "skip":
			if ev.Test == "" {
				if ev.Action == "fail" {
					failed++
					fmt.Fprint(w, strings.Join(outputs[key], ""))
					fmt.Fprintf(w, "FAIL\t%s\t%.1fs\n", pkg, ev.Elapsed)
				} else {
					fmt.Fprintf(w, "ok  \t%s\t%.1fs\n", pkg, ev.Elapsed)
				}
				delete(outputs, key)
				continue
			}
			times[key] = max(times[key], ev.Elapsed)
			if ev.Test != top {
				continue
			}
			if ev.Action == "fail" {
				fmt.Fprint(w, strings.Join(outputs[key], ""))
			}
			delete(outputs, key)
		}
	}
	return times, failed, nil
}

func tiRun(repo string, pkgs []string, goArgs []string, w io.Writer) (map[[2]string]float64, int, error) {
	args := append(append([]string{"test", "-json", "-timeout", "60m"}, pkgs...), goArgs...)
	fmt.Fprintln(w, "OPERATOR=true CORPUS=true go "+strings.Join(args, " "))
	cmd := exec.CommandContext(context.Background(), "go", args...) //nolint:gosec // go test with the caller's own arguments
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "OPERATOR=true", "CORPUS=true")
	cmd.Stderr = w
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, 0, fmt.Errorf("go test: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, 0, fmt.Errorf("go test: %w", err)
	}
	times, failed, scanErr := tiScan(stdout, w)
	waitErr := cmd.Wait()
	if scanErr != nil {
		return times, failed, scanErr
	}
	var ee *exec.ExitError
	if waitErr != nil && !errors.As(waitErr, &ee) {
		return times, failed, fmt.Errorf("go test: %w", waitErr)
	}
	if waitErr != nil && failed == 0 {
		failed = 1
	}
	return times, failed, nil
}

func tiRecord(path string, logs []string) int {
	if len(logs) == 0 {
		//go-only: a new tool with no original; record has nothing to read.
		return tiRefuse("record needs the go test -json logs of a full operator run")
	}
	all := map[[2]string]float64{}
	for _, l := range logs {
		f, err := os.Open(l) //nolint:gosec // a log path the operator named
		if err != nil {
			return tiRefuse(err.Error())
		}
		times, _, err := tiScan(f, io.Discard)
		f.Close()
		if err != nil {
			return tiRefuse(l + ": " + err.Error())
		}
		if len(times) == 0 {
			//go-only: a new tool with no original; an empty log would write an empty list.
			return tiRefuse(l + ": no test results")
		}
		for k, s := range times {
			all[k] = s
		}
	}
	var entries []tiEntry
	for k, s := range all {
		if s >= tiSlowSeconds {
			entries = append(entries, tiEntry{k[0], k[1], s})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].pkg != entries[j].pkg {
			return entries[i].pkg < entries[j].pkg
		}
		return entries[i].name < entries[j].name
	})
	var b strings.Builder
	fmt.Fprintf(&b, "# Operator-suite tests at or above %gs each, which `task test-operator-fast` skips.\n", tiSlowSeconds)
	b.WriteString("# Rewrite with `gopob test-tiers record <go test -json log>...` from a full\n")
	b.WriteString("# `OPERATOR=true CORPUS=true go test -json ./...`; seconds are that run's, host load included.\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "%s %s %.1f\n", e.pkg, e.name, e.seconds)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return tiRefuse(err.Error())
	}
	fmt.Printf("wrote %d slow tests to %s\n", len(entries), tiListFile)
	return 0
}

func testTiers(args []string) int {
	if len(args) == 0 {
		//go-only: a new tool with no original.
		return tiRefuse("usage: test-tiers fast|affected|slow [go test args] | record <log>...")
	}
	repo, err := root.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return root.RefusalCode
	}
	listPath := filepath.Join(repo, tiListFile)
	if args[0] == "record" {
		return tiRecord(listPath, args[1:])
	}
	entries, err := tiReadList(listPath)
	if err != nil {
		return tiRefuse(err.Error())
	}
	slow := map[[2]string]bool{}
	for _, e := range entries {
		slow[[2]string{e.pkg, e.name}] = true
	}
	var times map[[2]string]float64
	var failed int
	switch args[0] {
	case "fast":
		times, failed, err = tiRun(repo, []string{"./..."}, append([]string{"-skip", tiNamesRegex(entries)}, args[1:]...), os.Stdout)
	case "affected":
		_, dirs, rc := atAffected()
		if rc != 0 || len(dirs) == 0 {
			return rc
		}
		times, failed, err = tiRun(repo, dirs, append([]string{"-skip", tiNamesRegex(entries)}, args[1:]...), os.Stdout)
	case "slow":
		times, failed, err = tiRun(repo, []string{"./..."}, append([]string{"-run", tiNamesRegex(entries)}, args[1:]...), os.Stdout)
	default:
		//go-only: a new tool with no original.
		return tiRefuse(fmt.Sprintf("unknown mode %q; want fast, affected, slow or record", args[0]))
	}
	if err != nil {
		return tiRefuse(err.Error())
	}
	var crossed []string
	for k, s := range times {
		if s >= tiSlowSeconds && !slow[k] {
			crossed = append(crossed, fmt.Sprintf("  %s %s %.1fs", k[0], k[1], s))
		}
	}
	sort.Strings(crossed)
	fmt.Printf("test-tiers %s: %d tests, %d failing packages\n", args[0], len(times), failed)
	if len(crossed) > 0 {
		fmt.Printf("NEW SLOW: %d tests took >= %gs and are not in %s (host load inflates this; re-check before adding):\n%s\n",
			len(crossed), tiSlowSeconds, tiListFile, strings.Join(crossed, "\n"))
	}
	if failed > 0 {
		return 1
	}
	return 0
}
