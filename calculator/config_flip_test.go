package calculator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/builds/parsecache"
	"github.com/Vilsol/go-pob/internal/oraclebin"
	"github.com/Vilsol/go-pob/pob"
)

type configFlipClass string

const (
	flipAgreeMoves configFlipClass = "AGREE-MOVES"
	flipDisagree   configFlipClass = "DISAGREE"
	flipInertBoth  configFlipClass = "INERT-BOTH"
	flipOursOnly   configFlipClass = "OURS-ONLY"
	flipError      configFlipClass = "ERROR"
)

type configFlipStat struct {
	key            string
	pB, pV, oB, oV float64
}

type configFlipTally struct {
	agree, disagree, oursOnly, baseWrong int
	worst                                []configFlipStat
}

func classifyConfigFlip(stats []configFlipStat, tolerance float64) (configFlipClass, configFlipTally) {
	var t configFlipTally
	for _, s := range stats {
		pobMoved := !oracleClose(s.pV, s.pB, tolerance)
		baseOK := oracleClose(s.oB, s.pB, tolerance) && !math.IsNaN(s.oB)
		varOK := oracleClose(s.oV, s.pV, tolerance) && !math.IsNaN(s.oV) && !math.IsInf(s.oV, 0)
		if !baseOK {
			t.baseWrong++
		}
		switch {
		case pobMoved && varOK:
			t.agree++
		case pobMoved:
			t.disagree++
			t.worst = append(t.worst, s)
		case !varOK && baseOK:
			t.oursOnly++
			t.worst = append(t.worst, s)
		}
	}
	switch {
	case t.disagree > 0:
		return flipDisagree, t
	case t.oursOnly > 0:
		return flipOursOnly, t
	case t.agree > 0:
		return flipAgreeMoves, t
	}
	return flipInertBoth, t
}

func TestClassifyConfigFlip(t *testing.T) {
	for name, tc := range map[string]struct {
		stats []configFlipStat
		want  configFlipClass
	}{
		"agree":                  {[]configFlipStat{{"a", 10, 20, 10, 20}, {"b", 1, 1, 1, 1}}, flipAgreeMoves},
		"pob moves only":         {[]configFlipStat{{"a", 10, 20, 10, 10}}, flipDisagree},
		"wrong factor":           {[]configFlipStat{{"a", 10, 20, 10, 30}}, flipDisagree},
		"ours only":              {[]configFlipStat{{"a", 10, 10, 10, 12}}, flipOursOnly},
		"inert":                  {[]configFlipStat{{"a", 10, 10, 10, 10}}, flipInertBoth},
		"base wrong, both still": {[]configFlipStat{{"a", 10, 10, 11, 11}}, flipInertBoth},
		"boundary":               {[]configFlipStat{{"a", 1000, 1005, 1000, 1000}}, flipInertBoth},
		"past boundary":          {[]configFlipStat{{"a", 1000, 1005.1, 1000, 1000}}, flipDisagree},
		"nan variant":            {[]configFlipStat{{"a", 10, 10, 10, math.NaN()}}, flipOursOnly},
	} {
		if got, _ := classifyConfigFlip(tc.stats, oracleDumpTolerance); got != tc.want {
			t.Errorf("%s: %s, want %s", name, got, tc.want)
		}
	}
}

var (
	flipConfigOpenRe    = regexp.MustCompile(`<Config\b[^>]*>`)
	flipActiveSetRe     = regexp.MustCompile(`\bactiveConfigSet="(\d+)"`)
	flipConfigSetOpenRe = regexp.MustCompile(`<ConfigSet\b[^>]*>`)
	flipIDAttrRe        = regexp.MustCompile(`\bid="(\d+)"`)
)

func flipVariantXML(raw []byte, name, attr, value string) ([]byte, error) {
	s := string(raw)
	cfgStart := strings.Index(s, "<Config")
	for cfgStart >= 0 && !flipConfigOpenRe.MatchString(s[cfgStart:min(len(s), cfgStart+200)]) {
		next := strings.Index(s[cfgStart+1:], "<Config")
		if next < 0 {
			cfgStart = -1
			break
		}
		cfgStart += 1 + next
	}
	if cfgStart < 0 {
		return nil, fmt.Errorf("no <Config> element")
	}
	open := flipConfigOpenRe.FindStringIndex(s[cfgStart:])
	openTag := s[cfgStart+open[0] : cfgStart+open[1]]
	if strings.HasSuffix(openTag, "/>") {
		return nil, fmt.Errorf("self-closing <Config/>")
	}
	cfgEnd := strings.Index(s[cfgStart:], "</Config>")
	if cfgEnd < 0 {
		return nil, fmt.Errorf("no </Config>")
	}
	regionStart, regionEnd := cfgStart+open[1], cfgStart+cfgEnd

	active := "1"
	if m := flipActiveSetRe.FindStringSubmatch(openTag); m != nil {
		active = m[1]
	}
	sets := flipConfigSetOpenRe.FindAllStringIndex(s[regionStart:regionEnd], -1)
	if len(sets) > 0 {
		found := false
		for _, loc := range sets {
			tag := s[regionStart+loc[0] : regionStart+loc[1]]
			m := flipIDAttrRe.FindStringSubmatch(tag)
			if m == nil || m[1] != active {
				continue
			}
			if strings.HasSuffix(tag, "/>") {
				return nil, fmt.Errorf("active config set %s is self-closing", active)
			}
			end := strings.Index(s[regionStart+loc[1]:], "</ConfigSet>")
			if end < 0 {
				return nil, fmt.Errorf("no </ConfigSet> for set %s", active)
			}
			regionStart, regionEnd = regionStart+loc[1], regionStart+loc[1]+end
			found = true
			break
		}
		if !found {
			return nil, fmt.Errorf("active config set %s not found", active)
		}
	}

	inputRe := regexp.MustCompile(`[ \t]*<Input\b[^>]*\bname="` + regexp.QuoteMeta(name) + `"[^>]*/>\r?\n?`)
	region := inputRe.ReplaceAllString(s[regionStart:regionEnd], "")
	if attr != "" {
		var esc bytes.Buffer
		_ = xmlEscapeAttr(&esc, value)
		region = "\n\t\t\t<Input name=\"" + name + "\" " + attr + "=\"" + esc.String() + "\"/>" + region
	}
	return []byte(s[:regionStart] + region + s[regionEnd:]), nil
}

