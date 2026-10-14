package calculator

//pob:src/Data/ModCache.lua:1

import (
	"bufio"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

type luaVal struct {
	kind  byte
	str   string
	num   float64
	b     bool
	pos   []*luaVal
	named map[string]*luaVal
	raw   string
}

type luaParser struct {
	s string
	i int
}

func (p *luaParser) ws() {
	for p.i < len(p.s) && (p.s[p.i] == ' ' || p.s[p.i] == '\t' || p.s[p.i] == '\n' || p.s[p.i] == '\r') {
		p.i++
	}
}

func (p *luaParser) str() string {
	p.i++
	var b strings.Builder
	for p.i < len(p.s) {
		c := p.s[p.i]
		if c == '"' {
			p.i++
			return b.String()
		}
		if c == '\\' && p.i+1 < len(p.s) {
			p.i++
			switch n := p.s[p.i]; n {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				if n >= '0' && n <= '9' {
					j := p.i
					for j < len(p.s) && j < p.i+3 && p.s[j] >= '0' && p.s[j] <= '9' {
						j++
					}
					v, _ := strconv.Atoi(p.s[p.i:j])
					b.WriteByte(byte(v & 0xff))
					p.i = j
					continue
				}
				b.WriteByte(n)
			}
			p.i++
			continue
		}
		b.WriteByte(c)
		p.i++
	}
	return b.String()
}

func (p *luaParser) value() *luaVal {
	p.ws()
	start := p.i
	if p.i >= len(p.s) {
		return &luaVal{kind: 'z'}
	}
	switch c := p.s[p.i]; {
	case c == '"':
		s := p.str()
		return &luaVal{kind: 's', str: s, raw: p.s[start:p.i]}
	case c == '{':
		p.i++
		v := &luaVal{kind: 't', named: map[string]*luaVal{}}
		for {
			p.ws()
			if p.i >= len(p.s) {
				break
			}
			if p.s[p.i] == '}' {
				p.i++
				break
			}
			if p.s[p.i] == ',' {
				p.i++
				continue
			}
			if p.s[p.i] == '[' {
				p.i++
				p.ws()
				var key string
				if p.s[p.i] == '"' {
					key = p.str()
				} else {
					j := p.i
					for p.s[p.i] != ']' {
						p.i++
					}
					key = strings.TrimSpace(p.s[j:p.i])
				}
				p.ws()
				p.i++
				p.ws()
				p.i++
				val := p.value()
				if n, err := strconv.Atoi(key); err == nil && n == len(v.pos)+1 {
					v.pos = append(v.pos, val)
				} else {
					v.named[key] = val
				}
				continue
			}
			j := p.i
			for p.i < len(p.s) && (p.s[p.i] == '_' || p.s[p.i] == '.' || (p.s[p.i] >= 'a' && p.s[p.i] <= 'z') || (p.s[p.i] >= 'A' && p.s[p.i] <= 'Z') || (p.s[p.i] >= '0' && p.s[p.i] <= '9')) {
				p.i++
			}
			ident := p.s[j:p.i]
			p.ws()
			if p.i < len(p.s) && p.s[p.i] == '=' {
				p.i++
				v.named[ident] = p.value()
			} else {
				p.i = j
				v.pos = append(v.pos, p.value())
			}
		}
		v.raw = p.s[start:p.i]
		return v
	default:
		j := p.i
		for p.i < len(p.s) && p.s[p.i] != ',' && p.s[p.i] != '}' {
			p.i++
		}
		tok := strings.TrimSpace(p.s[j:p.i])
		switch tok {
		case "nil":
			return &luaVal{kind: 'z', raw: tok}
		case "true", "false":
			return &luaVal{kind: 'b', b: tok == "true", raw: tok}
		case "math.huge":
			return &luaVal{kind: 'n', num: math.Inf(1), raw: tok}
		case "-math.huge":
			return &luaVal{kind: 'n', num: math.Inf(-1), raw: tok}
		}
		f, err := strconv.ParseFloat(tok, 64)
		if err != nil {
			return &luaVal{kind: 's', str: "?" + tok, raw: tok}
		}
		return &luaVal{kind: 'n', num: f, raw: tok}
	}
}

func mcdNum(f float64) string {
	if math.IsInf(f, 1) {
		return "inf"
	}
	if math.IsInf(f, -1) {
		return "-inf"
	}
	return strconv.FormatFloat(f, 'g', 10, 64)
}

func mcdPobLeaves(v *luaVal, out *[]string) {
	switch v.kind {
	case 's':
		if v.str != "" {
			*out = append(*out, mcdLeafString(v.str))
		}
	case 'n':
		if v.num != 0 {
			*out = append(*out, mcdNum(v.num))
		}
	case 'b':
		if v.b {
			*out = append(*out, "1")
		}
	case 't':
		if _, ok := v.named["name"]; ok {
			if _, ok := v.named["type"]; ok {
				*out = append(*out, "mod("+mcdPobMod(v).full()+")")
				return
			}
		}
		for _, e := range v.pos {
			mcdPobLeaves(e, out)
		}
		for k, e := range v.named {
			if k == "keyOfScaledMod" {
				continue
			}
			mcdPobLeaves(e, out)
		}
	}
}

func mcdLeafString(s string) string {
	s = strings.ToLower(s)
	switch s {
	case "active skill", "active_skill", "grants_active_skill":
		return "skill"
	}
	return s
}

type mcdMod struct {
	name, typ, value, flags, tags string
}

func (m mcdMod) full() string {
	return m.name + "|" + m.typ + "|" + m.value + "|" + m.flags + "|" + m.tags
}

func mcdRenderTags(tags []canonTag) string {
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = t.String()
		if t.typ == "ItemCondition" || t.typ == "SkillName" {
			parts[i] = strings.ToLower(parts[i])
		}
		if t.typ == "DistanceRamp" {
			parts[i] = mcdRampRe.ReplaceAllString(parts[i], "{$1,$2}")
		}
	}
	sort.Strings(parts)
	return "[" + strings.Join(parts, " ") + "]"
}

