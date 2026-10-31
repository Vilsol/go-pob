package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed poe2configcorpus.lua
var p2ccLua []byte

const p2ccUsage = "usage: poe2-config-corpus <pob2-dir> <repo-root> <src-corpus-dir> <out-root> <version>  (env N, SEED, P, JOBS, LUAJIT)"

type p2ccOption struct {
	Index        int
	Var, Type    string
	DefaultState string
	DefaultIndex int
	List         []string
	Predicates   string
	ApplyLine    int
	Label        string
}

var (
	p2ccConfigSetRe = regexp.MustCompile(`<Config activeConfigSet="([0-9]+)"`)
	p2ccCounts      = []float64{0, 1, 2, 3, 5, 10, 20, 50, 100}
	p2ccIntegers    = []float64{-20, 0, 10, 50}
	p2ccFloats      = []float64{0, 0.5, 1, 2.5}
)

func p2ccParseOptions(tsv string) ([]p2ccOption, error) {
	var opts []p2ccOption
	seen := map[string]bool{}
	for i, line := range strings.Split(strings.TrimRight(tsv, "\n"), "\n") {
		if i == 0 {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 10 {
			return nil, fmt.Errorf("options row %d has %d fields, want 10", i, len(f))
		}
		o := p2ccOption{Var: f[1], Type: f[2], DefaultState: f[3], Predicates: f[6], Label: f[9]}
		o.Index, _ = strconv.Atoi(f[0])
		o.DefaultIndex, _ = strconv.Atoi(f[4])
		o.ApplyLine, _ = strconv.Atoi(f[8])
		if f[5] != "" {
			o.List = strings.Split(f[5], "\x1f")
		}
		if seen[o.Var] {
			continue
		}
		seen[o.Var] = true
		opts = append(opts, o)
	}
	return opts, nil
}

// p2ccEligible drops the quest rewards (the character's own state, already in the XML) and free text.
func p2ccEligible(o p2ccOption) bool {
	if strings.HasPrefix(o.Var, "quest") {
		return false
	}
	switch o.Type {
	case "check", "count", "countAllowZero", "integer", "float":
		return true
	case "list":
		return len(o.List) > 1
	}
	return false
}

func p2ccNumber(v float64) (string, string) { return "number", strconv.FormatFloat(v, 'g', -1, 64) }

// p2ccValue draws a non-default value: a check flips its default, a list takes another entry.
func p2ccValue(o p2ccOption, r *rand.Rand) (string, string) {
	switch o.Type {
	case "check":
		return "boolean", strconv.FormatBool(o.DefaultState != "b:true")
	case "count", "countAllowZero":
		return p2ccNumber(p2ccCounts[r.IntN(len(p2ccCounts))])
	case "integer":
		return p2ccNumber(p2ccIntegers[r.IntN(len(p2ccIntegers))])
	case "float":
		return p2ccNumber(p2ccFloats[r.IntN(len(p2ccFloats))])
	}
	def := max(o.DefaultIndex, 1) - 1
	pick := r.IntN(len(o.List) - 1)
	if pick >= def {
		pick++
	}
	v := o.List[pick]
	switch v[:2] {
	case "n:":
		return "number", v[2:]
	case "b:":
		return "boolean", v[2:]
	}
	return "string", v[2:]
}

func p2ccRand(seed uint64, build string) *rand.Rand {
	h := fnv.New64a()
	_, _ = h.Write([]byte(build))
	return rand.New(rand.NewPCG(seed, h.Sum64())) //nolint:gosec // a reproducible sample, not a secret
}

func p2ccXMLAttr(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;").Replace(s)
}

type p2ccSet struct{ Var, Attr, Value string }

// p2ccRewrite inserts the inputs at the top of the active ConfigSet; a var the build already sets is kept as is.
func p2ccRewrite(xml string, sets []p2ccSet) (string, []p2ccSet, error) {
	m := p2ccConfigSetRe.FindStringSubmatch(xml)
	if m == nil {
		return "", nil, fmt.Errorf("no <Config activeConfigSet>")
	}
	open := `<ConfigSet id="` + m[1] + `"`
	i := strings.Index(xml, open)
	if i < 0 {
		return "", nil, fmt.Errorf("no %s>", open)
	}
	j := strings.IndexByte(xml[i:], '>')
	if j < 0 || xml[i+j-1] == '/' {
		return "", nil, fmt.Errorf("%s> is empty or unterminated", open)
	}
	at := i + j + 1
	var b strings.Builder
	var kept []p2ccSet
	for _, s := range sets {
		if strings.Contains(xml, `<Input name="`+p2ccXMLAttr(s.Var)+`"`) {
			continue
		}
		kept = append(kept, s)
		fmt.Fprintf(&b, "\n\t\t\t<Input name=\"%s\" %s=\"%s\"/>", p2ccXMLAttr(s.Var), s.Attr, p2ccXMLAttr(s.Value))
	}
	return xml[:at] + b.String() + xml[at:], kept, nil
}

func p2ccReadVisible(path string) (map[string]bool, string, error) {
	b, err := os.ReadFile(path) //nolint:gosec // inside the operator-supplied output root
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", path, err)
	}
	s := string(b)
	if strings.HasPrefix(s, "#error\t") {
		return nil, strings.TrimSpace(strings.TrimPrefix(s, "#error\t")), nil
	}
	vis := map[string]bool{}
	for _, l := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		if l != "" {
			vis[l] = true
		}
	}
	return vis, "", nil
}