func xmlEscapeAttr(b *bytes.Buffer, v string) error {
	for _, r := range v {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
	return nil
}

func verifyFlipVariant(base, variant []byte, name, value string, removed bool) error {
	pb, err := builds.ParseBuild(base)
	if err != nil {
		return fmt.Errorf("base: %w", err)
	}
	pv, err := builds.ParseBuild(variant)
	if err != nil {
		return fmt.Errorf("variant: %w", err)
	}
	others := func(inputs []pob.Input) []string {
		var out []string
		for _, in := range inputs {
			if in.Name != name {
				out = append(out, in.Name+"="+configInputString(in))
			}
		}
		sort.Strings(out)
		return out
	}
	var mine []string
	for _, in := range pv.Config.Inputs {
		if in.Name == name {
			mine = append(mine, configInputString(in))
		}
	}
	if !slices.Equal(others(pb.Config.Inputs), others(pv.Config.Inputs)) {
		return fmt.Errorf("the other inputs changed")
	}
	switch {
	case removed && len(mine) != 0:
		return fmt.Errorf("%s still present: %v", name, mine)
	case !removed && (len(mine) != 1 || mine[0] != value):
		return fmt.Errorf("%s is %v, want [%s]", name, mine, value)
	}
	return nil
}

type configFlipValue struct {
	attr, value string
}

func (v configFlipValue) label() string {
	if v.attr == "" {
		return "<unset>"
	}
	return v.value
}

func configFlipValues(o configPinOption, base string, baseSet bool) []configFlipValue {
	var out []configFlipValue
	add := func(attr, v string) {
		if v != base || (attr == "" && baseSet) {
			out = append(out, configFlipValue{attr, v})
		}
	}
	dflt := o.configPinDefault()
	switch o.Type {
	case "check":
		if base == "true" {
			if dflt == "true" {
				add("boolean", "false")
			} else {
				add("", "")
			}
		} else {
			add("boolean", "true")
		}
	case "list":
		for _, e := range o.List {
			attr := "string"
			if _, ok := e.Val.(float64); ok {
				attr = "number"
			}
			add(attr, configValueString(e.Val))
		}
	default:
		var vals []string
		switch o.Type {
		case "count":
			vals = []string{"1", "5"}
		case "countAllowZero":
			vals = []string{"0", "5"}
		case "integer":
			vals = []string{"-10", "50"}
		case "float":
			vals = []string{"0.5", "2.5"}
		}
		for _, v := range vals {
			add("number", v)
		}
		if baseSet {
			add("", "")
		}
	}
	return out
}

type flipPoBRunner struct {
	pobSrc, luajit, script, arch, dumps, logs string
	shards                                    int
}

type flipJob struct {
	xml, out string
	vis      bool
}

func (r flipPoBRunner) run(jobs []flipJob) error {
	var todo []flipJob
	seen := map[string]bool{}
	for _, j := range jobs {
		if seen[j.out] {
			continue
		}
		seen[j.out] = true
		if _, err := os.Stat(j.out); err != nil {
			todo = append(todo, j)
		}
	}
	if len(todo) == 0 {
		return nil
	}
	shards := min(r.shards, len(todo))
	fmt.Printf("config-flip: PoB dumps %d new of %d (%d shards)\n", len(todo), len(seen), shards)
	errs := make([]error, shards)
	var wg sync.WaitGroup
	for s := range shards {
		var b strings.Builder
		for i := s; i < len(todo); i += shards {
			v := "0"
			if todo[i].vis {
				v = "1"
			}
			fmt.Fprintf(&b, "%s\t%s\t%s\n", todo[i].xml, todo[i].out, v)
		}
		jobsFile := filepath.Join(r.logs, fmt.Sprintf("jobs-%d.tsv", s))
		if err := os.WriteFile(jobsFile, []byte(b.String()), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", jobsFile, err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			logf, err := os.Create(filepath.Join(r.logs, fmt.Sprintf("shard-%d.log", s)))
			if err != nil {
				errs[s] = err
				return
			}
			defer logf.Close()
			cmd := exec.CommandContext(context.Background(), "setarch", r.arch, "-R", r.luajit, r.script, "dump", jobsFile) //nolint:gosec // the operator-supplied luajit and script
			cmd.Dir = r.pobSrc
			pob := filepath.Dir(r.pobSrc)
			cmd.Env = append(os.Environ(),
				"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;",
				"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;")
			cmd.Stdout, cmd.Stderr = logf, logf
			if err := cmd.Run(); err != nil {
				errs[s] = fmt.Errorf("shard %d: %w (log %s)", s, err, logf.Name())
			}
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	for _, j := range todo {
		if _, err := os.Stat(j.out); err != nil {
			return fmt.Errorf("PoB wrote no %s for %s", j.out, j.xml)
		}
	}
	return nil
}

type flipDump struct {
	Error      string           `json:"error"`
	Visible    []string         `json:"visible"`
	NonDefault map[string]any   `json:"-"`
	Tables     oraclebin.Record `json:"-"`
}

func readFlipDump(path string) (*flipDump, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var d flipDump
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	d.NonDefault, _ = generic["nonDefault"].(map[string]any)
	numeric := func(key string) map[string]float64 {
		m, ok := generic[key].(map[string]any)
		if !ok {
			return nil
		}
		out := make(map[string]float64, len(m))
		for k, v := range m {
			switch f := v.(type) {
			case float64:
				out[k] = f
			case nil:
				out[k] = math.NaN()
			}
		}
		return out
	}
	d.Tables = oraclebin.Record{
		Output: numeric("output"), MainHand: numeric("outputMainHand"), OffHand: numeric("outputOffHand"),
		EnemyOutput: numeric("enemyOutput"), Minion: numeric("minionOutput"),
		MinionMain: numeric("minionOutputMainHand"), MinionOff: numeric("minionOutputOffHand"),
	}
	return &d, nil
}

func pobTables(path string) oraclebin.Record {
	d, err := readFlipDump(path)
	if err != nil || d == nil {
		return oraclebin.Record{}
	}
	return d.Tables
}

func contentKey(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])[:24]
}

type flipOurs struct {
	tables map[string]map[string]float64
	err    string
}

func evalFlipOurs(raw []byte, guard string, pairs map[string][]string, cache *EnvironmentCache) flipOurs {
	build, err := parsecache.ParseCached(raw)
	if err != nil {
		return flipOurs{err: "parse: " + err.Error()}
	}
	cache.guardPick = guard
	cache.pairsPicks = pairs
	defer func() {
		cache.guardPick = ""
		cache.pairsPicks = nil
	}()
	var env *Environment
	var panicked any
	func() {
		defer func() { panicked = recover() }()
		env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
	}()
	if panicked != nil || env == nil {
		return flipOurs{err: fmt.Sprintf("panic: %v", panicked)}
	}
	out := flipOurs{tables: map[string]map[string]float64{}}
	for _, p := range oracleTablePairs(oraclebin.Record{}, env) {
		cp := make(map[string]float64, len(p.got))
		for k, v := range p.got {
			cp[k] = v
		}
		out.tables[p.prefix] = cp
	}
	return out
}

func flipStats(pBase, pVar oraclebin.Record, oB, oV flipOurs) []configFlipStat {
	prefixes := []string{"", "MainHand.", "OffHand.", "Enemy.", "Minion.", "Minion.MainHand.", "Minion.OffHand."}
	tables := func(r oraclebin.Record) []map[string]float64 {
		return []map[string]float64{r.Output, r.MainHand, r.OffHand, r.EnemyOutput, r.Minion, r.MinionMain, r.MinionOff}
	}
	pb, pv := tables(pBase), tables(pVar)
	out := make([]configFlipStat, 0, len(pb[0])+len(pv[0]))
	for i, prefix := range prefixes {
		keys := map[string]bool{}
		for k := range pb[i] {
			keys[k] = true
		}
		for k := range pv[i] {
			keys[k] = true
		}
		for _, k := range slices.Sorted(func(yield func(string) bool) {
			for k := range keys {
				if !yield(k) {
					return
				}
			}
		}) {
			if math.IsNaN(pb[i][k]) || math.IsNaN(pv[i][k]) {
				continue
			}
			out = append(out, configFlipStat{
				key: prefix + k, pB: pb[i][k], pV: pv[i][k],
				oB: oB.tables[prefix][k], oV: oV.tables[prefix][k],
			})
		}
	}
	return out
}

type flipRow struct {
	opt                 configPinOption
	val                 configFlipValue
	carrier             string
	visible             bool
	tried               int
	class               configFlipClass
	tally               configFlipTally
	note                string
	controlCarrierMoved bool
	pobBase, pobVariant oraclebin.Record
}

// TestConfigFlip is `task config-flip`.
func TestConfigFlip(t *testing.T) {
	requireOperatorRun(t, "runs headless PoB; `task config-flip`")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	env := func(k string) string {
		v := os.Getenv(k)
		if v == "" {
			t.Fatalf("REFUSING: %s is unset; run this through `task config-flip`", k)
		}
		return v
	}
	out, err := filepath.Abs(env("CONFIG_FLIP_OUT"))
	if err != nil {
		t.Fatal(err)
	}
	opts, err := loadConfigPinOptions(env("CONFIG_OPTIONS"))
	if err != nil {
		t.Fatal(err)
	}
	runner := flipPoBRunner{pobSrc: env("POB_SRC"), luajit: env("LUAJIT"), script: env("FLIP_SCRIPT"), shards: 6}
	if v, err := strconv.Atoi(os.Getenv("SHARDS")); err == nil && v > 0 {
		runner.shards = v
	}
	maxCarriers := 4
	stride := 25
	if v, err := strconv.Atoi(os.Getenv("STRIDE")); err == nil && v > 0 {
		stride = v
	}
	uname, err := exec.CommandContext(context.Background(), "uname", "-m").Output()
	if err != nil {
		t.Fatal(err)
	}
	runner.arch = strings.TrimSpace(string(uname))

	fp := sha256.New()
	for _, p := range []string{runner.script, filepath.Join(filepath.Dir(runner.script), "json_stable.lua")} {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		fp.Write(b)
	}
	pinHead, err := exec.CommandContext(context.Background(), "git", "-C", runner.pobSrc, "rev-parse", "HEAD").Output() //nolint:gosec // the operator-supplied pin checkout
	if err != nil {
		t.Fatalf("REFUSING: cannot read the pin's HEAD: %v", err)
	}
	fp.Write(pinHead)
	runner.dumps = filepath.Join(out, "pob-"+hex.EncodeToString(fp.Sum(nil))[:12])
	runner.logs = filepath.Join(out, "logs")
	varDir := filepath.Join(out, "variants")
	for _, d := range []string{runner.dumps, runner.logs, varDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	only := map[string]bool{}
	for _, n := range strings.Split(os.Getenv("OPTIONS"), ",") {
		if n != "" {
			only[n] = true
		}
	}
	if len(only) > 0 {
		opts = slices.DeleteFunc(opts, func(o configPinOption) bool { return !only[o.Var] })
		if len(opts) == 0 {
			t.Fatal("REFUSING: OPTIONS matched no pin option")
		}
	}
	optByVar := map[string]configPinOption{}
	for _, o := range opts {
		optByVar[o.Var] = o
	}

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatal(err)
	}
	corpusDir, names := corpusNamesForLoadedData(t)
	if corpusDir, err = filepath.Abs(corpusDir); err != nil {
		t.Fatal(err)
	}
	guardPicks, pairsPicks := loadGuardPicks(t), loadPairsPicks(t)
	var graded []string
	for _, n := range names {
		if rec, ok := oracle[n]; ok && rec.TreeSources >= oracleMinTreeSources {
			graded = append(graded, n)
		}
	}

	raws := map[string][]byte{}
	var rawMu sync.Mutex
	readRaw := func(name string) []byte {
		rawMu.Lock()
		defer rawMu.Unlock()
		if b, ok := raws[name]; ok {
			return b
		}
		b, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			t.Fatal(err)
		}
		raws[name] = b
		return b
	}

	pool := map[string]bool{}
	for i := 0; i < len(graded); i += stride {
		pool[graded[i]] = true
	}
	defaults := map[string]string{}
	for _, o := range opts {
		defaults[o.Var] = o.configPinDefault()
	}
	nonDefaultCarriers := map[string][]string{}
	for _, n := range graded {
		build, err := parsecache.ParseCached(readRaw(n))
		if err != nil {
			continue
		}
		for _, in := range build.Config.Inputs {
			if _, ok := optByVar[in.Name]; ok && configInputString(in) != defaults[in.Name] &&
				len(nonDefaultCarriers[in.Name]) < maxCarriers && !slices.Contains(nonDefaultCarriers[in.Name], n) {
				nonDefaultCarriers[in.Name] = append(nonDefaultCarriers[in.Name], n)
				pool[n] = true
			}
		}
	}
	forced := map[string][]string{}
	for _, kv := range strings.Split(os.Getenv("CARRIERS"), ",") {
		if v, n, ok := strings.Cut(kv, "="); ok {
			forced[v] = append(forced[v], n)
			pool[n] = true
		}
	}
	poolNames := slices.Sorted(func(yield func(string) bool) {
		for n := range pool {
			if !yield(n) {
				return
			}
		}
	})
	baseDump := func(n string) string { return filepath.Join(runner.dumps, contentKey(readRaw(n))+".json") }
	jobs := make([]flipJob, 0, len(poolNames))
	for _, n := range poolNames {
		jobs = append(jobs, flipJob{xml: filepath.Join(corpusDir, n), out: baseDump(n), vis: true})
	}
	if err := runner.run(jobs); err != nil {
		t.Fatal(err)
	}
	visibleOn := map[string][]string{}
	baseOf := map[string]*flipDump{}
	for _, n := range poolNames {
		d, err := readFlipDump(baseDump(n))
		if err != nil {
			t.Fatal(err)
		}
		if d.Error != "" {
			continue
		}
		baseOf[n] = d
		for _, v := range d.Visible {
			visibleOn[v] = append(visibleOn[v], n)
		}
	}
	if len(baseOf) == 0 {
		t.Fatal("REFUSING: every pool build errored in PoB")
	}
	fmt.Printf("config-flip: %d options, carrier pool %d builds (%d loaded in PoB)\n", len(opts), len(poolNames), len(baseOf))

	type carrier struct {
		name    string
		visible bool
	}
	carriersOf := func(o configPinOption) []carrier {
		var out []carrier
		vis := map[string]bool{}
		for _, n := range visibleOn[o.Var] {
			vis[n] = true
		}
		for _, n := range forced[o.Var] {
			if baseOf[n] != nil {
				out = append(out, carrier{n, vis[n]})
			}
		}
		for _, n := range nonDefaultCarriers[o.Var] {
			if vis[n] && baseOf[n] != nil {
				out = append(out, carrier{n, true})
			}
		}
		rest := slices.Clone(visibleOn[o.Var])
		hidden := false
		if len(rest) == 0 {
			hidden = true
			for n := range baseOf {
				rest = append(rest, n)
			}
		}
		hash := func(n string) string { return contentKey([]byte(o.Var + "\x00" + n)) }
		sort.Slice(rest, func(i, j int) bool { return hash(rest[i]) < hash(rest[j]) })
		for _, n := range rest {
			if len(out) >= maxCarriers {
				break
			}
			if !slices.ContainsFunc(out, func(c carrier) bool { return c.name == n }) {
				out = append(out, carrier{n, !hidden})
			}
		}
		return out
	}

	baseValue := func(n string, o configPinOption) (string, bool) {
		if v, ok := baseOf[n].NonDefault[o.Var]; ok {
			return configValueString(v), true
		}
		return o.configPinDefault(), false
	}

	type variant struct {
		opt     configPinOption
		val     configFlipValue
		carrier carrier
		path    string
		dump    string
		err     string
	}
	makeVariant := func(o configPinOption, val configFlipValue, c carrier) variant {
		v := variant{opt: o, val: val, carrier: c}
		raw := readRaw(c.name)
		vx, err := flipVariantXML(raw, o.Var, val.attr, val.value)
		if err == nil {
			err = verifyFlipVariant(raw, vx, o.Var, val.value, val.attr == "")
		}
		if err != nil {
			v.err = "variant: " + err.Error()
			return v
		}
		key := contentKey(vx)
		v.path = filepath.Join(varDir, key+".xml")
		v.dump = filepath.Join(runner.dumps, key+".json")
		if _, err := os.Stat(v.path); err != nil {
			if err := os.WriteFile(v.path, vx, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return v
	}

	controlOpt, hasControl := optByVar["resistancePenalty"]
	if !hasControl {
		all, _ := loadConfigPinOptions(env("CONFIG_OPTIONS"))
		for _, o := range all {
			if o.Var == "resistancePenalty" {
				controlOpt, hasControl = o, true
			}
		}
	}
	if !hasControl {
		t.Fatal("REFUSING: the pin has no resistancePenalty option to control with")
	}
	controlFor := func(c carrier) variant {
		b, _ := baseValue(c.name, controlOpt)
		val := configFlipValue{"number", "0"}
		if b == "0" {
			val = configFlipValue{"number", "-60"}
		}
		return makeVariant(controlOpt, val, carrier{c.name, true})
	}

	oursCache := map[string]flipOurs{}
	evalAll := func(items []variant) {
		type evalJob struct {
			key, carrier string
			raw          []byte
		}
		var ej []evalJob
		seen := map[string]bool{}
		add := func(key, carrier string, raw func() []byte) {
			if _, ok := oursCache[key]; ok || seen[key] {
				return
			}
			seen[key] = true
			ej = append(ej, evalJob{key, carrier, raw()})
		}
		for _, v := range items {
			if v.err != "" {
				continue
			}
			add("base:"+v.carrier.name, v.carrier.name, func() []byte { return readRaw(v.carrier.name) })
			add(v.path, v.carrier.name, func() []byte {
				b, err := os.ReadFile(v.path)
				if err != nil {
					t.Fatal(err)
				}
				return b
			})
		}
		results := make([]flipOurs, len(ej))
		runConcurrently(len(ej), evalWorkers(), func(i int, cache *EnvironmentCache) {
			results[i] = evalFlipOurs(ej[i].raw, guardPicks[ej[i].carrier], pairsPicks[ej[i].carrier], cache)
		})
		for i, j := range ej {
			oursCache[j.key] = results[i]
		}
	}

	grade := func(v variant) (configFlipClass, configFlipTally, string) {
		if v.err != "" {
			return flipError, configFlipTally{}, v.err
		}
		pv, err := readFlipDump(v.dump)
		if err != nil {
			t.Fatal(err)
		}
		if pv.Error != "" {
			return flipError, configFlipTally{}, "pob: " + firstLine(pv.Error)
		}
		oB, oV := oursCache["base:"+v.carrier.name], oursCache[v.path]
		if oB.err != "" || oV.err != "" {
			return flipError, configFlipTally{}, "ours: " + oB.err + oV.err
		}
		class, tally := classifyConfigFlip(flipStats(baseOf[v.carrier.name].Tables, pv.Tables, oB, oV), oracleDumpTolerance)
		return class, tally, ""
	}

	type pending struct {
		opt      configPinOption
		val      configFlipValue
		carriers []carrier
		next     int
	}
	var queue []*pending
	var rows []flipRow
	for _, o := range opts {
		cs := carriersOf(o)
		if len(cs) == 0 {
			rows = append(rows, flipRow{opt: o, class: flipError, note: "no carrier"})
			continue
		}
		b, set := baseValue(cs[0].name, o)
		for _, val := range configFlipValues(o, b, set) {
			queue = append(queue, &pending{opt: o, val: val, carriers: cs})
		}
	}

	controls := map[string]bool{}
	for round := 0; len(queue) > 0 && round < maxCarriers; round++ {
		var items []variant
		var owners []*pending
		ctrlNeeded := map[string]variant{}
		for _, p := range queue {
			c := p.carriers[p.next]
			b, set := baseValue(c.name, p.opt)
			val := p.val
			if val.value == b && !(val.attr == "" && set) {
				alts := configFlipValues(p.opt, b, set)
				if len(alts) == 0 {
					continue
				}
				val = alts[0]
			}
			items = append(items, makeVariant(p.opt, val, c))
			owners = append(owners, p)
			if _, ok := controls[c.name]; !ok {
				ctrlNeeded[c.name] = controlFor(c)
			}
		}
		for _, cv := range ctrlNeeded {
			items = append(items, cv)
		}
		var pobJobs []flipJob
		for _, v := range items {
			if v.err == "" {
				pobJobs = append(pobJobs, flipJob{xml: v.path, out: v.dump})
			}
		}
		if err := runner.run(pobJobs); err != nil {
			t.Fatal(err)
		}
		evalAll(items)
		for name, cv := range ctrlNeeded {
			class, _, _ := grade(cv)
			controls[name] = class == flipAgreeMoves || class == flipDisagree || class == flipOursOnly
		}

		var next []*pending
		for i, p := range owners {
			v := items[i]
			class, tally, note := grade(v)
			if class == flipInertBoth && p.next+1 < len(p.carriers) {
				p.next++
				next = append(next, p)
				continue
			}
			rows = append(rows, flipRow{
				opt: p.opt, val: v.val, carrier: v.carrier.name, visible: v.carrier.visible, tried: p.next + 1,
				class: class, tally: tally, note: note, controlCarrierMoved: controls[v.carrier.name],
				pobBase: baseOf[v.carrier.name].Tables, pobVariant: pobTables(v.dump),
			})
		}
		queue = next
		fmt.Printf("config-flip: round %d graded %d variants, %d go to another carrier\n", round+1, len(items), len(queue))
	}

	writeConfigFlipReport(t, out, rows)
	if fixture := os.Getenv("FLIP_FIXTURE"); fixture != "" {
		writeConfigFlipFixture(t, filepath.Join(out, "fixture.jsonl"), rows, strings.Split(fixture, ","))
	}
}

type configFlipFixtureRecord struct {
	Carrier string             `json:"carrier"`
	Var     string             `json:"var"`
	Attr    string             `json:"attr"`
	Value   string             `json:"value"`
	PoB     map[string]float64 `json:"pob"`
}

func writeConfigFlipFixture(t *testing.T, path string, rows []flipRow, vars []string) {
	t.Helper()
	var b strings.Builder
	n := 0
	for _, r := range rows {
		if !slices.Contains(vars, r.opt.Var) || r.carrier == "" || r.class == flipError || r.class == flipInertBoth {
			continue
		}
		rec := configFlipFixtureRecord{Carrier: r.carrier, Var: r.opt.Var, Attr: r.val.attr, Value: r.val.value, PoB: map[string]float64{}}
		for _, s := range flipStats(r.pobBase, r.pobVariant, flipOurs{}, flipOurs{}) {
			if !oracleClose(s.pV, s.pB, oracleDumpTolerance) {
				rec.PoB[s.key] = s.pV
			}
		}
		line, err := json.Marshal(rec)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(line)
		b.WriteByte('\n')
		n++
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("config-flip: %d fixture records -> %s\n", n, path)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func writeConfigFlipReport(t *testing.T, out string, rows []flipRow) {
	t.Helper()
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].opt.Var != rows[j].opt.Var {
			return rows[i].opt.Var < rows[j].opt.Var
		}
		return rows[i].val.label() < rows[j].val.label()
	})
	var b strings.Builder
	b.WriteString("var\ttype\tvalue\tclass\tcarrier\tvisible\ttried\tagree\tdisagree\tours_only\tbase_wrong\tcontrol_moved\tworst\tnote\n")
	perClass := map[configFlipClass]int{}
	optWorst := map[string]configFlipClass{}
	rank := map[configFlipClass]int{flipDisagree: 5, flipOursOnly: 4, flipError: 3, flipInertBoth: 2, flipAgreeMoves: 1}
	for _, r := range rows {
		sort.Slice(r.tally.worst, func(i, j int) bool {
			return flipRel(r.tally.worst[i]) > flipRel(r.tally.worst[j])
		})
		var worst []string
		for _, s := range r.tally.worst[:min(4, len(r.tally.worst))] {
			worst = append(worst, fmt.Sprintf("%s pob %.6g->%.6g ours %.6g->%.6g", s.key, s.pB, s.pV, s.oB, s.oV))
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\t%t\t%d\t%d\t%d\t%d\t%d\t%t\t%s\t%s\n", r.opt.Var, r.opt.Type, r.val.label(),
			r.class, r.carrier, r.visible, r.tried, r.tally.agree, r.tally.disagree, r.tally.oursOnly, r.tally.baseWrong,
			r.controlCarrierMoved, strings.Join(worst, "; "), r.note)
		perClass[r.class]++
		if rank[r.class] > rank[optWorst[r.opt.Var]] {
			optWorst[r.opt.Var] = r.class
		}
	}
	perOpt := map[configFlipClass]int{}
	for _, c := range optWorst {
		perOpt[c]++
	}
	var s strings.Builder
	fmt.Fprintf(&s, "config-flip: %d option-values over %d options\n", len(rows), len(optWorst))
	fmt.Fprintf(&s, "%-12s %8s %8s\n", "class", "values", "options")
	for _, c := range []configFlipClass{flipAgreeMoves, flipDisagree, flipOursOnly, flipInertBoth, flipError} {
		fmt.Fprintf(&s, "%-12s %8d %8d\n", c, perClass[c], perOpt[c])
	}
	if err := os.WriteFile(filepath.Join(out, "flip.tsv"), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "flip-summary.txt"), []byte(s.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	fmt.Print(s.String())
	fmt.Printf("written to %s (flip.tsv, flip-summary.txt)\n", out)
}

func flipRel(s configFlipStat) float64 {
	return math.Abs(s.oV-s.pV) / math.Max(math.Abs(s.pV), 1e-9)
}

func TestFlipVariantXMLEditsOnlyTheActiveSet(t *testing.T) {
	base := `<PathOfBuilding><Build/><Config activeConfigSet="2">
		<ConfigSet title="a" id="1">
			<Input name="conditionFullLife" boolean="true"/>
		</ConfigSet>
		<ConfigSet id="2" title="b">
			<Input boolean="true" name="conditionFullLife"/>
			<Input name="enemyIsBoss" string="Pinnacle"/>
		</ConfigSet>
	</Config></PathOfBuilding>`
	got, err := flipVariantXML([]byte(base), "conditionFullLife", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(got), `name="conditionFullLife"`) != 1 || !strings.Contains(string(got), `title="a" id="1">
			<Input name="conditionFullLife" boolean="true"/>`) {
		t.Errorf("removal touched the inactive set or missed the reordered attribute:\n%s", got)
	}
	got, err = flipVariantXML([]byte(base), "enemyIsBoss", "string", "None")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(got), `name="enemyIsBoss"`) != 1 || !strings.Contains(string(got), `<Input name="enemyIsBoss" string="None"/>`) {
		t.Errorf("replace failed:\n%s", got)
	}
	if _, err := flipVariantXML([]byte(`<PathOfBuilding/>`), "x", "", ""); err == nil {
		t.Error("a build with no <Config> must refuse")
	}
}

func TestVerifyFlipVariantRefusesAMissedEdit(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	dir := corpusDirForLoadedData(t)
	raw, err := os.ReadFile(filepath.Join(dir, "0001-necromancer-unknown.xml"))
	if err != nil {
		t.Fatal(err)
	}
	v, err := flipVariantXML(raw, "conditionFullLife", "boolean", "true")
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyFlipVariant(raw, v, "conditionFullLife", "true", false); err != nil {
		t.Fatalf("a correct edit failed verification: %v", err)
	}
	if err := verifyFlipVariant(raw, raw, "conditionFullLife", "true", false); err == nil {
		t.Error("an unedited build passed as the variant")
	}
	if err := verifyFlipVariant(raw, v, "conditionFullLife", "", true); err == nil {
		t.Error("a set input passed as removed")
	}
}

func flipPredicateNames(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		var out []string
		for _, e := range x {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func configRelevanceQueryOf(opts []configPinOption) ConfigRelevance {
	sets := map[string]map[string]bool{}
	add := func(set string, v any) {
		if sets[set] == nil {
			sets[set] = map[string]bool{}
		}
		for _, n := range flipPredicateNames(v) {
			sets[set][n] = true
		}
	}
	for _, o := range opts {
		p := o.Predicates
		add("Conds", p["ifCond"])
		add("Conds", o.implyCond)
		add("Conds", o.implyCondList)
		add("MinionConds", p["ifMinionCond"])
		add("MinionConds", o.implyMinionCond)
		add("EnemyConds", p["ifEnemyCond"])
		add("EnemyConds", o.implyEnemyCond)
		add("CondTrue", p["ifCondTrue"])
		add("Mults", p["ifMult"])
		add("EnemyMults", p["ifEnemyMult"])
		add("Stats", p["ifStat"])
		add("EnemyStats", p["ifEnemyStat"])
		add("TagTypes", p["ifTagType"])
		add("Flags", p["ifFlag"])
		add("Mods", p["ifMod"])
		if o.includeTransfigured {
			add("TransfiguredSkills", p["ifSkill"])
		} else {
			add("Skills", p["ifSkill"])
		}
		add("SkillFlags", p["ifSkillFlag"])
		add("SkillData", p["ifSkillData"])
	}
	list := func(k string) []string { return slices.Sorted(maps.Keys(sets[k])) }
	return ConfigRelevance{
		Conds: list("Conds"), MinionConds: list("MinionConds"), EnemyConds: list("EnemyConds"),
		CondTrue: list("CondTrue"), Mults: list("Mults"), EnemyMults: list("EnemyMults"), Stats: list("Stats"),
		EnemyStats: list("EnemyStats"), TagTypes: list("TagTypes"), Flags: list("Flags"), Mods: list("Mods"),
		Skills: list("Skills"), TransfiguredSkills: list("TransfiguredSkills"), SkillFlags: list("SkillFlags"),
		SkillData: list("SkillData"),
	}
}

func configShownGo(o configPinOption, truthy func(string) bool, rel map[string]map[string]bool) bool {
	implied := func() bool {
		if !truthy(o.Var) {
			return false
		}
		for _, c := range flipPredicateNames(o.implyCondList) {
			if rel["Conds"][c] {
				return true
			}
		}
		for _, c := range flipPredicateNames(o.implyCond) {
			if rel["Conds"][c] {
				return true
			}
		}
		for _, c := range flipPredicateNames(o.implyMinionCond) {
			if rel["MinionConds"][c] {
				return true
			}
		}
		for _, c := range flipPredicateNames(o.implyEnemyCond) {
			if rel["EnemyConds"][c] {
				return true
			}
		}
		return false
	}
	check := func(key string, pass func(string) bool) bool {
		v, ok := o.Predicates[key]
		if !ok {
			return true
		}
		return slices.ContainsFunc(flipPredicateNames(v), pass)
	}
	in := func(set string) func(string) bool { return func(n string) bool { return rel[set][n] } }
	withImply := func(set string) func(string) bool { return func(n string) bool { return implied() || rel[set][n] } }
	skills := "Skills"
	if o.includeTransfigured {
		skills = "TransfiguredSkills"
	}
	return check("ifOption", truthy) && check("ifCond", withImply("Conds")) &&
		check("ifMinionCond", withImply("MinionConds")) && check("ifEnemyCond", withImply("EnemyConds")) &&
		check("ifCondTrue", in("CondTrue")) && check("ifMult", withImply("Mults")) &&
		check("ifEnemyMult", withImply("EnemyMults")) && check("ifStat", withImply("Stats")) &&
		check("ifEnemyStat", withImply("EnemyStats")) && check("ifTagType", withImply("TagTypes")) &&
		check("ifFlag", in("Flags")) && check("ifMod", withImply("Mods")) && check("ifSkill", in(skills)) &&
		check("ifSkillFlag", in("SkillFlags")) && check("ifSkillData", in("SkillData"))
}

// TestConfigVisibility is `task config-visibility`: on every pool build of a config-flip run.
func TestConfigVisibility(t *testing.T) {
	requireOperatorRun(t, "reads a config-flip run; `task config-visibility`")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	out, optsPath := os.Getenv("CONFIG_FLIP_OUT"), os.Getenv("CONFIG_OPTIONS")
	if out == "" || optsPath == "" {
		t.Fatal("REFUSING: set CONFIG_FLIP_OUT (a config-flip run) and CONFIG_OPTIONS")
	}
	opts, err := loadConfigPinOptions(optsPath)
	if err != nil {
		t.Fatal(err)
	}
	query := configRelevanceQueryOf(opts)
	corpusDir, names := corpusNamesForLoadedData(t)
	dumpDirs, _ := filepath.Glob(filepath.Join(out, "pob-*"))
	if len(dumpDirs) != 1 {
		t.Fatalf("REFUSING: want exactly one pob-* dump dir in %s, found %v", out, dumpDirs)
	}
	type buildVis struct {
		name string
		pob  *flipDump
	}
	var bv []buildVis
	for _, n := range names {
		raw, err := os.ReadFile(filepath.Join(corpusDir, n))
		if err != nil {
			continue
		}
		d, err := readFlipDump(filepath.Join(dumpDirs[0], contentKey(raw)+".json"))
		if err != nil || d.Error != "" || d.Visible == nil {
			continue
		}
		bv = append(bv, buildVis{n, d})
	}
	if len(bv) == 0 {
		t.Fatal("REFUSING: no pool build of this run has a visibility dump")
	}
	type miss struct{ pobOnly, goOnly []string }
	results := make([]miss, len(bv))
	runConcurrently(len(bv), evalWorkers(), func(i int, cache *EnvironmentCache) {
		raw, _ := os.ReadFile(filepath.Join(corpusDir, bv[i].name))
		build, err := parsecache.ParseCached(raw)
		if err != nil {
			return
		}
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		if env == nil {
			return
		}
		ans := env.ConfigRelevance(query)
		rel := map[string]map[string]bool{}
		for k, v := range map[string][]string{
			"Conds": ans.Conds, "MinionConds": ans.MinionConds, "EnemyConds": ans.EnemyConds, "CondTrue": ans.CondTrue,
			"Mults": ans.Mults, "EnemyMults": ans.EnemyMults, "Stats": ans.Stats, "EnemyStats": ans.EnemyStats,
			"TagTypes": ans.TagTypes, "Flags": ans.Flags, "Mods": ans.Mods, "Skills": ans.Skills,
			"TransfiguredSkills": ans.TransfiguredSkills, "SkillFlags": ans.SkillFlags, "SkillData": ans.SkillData,
		} {
			rel[k] = map[string]bool{}
			for _, n := range v {
				rel[k][n] = true
			}
		}
		nonDefault := bv[i].pob.NonDefault
		truthy := func(name string) bool {
			if v, ok := nonDefault[name]; ok {
				return v != false
			}
			for _, o := range opts {
				if o.Var == name {
					d := o.configPinDefault()
					return d != "false" && d != "" && (o.Type == "list" || o.Type == "check" || o.DefaultState != nil)
				}
			}
			return false
		}
		pobVis := map[string]bool{}
		for _, v := range bv[i].pob.Visible {
			pobVis[v] = true
		}
		for _, o := range opts {
			if _, set := nonDefault[o.Var]; set {
				continue
			}
			g := configShownGo(o, truthy, rel)
			switch {
			case pobVis[o.Var] && !g:
				results[i].pobOnly = append(results[i].pobOnly, o.Var)
			case !pobVis[o.Var] && g:
				results[i].goOnly = append(results[i].goOnly, o.Var)
			}
		}
	})
	pobOnly, goOnly := map[string][]string{}, map[string][]string{}
	for i, r := range results {
		for _, v := range r.pobOnly {
			pobOnly[v] = append(pobOnly[v], bv[i].name)
		}
		for _, v := range r.goOnly {
			goOnly[v] = append(goOnly[v], bv[i].name)
		}
	}
	var b strings.Builder
	b.WriteString("var\tside\tbuilds\texample\tpredicates\n")
	predOf := map[string]string{}
	for _, o := range opts {
		j, _ := json.Marshal(o.Predicates)
		predOf[o.Var] = string(j)
	}
	write := func(side string, m map[string][]string) {
		for _, v := range slices.Sorted(maps.Keys(m)) {
			fmt.Fprintf(&b, "%s\t%s\t%d\t%s\t%s\n", v, side, len(m[v]), m[v][0], predOf[v])
		}
	}
	write("pob-only", pobOnly)
	write("ours-only", goOnly)
	if err := os.WriteFile(filepath.Join(out, "visibility.tsv"), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("config-visibility: %d builds; %d options PoB shows and we hide somewhere, %d we show and PoB hides; %s\n",
		len(bv), len(pobOnly), len(goOnly), filepath.Join(out, "visibility.tsv"))
	const visibilityCeiling = 5
	pairs := 0
	for _, m := range []map[string][]string{pobOnly, goOnly} {
		for _, b := range m {
			pairs += len(b)
		}
	}
	if pairs > visibilityCeiling {
		t.Errorf("%d option x build visibility differences, ceiling %d", pairs, visibilityCeiling)
	} else if pairs < visibilityCeiling {
		t.Logf("%d option x build visibility differences under ceiling %d, lower it", pairs, visibilityCeiling)
	}
}