func mcdPobMod(v *luaVal) mcdMod {
	m := mcdMod{}
	if n := v.named["name"]; n != nil {
		m.name = n.str
	}
	if n := v.named["type"]; n != nil {
		m.typ = n.str
	}
	f, kf := 0.0, 0.0
	if n := v.named["flags"]; n != nil {
		f = n.num
	}
	if n := v.named["keywordFlags"]; n != nil {
		kf = n.num
	}
	m.flags = fmt.Sprintf("f%d/k%d", int64(f), int64(kf))
	if val := v.named["value"]; val != nil {
		switch val.kind {
		case 'n':
			m.value = mcdNum(val.num)
		case 'b':
			m.value = strconv.FormatBool(val.b)
		case 's':
			if f, err := strconv.ParseFloat(val.str, 64); err == nil && m.typ != "LIST" {
				m.value = mcdNum(f)
				break
			}
			m.value = "{" + mcdLeafString(val.str) + "}"
		case 't':
			var leaves []string
			mcdPobLeaves(val, &leaves)
			sort.Strings(leaves)
			m.value = "{" + strings.Join(leaves, ",") + "}"
		default:
			m.value = "?"
		}
	}
	tags := make([]canonTag, 0, len(v.pos))
	for _, t := range v.pos {
		tags = append(tags, tdTag(t.raw))
	}
	m.tags = mcdRenderTags(tags)
	return m
}

var mcdRampRe = regexp.MustCompile(`1=([^/,}]+)/2=([^,}]+)`)

var mcdModType = reflect.TypeOf((*mod.Mod)(nil)).Elem()

func mcdOurLeaves(v reflect.Value, out *[]string, depth int) {
	if !v.IsValid() || depth > 8 {
		return
	}
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		if v.Kind() == reflect.Interface && v.Elem().Type().Implements(mcdModType) && v.CanInterface() {
			*out = append(*out, "mod("+mcdOurMod(v.Interface().(mod.Mod)).full()+")")
			return
		}
		mcdOurLeaves(v.Elem(), out, depth+1)
		return
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type() == reflect.TypeOf(mod.ExtraSkill{}) && v.Type().Field(i).Name == "SkillName" {
				continue
			}
			if v.Type().Field(i).Name == "KeyOfScaledMod" {
				continue
			}
			if v.Type() == reflect.TypeOf(mod.WeaponData{}) && v.Type().Field(i).Name == "Unset" {
				continue
			}
			mcdOurLeaves(v.Field(i), out, depth+1)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			mcdOurLeaves(v.Index(i), out, depth+1)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			mcdOurLeaves(v.MapIndex(k), out, depth+1)
		}
	case reflect.String:
		if s := v.String(); s != "" {
			*out = append(*out, mcdLeafString(s))
		}
	case reflect.Float32, reflect.Float64:
		if f := v.Float(); f != 0 {
			*out = append(*out, mcdNum(f))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n := v.Int(); n != 0 {
			*out = append(*out, mcdNum(float64(n)))
		}
	case reflect.Bool:
		if v.Bool() {
			*out = append(*out, "1")
		}
	}
}

