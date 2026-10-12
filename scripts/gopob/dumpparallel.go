package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const dpUsage = "usage: dump_parallel.sh <pob-src-parent> <repo-root> <builds-dir> <out-base> <runs>"

var dpArgNames = []string{"<pob-src-parent>", "<repo-root>", "<builds-dir>", "<out-base>", "<runs>"}

func dpFatal(format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	return 2
}

func dpIsUint(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func dpNonEmpty(dir string) bool {
	ents, err := os.ReadDir(dir) //nolint:gosec // an operator-supplied output path, as in the original
	if err != nil {
		return false
	}
	return len(ents) > 0
}

func dpVisibleCount(dir string) int {
	ents, err := os.ReadDir(dir) //nolint:gosec // an operator-supplied output path, as in the original
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range ents {
		if !strings.HasPrefix(e.Name(), ".") {
			n++
		}
	}
	return n
}

func dpEnvOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func dumpParallel(argv []string) int {
	for i, name := range dpArgNames {
		if len(argv) < i+1 {
			fmt.Fprintf(os.Stderr, "FATAL: missing %s\n%s\n", name, dpUsage)
			return 2
		}
	}
	pob, here, builds, outBase, runsArg := argv[0], argv[1], argv[2], argv[3], argv[4]
	niceLevel := dpEnvOr("NICE", "10")
	luajit := dpEnvOr("LUAJIT", "luajit")

	if !dpIsUint(runsArg) {
		return dpFatal("<runs> must be a positive integer, not '%s'", runsArg)
	}
	runs, err := strconv.Atoi(runsArg)
	if err != nil || runs < 1 {
		return dpFatal("<runs> must be at least 1, not %s", runsArg)
	}

	if fi, err := os.Stat(pyPathJoin(pob, "src")); err != nil || !fi.IsDir() { //nolint:gosec // an operator-supplied checkout path, as in the original
		return dpFatal("no %s/src", pob)
	}
	if fi, err := os.Stat(builds); err != nil || !fi.IsDir() { //nolint:gosec // an operator-supplied path, as in the original
		return dpFatal("no builds dir %s", builds)
	}

	for i := 1; i <= runs; i++ {
		d := fmt.Sprintf("%s-%d", outBase, i)
		if dpNonEmpty(d) {
			fmt.Fprintf(os.Stderr,
				"FATAL: %s exists and is NOT EMPTY — refusing to overwrite another run's output.\n", d)
			fmt.Fprintln(os.Stderr, "       Remove it yourself, or pick a different OUT base.")
			return 3
		}
	}

	fmt.Printf("dump_parallel: %d runs, builds=%s, out=%s-{1..%d}\n", runs, builds, outBase, runs)
	fmt.Printf("dump_parallel: FORCE_CLUSTER_KEY=%s EMPTY_MODCACHE=%s\n",
		dpEnvOr("FORCE_CLUSTER_KEY", "unset"), dpEnvOr("EMPTY_MODCACHE", "unset"))

	machine := ""
	if out, err := exec.Command("uname", "-m").Output(); err == nil { //nolint:gosec,noctx // as in the original
		machine = strings.TrimRight(string(out), "\n")
	}

	type run struct {
		cmd *exec.Cmd
		log *os.File
	}
	started := make([]run, 0, runs)
	for i := 1; i <= runs; i++ {
		d := fmt.Sprintf("%s-%d", outBase, i)
		_ = os.MkdirAll(d, 0o750) //nolint:gosec // an operator-supplied output path; unchecked in the original, which runs without `set -e`
		script := fmt.Sprintf("ip link set lo up; cd '%s/src' && exec nice -n %s '%s' '%s' '%s' '%s'",
			pob, niceLevel, luajit, pyPathJoin(here, "oracle", "dump_build.lua"), builds, d)
		cmd := exec.Command("setarch", machine, "-R", "unshare", "-rn", "--", "sh", "-c", script) //nolint:gosec,noctx // as in the original
		logPath := fmt.Sprintf("%s-%d.log", outBase, i)
		lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) //nolint:gosec // an operator-supplied output path
		if err == nil {
			cmd.Stdout, cmd.Stderr = lf, lf
		}
		_ = cmd.Start()
		if cmd.Process != nil {
			_ = os.WriteFile(fmt.Sprintf("%s-%d.pid", outBase, i), //nolint:gosec // an operator-supplied output path
				[]byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0o600)
		}
		started = append(started, run{cmd, lf})
		fmt.Printf("dump_parallel: run %d -> %s\n", i, d)
	}

	rc := 0
	for i, r := range started {
		if err := r.cmd.Wait(); err != nil {
			rc = 1
			fmt.Fprintf(os.Stderr, "dump_parallel: run %d FAILED\n", i+1)
		}
		if r.log != nil {
			_ = r.log.Close()
		}
	}

	for i := 1; i <= runs; i++ {
		fmt.Printf("dump_parallel: run %d -> %d files\n", i, dpVisibleCount(fmt.Sprintf("%s-%d", outBase, i)))
	}
	fmt.Printf("dump_parallel: rc=%d\n", rc)
	return rc
}