func p2ccSample(src string, n int, seed uint64) ([]string, error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", src, err)
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xml") {
			names = append(names, strings.TrimSuffix(e.Name(), ".xml"))
		}
	}
	sort.Strings(names)
	r := rand.New(rand.NewPCG(seed, 0)) //nolint:gosec // a reproducible sample, not a secret
	r.Shuffle(len(names), func(a, b int) { names[a], names[b] = names[b], names[a] })
	if n < len(names) {
		names = names[:n]
	}
	sort.Strings(names)
	return names, nil
}

func p2ccChunks(names []string, size int) [][]string {
	var out [][]string
	for len(names) > 0 {
		k := min(size, len(names))
		out = append(out, names[:k])
		names = names[k:]
	}
	return out
}

func poe2ConfigCorpus(argv []string) int {
	if len(argv) != 5 {
		fmt.Fprintln(os.Stderr, p2ccUsage)
		return 2
	}
	pob, here, src, outRoot, version := argv[0], argv[1], argv[2], argv[3], argv[4]
	n, seed, p, jobs := 2000, uint64(1), 0.3, 4
	if v, err := strconv.Atoi(os.Getenv("N")); err == nil && v > 0 {
		n = v
	}
	if v, err := strconv.ParseUint(os.Getenv("SEED"), 10, 64); err == nil {
		seed = v
	}
	if v, err := strconv.ParseFloat(os.Getenv("P"), 64); err == nil && v > 0 && v <= 1 {
		p = v
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
	for _, k := range []string{"corpus", "oracle", "dumps", "visible", "logs", "work"} {
		dirs[k] = filepath.Join(outRoot, k+"-config")
	}
	if abs, _ := filepath.Abs(src); abs == dirs["corpus"] {
		return p2cRefuse(2, "the source corpus is the output corpus")
	}
	head, err := exec.Command("git", "-C", here, "rev-parse", "HEAD").Output() //nolint:gosec,noctx // an operator-supplied checkout
	if err != nil {
		return p2cRefuse(2, "repo HEAD of %s: %v", here, err)
	}
	dirty := exec.Command("git", "-C", here, "diff", "--quiet", "HEAD", "--", "scripts/gopob/").Run() != nil //nolint:gosec,noctx // an operator-supplied checkout
	stamp, err := p2cStamp(pob, filepath.Join(here, "scripts", "gopob", "poe2configcorpus.go"), dumper)
	if err != nil {
		return p2cRefuse(2, "cannot fingerprint the generator: %v", err)
	}
	luaSum := sha256.Sum256(p2ccLua)
	gen := map[string]any{"version": version, "source": src, "n": n, "seed": seed, "p": p, "luaSha256": hex.EncodeToString(luaSum[:])}
	genJSON, _ := json.Marshal(gen)
	stamp += string(genJSON) + "\n"
	gen["generatorStamp"], gen["generatorCommit"], gen["generatorDirty"] = stamp, strings.TrimSpace(string(head)), dirty
	stampPath := filepath.Join(dirs["corpus"], ".poe2-config-stamp")
	if dpNonEmpty(dirs["corpus"]) {
		old, _ := os.ReadFile(stampPath) //nolint:gosec // inside the operator-supplied output root
		if string(old) != stamp {
			return p2cRefuse(3, "%s is NOT EMPTY and was not built by this generator, seed, N, p, dumper and pin — refusing to mix runs", dirs["corpus"])
		}
		fmt.Fprintf(os.Stderr, "poe2-config-corpus: resuming in %s\n", outRoot)
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o750); err != nil {
			return p2cRefuse(3, "cannot create %s: %v", d, err)
		}
	}
	if err := os.WriteFile(stampPath, []byte(stamp), 0o600); err != nil {
		return p2cRefuse(3, "cannot write %s: %v", stampPath, err)
	}
	script := filepath.Join(dirs["work"], "poe2configcorpus.lua")
	if err := os.WriteFile(script, p2ccLua, 0o600); err != nil {
		return p2cRefuse(3, "cannot write %s: %v", script, err)
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

	optPath := filepath.Join(dirs["oracle"], "options.tsv")
	if log, err := r.run(script, "options", optPath); err != nil {
		return p2cRefuse(4, "enumerating ConfigOptions failed: %v\n%s", err, log)
	}
	optRaw, err := os.ReadFile(optPath) //nolint:gosec // inside the operator-supplied output root
	if err != nil {
		return p2cRefuse(4, "%v", err)
	}
	opts, err := p2ccParseOptions(string(optRaw))
	if err != nil {
		return p2cRefuse(4, "%v", err)
	}
	var eligible []p2ccOption
	for _, o := range opts {
		if p2ccEligible(o) {
			eligible = append(eligible, o)
		}
	}
	names, err := p2ccSample(src, n, seed)
	if err != nil {
		return p2cRefuse(4, "%v", err)
	}
	fmt.Fprintf(os.Stderr, "poe2-config-corpus: %d options, %d eligible, %d builds, p=%g, seed=%d\n", len(opts), len(eligible), len(names), p, seed)

	var pending []string
	for _, nm := range names {
		if _, err := os.Stat(filepath.Join(dirs["visible"], nm+".txt")); err != nil {
			pending = append(pending, nm)
		}
	}
	chunks := p2ccChunks(pending, 20)
	keys := make([]string, len(chunks))
	for i := range chunks {
		keys[i] = strconv.Itoa(i)
	}
	p2cParallel(keys, jobs, func(k string) {
		i, _ := strconv.Atoi(k)
		args := make([]string, 0, 2+len(chunks[i]))
		args = append(args, "visible", dirs["visible"])
		for _, nm := range chunks[i] {
			args = append(args, filepath.Join(src, nm+".xml"))
		}
		if log, err := r.run(script, args...); err != nil {
			_ = os.WriteFile(filepath.Join(dirs["logs"], "visible-"+k+".log"), log, 0o600) //nolint:gosec // inside the output root
		}
	})

	var tsv strings.Builder
	tsv.WriteString("build\tvar\tattr\tvalue\n")
	visErrors := map[string]int{}
	var generated []string
	setCount := 0
	for _, nm := range names {
		vis, visErr, err := p2ccReadVisible(filepath.Join(dirs["visible"], nm+".txt"))
		if err != nil {
			visErrors["visibility: no output"]++
			continue
		}
		if visErr != "" {
			visErrors[p2cErrorKind("visibility", visErr)]++
			continue
		}
		rng := p2ccRand(seed, nm)
		var sets []p2ccSet
		for _, o := range eligible {
			if !vis[o.Var] || rng.Float64() >= p {
				continue
			}
			attr, val := p2ccValue(o, rng)
			sets = append(sets, p2ccSet{o.Var, attr, val})
		}
		raw, err := os.ReadFile(filepath.Join(src, nm+".xml")) //nolint:gosec // an operator-supplied corpus
		if err != nil {
			visErrors["read: "+err.Error()]++
			continue
		}
		out, kept, err := p2ccRewrite(string(raw), sets)
		if err != nil {
			visErrors["rewrite: "+err.Error()]++
			continue
		}
		if err := os.WriteFile(filepath.Join(dirs["corpus"], nm+".xml"), []byte(out), 0o600); err != nil { //nolint:gosec // inside the operator-supplied output root
			return p2cRefuse(3, "%v", err)
		}
		for _, s := range kept {
			fmt.Fprintf(&tsv, "%s.xml\t%s\t%s\t%s\n", nm, s.Var, s.Attr, s.Value)
		}
		setCount += len(kept)
		generated = append(generated, nm)
	}
	if err := os.WriteFile(filepath.Join(dirs["oracle"], "configs.tsv"), []byte(tsv.String()), 0o600); err != nil {
		return p2cRefuse(3, "%v", err)
	}
	gen["options"], gen["eligibleOptions"], gen["generated"] = len(opts), len(eligible), len(generated)
	gen["skipped"], gen["inputsSet"] = visErrors, setCount
	if o, err := exec.Command("git", "-C", pob, "rev-parse", "HEAD").Output(); err == nil { //nolint:gosec,noctx // an operator-supplied checkout
		gen["pobHead"] = strings.TrimSpace(string(o))
	}
	gb, _ := json.MarshalIndent(gen, "", "  ")
	if err := os.WriteFile(filepath.Join(dirs["corpus"], "MANIFEST.json"), append(gb, '\n'), 0o600); err != nil {
		return p2cRefuse(3, "%v", err)
	}
	fmt.Fprintf(os.Stderr, "poe2-config-corpus: wrote %d builds, %d inputs (%.1f per build), skipped %v in %s\n",
		len(generated), setCount, float64(setCount)/float64(max(1, len(generated))), visErrors, time.Since(start).Round(time.Second))
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
	if err := p2cGzip(tmp, filepath.Join(dirs["oracle"], "output-"+version+"-config.jsonl.gz")); err != nil {
		return p2cRefuse(6, "cannot gzip %s: %v", tmp, err)
	}
	_ = os.Remove(tmp) //nolint:gosec // inside the operator-supplied output root
	gen["dumperSha256"], _ = p2cSha256File(dumper)
	gb, _ = json.MarshalIndent(gen, "", "  ")
	if err := os.WriteFile(filepath.Join(dirs["oracle"], "MANIFEST.json"), append(gb, '\n'), 0o600); err != nil {
		return p2cRefuse(3, "%v", err)
	}
	_, _ = os.Stdout.Write(append(gb, '\n'))
	fmt.Fprintf(os.Stderr, "poe2-config-corpus: done in %s\n", time.Since(start).Round(time.Second))
	return 0
}