func mcdOurMod(m mod.Mod) mcdMod {
	r := mcdMod{name: m.Name(), typ: string(m.Type())}
	r.flags = fmt.Sprintf("f%d/k%d", int64(m.Flags()), int64(m.KeywordFlags()))
	if val := m.Value(); val != nil {
		switch val.Type() {
		case mod.ModValueMultiTypeFloat:
			r.value = mcdNum(val.Float())
		case mod.ModValueMultiTypeFlag:
			r.value = strconv.FormatBool(val.Flag())
		case mod.ModValueMultiTypeList:
			var leaves []string
			mcdOurLeaves(reflect.ValueOf(val.List()), &leaves, 0)
			sort.Strings(leaves)
			r.value = "{" + strings.Join(leaves, ",") + "}"
		}
	}
	tags := make([]canonTag, 0, len(m.Tags()))
	for _, t := range m.Tags() {
		tags = append(tags, canonOurTag(t))
	}
	r.tags = mcdRenderTags(tags)
	return r
}

func mcdField(ms []mcdMod, f func(mcdMod) string) string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = f(m)
	}
	sort.Strings(out)
	return strings.Join(out, " ; ")
}

func mcdClassify(pob []mcdMod, pobWhole bool, ours []mcdMod, oursWhole bool) string {
	switch {
	case pobWhole && len(pob) > 0 && len(ours) == 0:
		return "missing"
	case pobWhole && len(pob) > 0 && !oursWhole:
		return "partial"
	case pobWhole && len(pob) == 0:
		if !oursWhole {
			return "missing-empty"
		}
		if len(ours) > 0 {
			return "count"
		}
		return ""
	case !pobWhole && oursWhole && len(ours) > 0:
		return "go-only"
	case !pobWhole:
		return ""
	}
	if len(pob) != len(ours) {
		return "count"
	}
	if mcdField(pob, func(m mcdMod) string { return m.name }) != mcdField(ours, func(m mcdMod) string { return m.name }) {
		return "wrong-name"
	}
	nt := func(m mcdMod) string { return m.name + "|" + m.typ }
	if mcdField(pob, nt) != mcdField(ours, nt) {
		return "wrong-type"
	}
	nv := func(m mcdMod) string { return m.name + "|" + m.typ + "|" + m.value }
	if mcdField(pob, nv) != mcdField(ours, nv) {
		if strings.Contains(mcdField(pob, nv), "{") || strings.Contains(mcdField(ours, nv), "{") {
			return "wrong-list"
		}
		return "wrong-value"
	}
	nf := func(m mcdMod) string { return nv(m) + "|" + m.flags }
	if mcdField(pob, nf) != mcdField(ours, nf) {
		return "wrong-flags"
	}
	if mcdField(pob, mcdMod.full) != mcdField(ours, mcdMod.full) {
		return "wrong-tag"
	}
	return ""
}

type mcdRow struct {
	line, class, pob, ours string
}

