package main

import (
	"archive/zip"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const p2cUsage = "usage: poe2-corpus <pob2-dir> <repo-root> <characters.zip> <out-root> <version>"

var p2cConvertHeader = []string{"character", "ok", "stage", "error", "groups", "guess", "guessSkill", "guessDPS",
	"pick", "pickSkill", "pickDPS", "sweepErrors", "timeless", "secs"}

var p2cDigits = regexp.MustCompile(`[0-9]+`)

func p2cRefuse(code int, format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "poe2-corpus: REFUSING: "+format+"\n", args...)
	return code
}

func p2cSha256File(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec // an operator-supplied path
	if err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func p2cExtract(zipPath, dir string, limit int) ([]string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("extract %s: %w", zipPath, err)
	}
	defer func() { _ = zr.Close() }()
	files := map[string]*zip.File{}
	var names []string
	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		if f.FileInfo().IsDir() || !strings.HasSuffix(base, ".json") || base != f.Name {
			continue
		}
		n := strings.TrimSuffix(base, ".json")
		files[n] = f
		names = append(names, n)
	}
	sort.Strings(names)
	if limit > 0 && limit < len(names) {
		names = names[:limit]
	}
	if err := os.MkdirAll(dir, 0o750); err != nil { //nolint:gosec // an operator-supplied output root
		return nil, fmt.Errorf("extract %s: %w", zipPath, err)
	}
	for _, n := range names {
		if _, err := os.Stat(filepath.Join(dir, n+".json")); err == nil { //nolint:gosec // an operator-supplied output root
			continue
		}
		rc, err := files[n].Open()
		if err != nil {
			return nil, fmt.Errorf("extract %s: %w", zipPath, err)
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("extract %s: %w", zipPath, err)
		}
		if err := os.WriteFile(filepath.Join(dir, n+".json"), b, 0o600); err != nil { //nolint:gosec // inside the operator-supplied output root
			return nil, fmt.Errorf("extract %s: %w", zipPath, err)
		}
	}
	return names, nil
}

type p2cRunner struct {
	pob, luajit, machine, nice, memMax string
	env                                []string
}

func (r p2cRunner) run(script string, args ...string) ([]byte, error) {
	quoted := make([]string, 0, len(args)+2)
	for _, a := range append([]string{r.luajit, script}, args...) {
		quoted = append(quoted, "'"+a+"'")
	}
	sh := fmt.Sprintf("ip link set lo up; cd '%s/src' && exec %s", r.pob, strings.Join(quoted, " "))
	cmd := exec.Command("systemd-run", "--user", "--scope", "--quiet", "-p", "MemoryMax="+r.memMax, "--", //nolint:gosec,noctx // operator-supplied paths, as dump-shard
		"nice", "-n", r.nice, "setarch", r.machine, "-R", "unshare", "-rn", "--", "sh", "-c", sh)
	cmd.Env = r.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s: %w", script, err)
	}
	return out, nil
}

func p2cStamp(pob, converter, dumper string) (string, error) {
	var b strings.Builder
	for _, p := range []string{converter, dumper} {
		h, err := p2cSha256File(p)
		if err != nil {
			return "", err
		}
		b.WriteString(filepath.Base(p) + " " + h + "\n")
	}
	o, err := exec.Command("git", "-C", pob, "rev-parse", "HEAD").Output() //nolint:gosec,noctx // an operator-supplied checkout
	if err != nil {
		return "", fmt.Errorf("pin HEAD of %s: %w", pob, err)
	}
	b.WriteString("pob " + strings.TrimSpace(string(o)) + "\n")
	return b.String(), nil
}

// a row the converter wrote is final, success or a Lua error; a "process" row is the runner's own and is retried
func p2cConverted(row string) bool {
	b, err := os.ReadFile(row) //nolint:gosec // inside the operator-supplied output root
	if err != nil {
		return false
	}
	f := strings.Split(strings.TrimRight(string(b), "\n"), "\t")
	return len(f) > 2 && f[2] != "process" && strings.Count(string(b), "\n") == 1
}

