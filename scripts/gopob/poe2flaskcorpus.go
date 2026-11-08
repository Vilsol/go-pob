package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const p2fcUsage = "usage: poe2-flask-corpus <pob2-dir> <repo-root> <src-corpus-dir> <out-root> <version>  (env N, SEED, JOBS, LUAJIT)"

var p2fcSlotRe = regexp.MustCompile(`<Slot ([^>]*?)name="(Flask|Charm) [0-9]+"([^>]*?)/>`)

// p2fcActivate ticks every equipped flask and charm slot; the count is how many it ticked.
func p2fcActivate(xml string) (string, int) {
	n := 0
	out := p2fcSlotRe.ReplaceAllStringFunc(xml, func(s string) string {
		if strings.Contains(s, ` active="`) || strings.Contains(s, `itemId="0"`) {
			return s
		}
		n++
		return strings.TrimSuffix(s, "/>") + ` active="true"/>`
	})
	return out, n
}

func poe2FlaskCorpus(argv []string) int {
	if len(argv) != 5 {
		fmt.Fprintln(os.Stderr, p2fcUsage)
		return 2
	}
	pob, here, src, outRoot, version := argv[0], argv[1], argv[2], argv[3], argv[4]
	n, seed, jobs := 1000, uint64(1), 4
	if v, err := strconv.Atoi(os.Getenv("N")); err == nil && v > 0 {
		n = v
	}
	if v, err := strconv.ParseUint(os.Getenv("SEED"), 10, 64); err == nil {
		seed = v
	}
	if v, err := strconv.Atoi(os.Getenv("JOBS")); err == nil && v > 0 {
		jobs = v
	}
	dumper := filepath.Join(here, "oracle", "dump_build.lua")
	for _, f := range []string{filepath.Join(pob, "src", "HeadlessWrapper.lua"), dumper} {
		if _, err := os.Stat(f); err != nil { //nolint:gosec // an operator-supplied path
			return p2cRefuse(2, "no %s", f)
		}
	}
	if fi, err := os.Stat(src); err != nil || !fi.IsDir() { //nolint:gosec // an operator-supplied path
		return p2cRefuse(2, "no source corpus at %s", src)
	}
	luajit := os.Getenv("LUAJIT")
	if !strings.Contains(luajit, "luajit-det") {
		return p2cRefuse(2, "LUAJIT=%q is not the deterministic build; point it at .luajit-det/bin/luajit", luajit)
	}
	dirs := map[string]string{}
	for _, k := range []string{"corpus", "oracle", "dumps", "logs", "work"} {
		dirs[k] = filepath.Join(outRoot, k+"-flasks")
	}
	if abs, _ := filepath.Abs(src); abs == dirs["corpus"] {
		return p2cRefuse(2, "the source corpus is the output corpus")
	}
	stamp, err := p2cStamp(pob, filepath.Join(here, "scripts", "gopob", "poe2flaskcorpus.go"), dumper)
	if err != nil {
		return p2cRefuse(2, "cannot fingerprint the generator: %v", err)
	}
	gen := map[string]any{"version": version, "source": src, "n": n, "seed": seed}
	genJSON, _ := json.Marshal(gen)
	stamp += string(genJSON) + "\n"
	stampPath := filepath.Join(dirs["corpus"], ".poe2-flask-stamp")
	if dpNonEmpty(dirs["corpus"]) {
		old, _ := os.ReadFile(stampPath) //nolint:gosec // inside the operator-supplied output root
		if string(old) != stamp {
			return p2cRefuse(3, "%s is NOT EMPTY and was not built by this generator, seed, N, dumper and pin — refusing to mix runs", dirs["corpus"])
		}
		fmt.Fprintf(os.Stderr, "poe2-flask-corpus: resuming in %s\n", outRoot)
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o750); err != nil {
			return p2cRefuse(3, "cannot create %s: %v", d, err)
		}
	}
	if err := os.WriteFile(stampPath, []byte(stamp), 0o600); err != nil {
		return p2cRefuse(3, "cannot write %s: %v", stampPath, err)
	}
	machine := ""
	if o, err := exec.Command("uname", "-m").Output(); err == nil { //nolint:noctx // as poe2-corpus
		machine = strings.TrimRight(string(o), "\n")
	}
	r := p2cRunner{pob: pob, luajit: luajit, machine: machine, nice: dpEnvOr("NICE", "15"), memMax: dpEnvOr("MEMORY_MAX", "8G"),
		env: append(os.Environ(),
			"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
			"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")}
	start := time.Now()

	names, err := p2ccSample(src, n, seed)
	if err != nil {
		return p2cRefuse(4, "%v", err)
	}
	var generated []string
	ticked, without := 0, 0
	for _, nm := range names {
		raw, err := os.ReadFile(filepath.Join(src, nm+".xml")) //nolint:gosec // an operator-supplied corpus
		if err != nil {
			return p2cRefuse(4, "%v", err)
		}
		out, k := p2fcActivate(string(raw))
		if k == 0 {
			without++
			continue
		}
		if err := os.WriteFile(filepath.Join(dirs["corpus"], nm+".xml"), []byte(out), 0o600); err != nil { //nolint:gosec // inside the operator-supplied output root
			return p2cRefuse(3, "%v", err)
		}
		ticked += k
		generated = append(generated, nm)
	}
	gen["generated"], gen["slotsActivated"], gen["noFlaskOrCharm"] = len(generated), ticked, without
	fmt.Fprintf(os.Stderr, "poe2-flask-corpus: %d builds, %d slots activated, %d sampled builds carry none\n", len(generated), ticked, without)
	if len(generated) == 0 {
		return p2cRefuse(5, "no build generated")
	}

	work := filepath.Join(dirs["work"], "dumps")
	_ = os.RemoveAll(work)
	p2cParallel(generated, jobs, func(nm string) {
		if _, reason := pyLoadJSONObject(filepath.Join(dirs["dumps"], nm+".json")); reason == "" {
			return
		}
		in := filepath.Join(work, nm)
		if err := os.MkdirAll(in, 0o750); err != nil {
			return
		}
		_ = os.Symlink(filepath.Join(dirs["corpus"], nm+".xml"), filepath.Join(in, nm+".xml"))
		if log, err := r.run(dumper, in, dirs["dumps"]); err != nil {
			_ = os.WriteFile(filepath.Join(dirs["logs"], nm+".dump.log"), log, 0o600) //nolint:gosec // inside the output root
		}
	})
	_ = os.RemoveAll(work)

	gen["dumps"] = p2cReadDumps(dirs["dumps"], generated)
	tmp := filepath.Join(dirs["work"], "output.jsonl")
	if rc := extractOutputOracle([]string{dirs["dumps"], tmp}); rc != 0 {
		return p2cRefuse(6, "extract-output-oracle failed on %s", dirs["dumps"])
	}
	if err := p2cGzip(tmp, filepath.Join(dirs["oracle"], "output-"+version+"-flasks.jsonl.gz")); err != nil {
		return p2cRefuse(6, "cannot gzip %s: %v", tmp, err)
	}
	_ = os.Remove(tmp) //nolint:gosec // inside the operator-supplied output root
	gen["dumperSha256"], _ = p2cSha256File(dumper)
	if o, err := exec.Command("git", "-C", pob, "rev-parse", "HEAD").Output(); err == nil { //nolint:gosec,noctx // an operator-supplied checkout
		gen["pobHead"] = strings.TrimSpace(string(o))
	}
	gb, _ := json.MarshalIndent(gen, "", "  ")
	if err := os.WriteFile(filepath.Join(dirs["oracle"], "MANIFEST.json"), append(gb, '\n'), 0o600); err != nil {
		return p2cRefuse(3, "%v", err)
	}
	_, _ = os.Stdout.Write(append(gb, '\n'))
	fmt.Fprintf(os.Stderr, "poe2-flask-corpus: done in %s\n", time.Since(start).Round(time.Second))
	return 0
}