func mcdDiff(t *testing.T) ([]mcdRow, int, map[string]bool) {
	t.Helper()
	path := os.Getenv("MCD_CACHE")
	if path == "" {
		pobRoot := os.Getenv("POB_DIR")
		if pobRoot == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				t.Skipf("NOT-RUN: no $HOME: %v", err)
			}
			pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
		}
		path = filepath.Join(pobRoot, "src", "Data", "ModCache.lua")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("NOT-RUN: %v", err)
	}
	only := strings.ToLower(os.Getenv("MCD_ONLY"))
	var rows []mcdRow
	n := 0
	keys := map[string]bool{}
	for _, ln := range strings.Split(string(raw), "\n") {
		ln = strings.TrimPrefix(ln, "local c=...")
		if !strings.HasPrefix(ln, `c["`) {
			continue
		}
		p := &luaParser{s: ln, i: 2}
		key := p.str()
		if !strings.HasPrefix(ln[p.i:], "]=") {
			t.Fatalf("REFUSING: unparseable ModCache key in %q", ln)
		}
		p.i += 2
		keys[key] = true
		if only != "" && !strings.Contains(strings.ToLower(key), only) {
			continue
		}
		rhs := p.value()
		if rhs.kind != 't' || len(rhs.pos) < 1 {
			t.Fatalf("REFUSING: unparseable ModCache row %q", ln)
		}
		n++
		var pob []mcdMod
		pobWhole := rhs.pos[0].kind == 't' && (len(rhs.pos) < 2 || rhs.pos[1].kind == 'z')
		if rhs.pos[0].kind == 't' {
			for _, m := range rhs.pos[0].pos {
				pob = append(pob, mcdPobMod(m))
			}
		}
		e := ParseMod(key, false)
		var ours []mcdMod
		oursWhole := false
		if e != nil {
			oursWhole = strings.TrimSpace(e.Extra) == ""
			for _, m := range e.ModList {
				ours = append(ours, mcdOurMod(m))
			}
		}
		if c := mcdClassify(pob, pobWhole, ours, oursWhole); c != "" {
			rows = append(rows, mcdRow{key, c, mcdField(pob, mcdMod.full), mcdField(ours, mcdMod.full)})
		}
	}
	return rows, n, keys
}

func TestModCacheDifferential(t *testing.T) {
	rows, n, keys := mcdDiff(t)
	if os.Getenv("MCD_ONLY") == "" && n < 10000 {
		t.Fatalf("REFUSING: only %d ModCache lines compared — the reader is broken, not the tree", n)
	}
	byClass := map[string]int{}
	for _, r := range rows {
		byClass[r.class]++
	}
	classes := make([]string, 0, len(byClass))
	for c := range byClass {
		classes = append(classes, c)
	}
	sort.Strings(classes)
	t.Logf("ModCache lines compared: %d, differing: %d", n, len(rows))
	for _, c := range classes {
		t.Logf("  %-14s %6d", c, byClass[c])
	}
	show := os.Getenv("MCD_SHOW")
	for _, r := range rows {
		if show == "all" || r.class == show || os.Getenv("MCD_ONLY") != "" {
			t.Logf("[%s] %q\n    pob:  %s\n    ours: %s", r.class, r.line, r.pob, r.ours)
		}
	}
	if os.Getenv("MCD_RANK") != "" || os.Getenv("MCD_EXTRA_OUT") != "" {
		mcdRank(t, rows, keys)
	}
}

var (
	mcdNumRe            = regexp.MustCompile(`\(-?[0-9.]+-[0-9.]+\)|[0-9]+(\.[0-9]+)?`)
	mcdPrefixRe         = regexp.MustCompile(`^(\{[^}]*\})+`)
	mcdNodesRe          = regexp.MustCompile(`\bnodes="([0-9,]*)"`)
	mcdMastAttr         = regexp.MustCompile(`masteryEffects="([^"]*)"`)
	mcdMastRe           = regexp.MustCompile(`\{([0-9]+),([0-9]+)\}`)
	mcdMetaRe           = regexp.MustCompile(`^[A-Z][A-Za-z ]*:`)
	mcdClusterNotableRe = regexp.MustCompile(`^1 Added Passive Skill is (.+)$`)
)

func mcdTemplate(s string) string {
	return mcdNumRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "#")
}

