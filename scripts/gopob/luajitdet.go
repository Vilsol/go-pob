package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	ljCommit  = "eec7a8016c3381b949b5d84583800d05897fa960"
	ljVersion = "LuaJIT 2.1.1744318430"
	ljRuns    = 20
)

const ljProbe = `local s, so = {}, {}
for _, k in ipairs({"affliction_curse_effect", "affliction_curse_effect_small", "alpha", "beta",
                    "gamma", "delta", "epsilon", "zeta", "eta", "theta"}) do s[k] = true end
for k in pairs(s) do so[#so+1] = k end
local t, to, keys = {}, {}, {}
for i = 1, 10 do keys[i] = {}; t[keys[i]] = i end
for _, v in pairs(t) do to[#to+1] = v end
io.write(table.concat(so, ","), "|", table.concat(to, ","), "\n")
`

func ljFatal(code int, format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	return code
}

func ljDistinct(bin, probe string, runs int) int {
	machine := ""
	if o, err := exec.Command("uname", "-m").Output(); err == nil { //nolint:gosec,noctx // as in the original
		machine = strings.TrimRight(string(o), "\n")
	}
	seen := map[string]bool{}
	for range runs {
		cmd := exec.Command("setarch", machine, "-R", bin, probe) //nolint:gosec,noctx // as in the original
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		seen[string(out)] = true
	}
	return len(seen)
}

func ljVerify(bin string) int {
	if unix.Access(bin, unix.X_OK) != nil {
		return ljFatal(2, "no binary at %s — run without --verify first.", bin)
	}

	vcmd := exec.Command(bin, "-v") //nolint:gosec,noctx // the operator's own build output
	vout, _ := vcmd.CombinedOutput()
	got := ""
	if lines := pyUniversalLines(string(vout)); len(lines) > 0 {
		fields := strings.Fields(strings.TrimRight(lines[0], "\n"))
		if len(fields) > 2 {
			fields = fields[:2]
		}
		got = strings.Join(fields, " ")
	}
	if got != ljVersion {
		fmt.Fprintf(os.Stderr, "FATAL: built %s, expected %s.\n", got, ljVersion)
		fmt.Fprintln(os.Stderr, "       A different LuaJIT changes more than iteration order.")
		return 3
	}

	probe, err := os.CreateTemp("", "")
	if err != nil {
		return ljFatal(2, "cannot write the probe: %s", errnoText(err))
	}
	defer func() { _ = os.Remove(probe.Name()) }()
	if _, err := probe.WriteString(ljProbe); err != nil {
		return ljFatal(2, "cannot write the probe: %s", errnoText(err))
	}
	_ = probe.Close()

	distinct := ljDistinct(bin, probe.Name(), ljRuns)
	if distinct != 1 {
		return ljFatal(4, "%d distinct iteration orders across %d processes — NOT deterministic.",
			distinct, ljRuns)
	}

	if stockPath, err := exec.LookPath("luajit"); err == nil {
		stock := ljDistinct(stockPath, probe.Name(), ljRuns)
		if stock == 1 {
			fmt.Fprintln(os.Stderr, "WARNING: the stock luajit is ALSO deterministic under this probe.")
			fmt.Fprintln(os.Stderr, "         Either it is already built without the security hashing, or the probe")
			fmt.Fprintln(os.Stderr, "         has stopped discriminating. Do not trust the result until you know which.")
		} else {
			fmt.Printf("  control: stock luajit gives %d distinct order(s) over %d runs (expected >1)\n",
				stock, ljRuns)
		}
	}

	fmt.Printf("  verified: 1 iteration order over %d processes, %s\n", ljRuns, got)
	return 0
}

func buildLuajitDet(argv []string) int {
	bin := detLuajitBin()
	dest := filepath.Dir(filepath.Dir(bin))

	if len(argv) > 0 && argv[0] == "--verify" {
		return ljVerify(bin)
	}

	const miseTools = "conda:gcc conda:make@4.4.1"
	src := pyPathJoin(dest, "src-checkout")
	_ = os.MkdirAll(pyPathJoin(dest, "bin"), 0o750)                             //nolint:gosec // inside the repo, as in the original
	if fi, err := os.Stat(pyPathJoin(src, ".git")); err != nil || !fi.IsDir() { //nolint:gosec // inside the repo
		fmt.Println("cloning LuaJIT...")
		_ = exec.Command("git", "clone", "-q", //nolint:gosec,noctx // as in the original
			"https://github.com/LuaJIT/LuaJIT.git", src).Run()
	}
	_, _, _ = awGit(src, "fetch", "-q", "origin")
	_, _, _ = awGit(src, "checkout", "-q", ljCommit)

	fmt.Printf("building LuaJIT %s without security hashing...\n", ljCommit)
	script := fmt.Sprintf("cd '%s' && make clean >/dev/null 2>&1; make -j$(nproc) "+
		"XCFLAGS='-DLUAJIT_SECURITY_PRNG=0 -DLUAJIT_SECURITY_STRHASH=0 -DLUAJIT_SECURITY_STRID=0'",
		src)
	build := exec.Command("mise", append(append([]string{"x"}, //nolint:gosec,noctx // as in the original
		strings.Fields(miseTools)...), "--", "sh", "-c", script)...)
	logPath := pyPathJoin(dest, "build.log")
	lf, err := os.Create(logPath) //nolint:gosec // inside the repo, as in the original
	if err == nil {
		build.Stdout, build.Stderr = lf, lf
	}
	runErr := build.Run()
	if lf != nil {
		_ = lf.Close()
	}
	if runErr != nil {
		return ljFatal(5, "build failed, see %s", logPath)
	}

	if b, err := os.ReadFile(pyPathJoin(src, "src", "luajit")); err == nil { //nolint:gosec // the build output
		_ = os.WriteFile(bin, b, 0o700) //nolint:gosec // an executable is the point
	}
	if rc := ljVerify(bin); rc != 0 {
		return rc
	}
	fmt.Printf("built %s\n", bin)
	return 0
}

func detLuajitBin() string {
	//go-only: the python and bash originals predate worktrees and resolve from their own
	return filepath.Join(detLuajitRoot(), ".luajit-det", "bin", "luajit")
}

func detLuajitRoot() string {
	if v := os.Getenv("GOPOB_ROOT"); v != "" {
		return v
	}
	return mainWorktreeRoot()
}

func requireDetLuajit(args []string) int {
	who := "this task"
	if len(args) > 0 && args[0] != "" {
		who = args[0]
	}
	bin := detLuajitBin()
	if len(args) > 1 && args[1] == "--path" {
		fmt.Println(bin)
	}
	st, err := os.Stat(bin)
	if err == nil && !st.IsDir() && st.Mode().Perm()&0o111 != 0 {
		return 0
	}
	fmt.Fprintf(os.Stderr, "%s: %s is missing or not executable — build it with `task luajit-det`.\n"+
		"  A stock LuaJIT varies pairs() order per PROCESS, so the output is not reproducible and a\n"+
		"  ranking taken from it can move on its own. Refusing rather than producing one.\n", who, bin)
	return 1
}

func mainWorktreeRoot() string {
	out, err := exec.CommandContext(context.Background(),
		"git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err == nil {
		if p := strings.TrimSpace(string(out)); p != "" {
			return filepath.Dir(p)
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
