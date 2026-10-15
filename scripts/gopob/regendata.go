package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type rdGenerator struct {
	name, out            string
	emit                 string
	lua                  string
	dump                 string
	luaArgs              []string
	source               rdSource
	stdinGz              string
	emitArgs             []string
	emitArgsAreRepoPaths bool
	post                 string
	postArgs             []string
	twoTree              bool
	needsPob             bool
	note                 string
}

type rdSource int

const (
	rdUnclassified rdSource = iota
	rdPoBAuthored
	rdGameData
	rdGameDataFromPoB
	rdGameDataOverlay
	rdPoBOracle
)

type rdRefusalError struct{ msg string }

func (e *rdRefusalError) Error() string { return e.msg }

func rdRefuse(format string, args ...any) error {
	return &rdRefusalError{msg: fmt.Sprintf(format, args...)}
}

var rdStderr io.Writer = os.Stderr

func rdStderrForTest(w io.Writer) func() {
	prev := rdStderr
	rdStderr = w
	return func() { rdStderr = prev }
}

func rdRepo() string {
	if v := os.Getenv("GOPOB_ROOT"); v != "" {
		return v
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for _, anchor := range [][]string{{"scripts", "port_manifest.txt"}, {"go.mod"}} {
		for dir := wd; ; {
			if fi, err := os.Stat(pyPathJoin(append([]string{dir}, anchor...)...)); err == nil && !fi.IsDir() { //nolint:gosec // walking up from the CWD
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return wd
}

func rdExpandUser(p string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}
	home := os.Getenv("HOME")
	if home == "" {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return home + p[1:]
	}
	return p
}

type rdRun struct {
	repo, pob string
	check     bool
	only      string
	exitCode  int
	tmpRoot   string
}

func (r *rdRun) redactTmp(parts []string) []string {
	if r.tmpRoot == "" {
		return parts
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.ReplaceAll(p, r.tmpRoot, "<tmp>"))
	}
	return out
}

func (r *rdRun) runCmd(argv []string, dir string, env []string, stdout *os.File, stdin *os.File) error {
	cmd := exec.Command(argv[0], argv[1:]...) //nolint:gosec,noctx // the generator table's own pipeline
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if stdout != nil {
		cmd.Stdout = stdout
	}
	if stdin != nil {
		cmd.Stdin = stdin
	}
	if err := cmd.Run(); err != nil {
		return rdRefuse("command failed (%d): %s", rdExitCode(err),
			strings.Join(r.redactTmp(argv), " "))
	}
	return nil
}

func rdExitCode(err error) int {
	var ee *exec.ExitError
	if asExitError(err, &ee) {
		return ee.ExitCode()
	}
	return 1
}

func rdLuajitEnv(pob string) []string {
	env := os.Environ()
	src := pyPathJoin(pob, "src")
	for _, q := range []struct{ expr, key string }{
		{"io.write(package.path)", "LUA_PATH"},
		{"io.write(package.cpath)", "LUA_CPATH"},
	} {
		cmd := exec.Command("mise", "x", "--", "luajit", "-e", q.expr) //nolint:gosec,noctx // as in the original
		cmd.Dir = src
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			env = append(env, q.key+"="+string(out))
		}
	}
	return env
}

func (r *rdRun) luajitCmd() []string {
	det := detLuajitBin()
	if _, err := os.Stat(det); err == nil { //nolint:gosec // inside the repo
		return []string{"setarch", "-R", det}
	}
	fmt.Fprintln(os.Stderr,
		"  ⚠️ .luajit-det is absent -- falling back to stock luajit. pairs() order varies per\n"+
			"     process, so --check may report spurious changes. Build it with `task luajit-det`.")
	return []string{"mise", "x", "--", "luajit"}
}

func (r *rdRun) runLua(script, out, dataPath string, args ...string) error {
	lj := r.luajitCmd()
	env := rdLuajitEnv(r.pob)
	logical := append(append(append([]string{}, lj...), pyPathJoin(r.repo, script), out), args...)

	stage, err := newLuaStage(r.repo)
	if err != nil {
		//go-only: the stage has no counterpart in the original, which passed the repo paths through.
		return rdRefuse("%s", err)
	}
	defer stage.remove()
	if lj[0] == "setarch" {
		if lj[2], err = stage.link("luajit", lj[2]); err != nil {
			//go-only: as above, the stage.
			return rdRefuse("%s", err)
		}
	}
	if dataPath != "" {
		staged, err := stage.link("data.json", dataPath)
		if err != nil {
			//go-only: as above, the stage.
			return rdRefuse("%s", err)
		}
		env = append(env, "GAMEDATA_DUMP="+staged)
	}

	staged := append(append(append([]string{}, lj...), stage.path(script), stage.path("out.json")), args...)
	cmd := exec.Command(staged[0], staged[1:]...) //nolint:gosec,noctx // the generator table's own pipeline
	cmd.Dir = pyPathJoin(r.pob, "src")
	cmd.Env = env
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		return rdRefuse("command failed (%d): %s", rdExitCode(err),
			strings.Join(r.redactTmp(logical), " "))
	}
	b, err := os.ReadFile(stage.path("out.json")) //nolint:gosec // the stage this run created
	if err != nil {
		return nil //nolint:nilerr // see above
	}
	if err := os.WriteFile(out, b, 0o600); err != nil { //nolint:gosec // this run's temp tree
		//go-only: as above, the stage.
		return rdRefuse("could not write %s: %s", out, errnoText(err))
	}
	return nil
}