func mcdCorpus(t *testing.T) (map[string][]string, map[string]string, int) {
	t.Helper()
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	clusterNotables := clusterNodeMap(data.LatestTreeVersion)
	nodeLines := func(id string, eff int64) []string {
		n, ok := tree.Nodes[id]
		if !ok {
			return nil
		}
		if eff == 0 {
			return n.Stats
		}
		for _, e := range n.MasteryEffects {
			if e.Effect == eff {
				return e.Stats
			}
		}
		return nil
	}
	dir := corpusDirForLoadedData(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	carriers := map[string][]string{}
	repr := map[string]string{}
	builds := 0
	for _, de := range entries {
		name := de.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		builds++
		seen := map[string]bool{}
		hit := func(line string) {
			for _, part := range strings.Split(line, "\n") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				tm := mcdTemplate(part)
				if seen[tm] {
					continue
				}
				seen[tm] = true
				carriers[tm] = append(carriers[tm], name)
				if _, ok := repr[tm]; !ok {
					repr[tm] = part
				}
			}
		}
		text := string(b)
		inItem := false
		for _, ln := range strings.Split(text, "\n") {
			tl := strings.TrimSpace(ln)
			switch {
			case strings.HasPrefix(tl, "<Item "):
				inItem = true
				continue
			case strings.HasPrefix(tl, "</Item>"):
				inItem = false
				continue
			}
			if !inItem || tl == "" || tl[0] == '<' {
				continue
			}
			tl = mcdPrefixRe.ReplaceAllString(html.UnescapeString(tl), "")
			if mcdMetaRe.MatchString(tl) {
				continue
			}
			hit(tl)
			if m := mcdClusterNotableRe.FindStringSubmatch(tl); m != nil {
				if node, ok := clusterNotables[m[1]]; ok {
					for _, l := range node.Stats {
						hit(l)
					}
				}
			}
		}
		for _, m := range mcdNodesRe.FindAllStringSubmatch(text, -1) {
			for _, id := range strings.Split(m[1], ",") {
				for _, l := range nodeLines(id, 0) {
					hit(l)
				}
			}
		}
		for _, m := range mcdMastAttr.FindAllStringSubmatch(text, -1) {
			for _, p := range mcdMastRe.FindAllStringSubmatch(m[1], -1) {
				eff, _ := strconv.ParseInt(p[2], 10, 64)
				for _, l := range nodeLines(p[1], eff) {
					hit(l)
				}
			}
		}
	}
	if builds < 10000 {
		t.Fatalf("REFUSING: only %d corpus builds read", builds)
	}
	return carriers, repr, builds
}

func mcdRank(t *testing.T, rows []mcdRow, keys map[string]bool) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("MCD_RANK / MCD_EXTRA_OUT need CORPUS=true")
	}
	requireGameData(t, "3.29")
	carriers, repr, builds := mcdCorpus(t)
	t.Logf("corpus builds read: %d, distinct line templates: %d", builds, len(carriers))

	if p := os.Getenv("MCD_EXTRA_OUT"); p != "" {
		covered := map[string]bool{}
		for k := range keys {
			covered[mcdTemplate(k)] = true
		}
		var extra []string
		for tm, line := range repr {
			if !covered[tm] && !keys[line] {
				extra = append(extra, line)
			}
		}
		sort.Strings(extra)
		if err := os.WriteFile(p, []byte(strings.Join(extra, "\n")+"\n"), 0o600); err != nil {
			t.Fatalf("MCD_EXTRA_OUT: %v", err)
		}
		t.Logf("MCD_EXTRA_OUT: %d uncached corpus lines written", len(extra))
		if os.Getenv("MCD_RANK") == "" {
			return
		}
	}

	wl := os.Getenv("MCD_WL")
	if wl == "" {
		t.Fatal("MCD_RANK needs MCD_WL=<worklist dir> (from WORKLIST=<dir> task gate)")
	}
	wrong := map[string]int{}
	bf, err := os.Open(filepath.Join(wl, "builds.tsv"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	sc := bufio.NewScanner(bf)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) < 2 || f[0] == "build" {
			continue
		}
		w, _ := strconv.Atoi(f[1])
		wrong[f[0]] = w
	}
	_ = bf.Close()
	if len(wrong) == 0 {
		t.Fatal("REFUSING: builds.tsv has no rows")
	}

	lits := mcdConsumerLiterals(t)
	type ranked struct {
		r                                 mcdRow
		lines, carriers, wrongC, residual int
		consumer                          string
	}
	byTmpl := map[string]*ranked{}
	var order []string
	for _, r := range rows {
		var parts []string
		for _, part := range strings.Split(r.line, "\n") {
			parts = append(parts, mcdTemplate(part))
		}
		key := r.class + "\x00" + strings.Join(parts, "\n")
		if rk, ok := byTmpl[key]; ok {
			rk.lines++
			continue
		}
		set := map[string]bool{}
		for _, tm := range parts {
			for _, b := range carriers[tm] {
				set[b] = true
			}
		}
		rk := &ranked{r: r, lines: 1, carriers: len(set), consumer: mcdConsumerColumn(r.pob, lits)}
		for b := range set {
			if w := wrong[b]; w > 0 {
				rk.wrongC++
				rk.residual += w
			}
		}
		byTmpl[key] = rk
		order = append(order, key)
	}
	out := make([]ranked, 0, len(order))
	for _, k := range order {
		out = append(out, *byTmpl[k])
	}
	sort.SliceStable(out, func(i, j int) bool {
		ei, ej := out[i].r.class == "missing-empty", out[j].r.class == "missing-empty"
		if ei != ej {
			return ej
		}
		if out[i].wrongC != out[j].wrongC {
			return out[i].wrongC > out[j].wrongC
		}
		if out[i].residual != out[j].residual {
			return out[i].residual > out[j].residual
		}
		return out[i].carriers > out[j].carriers
	})
	var sb strings.Builder
	sb.WriteString("line\tclass\tlines\tcarriers\twrongCarriers\tresidual\tconsumer\tpob\tours\n")
	withCarrier, withWrong := map[string]int{}, map[string]int{}
	for _, o := range out {
		if o.carriers > 0 {
			withCarrier[o.r.class]++
		}
		if o.wrongC > 0 {
			withWrong[o.r.class]++
		}
		fmt.Fprintf(&sb, "%s\t%s\t%d\t%d\t%d\t%d\t%s\t%s\t%s\n", strings.ReplaceAll(o.r.line, "\n", `\n`), o.r.class,
			o.lines, o.carriers, o.wrongC, o.residual, o.consumer, o.r.pob, o.r.ours)
	}
	t.Logf("differing TEMPLATES: %d; with >=1 carrier, by class: %v", len(out), withCarrier)
	t.Logf("differing TEMPLATES with >=1 WRONG carrier, by class: %v", withWrong)
	for i, o := range out {
		if i >= 80 {
			break
		}
		t.Logf("%3d %5d %5d %7d %-6s [%s] %q", o.lines, o.carriers, o.wrongC, o.residual, o.consumer, o.r.class, o.r.line)
	}
	if p := os.Getenv("MCD_OUT"); p != "" {
		if err := os.WriteFile(p, []byte(sb.String()), 0o600); err != nil {
			t.Fatalf("MCD_OUT: %v", err)
		}
	}
}

