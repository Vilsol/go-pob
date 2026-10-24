package main

//	               actual calculation, whose own pairs() draws move real numbers (measured: 18

import (
	"crypto/md5" //nolint:gosec // a content fingerprint, matching the original's md5sum
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var pdmProbes = []struct {
	name string
	lua  string
	args []string
}{
	{"item_tools", "item_tools.lua", []string{"{TD}/item_tools_cases.json"}},
	{"many_builds", "build_stats.lua", []string{"{REPO}/testdata/many-builds"}},
	{"jewel_radius", "jewel_radius.lua", nil},
	{"item_modlist", "item_modlist.lua", []string{"{TD}/item_parse_cases.json"}},
	{"item_data", "item_data.lua", []string{"{TD}/item_parse_cases.json"}},
	{"item_bases", "item_bases.lua", []string{"{TD}/item_bases_cases.json"}},
	{"item_parse", "item_parse.lua", []string{"{TD}/item_parse_cases.json"}},
	{"mod_parse", "mod_parse.lua", []string{"{TD}/item_parse_cases.json"}},
}

const pdmCanary = `local out = arg[1]
local json = require("dkjson")
local rec = {}
for i = 1, 12 do rec["k" .. i] = "v" .. i end
local fh = assert(io.open(out, "w"))
fh:write(json.encode({ rec }, { indent = true }))
fh:close()
`

func pdmRefuse(lines ...string) int {
	for _, l := range lines {
		fmt.Fprintln(os.Stderr, l)
	}
	return 2
}

func pdmDistinct(paths []string) int {
	seen := map[string]bool{}
	for _, p := range paths {
		b, err := os.ReadFile(p) //nolint:gosec // a path this tool just wrote
		if err != nil {
			continue
		}
		sum := md5.Sum(b) //nolint:gosec // a content fingerprint, as in the original
		seen[hex.EncodeToString(sum[:])] = true
	}
	return len(seen)
}

func probeDeterminism(argv []string) int {
	here := pdmHere()
	repo := pyPathStr(pyPathJoin(here, ".."))

	pob := pobPinDir()
	if fi, err := os.Stat(pyPathJoin(pob, "src")); err != nil || !fi.IsDir() { //nolint:gosec // an operator-supplied checkout path, as in the original
		return pdmRefuse(fmt.Sprintf("no PoB checkout at %s (set POB_DIR)", pob))
	}
	td := pyPathJoin(repo, "testdata", "oracle")

	probe := "item_tools"
	if len(argv) > 0 && argv[0] != "" {
		probe = argv[0]
	}
	runsArg := "8"
	if len(argv) > 1 && argv[1] != "" {
		runsArg = argv[1]
	}
	if !dpIsUint(runsArg) {
		return pdmRefuse(fmt.Sprintf("REFUSED: <runs> must be a positive integer, not '%s'", runsArg))
	}
	runs, err := strconv.Atoi(runsArg)
	if err != nil || runs < 2 {
		return pdmRefuse("REFUSED: <runs> must be at least 2; one process cannot show a per-process draw")
	}

	work, err := os.MkdirTemp("", "")
	if err != nil {
		//go-only: the shell writes `WORK="$(mktemp -d)"` under `set -u` WITHOUT `set -e`, so a
		return pdmRefuse(fmt.Sprintf("REFUSED: cannot create a work directory: %s", errnoText(err)))
	}
	defer func() { _ = os.RemoveAll(work) }()

	luajit := os.Getenv("LUAJIT")
	var runCmd []string
	if luajit != "" {
		for _, v := range []struct{ name, expr string }{
			{"LUA_PATH", "io.write(package.path)"},
			{"LUA_CPATH", "io.write(package.cpath)"},
		} {
			c := exec.Command("mise", "x", "--", "luajit", "-e", v.expr) //nolint:gosec,noctx // as in the original
			c.Dir = pyPathJoin(pob, "src")
			c.Stderr = os.Stderr
			out, _ := c.Output()
			_ = os.Setenv(v.name, string(out))
		}
		machine := ""
		if o, err := exec.Command("uname", "-m").Output(); err == nil { //nolint:gosec,noctx // as in the original
			machine = strings.TrimRight(string(o), "\n")
		}
		runCmd = []string{"setarch", machine, "-R", cdrRealpathM(luajit)}
	} else {
		runCmd = []string{"mise", "x", "--", "luajit"}
	}

	runOnce := func(args ...string) {
		full := append(append([]string{}, runCmd[1:]...), args...)
		c := exec.Command(runCmd[0], full...) //nolint:gosec,noctx // as in the original
		c.Dir = pyPathJoin(pob, "src")
		_ = c.Run()
	}

	canaryLua := pyPathJoin(work, "canary.lua")
	if err := os.WriteFile(canaryLua, []byte(pdmCanary), 0o600); err != nil {
		//go-only: the shell's `cat >"$WORK/canary.lua"` failing prints a redirect error and the
		return pdmRefuse(fmt.Sprintf("REFUSED: cannot write the canary: %s", errnoText(err)))
	}
	canaryOut := make([]string, 0, runs)
	for i := 1; i <= runs; i++ {
		out := pyPathJoin(work, fmt.Sprintf("canary-%d.json", i))
		runOnce(canaryLua, out)
		canaryOut = append(canaryOut, out)
	}
	canary := pdmDistinct(canaryOut)
	if luajit == "" && canary <= 1 {
		return pdmRefuse(
			fmt.Sprintf("REFUSED: the canary produced %d distinct md5 of %d runs.", canary, runs),
			"         Stock LuaJIT must vary here; a green result would prove nothing.")
	}
	fmt.Printf("canary: %d distinct md5 of %d runs (must be >1 on stock luajit)\n", canary, runs)

	known := map[string]bool{"all": true}
	for _, p := range pdmProbes {
		known[p.name] = true
	}
	if !known[probe] {
		names := make([]string, 0, len(pdmProbes))
		for _, p := range pdmProbes {
			names = append(names, p.name)
		}
		return pdmRefuse(fmt.Sprintf("unknown probe: %s (one of %s all)", probe, strings.Join(names, " ")))
	}

	failed := 0
	for _, p := range pdmProbes {
		if probe != "all" && probe != p.name {
			continue
		}
		outs := make([]string, 0, runs)
		ok := true
		for i := 1; i <= runs; i++ {
			out := pyPathJoin(work, fmt.Sprintf("%s-%d.json", p.name, i))
			args := make([]string, 0, len(p.args)+2)
			args = append(args, pyPathJoin(here, p.lua))
			for _, a := range p.args {
				a = strings.ReplaceAll(a, "{TD}", td)
				args = append(args, strings.ReplaceAll(a, "{REPO}", repo))
			}
			args = append(args, out)
			runOnce(args...)
			if _, err := os.Stat(out); err != nil { //nolint:gosec // a path this tool just wrote
				fmt.Fprintf(os.Stderr, "%s: run %d FAILED\n", p.name, i)
				ok = false
				break
			}
			outs = append(outs, out)
		}
		if !ok {
			continue
		}
		n := pdmDistinct(outs)
		fmt.Printf("%-14s %d distinct md5 of %d runs", p.name, n, runs)
		if n == 1 {
			fmt.Println("   OK")
		} else {
			fmt.Println("   NOT REPRODUCIBLE")
			failed = 1
		}
		for _, o := range outs {
			_ = os.Remove(o)
		}
	}
	return failed
}

func pdmHere() string {
	if v := os.Getenv("GOPOB_ROOT"); v != "" {
		return pyPathJoin(v, "oracle")
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for dir := wd; ; {
		if fi, err := os.Stat(pyPathJoin(dir, "oracle", "probe_determinism.sh")); err == nil && !fi.IsDir() { //nolint:gosec // walking up from the CWD, as the original resolves from $0
			return pyPathJoin(dir, "oracle")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd
		}
		dir = parent
	}
}
