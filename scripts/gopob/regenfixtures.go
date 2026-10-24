package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var rfOrder = []string{
	"item_tools", "build_stats", "jewel_radius", "item_modlist",
	"item_data", "item_bases", "item_parse", "mod_parse",
}

type rfProbe struct {
	in  []string
	out string
}

var rfFixtures = map[string]rfProbe{
	"item_tools":   {[]string{"{OUT}/item_tools_cases.json"}, "{OUT}/item_tools_expected.json"},
	"build_stats":  {[]string{"{HERE}/testdata/many-builds"}, "{OUT}/many_builds_expected.json"},
	"jewel_radius": {nil, "{OUT}/jewel_radius_expected.json"},
	"item_modlist": {[]string{"{OUT}/item_parse_cases.json"}, "{OUT}/item_modlist_expected.json"},
	"item_data":    {[]string{"{OUT}/item_parse_cases.json"}, "{OUT}/item_data_expected.json"},
	"item_bases":   {[]string{"{OUT}/item_bases_cases.json"}, "{OUT}/item_bases_expected.json"},
	"item_parse":   {[]string{"{OUT}/item_parse_cases.json"}, "{OUT}/item_parse_expected.json"},
	"mod_parse":    {[]string{"{OUT}/item_parse_cases.json"}, "{OUT}/mod_parse_expected.json"},
}

const rfModBrowser = "custom_mod_browser"

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
		if fi, err := os.Stat(pyPathJoin(dir, "oracle", "build_stats.lua")); err == nil && !fi.IsDir() { //nolint:gosec // walking up from the CWD, as the original resolves from BASH_SOURCE
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
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

func rfRun(name, pob, here, luajit string, in []string, out string) bool {
	fmt.Printf("%-14s ", name)
	errPath := fmt.Sprintf("/tmp/fx-%s.err", name)
	ef, err := os.Create(errPath) //nolint:gosec // a fixed diagnostic path, as in the original
	if err != nil {
		fmt.Printf("FAILED — %s\n", errPath)
		return false
	}
	stage, err := newLuaStage(here)
	if err != nil {
		_ = ef.Close()
		fmt.Printf("FAILED — %s\n", err)
		return false
	}
	defer stage.remove()
	bin, err := stage.link("luajit", luajit)
	if err != nil {
		_ = ef.Close()
		fmt.Printf("FAILED — %s\n", err)
		return false
	}
	probe := []string{"-R", bin, stage.path(pyPathJoin("oracle", name+".lua"))}
	for _, p := range in {
		staged, err := stage.link(filepath.Base(p), p)
		if err != nil {
			_ = ef.Close()
			fmt.Printf("FAILED — %s\n", err)
			return false
		}
		probe = append(probe, staged)
	}
	probe = append(probe, stage.path(filepath.Base(out)))
	cmd := exec.Command("setarch", probe...) //nolint:gosec,noctx // as in the original
	cmd.Dir = pyPathJoin(pob, "src")
	cmd.Stdout, cmd.Stderr = nil, ef
	runErr := cmd.Run()
	if b, err := os.ReadFile(stage.path(filepath.Base(out))); err == nil { //nolint:gosec // the stage this run created
		if err := os.WriteFile(out, b, 0o644); err != nil && runErr == nil { //nolint:gosec // a committed fixture
			runErr = err
		}
	}
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

func rfRunModBrowser(pob, here, out string) bool {
	fmt.Printf("%-14s ", rfModBrowser)
	tmp, err := os.MkdirTemp("", "fx-mod-browser")
	if err != nil {
		fmt.Printf("FAILED — %s\n", err)
		return false
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	dump := pyPathJoin(tmp, "out.json")
	r := &rdRun{repo: here, pob: pob}
	if err := r.runLua("oracle/custom_mod_browser.lua", dump, "", modBrowserSearches...); err != nil {
		fmt.Printf("FAILED — %s\n", err)
		return false
	}
	var d struct {
		Searches json.RawMessage `json:"searches"`
	}
	b, err := os.ReadFile(dump) //nolint:gosec // this run's temp tree
	if err == nil {
		err = json.Unmarshal(b, &d)
	}
	if err == nil && len(d.Searches) == 0 {
		err = fmt.Errorf("%s has no searches", dump)
	}
	var buf bytes.Buffer
	if err == nil {
		err = json.Indent(&buf, []byte(`{"searches":`+string(d.Searches)+`}`), "", "\t")
	}
	if err == nil {
		buf.WriteByte('\n')
		err = os.WriteFile(pyPathJoin(out, "custom_mod_browser_search.json"), buf.Bytes(), 0o644) //nolint:gosec // a committed fixture
	}
	if err != nil {
		fmt.Printf("FAILED — %s\n", err)
		return false
	}
	fmt.Println("ok")
	return true
}

func regenFixtures(argv []string) int {
	here := rfHere()
	pob := pobPinDir()
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
		if name == rfModBrowser {
			if !rfRunModBrowser(pob, here, out) {
				fail = 1
			}
			continue
		}
		spec, ok := rfFixtures[name]
		if !ok {
			fmt.Fprintf(os.Stderr, "no such fixture: %s\n", name)
			return 2
		}
		fill := func(a string) string {
			return strings.ReplaceAll(strings.ReplaceAll(a, "{OUT}", out), "{HERE}", here)
		}
		in := make([]string, 0, len(spec.in))
		for _, a := range spec.in {
			in = append(in, fill(a))
		}
		if !rfRun(name, pob, here, luajit, in, fill(spec.out)) {
			fail = 1
		}
	}
	return fail
}