func rdGopob() []string {
	if v := os.Getenv("GOPOB_BIN"); v != "" {
		return []string{v}
	}
	exe, err := os.Executable()
	if err != nil {
		return []string{"go", "run", "./scripts/gopob"}
	}
	return []string{exe}
}

func rdSub(emit string) string {
	base := strings.TrimSuffix(filepath.Base(emit), ".py")
	if base == "skill_stat_map_emit" {
		return "skill-stat-maps-emit"
	}
	return strings.ReplaceAll(base, "_", "-")
}

func (r *rdRun) verifyPin() (string, error) {
	argv := append(rdGopob(), "pob-pin")
	cmd := exec.Command(argv[0], argv[1:]...) //nolint:gosec,noctx
	cmd.Dir = r.repo
	cmd.Env = append(os.Environ(), "POB_DIR="+r.pob)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	if err := cmd.Run(); err != nil {
		return "", rdRefuse("the PoB checkout does not match scripts/pob_pin.json:\n%s\n\n"+
			"Regenerating against the wrong checkout rewrites every table with plausible wrong "+
			"data. Fix the checkout, or re-pin deliberately.", strings.TrimSpace(buf.String()))
	}
	b, err := os.ReadFile(pyPathJoin(r.repo, "scripts", "pob_pin.json")) //nolint:gosec // this repo's own pin
	if err != nil {
		return "unknown", nil //nolint:nilerr // "unknown" is the original's fallback, not an error
	}
	var pin map[string]any
	if json.Unmarshal(b, &pin) != nil {
		return "unknown", nil //nolint:nilerr // same fallback: a malformed pin file is not a refusal
	}
	if v, ok := pin["content_sha256"].(string); ok {
		return v, nil
	}
	return "unknown", nil
}

