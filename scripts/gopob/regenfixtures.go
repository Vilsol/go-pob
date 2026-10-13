package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var rfOrder = []string{
	"item_tools", "build_stats", "jewel_radius", "item_modlist",
	"item_data", "item_bases", "item_parse", "mod_parse",
}

var rfFixtures = map[string][]string{
	"item_tools":   {"{OUT}/item_tools_cases.json", "{OUT}/item_tools_expected.json"},
	"build_stats":  {"{HERE}/testdata/many-builds", "{OUT}/many_builds_expected.json"},
	"jewel_radius": {"{OUT}/jewel_radius_expected.json"},
	"item_modlist": {"{OUT}/item_parse_cases.json", "{OUT}/item_modlist_expected.json"},
	"item_data":    {"{OUT}/item_parse_cases.json", "{OUT}/item_data_expected.json"},
	"item_bases":   {"{OUT}/item_bases_cases.json", "{OUT}/item_bases_expected.json"},
	"item_parse":   {"{OUT}/item_parse_cases.json", "{OUT}/item_parse_expected.json"},
	"mod_parse":    {"{OUT}/item_parse_cases.json", "{OUT}/mod_parse_expected.json"},
}

func rfFatal(format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	return 2
}

func rfHere() string {
	if v := os.Getenv("GOPOB_ROOT"); v != "" {
		return v
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for dir := wd; ; {
		if fi, err := os.Stat(pyPathJoin(dir, "scripts", "regen_fixtures.sh")); err == nil && !fi.IsDir() { //nolint:gosec // walking up from the CWD, as the original resolves from BASH_SOURCE
			return dir
		}
		parent := pyPathStr(pyPathJoin(dir, ".."))
		if parent == dir || dir == "/" {
			return wd
		}
		dir = parent
	}
}

func rfShellUnquote(s string) string {
	if !strings.HasPrefix(s, "'") {
		return s
	}
	var b strings.Builder
	for i := 1; i < len(s); {
		if s[i] == '\'' {
			if strings.HasPrefix(s[i:], `'\''`) {
				b.WriteByte('\'')
				i += 4
				continue
			}
			break
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func rfApplyMiseLuaEnv(pob string) {
	cmd := exec.Command("mise", "env", "-s", "bash") //nolint:gosec,noctx // as in the original
	cmd.Dir = pob
	out, err := cmd.Output()
	if err != nil {
		return
	}
	for _, line := range pyUniversalLines(string(out)) {
		line = strings.TrimRight(line, "\n")
		for _, name := range []string{"LUA_PATH", "LUA_CPATH"} {
			prefix := "export " + name + "="
			if strings.HasPrefix(line, prefix) {
				_ = os.Setenv(name, rfShellUnquote(strings.TrimPrefix(line, prefix)))
			}
		}
	}
}

func rfRun(name, pob, here, luajit string, args []string) bool {
	fmt.Printf("%-14s ", name)
	errPath := fmt.Sprintf("/tmp/fx-%s.err", name)
	ef, err := os.Create(errPath) //nolint:gosec // a fixed diagnostic path, as in the original
	if err != nil {
		fmt.Printf("FAILED — %s\n", errPath)
		return false
	}
	probe := append([]string{"-R", luajit, pyPathJoin(here, "oracle", name+".lua")}, args...)
	cmd := exec.Command("setarch", probe...) //nolint:gosec,noctx // as in the original
	cmd.Dir = pyPathJoin(pob, "src")
	cmd.Stdout, cmd.Stderr = nil, ef
	runErr := cmd.Run()
	_ = ef.Close()
	if runErr == nil {
		fmt.Println("ok")
		return true
	}
	fmt.Printf("FAILED — %s\n", errPath)
	body, _ := os.ReadFile(errPath) //nolint:gosec // the file just written above
	lines := pyUniversalLines(string(body))
	if len(lines) > 3 {
		lines = lines[len(lines)-3:]
	}
	for _, l := range lines {
		fmt.Print(l)
		if !strings.HasSuffix(l, "\n") {
			fmt.Println()
		}
	}
	return false
}

func regenFixtures(argv []string) int {
	here := rfHere()
	pob := os.Getenv("POB_DIR")
	if pob == "" {
		home, _ := pyUserHome()
		pob = pyPathJoin(home, "PathOfBuilding")
	}
	luajit := os.Getenv("LUAJIT")
	if luajit == "" {
		luajit = detLuajitBin()
	}
	out := pyPathJoin(here, "testdata", "oracle")

	if !executableByMe(luajit) {
		return rfFatal("no deterministic luajit at %s — run `task luajit-det`", luajit)
	}
	if fi, err := os.Stat(pyPathJoin(pob, "src")); err != nil || !fi.IsDir() { //nolint:gosec // an operator-supplied checkout path, as in the original
		return rfFatal("no %s/src", pob)
	}

	rfApplyMiseLuaEnv(pob)
	if os.Getenv("LUA_PATH") == "" {
		return rfFatal("LUA_PATH not resolved from %s", pob)
	}

	targets := argv
	if len(targets) == 0 || targets[0] == "" {
		targets = rfOrder
	}

	fail := 0
	for _, name := range targets {
		spec, ok := rfFixtures[name]
		if !ok {
			fmt.Fprintf(os.Stderr, "no such fixture: %s\n", name)
			return 2
		}
		args := make([]string, 0, len(spec))
		for _, a := range spec {
			a = strings.ReplaceAll(a, "{OUT}", out)
			args = append(args, strings.ReplaceAll(a, "{HERE}", here))
		}
		if !rfRun(name, pob, here, luajit, args) {
			fail = 1
		}
	}
	return fail
}