var mcdLitRe = regexp.MustCompile(`"([A-Za-z][A-Za-z0-9:_]*)"`)

func mcdConsumerLiterals(t *testing.T) map[string]bool {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: cannot list calculator/*.go: %v", err)
	}
	lits := map[string]bool{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == "mod_parser.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, ln := range strings.Split(string(b), "\n") {
			if i := strings.Index(ln, "//"); i >= 0 {
				ln = ln[:i]
			}
			for _, m := range mcdLitRe.FindAllStringSubmatch(ln, -1) {
				lits[m[1]] = true
			}
		}
	}
	return lits
}

func mcdConsumerColumn(pob string, lits map[string]bool) string {
	names := map[string]bool{}
	for _, seg := range strings.Split(strings.ReplaceAll(pob, "mod(", " ; "), " ; ") {
		seg = strings.TrimSpace(seg)
		if i := strings.Index(seg, "|"); i > 0 {
			names[seg[:i]] = true
		}
	}
	read := 0
	for n := range names {
		if lits[n] {
			read++
		}
	}
	return fmt.Sprintf("%d/%d", read, len(names))
}

// TestModCacheDifferentialFixedLinesStayFixed is the differential as a GUARD over the lines r39 moved from differing to matching.
func TestModCacheDifferentialFixedLinesStayFixed(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "modcache_diff_fixed_lines.txt"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	want := map[string]bool{}
	for _, l := range strings.Split(string(raw), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			want[l] = true
		}
	}
	rows, _, keys := mcdDiff(t)
	missing := 0
	for l := range want {
		if !keys[l] {
			missing++
			t.Errorf("fixture line %q is not in Data/ModCache.lua — the pin moved; re-derive the list", l)
		}
	}
	if len(want)-missing == 0 {
		t.Fatal("REFUSING: no fixture line resolved against the cache")
	}
	for _, r := range rows {
		if want[r.line] {
			t.Errorf("[%s] %q diverges from the pin again\n    pob:  %s\n    ours: %s", r.class, r.line, r.pob, r.ours)
		}
	}
}