func (r *rdRun) generate(g rdGenerator, outRoot, tmp string) error {
	outPath := pyPathJoin(outRoot, g.out)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o750); err != nil {
		return rdRefuse("could not create %s: %s", filepath.Dir(outPath), errnoText(err))
	}

	if g.stdinGz != "" {
		src := pyPathJoin(outRoot, g.stdinGz)
		if _, err := os.Stat(src); err != nil { //nolint:gosec // this repo's own testdata
			src = pyPathJoin(r.repo, g.stdinGz)
		}
		dst, err := os.Create(outPath) //nolint:gosec // the declared artifact path
		if err != nil {
			//go-only: the original writes `with open(out_path, "wb")` unguarded, so an
			return rdRefuse("could not write %s: %s", outPath, errnoText(err))
		}
		unzip := exec.Command("gzip", "-dc", src) //nolint:gosec,noctx // as in the original
		pipe, err := unzip.StdoutPipe()
		if err != nil {
			_ = dst.Close()
			return rdRefuse("could not read %s: %s", src, errnoText(err))
		}
		unzip.Stderr = os.Stderr
		if err := unzip.Start(); err != nil {
			_ = dst.Close()
			return rdRefuse("could not read %s: %s", src, errnoText(err))
		}
		ecmd := append(rdGopob(), rdSub(g.emit))
		emitCmd := exec.Command(ecmd[0], ecmd[1:]...) //nolint:gosec,noctx
		emitCmd.Dir = r.repo
		emitCmd.Stdin, emitCmd.Stdout, emitCmd.Stderr = pipe, dst, os.Stderr
		emitErr := emitCmd.Run()
		_ = unzip.Wait()
		_ = dst.Close()
		if emitErr != nil {
			return rdRefuse("command failed (%d): %s", rdExitCode(emitErr),
				strings.Join(r.redactTmp(ecmd), " "))
		}
		return r.runCmd([]string{"gofmt", "-w", outPath}, "", os.Environ(), nil, nil)
	}

	if g.twoTree {
		ours := pyPathJoin(tmp, "tree-ours.json")
		pobj := pyPathJoin(tmp, "tree-pob.json")
		if err := r.runCmd([]string{"go", "run", "./oracle/tree_ours_dump", ours}, r.repo, os.Environ(), nil, nil); err != nil {
			return err
		}
		if err := r.runLua("oracle/tree_stats.lua", pobj, ""); err != nil {
			return err
		}
		second := pyPathJoin(outRoot, "calculator", "tree_stat_text_generated.go")
		if err := os.MkdirAll(filepath.Dir(second), 0o750); err != nil {
			return rdRefuse("could not create %s: %s", filepath.Dir(second), errnoText(err))
		}
		if err := r.runCmd(append(rdGopob(), rdSub(g.emit), ours, pobj, outPath, second),
			r.repo, os.Environ(), nil, nil); err != nil {
			return err
		}
		return r.runCmd([]string{"gofmt", "-w", outPath, second}, "", os.Environ(), nil, nil)
	}

	jsonPath := pyPathJoin(tmp, g.name+".json")
	var dataPath string
	if g.lua != "" && g.dump != "" {
		dataPath = pyPathJoin(tmp, g.name+"-data.json")
		if err := r.runCmd([]string{"go", "run", "./scripts/gamedatadump", g.dump, dataPath}, r.repo, append(os.Environ(), "POB_DIR="+r.pob), nil, nil); err != nil {
			return err
		}
	}
	if g.lua != "" {
		if err := r.runLua(g.lua, jsonPath, dataPath, g.luaArgs...); err != nil {
			return err
		}
	}
	if g.dump != "" && g.lua == "" {
		if err := r.runCmd([]string{"go", "run", "./scripts/gamedatadump", g.dump, jsonPath}, r.repo, append(os.Environ(), "POB_DIR="+r.pob), nil, nil); err != nil {
			return err
		}
	}

	if strings.HasSuffix(g.out, ".json.gz") {
		dst, err := os.Create(outPath) //nolint:gosec // the declared artifact path
		if err != nil {
			//go-only: as above — the original's `with open(out_path, "wb")` is unguarded and
			return rdRefuse("could not write %s: %s", outPath, errnoText(err))
		}
		err = r.runCmd([]string{"gzip", "-9", "-n", "-c", jsonPath}, "", os.Environ(), dst, nil)
		_ = dst.Close()
		return err
	}

	if g.emit != "" {
		env := append(os.Environ(), "POB_DIR="+r.pob)
		args := append(rdGopob(), rdSub(g.emit))
		if g.lua != "" || g.dump != "" {
			args = append(args, jsonPath, outPath)
		} else {
			args = append(args, outPath)
		}
		for _, a := range g.emitArgs {
			if g.emitArgsAreRepoPaths {
				a = pyPathJoin(r.repo, a)
			}
			args = append(args, a)
		}
		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec,noctx // the generator table's own emitter
		cmd.Dir, cmd.Env = r.repo, env
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return rdRefuse("emitter failed (%d): %s", rdExitCode(err), rdSub(g.emit))
		}

		if g.post != "" {
			pargs := append(append(rdGopob(), rdSub(g.post), outPath), g.postArgs...)
			pcmd := exec.Command(pargs[0], pargs[1:]...) //nolint:gosec,noctx // the table's own post-processor
			pcmd.Dir, pcmd.Env = r.repo, env
			pcmd.Stdout, pcmd.Stderr = os.Stdout, os.Stderr
			if err := pcmd.Run(); err != nil {
				return rdRefuse("post-processor failed (%d): %s", rdExitCode(err), rdSub(g.post))
			}
		}

		return r.runCmd([]string{"gofmt", "-w", outPath}, "", os.Environ(), nil, nil)
	}
	return nil
}

