package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const dsUsage = "usage: dump_shard.sh <pob-dir> <repo-root> <builds-dir> <out-dir> <shards>"

var dsArgNames = []string{"<pob-dir>", "<repo-root>", "<builds-dir>", "<out-dir>", "<shards>"}

func dumpShard(argv []string) int {
	for i, name := range dsArgNames {
		if len(argv) < i+1 {
			fmt.Fprintf(os.Stderr, "FATAL: missing %s\n%s\n", name, dsUsage)
			return 2
		}
	}
	pob, here, builds, out, shardsArg := argv[0], argv[1], argv[2], argv[3], argv[4]
	niceLevel := dpEnvOr("NICE", "10")
	luajit := dpEnvOr("LUAJIT", "luajit")

	if !dpIsUint(shardsArg) {
		return dpFatal("<shards> must be a positive integer, not '%s'", shardsArg)
	}
	shards, err := strconv.Atoi(shardsArg)
	if err != nil || shards < 1 {
		return dpFatal("<shards> must be at least 1, not %s", shardsArg)
	}

	if fi, err := os.Stat(pyPathJoin(pob, "src")); err != nil || !fi.IsDir() { //nolint:gosec // an operator-supplied checkout path, as in the original
		return dpFatal("no %s/src", pob)
	}
	if fi, err := os.Stat(builds); err != nil || !fi.IsDir() { //nolint:gosec // an operator-supplied path, as in the original
		return dpFatal("no builds dir %s", builds)
	}

	if dpNonEmpty(out) {
		fmt.Fprintf(os.Stderr,
			"FATAL: %s exists and is NOT EMPTY — refusing to overwrite another run's output.\n", out)
		return 3
	}
	_ = os.MkdirAll(out, 0o750) //nolint:gosec // an operator-supplied output path, as in the original

	work, err := os.MkdirTemp("", "")
	if err != nil {
		return dpFatal("cannot create a work directory: %s", errnoText(err))
	}
	defer func() { _ = os.RemoveAll(work) }()

	ents, err := os.ReadDir(builds)
	if err != nil {
		return dpFatal("no builds dir %s", builds)
	}
	files := make([]string, 0, len(ents))
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".xml") {
			files = append(files, pyPathJoin(builds, e.Name()))
		}
	}
	sort.Strings(files)
	total := len(files)
	if total == 0 {
		fmt.Fprintf(os.Stderr,
			"FATAL: no .xml builds in %s — a sharded run over nothing would look like success.\n",
			builds)
		return 4
	}

	for i := range shards {
		_ = os.MkdirAll(filepath.Join(work, fmt.Sprintf("shard-%d", i)), 0o750)
	}
	for idx, f := range files {
		abs, err := filepath.Abs(f)
		if err == nil {
			if resolved, err := filepath.EvalSymlinks(abs); err == nil {
				abs = resolved
			}
		}
		link := filepath.Join(work, fmt.Sprintf("shard-%d", idx%shards), filepath.Base(f))
		_ = os.Symlink(abs, link)
	}

	fmt.Printf("dump_shard: %d builds over %d shard(s) -> %s\n", total, shards, out)
	fmt.Printf("dump_shard: FORCE_CLUSTER_KEY=%s EMPTY_MODCACHE=%s\n",
		dpEnvOr("FORCE_CLUSTER_KEY", "unset"), dpEnvOr("EMPTY_MODCACHE", "unset"))
	if strings.Contains(luajit, "luajit-det") {
		fmt.Printf("dump_shard: LUAJIT=%s (deterministic) — sharding is sound\n", luajit)
	} else {
		fmt.Fprintf(os.Stderr,
			"dump_shard: ⚠️ LUAJIT=%s is not the deterministic build — this dump is a MOSAIC.\n", luajit)
		fmt.Fprintln(os.Stderr,
			"dump_shard:   exploration only; build one with scripts/build_luajit_deterministic.sh")
	}

	machine := ""
	if o, err := exec.Command("uname", "-m").Output(); err == nil { //nolint:gosec,noctx // as in the original
		machine = strings.TrimRight(string(o), "\n")
	}
	type running struct {
		cmd *exec.Cmd
		log *os.File
	}
	started := make([]running, 0, shards)
	for i := range shards {
		script := fmt.Sprintf("ip link set lo up; cd '%s/src' && exec nice -n %s '%s' '%s' '%s' '%s'",
			pob, niceLevel, luajit, pyPathJoin(here, "oracle", "dump_build.lua"),
			filepath.Join(work, fmt.Sprintf("shard-%d", i)), out)
		cmd := exec.Command("setarch", machine, "-R", "unshare", "-rn", "--", "sh", "-c", script) //nolint:gosec,noctx // as in the original
		lf, err := os.Create(fmt.Sprintf("%s.shard-%d.log", out, i))                              //nolint:gosec // an operator-supplied output path
		if err == nil {
			cmd.Stdout, cmd.Stderr = lf, lf
		}
		_ = cmd.Start()
		started = append(started, running{cmd, lf})
	}
	fmt.Printf("dump_shard: %d process(es) launched\n", shards)

	rc := 0
	for _, r := range started {
		if err := r.cmd.Wait(); err != nil {
			rc = 1
		}
		if r.log != nil {
			_ = r.log.Close()
		}
	}

	got := 0
	if produced, err := os.ReadDir(out); err == nil {
		for _, e := range produced {
			if strings.HasSuffix(e.Name(), ".json") {
				got++
			}
		}
	}
	fmt.Printf("dump_shard: %d of %d dumped (exit %d)\n", got, total, rc)
	if got != total {
		fmt.Fprintf(os.Stderr,
			"dump_shard: ⚠️ SHORTFALL of %d — check %s.shard-*.log\n", total-got, out)
		return 5
	}
	return rc
}