func p2cParallel(names []string, jobs int, fn func(string)) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs)
	done := 0
	var mu sync.Mutex
	for _, n := range names {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			fn(n)
			mu.Lock()
			done++
			if done%500 == 0 {
				fmt.Fprintf(os.Stderr, "poe2-corpus: %d/%d\n", done, len(names))
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
}

func p2cErrorKind(stage, msg string) string {
	if i := strings.Index(msg, ": "); i > 0 && strings.Contains(msg[:i], ".lua:") {
		msg = msg[i+2:]
	}
	if len(msg) > 90 {
		msg = msg[:90]
	}
	return stage + ": " + p2cDigits.ReplaceAllString(msg, "N")
}

func p2cGzip(src, dst string) error {
	in, err := os.Open(src) //nolint:gosec // inside the operator-supplied output root
	if err != nil {
		return fmt.Errorf("gzip %s: %w", dst, err)
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst) //nolint:gosec // inside the operator-supplied output root
	if err != nil {
		return fmt.Errorf("gzip %s: %w", dst, err)
	}
	zw, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		_ = out.Close()
		return fmt.Errorf("gzip %s: %w", dst, err)
	}
	if _, err := io.Copy(zw, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("gzip %s: %w", dst, err)
	}
	if err := zw.Close(); err != nil {
		_ = out.Close()
		return fmt.Errorf("gzip %s: %w", dst, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("gzip %s: %w", dst, err)
	}
	return nil
}

type p2cDumpStats struct {
	Dumped  int            `json:"dumped"`
	Missing int            `json:"missing"`
	Errors  map[string]int `json:"errors"`
	Usable  int            `json:"usable"`
}

func p2cReadDumps(dir string, names []string) p2cDumpStats {
	st := p2cDumpStats{Errors: map[string]int{}}
	for _, n := range names {
		m, reason := pyLoadJSONObject(filepath.Join(dir, n+".json"))
		if reason != "" {
			st.Missing++
			continue
		}
		st.Dumped++
		if e, ok := m["error"].(string); ok && e != "" {
			st.Errors[p2cErrorKind("dump", e)]++
			continue
		}
		if o, ok := m["output"].(map[string]any); ok {
			if life, ok := o["Life"].(json.Number); ok {
				if f, err := life.Float64(); err == nil && f > 0 {
					st.Usable++
				}
			}
		}
	}
	return st
}

func poe2Corpus(argv []string) int {
	if len(argv) != 5 {
		fmt.Fprintln(os.Stderr, p2cUsage)
		return 2
	}
	pob, here, zipPath, outRoot, version := argv[0], argv[1], argv[2], argv[3], argv[4]
	if fi, err := os.Stat(zipPath); err != nil || fi.IsDir() { //nolint:gosec // an operator-supplied path
		return p2cRefuse(2, "no character zip at %s", zipPath)
	}
	converter := filepath.Join(here, "oracle", "poe2_import.lua")
	dumper := filepath.Join(here, "oracle", "dump_build.lua")
	for _, p := range []string{filepath.Join(pob, "src", "HeadlessWrapper.lua"), converter, dumper} {
		if _, err := os.Stat(p); err != nil { //nolint:gosec // an operator-supplied path
			return p2cRefuse(2, "no %s", p)
		}
	}
	luajit := os.Getenv("LUAJIT")
	if !strings.Contains(luajit, "luajit-det") {
		return p2cRefuse(2, "LUAJIT=%q is not the deterministic build; point it at .luajit-det/bin/luajit", luajit)
	}
	stamp, err := p2cStamp(pob, converter, dumper)
	if err != nil {
		return p2cRefuse(2, "cannot fingerprint the converter: %v", err)
	}
	stampPath := filepath.Join(outRoot, ".poe2-corpus-stamp")
	if dpNonEmpty(outRoot) {
		old, _ := os.ReadFile(stampPath) //nolint:gosec // inside the operator-supplied output root
		if string(old) != stamp {
			return p2cRefuse(3, "%s is NOT EMPTY and was not built by this converter, dumper and pin — refusing to mix runs", outRoot)
		}
		fmt.Fprintf(os.Stderr, "poe2-corpus: resuming in %s; finished characters and dumps are kept\n", outRoot)
	} else if err := os.MkdirAll(outRoot, 0o750); err != nil { //nolint:gosec // an operator-supplied output root
		return p2cRefuse(3, "cannot create %s: %v", outRoot, err)
	} else if err := os.WriteFile(stampPath, []byte(stamp), 0o600); err != nil { //nolint:gosec // an operator-supplied output root
		return p2cRefuse(3, "cannot write %s: %v", stampPath, err)
	}
	jobs, limit := 6, 0
	if v, err := strconv.Atoi(os.Getenv("JOBS")); err == nil && v > 0 {
		jobs = v
	}
	if v, err := strconv.Atoi(os.Getenv("LIMIT")); err == nil && v > 0 {
		limit = v
	}
	machine := ""
	if o, err := exec.Command("uname", "-m").Output(); err == nil { //nolint:noctx // as dump-shard
		machine = strings.TrimRight(string(o), "\n")
	}
	r := p2cRunner{pob: pob, luajit: luajit, machine: machine, nice: dpEnvOr("NICE", "15"), memMax: dpEnvOr("MEMORY_MAX", "8G"),
		env: append(os.Environ(),
			"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
			"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")}

	start := time.Now()
	jsonDir := filepath.Join(outRoot, "json")
	names, err := p2cExtract(zipPath, jsonDir, limit)
	if err != nil {
		return p2cRefuse(4, "cannot extract %s: %v", zipPath, err)
	}
	if len(names) == 0 {
		return p2cRefuse(4, "%s holds no character .json at its root", zipPath)
	}
	fmt.Fprintf(os.Stderr, "poe2-corpus: %d characters, %d jobs\n", len(names), jobs)

	dirs := map[string]string{}
	for _, k := range []string{"corpus", "corpus-skillsoff", "rows", "logs", "work", "dumps", "dumps-skillsoff", "oracle"} {
		dirs[k] = filepath.Join(outRoot, k)
		if err := os.MkdirAll(dirs[k], 0o750); err != nil {
			return p2cRefuse(3, "cannot create %s: %v", dirs[k], err)
		}
	}

	p2cParallel(names, jobs, func(n string) {
		row := filepath.Join(dirs["rows"], n+".tsv")
		if p2cConverted(row) {
			return
		}
		_ = os.Remove(row)
		log, err := r.run(converter, filepath.Join(jsonDir, n+".json"), filepath.Join(dirs["corpus"], n+".xml"),
			filepath.Join(dirs["corpus-skillsoff"], n+".xml"), row)
		if err != nil {
			_ = os.WriteFile(filepath.Join(dirs["logs"], n+".convert.log"), log, 0o600) //nolint:gosec // inside the output root
		}
		if _, serr := os.Stat(row); serr != nil {
			_ = os.WriteFile(row, []byte(n+"\t0\tprocess\t"+strings.ReplaceAll(fmt.Sprint(err), "\t", " ")+"\n"), 0o600) //nolint:gosec // inside the output root
		}
	})

	var converted []string
	kinds := map[string]int{}
	timeless := 0
	var tsv strings.Builder
	tsv.WriteString(strings.Join(p2cConvertHeader, "\t") + "\n")
	for _, n := range names {
		b, _ := os.ReadFile(filepath.Join(dirs["rows"], n+".tsv")) //nolint:gosec // inside the output root
		line := strings.TrimRight(string(b), "\n")
		tsv.WriteString(line + "\n")
		f := strings.Split(line, "\t")
		if len(f) > 1 && f[1] == "1" {
			converted = append(converted, n)
			if len(f) > 12 && f[12] != "" && f[12] != "0" {
				timeless++
			}
			continue
		}
		stage, msg := "process", ""
		if len(f) > 2 {
			stage = f[2]
		}
		if len(f) > 3 {
			msg = f[3]
		}
		kinds[p2cErrorKind(stage, msg)]++
	}
	if err := os.WriteFile(filepath.Join(dirs["oracle"], "convert.tsv"), []byte(tsv.String()), 0o600); err != nil {
		return p2cRefuse(3, "cannot write convert.tsv: %v", err)
	}
	fmt.Fprintf(os.Stderr, "poe2-corpus: converted %d/%d in %s\n", len(converted), len(names), time.Since(start).Round(time.Second))
	if len(converted) == 0 {
		return p2cRefuse(5, "no character converted; see %s", dirs["logs"])
	}

	modes := []struct{ corpus, dumps string }{{"corpus", "dumps"}, {"corpus-skillsoff", "dumps-skillsoff"}}
	var dumpJobs []string
	for _, n := range converted {
		for i := range modes {
			dumpJobs = append(dumpJobs, strconv.Itoa(i)+":"+n)
		}
	}
	_ = os.RemoveAll(dirs["work"])
	p2cParallel(dumpJobs, jobs, func(j string) {
		mode, n := modes[j[0]-'0'], j[2:]
		if _, reason := pyLoadJSONObject(filepath.Join(dirs[mode.dumps], n+".json")); reason == "" {
			return
		}
		in := filepath.Join(dirs["work"], mode.dumps, n)
		if err := os.MkdirAll(in, 0o750); err != nil {
			return
		}
		_ = os.Symlink(filepath.Join(dirs[mode.corpus], n+".xml"), filepath.Join(in, n+".xml"))
		if log, err := r.run(dumper, in, dirs[mode.dumps]); err != nil {
			_ = os.WriteFile(filepath.Join(dirs["logs"], n+"."+mode.dumps+".log"), log, 0o600) //nolint:gosec // inside the output root
		}
	})
	_ = os.RemoveAll(dirs["work"])

	manifest := map[string]any{"version": version, "characters": len(names), "converted": len(converted),
		"convertFailures": kinds, "timelessCharacters": timeless, "luajit": luajit, "limit": limit}
	for i, m := range modes {
		st := p2cReadDumps(dirs[m.dumps], converted)
		manifest[m.dumps] = st
		suffix := []string{"", "-skillsoff"}[i]
		tmp := filepath.Join(outRoot, "output"+suffix+".jsonl")
		if rc := extractOutputOracle([]string{dirs[m.dumps], tmp}); rc != 0 {
			return p2cRefuse(6, "extract-output-oracle failed on %s", dirs[m.dumps])
		}
		if err := p2cGzip(tmp, filepath.Join(dirs["oracle"], "output-"+version+suffix+".jsonl.gz")); err != nil {
			return p2cRefuse(6, "cannot gzip %s: %v", tmp, err)
		}
		_ = os.Remove(tmp) //nolint:gosec // inside the operator-supplied output root
	}
	if o, err := exec.Command("git", "-C", pob, "rev-parse", "HEAD").Output(); err == nil { //nolint:gosec,noctx // an operator-supplied checkout
		manifest["pobHead"] = strings.TrimSpace(string(o))
	}
	for k, p := range map[string]string{"converterSha256": converter, "dumperSha256": dumper, "zipSha256": zipPath,
		"pinRecordSha256": filepath.Join(here, "scripts", "pob_pin_poe2.json")} {
		if h, err := p2cSha256File(p); err == nil {
			manifest[k] = h
		}
	}
	b, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(dirs["oracle"], "MANIFEST.json"), append(b, '\n'), 0o600); err != nil {
		return p2cRefuse(3, "cannot write the manifest: %v", err)
	}
	w := bufio.NewWriter(os.Stdout)
	_, _ = w.Write(append(b, '\n'))
	_ = w.Flush()
	fmt.Fprintf(os.Stderr, "poe2-corpus: done in %s\n", time.Since(start).Round(time.Second))
	return 0
}