var rdNotGenerable = map[string]string{}

func (r *rdRun) manifestCoverage() ([]string, [][2]string) {
	var uncovered []string
	var handwritten [][2]string
	produced := map[string]bool{}
	for _, g := range rdGenerators {
		produced[g.out] = true
	}
	b, err := os.ReadFile(pyPathJoin(r.repo, "scripts", "port_manifest.txt")) //nolint:gosec // this repo's own manifest
	if err != nil {
		return nil, nil
	}
	for _, line := range insLines(b) {
		parts := strings.Fields(line)
		if len(parts) < 2 || strings.HasPrefix(line, "#") {
			continue
		}
		path, cls := parts[0], parts[1]
		src := ""
		if len(parts) > 2 {
			src = parts[2]
		}
		switch {
		case cls == "GENERATED" && !produced[path]:
			uncovered = append(uncovered, path)
		case cls == "DATA" && strings.HasSuffix(path, ".go") && rdNotGenerable[path] == "":
			handwritten = append(handwritten, [2]string{path, src})
		}
	}
	return uncovered, handwritten
}

func rdParseArgs(r *rdRun, argv []string, pobArg string) (string, bool) {
	for i := 0; i < len(argv); i++ {
		switch {
		case argv[i] == "--check":
			r.check = true
		case argv[i] == "--pob" && i+1 < len(argv):
			i++
			pobArg = argv[i]
		case strings.HasPrefix(argv[i], "--pob="):
			pobArg = strings.TrimPrefix(argv[i], "--pob=")
		case argv[i] == "--only" && i+1 < len(argv):
			i++
			r.only = argv[i]
		case strings.HasPrefix(argv[i], "--only="):
			r.only = strings.TrimPrefix(argv[i], "--only=")
		default:
			//go-only: the original's argparse rejects an unknown option itself; this loop is a
			fmt.Fprintf(rdStderr, "REFUSING: unknown option %q\n\n"+
				"usage: regen-data [--check] [--pob DIR] [--only NAME]\n\n"+
				"⚠️  Without --check this REGENERATES every committed data table.\n", argv[i])
			return pobArg, false
		}
	}
	return pobArg, true
}

func regenData(argv []string) int {
	r := &rdRun{repo: rdRepo()}
	pobArg := os.Getenv("POB_DIR")
	if pobArg == "" {
		pobArg = rdExpandUser("~/PathOfBuilding")
	}
	pobArg, ok := rdParseArgs(r, argv, pobArg)
	if !ok {
		return 3
	}
	r.pob = cdrRealpathM(rdExpandUser(pobArg))

	if err := r.main(); err != nil {
		var ref *rdRefusalError
		if ok := rdAsRefusal(err, &ref); ok {
			fmt.Fprintf(rdStderr, "REFUSING: %s\n", ref.msg)
			return 3
		}
		return 1
	}
	return r.exitCode
}

func rdAsRefusal(err error, out **rdRefusalError) bool {
	if r, ok := err.(*rdRefusalError); ok { //nolint:errorlint // the concrete type is the point
		*out = r
		return true
	}
	return false
}

func (r *rdRun) main() error {
	pin, err := r.verifyPin()
	if err != nil {
		return err
	}
	short := pin
	if len(short) > 16 {
		short = short[:16]
	}
	fmt.Printf("pin OK  content_sha256=%s…  %s\n", short, r.pob)

	gens := rdGenerators
	if r.only != "" {
		var picked []rdGenerator
		for _, g := range rdGenerators {
			if g.name == r.only {
				picked = append(picked, g)
			}
		}
		if len(picked) == 0 {
			names := make([]string, 0, len(rdGenerators))
			for _, g := range rdGenerators {
				names = append(names, g.name)
			}
			return rdRefuse("no generator named %s; have: %s", "'"+r.only+"'", strings.Join(names, ", "))
		}
		gens = picked
	}

	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		//go-only: tempfile.TemporaryDirectory() raises rather than refusing, so the original
		return rdRefuse("could not create a temp directory: %s", errnoText(err))
	}
	r.tmpRoot = tmp
	defer func() { _ = os.RemoveAll(tmp) }()

	outRoot := r.repo
	if r.check {
		outRoot = pyPathJoin(tmp, "out")
		for _, g := range gens {
			src := pyPathJoin(r.repo, g.out)
			dst := pyPathJoin(outRoot, g.out)
			if b, err := os.ReadFile(src); err == nil { //nolint:gosec // this repo's own artifacts
				if err := os.MkdirAll(filepath.Dir(dst), 0o750); err == nil {
					_ = os.WriteFile(dst, b, 0o600) //nolint:gosec // inside this run's temp tree
				}
			}
		}
	}

	var changed, unchanged int
	for _, g := range gens {
		var before []byte
		haveBefore := false
		if b, err := os.ReadFile(pyPathJoin(r.repo, g.out)); err == nil { //nolint:gosec // this repo's own artifacts
			before, haveBefore = b, true
		}

		if err := r.generate(g, outRoot, tmp); err != nil {
			return err
		}

		after, err := os.ReadFile(pyPathJoin(outRoot, g.out)) //nolint:gosec // the artifact just produced
		if err != nil {
			return rdRefuse("could not read %s: %s", pyPathJoin(outRoot, g.out), errnoText(err))
		}

		if haveBefore && bytes.Equal(before, after) {
			unchanged++
			fmt.Printf("  = %-22s %s\n", g.name, g.out)
		} else {
			delta := ""
			if haveBefore && !strings.HasSuffix(g.out, ".gz") {
				delta = fmt.Sprintf("  (%+d lines)", rdCountLines(after)-rdCountLines(before))
			}
			changed++
			fmt.Printf("  ~ %-22s %s%s\n", g.name, g.out, delta)
		}
	}

	uncovered, handwritten := r.manifestCoverage()

	fmt.Println()
	fmt.Printf("%d changed, %d unchanged, %d still hand-written, %d deliberately not generable\n",
		changed, unchanged, len(handwritten), len(rdNotGenerable))

	if len(uncovered) > 0 {
		fmt.Println()
		fmt.Println("⚠️ GENERATED per the manifest but NOT produced by this tool — it cannot")
		fmt.Println("   refresh them, so a pin bump leaves them stale:")
		for _, p := range uncovered {
			fmt.Printf("     %s\n", p)
		}
	}

	if len(handwritten) > 0 {
		fmt.Println()
		fmt.Println("STILL HAND-WRITTEN (each needs an emitter; a pin bump rots them silently):")
		for _, hw := range handwritten {
			fmt.Printf("     %-40s <- %s\n", hw[0], hw[1])
		}
	} else {
		fmt.Println()
		fmt.Println("No hand-written data table is awaiting an emitter.")
	}

	if len(rdNotGenerable) > 0 {
		fmt.Println()
		fmt.Println("DELIBERATELY NOT GENERABLE (a property of the data, not a TODO):")
		for _, p := range insSortedKeys(rdBoolKeys(rdNotGenerable)) {
			fmt.Printf("     %s\n", p)
		}
	}

	if r.check && changed > 0 {
		fmt.Println()
		fmt.Println("STALE: the committed artifacts differ from what the pinned PoB produces.")
		fmt.Println("Run `task data-regen`, review the diff, and commit it.")
		r.exitCode = 1
	}
	return nil
}

func rdBoolKeys(m map[string]string) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}

func rdCountLines(b []byte) int {
	n := 0
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '\n':
			n++
		case '\r':
			n++
			if i+1 < len(b) && b[i+1] == '\n' {
				i++
			}
		}
	}
	if len(b) > 0 && b[len(b)-1] != '\n' && b[len(b)-1] != '\r' {
		n++
	}
	return n
}
