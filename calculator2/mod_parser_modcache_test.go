package calculator2

//pob2:src/Data/ModCache.lua:1

import (
	"encoding/json"
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

	"github.com/Vilsol/go-pob/mod"
)

type mpLuaVal struct {
	kind  byte
	str   string
	num   float64
	b     bool
	pos   []*mpLuaVal
	named map[string]*mpLuaVal
	raw   string
}

type mpLuaParser struct {
	s string
	i int
}

func (p *mpLuaParser) ws() {
	for p.i < len(p.s) && (p.s[p.i] == ' ' || p.s[p.i] == '\t' || p.s[p.i] == '\n' || p.s[p.i] == '\r') {
		p.i++
	}
}

func (p *mpLuaParser) str() string {
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

func (p *mpLuaParser) value() *mpLuaVal {
	p.ws()
	start := p.i
	if p.i >= len(p.s) {
		return &mpLuaVal{kind: 'z'}
	}
	switch c := p.s[p.i]; {
	case c == '"':
		s := p.str()
		return &mpLuaVal{kind: 's', str: s, raw: p.s[start:p.i]}
	case c == '{':
		p.i++
		v := &mpLuaVal{kind: 't', named: map[string]*mpLuaVal{}}
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
			return &mpLuaVal{kind: 'z', raw: tok}
		case "true", "false":
			return &mpLuaVal{kind: 'b', b: tok == "true", raw: tok}
		case "math.huge":
			return &mpLuaVal{kind: 'n', num: math.Inf(1), raw: tok}
		case "-math.huge":
			return &mpLuaVal{kind: 'n', num: math.Inf(-1), raw: tok}
		}
		f, err := strconv.ParseFloat(tok, 64)
		if err != nil {
			return &mpLuaVal{kind: 's', str: "?" + tok, raw: tok}
		}
		return &mpLuaVal{kind: 'n', num: f, raw: tok}
	}
}

func mpcNum(f float64) string {
	if math.IsInf(f, 1) {
		return "inf"
	}
	if math.IsInf(f, -1) {
		return "-inf"
	}
	return strconv.FormatFloat(f, 'g', 10, 64)
}

func mpcPobLeaves(v *mpLuaVal, out *[]string) {
	switch v.kind {
	case 's':
		if v.str != "" {
			*out = append(*out, mpcLeafString(v.str))
		}
	case 'n':
		if v.num != 0 {
			*out = append(*out, mpcNum(v.num))
		}
	case 'b':
		if v.b {
			*out = append(*out, "1")
		}
	case 't':
		if _, ok := v.named["name"]; ok {
			if _, ok := v.named["type"]; ok {
				*out = append(*out, "mod("+mpcPobMod(v).full()+")")
				return
			}
		}
		for _, e := range v.pos {
			mpcPobLeaves(e, out)
		}
		for k, e := range v.named {
			if k == "keyOfScaledMod" {
				continue
			}
			mpcPobLeaves(e, out)
		}
	}
}

func mpcLeafString(s string) string {
	s = strings.ToLower(s)
	switch s {
	case "active skill", "active_skill", "grants_active_skill":
		return "skill"
	}
	return s
}

type mpcMod struct {
	name, typ, value, flags, tags string
}

func (m mpcMod) full() string {
	return m.name + "|" + m.typ + "|" + m.value + "|" + m.flags + "|" + m.tags
}

func mpcRenderTags(tags []mpCanonTag) string {
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = t.String()
		if t.typ == "ItemCondition" || t.typ == "SkillName" {
			parts[i] = strings.ToLower(parts[i])
		}
		if t.typ == "DistanceRamp" {
			parts[i] = mpcRampRe.ReplaceAllString(parts[i], "{$1,$2}")
		}
	}
	sort.Strings(parts)
	return "[" + strings.Join(parts, " ") + "]"
}

func mpcPobMod(v *mpLuaVal) mpcMod {
	m := mpcMod{}
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
			m.value = mpcNum(val.num)
			if m.typ == "FLAG" && val.num != 0 {
				// a FLAG mod's value is only ever read for truthiness, so a number reads as true
				m.value = "true"
			}
		case 'b':
			m.value = strconv.FormatBool(val.b)
		case 's':
			if f, err := strconv.ParseFloat(val.str, 64); err == nil && m.typ != "LIST" {
				m.value = mpcNum(f)
				break
			}
			m.value = "{" + mpcLeafString(val.str) + "}"
		case 't':
			var leaves []string
			mpcPobLeaves(val, &leaves)
			sort.Strings(leaves)
			m.value = "{" + strings.Join(leaves, ",") + "}"
		default:
			m.value = "?"
		}
	}
	tags := make([]mpCanonTag, 0, len(v.pos))
	for _, t := range v.pos {
		tags = append(tags, mpTdTag(t.raw))
	}
	m.tags = mpcRenderTags(tags)
	return m
}

var mpcRampRe = regexp.MustCompile(`1=([^/,}]+)/2=([^,}]+)`)

var mpcModType = reflect.TypeOf((*mod.Mod)(nil)).Elem()

func mpcOurLeaves(v reflect.Value, out *[]string, depth int) {
	if !v.IsValid() || depth > 8 {
		return
	}
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		if v.Kind() == reflect.Interface && v.Elem().Type().Implements(mpcModType) && v.CanInterface() {
			*out = append(*out, "mod("+mpcOurMod(v.Interface().(mod.Mod)).full()+")")
			return
		}
		mpcOurLeaves(v.Elem(), out, depth+1)
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
			mpcOurLeaves(v.Field(i), out, depth+1)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			mpcOurLeaves(v.Index(i), out, depth+1)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			mpcOurLeaves(v.MapIndex(k), out, depth+1)
		}
	case reflect.String:
		if s := v.String(); s != "" {
			*out = append(*out, mpcLeafString(s))
		}
	case reflect.Float32, reflect.Float64:
		if f := v.Float(); f != 0 {
			*out = append(*out, mpcNum(f))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n := v.Int(); n != 0 {
			*out = append(*out, mpcNum(float64(n)))
		}
	case reflect.Bool:
		if v.Bool() {
			*out = append(*out, "1")
		}
	}
}

func mpcOurMod(m mod.Mod) mpcMod {
	r := mpcMod{name: m.Name(), typ: string(m.Type())}
	r.flags = fmt.Sprintf("f%d/k%d", int64(m.Flags()), int64(m.KeywordFlags()))
	if val := m.Value(); val != nil {
		switch val.Type() {
		case mod.ModValueMultiTypeFloat:
			r.value = mpcNum(val.Float())
		case mod.ModValueMultiTypeFlag:
			r.value = strconv.FormatBool(val.Flag())
		case mod.ModValueMultiTypeList:
			var leaves []string
			mpcOurLeaves(reflect.ValueOf(val.List()), &leaves, 0)
			sort.Strings(leaves)
			r.value = "{" + strings.Join(leaves, ",") + "}"
		}
	}
	tags := make([]mpCanonTag, 0, len(m.Tags()))
	for _, t := range m.Tags() {
		tags = append(tags, mpCanonOurTag(t))
	}
	r.tags = mpcRenderTags(tags)
	return r
}

func mpcField(ms []mpcMod, f func(mpcMod) string) string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = f(m)
	}
	sort.Strings(out)
	return strings.Join(out, " ; ")
}

func mpcClassify(pob []mpcMod, pobWhole bool, ours []mpcMod, oursWhole bool) string {
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
	if mpcField(pob, func(m mpcMod) string { return m.name }) != mpcField(ours, func(m mpcMod) string { return m.name }) {
		return "wrong-name"
	}
	nt := func(m mpcMod) string { return m.name + "|" + m.typ }
	if mpcField(pob, nt) != mpcField(ours, nt) {
		return "wrong-type"
	}
	nv := func(m mpcMod) string { return m.name + "|" + m.typ + "|" + m.value }
	if mpcField(pob, nv) != mpcField(ours, nv) {
		if strings.Contains(mpcField(pob, nv), "{") || strings.Contains(mpcField(ours, nv), "{") {
			return "wrong-list"
		}
		return "wrong-value"
	}
	nf := func(m mpcMod) string { return nv(m) + "|" + m.flags }
	if mpcField(pob, nf) != mpcField(ours, nf) {
		return "wrong-flags"
	}
	if mpcField(pob, mpcMod.full) != mpcField(ours, mpcMod.full) {
		return "wrong-tag"
	}
	return ""
}

type mpcRow struct {
	line, class, pob, ours string
}

func mpcPob2Root(t *testing.T) string {
	t.Helper()
	if root := os.Getenv("POB2_DIR"); root != "" {
		return root
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("NOT-RUN: no $HOME: %v", err)
	}
	return filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding-PoE2")
}

func mpcCachePath(t *testing.T) string {
	t.Helper()
	if path := os.Getenv("MCD_CACHE"); path != "" {
		return path
	}
	return filepath.Join(mpcPob2Root(t), "src", "Data", "ModCache.lua")
}

func mpcEach(t *testing.T, fn func(key string, pob []mpcMod, pobWhole, pobNil bool, pobExtra string)) {
	t.Helper()
	raw, err := os.ReadFile(mpcCachePath(t))
	if err != nil {
		t.Skipf("NOT-RUN: %v", err)
	}
	for _, ln := range strings.Split(string(raw), "\n") {
		ln = strings.TrimPrefix(ln, "local c=...")
		if !strings.HasPrefix(ln, `c["`) {
			continue
		}
		p := &mpLuaParser{s: ln, i: 2}
		key := p.str()
		if !strings.HasPrefix(ln[p.i:], "]=") {
			t.Fatalf("REFUSING: unparseable ModCache key in %q", ln)
		}
		p.i += 2
		rhs := p.value()
		if rhs.kind != 't' || len(rhs.pos) < 1 {
			t.Fatalf("REFUSING: unparseable ModCache row %q", ln)
		}
		var pob []mpcMod
		pobWhole := rhs.pos[0].kind == 't' && (len(rhs.pos) < 2 || rhs.pos[1].kind == 'z')
		if rhs.pos[0].kind == 't' {
			for _, m := range rhs.pos[0].pos {
				pob = append(pob, mpcPobMod(m))
			}
		}
		pobExtra := ""
		if len(rhs.pos) >= 2 && rhs.pos[1].kind == 's' {
			pobExtra = rhs.pos[1].str
		}
		fn(key, pob, pobWhole, rhs.pos[0].kind != 't', pobExtra)
	}
}

func mpcDiff(t *testing.T) ([]mpcRow, int, map[string]bool) {
	t.Helper()
	only := strings.ToLower(os.Getenv("MCD_ONLY"))
	var rows []mpcRow
	n := 0
	keys := map[string]bool{}
	mpcEach(t, func(key string, pob []mpcMod, pobWhole, pobNil bool, pobExtra string) {
		keys[key] = true
		if only != "" && !strings.Contains(strings.ToLower(key), only) {
			return
		}
		n++
		modList, extra := ParseMod(key)
		var ours []mpcMod
		oursWhole := false
		if modList != nil {
			oursWhole = strings.TrimSpace(extra) == ""
			for _, m := range modList {
				ours = append(ours, mpcOurMod(m))
			}
		}
		c := mpcClassify(pob, pobWhole, ours, oursWhole)
		if c == "" && !pobWhole {
			// PoB leaves a remainder: grade the remainder, the nil-ness of the list and the partial mods too.
			switch {
			case pobNil != (modList == nil):
				c = "nil-mismatch"
			case strings.TrimSpace(pobExtra) != strings.TrimSpace(extra):
				c = "remainder"
			case mpcField(pob, mpcMod.full) != mpcField(ours, mpcMod.full):
				c = "partial-mods"
			}
		}
		if c != "" {
			rows = append(rows, mpcRow{key, c, mpcField(pob, mpcMod.full) + " ⟨" + strings.TrimSpace(pobExtra) + "⟩", mpcField(ours, mpcMod.full) + " ⟨" + strings.TrimSpace(extra) + "⟩"})
		}
	})
	return rows, n, keys
}

type mpCanonTag struct {
	typ      string
	params   string
	unmapped bool
}

func (c mpCanonTag) String() string {
	if c.params == "" {
		return c.typ
	}
	return c.typ + "/" + c.params
}

func mpCanonParams(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if v == "" || v == "false" ||
			(k == "div" && (mpCanonNumber(v) == "1" || mpCanonNumber(v) == "0")) ||
			(k == "base" && mpCanonNumber(v) == "0") ||
			(k == "globalLimit" && mpCanonNumber(v) == "0") {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + mpCanonNumber(m[k])
	}
	return strings.Join(parts, "/")
}

func mpCanonNumber(v string) string {
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return fmt.Sprintf("%.14g", f)
	}
	return v
}

func mpCanonList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return "{" + strings.Join(items, ",") + "}"
}

func mpCanonPobTag(typ string, params map[string]string) mpCanonTag {
	fold := func(singular, list, into string) {
		if v, ok := params[list]; ok {
			delete(params, list)
			params[into] = strings.TrimSuffix(strings.TrimPrefix(v, "{"), "}")
		}
		if v, ok := params[singular]; ok {
			delete(params, singular)
			if _, taken := params[into]; !taken {
				params[into] = v
			}
		}
	}
	fold("var", "varList", "vars")
	fold("stat", "statList", "stats")
	fold("skillName", "skillNameList", "skillNames")
	fold("skillType", "skillTypeList", "skillTypes")
	fold("skillPart", "skillPartList", "skillParts")
	if typ == string(mod.TypeSlotName) {
		fold("slotName", "slotNameList", "slotNames")
	}

	if v, ok := params["skillTypes"]; ok {
		parts := strings.Split(v, ",")
		for i := range parts {
			parts[i] = mpCanonNumber(parts[i])
		}
		params["skillTypes"] = strings.Join(parts, ",")
	}
	return mpCanonTag{typ: typ, params: mpCanonParams(params)}
}

func mpCanonOurTag(t mod.Tag) mpCanonTag {
	p := map[string]string{}
	set := func(k, v string) {
		if v != "" {
			p[k] = v
		}
	}
	setNum := func(k string, v float64) { p[k] = strconv.FormatFloat(v, 'g', -1, 64) }
	setBool := func(k string, v bool) {
		if v {
			p[k] = "true"
		}
	}
	setNumPtr := func(k string, v *float64) {
		if v != nil {
			setNum(k, *v)
		}
	}
	setStrPtr := func(k string, v *string) {
		if v != nil {
			set(k, *v)
		}
	}

	switch tag := t.(type) {
	case *mod.ConditionTag:
		set("vars", strings.Join(tag.VarList, ","))
		setBool("neg", tag.Negative)
	case *mod.ActorConditionTag:
		setStrPtr("actor", tag.Actor)
		set("vars", strings.Join(tag.VariableList, ","))
		setBool("neg", tag.Negative)
	case *mod.MultiplierTag:
		set("vars", strings.Join(tag.VariableList, ","))
		setNum("base", tag.TagBase)
		setNum("div", tag.Division)
		setNumPtr("limit", tag.TagLimit)
		setStrPtr("limitVar", tag.TagLimitVariable)
		setBool("limitTotal", tag.TagLimitTotal)
		set("actor", tag.TagActor)
		setNumPtr("globalLimit", tag.TagGlobalLimit)
		setStrPtr("globalLimitKey", tag.TagGlobalLimitKey)
		setStrPtr("divVar", tag.TagDivVariable)
		setBool("noFloor", tag.TagNoFloor)
		setStrPtr("limitStat", tag.TagLimitStat)
		setBool("limitNegTotal", tag.TagLimitNegTotal)
		set("limitActor", tag.TagLimitActor)
	case *mod.MultiplierThresholdTag:
		if len(tag.VariableList) > 0 {
			set("vars", strings.Join(tag.VariableList, ","))
		} else {
			set("vars", tag.Variable)
		}
		setNumPtr("threshold", tag.TagThreshold)
		setStrPtr("thresholdVar", tag.ThresholdVariable)
		setBool("upper", tag.TagUpper)
		setBool("equals", tag.TagEquals)
		set("actor", tag.TagActor)
	case *mod.PerStatTag:
		set("stats", strings.Join(tag.StatList, ","))
		setNum("div", tag.Divide)
		setNum("base", tag.Base)
		setNumPtr("limit", tag.TagLimit)
		setStrPtr("limitVar", tag.TagLimitVariable)
		setBool("limitTotal", tag.TagLimitTotal)
		set("actor", tag.TagActor)
		setNum("globalLimit", tag.TagGlobalLimit)
		set("globalLimitKey", tag.TagGlobalLimitKey)
	case *mod.PercentStatTag:
		set("stats", tag.Stat)
		if tag.PercentVar != "" {
			set("percentVar", tag.PercentVar)
		} else {
			setNum("percent", tag.Percent)
		}
		set("actor", tag.TagActor)
		setBool("floor", tag.TagFloor)
	case *mod.StatThresholdTag:
		set("stats", tag.Stat)
		if tag.TagThresholdStat != "" {
			set("thresholdStat", tag.TagThresholdStat)
		} else {
			setNum("threshold", tag.Threshold)
		}
		setBool("upper", tag.TagUpper)
		if tag.ThresholdPercent != 0 {
			setNum("thresholdPercent", tag.ThresholdPercent)
		}
	case *mod.SkillTypeTag:
		nums := make([]string, 0, len(tag.SkillTypeList))
		for _, st := range tag.SkillTypeList {
			n, ok := mpSkillTypeNumber[st]
			if !ok {
				return mpCanonTag{typ: string(mod.TypeSkillType), unmapped: true}
			}
			nums = append(nums, strconv.Itoa(n))
		}
		set("skillTypes", strings.Join(nums, ","))
		setBool("neg", tag.Negative)
	case *mod.MonsterTagTag:
		set("monsterTag", strings.Join(tag.MonsterTagList, ","))
		setBool("neg", tag.Negative)
	case *mod.GemTagTag:
		set("gemTags", strings.Join(tag.GemTagList, ","))
		setBool("neg", tag.Negative)
	case *mod.BaseFlagTag:
		set("baseFlag", tag.BaseFlag)
		setBool("neg", tag.Negative)
	case *mod.SkillNameTag:
		set("skillNames", strings.Join(tag.SkillNameList, ","))
		setBool("neg", tag.Negative)
		setBool("summonSkill", tag.TagSummonSkill)
		setBool("includeTransfigured", tag.IncludeTransfigured)
		setBool("partialMatch", tag.TagPartialMatch)
	case *mod.SkillIDTag:
		set("skillId", tag.ID())
	case *mod.SkillPartTag:
		if len(tag.PartList) > 0 {
			parts := make([]string, len(tag.PartList))
			for i, v := range tag.PartList {
				parts[i] = strconv.Itoa(v)
			}
			set("skillParts", strings.Join(parts, ","))
		} else {
			set("skillParts", strconv.Itoa(tag.Part))
		}
		setBool("neg", tag.Negative)
	case *mod.SocketedInTag:
		set("slotName", tag.SlotName)
		set("slotType", tag.TagSlotType)
		set("keyword", tag.TagKeyword)
		set("socketColor", tag.TagSocketColor)
		if len(tag.TagSockets) > 0 {
			nums := make([]string, len(tag.TagSockets))
			for i, n := range tag.TagSockets {
				nums[i] = strconv.Itoa(n)
			}
			set("sockets", mpCanonList(nums))
		}
		if tag.TagSocketsAll {
			set("sockets", "all")
		}
		if tag.TagSocketsCount != 0 {
			setNum("sockets", tag.TagSocketsCount)
		}
		setBool("neg", tag.TagNeg)
	case *mod.SlotNameTag:
		set("slotNames", strings.Join(tag.SlotNameList, ","))
	case *mod.SlotNumberTag:
		setNum("num", float64(tag.N))
	case *mod.GlobalEffectTag:
		if len(tag.GlobalEffectList) > 0 {
			set("effectType", tag.GlobalEffectList[0])
		}
		set("effectName", tag.NameTag)
		set("effectCond", tag.EffectCondTag)
		set("modCond", tag.ModCondTag)
		setBool("unscalable", tag.UnscalableTag)
		set("effectEnemyCond", tag.EffectEnemyCondTag)
		set("effectStackVar", tag.EffectStackVarTag)
		if tag.EffectStackLimitTag != 0 {
			setNum("effectStackLimit", tag.EffectStackLimitTag)
		}
		set("effectStackLimitVar", tag.EffectStackLimitVarTag)
		setBool("applyNotPlayer", tag.ApplyNotPlayerTag)
		setBool("applyMinions", tag.ApplyMinionsTag)
		setBool("allowTotemBuff", tag.AllowTotemBuffTag)
		if tag.DivSet {
			setNum("div", tag.DivTag)
		}
		if tag.LimitSet {
			setNum("limit", tag.LimitTag)
		}
		setBool("neg", tag.Negative)
	case *mod.DisablesItemTag:
		set("slotName", tag.TagSlotName)
		set("excludeItemType", tag.TagExcludeItemType)
	case *mod.ItemConditionTag:
		set("itemSlot", tag.ItemSlot)
		set("searchCond", tag.SearchCond)
		set("nameCond", tag.NameCond)
		set("rarityCond", tag.RarityCond)
		if tag.CorruptedCond != nil {
			setBool("corruptedCond", *tag.CorruptedCond)
		}
		setBool("allSlots", tag.AllSlots)
		setBool("bothSlots", tag.BothSlots)
		setBool("excludeSelf", tag.ExcludeSelf)
		setBool("shaperCond", tag.ShaperCond)
		setBool("elderCond", tag.ElderCond)
		setBool("neg", tag.Neg)
	case *mod.GlobalTag:
		setBool("neg", tag.Negative)
	case *mod.IgnoreCondTag, mod.IgnoreCondTag:
		return mpCanonTag{typ: string(mod.TypeIgnoreCond)}
	case *mod.ModFlagTag:
		setNum("modFlags", float64(tag.Flag))
	case *mod.ModFlagOrTag:
		setNum("modFlags", float64(tag.Flag))
	case *mod.KeywordFlagAndTag:
		setNum("keywordFlags", float64(tag.KeywordFlags))
	case *mod.DistanceRampTag:
		set("ramp", mpCanonRampFloat(tag.Ramp))
	case *mod.MeleeProximityTag:
		steps := make([]string, len(tag.Ramp))
		for i, v := range tag.Ramp {
			steps[i] = strconv.Itoa(v)
		}
		set("ramp", mpCanonList(steps))
	case *mod.LimitTag:
		if tag.LimitVar != "" {
			set("limitVar", tag.LimitVar)
		} else {
			setNum("limit", tag.TagLimit)
		}
		setBool("neg", tag.Negative)
	case *mod.InSlotTag:
		setNum("num", float64(tag.N))
	case mod.InSlotTag:
		setNum("num", float64(tag.N))
	case *mod.FlagTag:
		setBool("value", tag.Value)
	case mod.FlagTag:
		setBool("value", tag.Value)
	default:
		return mpCanonTag{typ: string(t.Type()), unmapped: true}
	}
	return mpCanonTag{typ: string(t.Type()), params: mpCanonParams(p)}
}

func mpCanonRampFloat(ramp [][]float64) string {
	if len(ramp) == 0 {
		return ""
	}
	steps := make([]string, 0, len(ramp))
	for _, pair := range ramp {
		kv := make([]string, 0, len(pair))
		for i, v := range pair {
			kv = append(kv, strconv.Itoa(i+1)+"="+strconv.FormatFloat(v, 'g', -1, 64))
		}
		steps = append(steps, strings.Join(kv, "/"))
	}
	return mpCanonList(steps)
}

func mpTdSplitTop(body string) []string {
	var out []string
	depth, start := 0, 0
	inStr := false
	for i := 0; i < len(body); i++ {
		c := body[i]
		if inStr {
			if c == '\\' {
				i++
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, body[start:i])
				start = i + 1
			}
		}
	}
	return append(out, body[start:])
}

func mpTdInner(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return "", false
	}
	return s[1 : len(s)-1], true
}

func mpTdPositional(entry string) (string, bool) {
	e := strings.TrimSpace(entry)
	if !strings.HasPrefix(e, "[") {
		return "", false
	}
	end := strings.Index(e, "]=")
	if end < 0 {
		return "", false
	}
	k := e[1:end]
	if k == "" {
		return "", false
	}
	for i := 0; i < len(k); i++ {
		if k[i] < '0' || k[i] > '9' {
			return "", false
		}
	}
	return e[end+2:], true
}

func mpTdValue(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return strings.ReplaceAll(v[1:len(v)-1], `\"`, `"`)
	}
	if body, ok := mpTdInner(v); ok {
		var items []string
		for _, e := range mpTdSplitTop(body) {
			if rhs, isPos := mpTdPositional(e); isPos {
				items = append(items, mpTdValue(rhs))
			} else {
				items = append(items, mpTdValue(e))
			}
		}
		return "{" + strings.Join(items, ",") + "}"
	}
	return v
}

func mpTdTag(lit string) mpCanonTag {
	body, ok := mpTdInner(lit)
	if !ok {
		return mpCanonTag{typ: "UNPARSED"}
	}
	typ := ""
	params := map[string]string{}
	for _, f := range mpTdSplitTop(body) {
		k, v, found := strings.Cut(strings.TrimSpace(f), "=")
		if !found {
			continue
		}
		if k == "type" {
			typ = mpTdValue(v)
			continue
		}
		params[k] = mpTdValue(v)
	}
	return mpCanonPobTag(typ, params)
}

//pob2:src/Data/Global.lua:335-626
var mpSkillTypeNumber = map[string]int{
	"Attack":                               1,
	"Spell":                                2,
	"Projectile":                           3,
	"DualWieldOnly":                        4,
	"Buff":                                 5,
	"Minion":                               6,
	"Damage":                               7,
	"Area":                                 8,
	"Duration":                             9,
	"RequiresShield":                       10,
	"ProjectileSpeed":                      11,
	"HasReservation":                       12,
	"ReservationBecomesCost":               13,
	"Trappable":                            14,
	"Totemable":                            15,
	"Mineable":                             16,
	"ElementalStatus":                      17,
	"MinionsCanExplode":                    18,
	"Chains":                               19,
	"Melee":                                20,
	"MeleeSingleTarget":                    21,
	"Multicastable":                        22,
	"TotemCastsAlone":                      23,
	"CausesBurning":                        24,
	"SummonsTotem":                         25,
	"TotemCastsWhenNotDetached":            26,
	"Physical":                             27,
	"Fire":                                 28,
	"Cold":                                 29,
	"Lightning":                            30,
	"Triggerable":                          31,
	"Triggers":                             32,
	"Trapped":                              33,
	"Movement":                             34,
	"DamageOverTime":                       35,
	"RemoteMined":                          36,
	"Triggered":                            37,
	"Vaal":                                 38,
	"Aura":                                 39,
	"CanTargetUnusableCorpse":              40,
	"RangedAttack":                         41,
	"Chaos":                                42,
	"FixedSpeedProjectile":                 43,
	"ThresholdJewelArea":                   44,
	"ThresholdJewelProjectile":             45,
	"ThresholdJewelDuration":               46,
	"ThresholdJewelRangedAttack":           47,
	"Channel":                              48,
	"DegenOnlySpellDamage":                 49,
	"InbuiltTrigger":                       50,
	"Golem":                                51,
	"Herald":                               52,
	"AuraAffectsEnemies":                   53,
	"NoRuthless":                           54,
	"ThresholdJewelSpellDamage":            55,
	"Cascadable":                           56,
	"ProjectilesFromUser":                  57,
	"MirageArcherCanUse":                   58,
	"ProjectileSpiral":                     59,
	"SingleMainProjectile":                 60,
	"MinionsPersistWhenSkillRemoved":       61,
	"ProjectileNumber":                     62,
	"Warcry":                               63,
	"Instant":                              64,
	"Brand":                                65,
	"TargetsDestructibleCorpses":           66,
	"NonHitChill":                          67,
	"ChillingArea":                         68,
	"AppliesCurse":                         69,
	"Barrageable":                          70,
	"AuraDuration":                         71,
	"AreaSpell":                            72,
	"OR":                                   73,
	"AND":                                  74,
	"NOT":                                  75,
	"AppliesMaim":                          76,
	"CreatesMinion":                        77,
	"Guard":                                78,
	"Travel":                               79,
	"Blink":                                80,
	"CanHaveBlessing":                      81,
	"ProjectilesNotFromUser":               82,
	"AttackInPlace":                        83,
	"AttackInPlaceIsDefault":               84,
	"Nova":                                 85,
	"InstantNoRepeatWhenHeld":              86,
	"InstantShiftAttackForLeftMouse":       87,
	"AuraNotOnCaster":                      88,
	"Banner":                               89,
	"Rain":                                 90,
	"Cooldown":                             91,
	"ThresholdJewelChaining":               92,
	"Slam":                                 93,
	"Stance":                               94,
	"NonRepeatable":                        95,
	"UsedByTotem":                          96,
	"Steel":                                97,
	"Hex":                                  98,
	"Mark":                                 99,
	"Aegis":                                100,
	"Orb":                                  101,
	"KillNoDamageModifiers":                102,
	"RandomElement":                        103,
	"LateConsumeCooldown":                  104,
	"Arcane":                               105,
	"FixedCastTime":                        106,
	"RequiresOffHandNotWeapon":             107,
	"Link":                                 108,
	"Blessing":                             109,
	"DynamicCooldown":                      110,
	"Microtransaction":                     111,
	"OwnerCannotUse":                       112,
	"ProjectilesNumberModifiersNotApplied": 113,
	"TotemsAreBallistae":                   114,
	"SkillGrantedBySupport":                115,
	"CrossbowSkill":                        116,
	"CrossbowAmmoSkill":                    117,
	"UseGlobalStats":                       118,
	"ModifiesNextSkill":                    119,
	"OngoingSkill":                         120,
	"UsableWhileShapeshifted":              121,
	"Meta":                                 122,
	"Bear":                                 123,
	"Wolf":                                 124,
	"Invokable":                            125,
	"CreatesSkeletonMinion":                126,
	"CreatesUndeadMinion":                  127,
	"CreatesDemonMinion":                   128,
	"CommandsMinions":                      129,
	"ReservesManually":                     130,
	"ConsumesCharges":                      131,
	"ManualCooldownConsumption":            132,
	"SupportedByHourglass":                 133,
	"SupportedByBreachlordsAmalgam":        134,
	"ConsumesFullyBrokenArmour":            135,
	"SkillConsumesFreeze":                  136,
	"SkillConsumesIgnite":                  137,
	"SkillConsumesShock":                   138,
	"Wall":                                 139,
	"Persistent":                           140,
	"UsableWhileMoving":                    141,
	"CanBecomeArrowRain":                   142,
	"MultipleReservation":                  143,
	"SupportedByElementalDischarge":        144,
	"Limit":                                145,
	"Singlular":                            146,
	"GeneratesCharges":                     147,
	"EmpowersOtherSkill":                   148,
	"PerformsFinalStrike":                  149,
	"PerfectTiming":                        150,
	"CanHaveMultipleOngoingSkillInstances": 151,
	"Sustained":                            152,
	"ComboStacking":                        153,
	"SupportedByComboFinisher":             154,
	"Offering":                             155,
	"Retaliation":                          156,
	"Shapeshift":                           157,
	"Invocation":                           158,
	"Grenade":                              159,
	"NoDualWield":                          160,
	"Jumping":                              161,
	"CannotChain":                          162,
	"CreatesGroundRune":                    163,
	"CreatesFissure":                       164,
	"SummonsAttackTotem":                   165,
	"NonWeaponAttack":                      166,
	"CreatesGroundEffect":                  167,
	"SupportedByComboMastery":              168,
	"IceCrystal":                           169,
	"SkillConsumesPowerChargesOnUse":       170,
	"SkillConsumesFrenzyChargesOnUse":      171,
	"SkillConsumesEnduranceChargesOnUse":   172,
	"SupportedByFerocity":                  173,
	"SupportedByPotential":                 174,
	"ProjectileNoCollision":                175,
	"SupportedByExcise":                    176,
	"SupportedByExpanse":                   177,
	"SupportedByExecrate":                  178,
	"IsBlasphemy":                          179,
	"PersistentShowsCastTime":              180,
	"GeneratesEnergy":                      181,
	"GeneratesRemnants":                    182,
	"CommandableMinion":                    183,
	"Bow":                                  184,
	"AffectsPresence":                      185,
	"GainsStages":                          186,
	"HasSeals":                             187,
	"SupportedByExpand":                    188,
	"SupportedByUnleash":                   189,
	"SupportedBySalvo":                     190,
	"Spear":                                191,
	"GroundTargetedProjectile":             192,
	"SupportedByFusillade":                 193,
	"HasUsageCondition":                    194,
	"SupportedByMobileAssault":             195,
	"RequiresBuckler":                      196,
	"UsableWhileMounted":                   197,
	"Companion":                            198,
	"ConsumesInstillment":                  199,
	"CanCancelActions":                     200,
	"SupportedByUnmoving":                  201,
	"SupportedByCleanse":                   202,
	"Hazard":                               203,
	"SupportedByRally":                     204,
	"SupportedByFlamepierce":               205,
	"SupportedByStormchain":                206,
	"SupportedByFreezefork":                207,
	"Palm":                                 208,
	"CannotSpiritStrike":                   209,
	"SkillConsumesBleeding":                210,
	"SkillConsumesPoison":                  211,
	"TargetsDestructibleRareCorpses":       212,
	"SupportedByAncestralAid":              213,
	"MinionsAreUndamagable":                214,
	"GeneratesInfusion":                    215,
	"SkillConsumesParried":                 216,
	"DetonatesAfterTime":                   217,
	"NoAttackOrCastTime":                   218,
	"CreatesCompanion":                     219,
	"CannotTerrainChain":                   220,
	"SupportedByTumult":                    221,
	"RequiresCharges":                      222,
	"CannotConsumeCharges":                 223,
	"ConsumesRage":                         224,
	"NonDamageArmourBreak":                 225,
	"Necrotic":                             226,
	"Nature":                               227,
	"NoAttackInPlace":                      228,
	"DodgeReplacement":                     229,
	"SupportedByDurationThree":             230,
	"ToggleSpawnedObjectTargetable_DefaultOn":  231,
	"ToggleSpawnedObjectTargetable_DefaultOff": 232,
	"ReserveInAllSets":                         233,
	"Unleashable":                              234,
	"CannotCreateJaggedGround":                 235,
	"SingleLevelSkill":                         236,
	"SupportedByZarokh":                        237,
	"SupportedByGarukhan":                      238,
	"FrozenSpite":                              239,
	"ObjectDurability":                         240,
	"Detonator":                                241,
	"UnlimitedTotems":                          242,
	"SupportedByHaemoCrystals":                 243,
	"SupportedByFlamePillar":                   244,
	"CanCreateStoneElementals":                 245,
	"RemnantCannotBeShared":                    246,
	"GamepadDoNotForceSkillAtLocation":         247,
	"GamepadDeflectable":                       248,
	"GamepadForceAllowInteraction":             249,
	"Wyvern":                                   250,
	"Plant":                                    251,
	"Wind":                                     252,
	"SupportedByHayoxi":                        253,
	"Storm":                                    254,
	"DisableUpdateActionLocationAfterRelease":  255,
	"InteractsWithElementalGround":             256,
	"SupportedByNovaProjectiles":               257,
	"UsedByProxy":                              258,
	"SupportedByEchoingCry":                    259,
	"SpecialAncestralBoost":                    260,
	"Runic":                                    261,
	"ActiveBlock":                              262,
	"SupportedByVruunsInevitablity":            263,
	"SupportedByTulsAvalanche":                 264,
	"IndeterminateEmpowermentAmount":           265,
	"GamepadDoNotChannelSkillAtLocation":       266,
	"AffectedByCooldownRate":                   267,
	"UsedByClone":                              268,
	"SupportedByAncestralWarriorTotem":         269,
	"SupportedBySpellTotem":                    270,
	"SupportedByBallistaTotem":                 271,
	"SupportedByMortarTotem":                   272,
	"SupportedByFeralInvocation":               273,
	"SupportedByMirageArcher":                  274,
	"SupportedByMirageDeadeye":                 275,
	"SupportedByHollowForm":                    276,
	"SupportedByAnimusSplinters":               277,
	"HasNoCost":                                278,
	"SupportedByBattershout":                   279,
	"SupportedByRuneforgedBlades":              280,
	"SupportedByExploitWeakness":               281,
	"SupportedByDrainedAilment":                282,
	"SkillConsumesDazed":                       283,
	"SupportedByDazzle":                        284,
	"SupportedByFrostfire":                     285,
	"SupportedByBitingFrost":                   286,
	"SupportedByJaggedGround":                  287,
	"SupportedByAbidingHex":                    288,
	"SupportedByFrenziedRiposte":               289,
	"SupportedByCreepingChill":                 290,
}

// mpModCacheAgreeFloor is the number of ModCache lines ParseMod must agree with PoB-PoE2 on; it only goes up.
const mpModCacheAgreeFloor = 9951

func TestModCacheVsPoB(t *testing.T) {
	rows, n, keys := mpcDiff(t)
	if os.Getenv("MCD_ONLY") == "" && n < 9000 {
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
	agree := n - len(rows)
	t.Logf("ModCache lines compared: %d, agree: %d, differing: %d", n, agree, len(rows))
	for _, c := range classes {
		t.Logf("  %-14s %6d", c, byClass[c])
	}
	show := os.Getenv("MCD_SHOW")
	for _, r := range rows {
		if show == "all" || r.class == show || os.Getenv("MCD_ONLY") != "" {
			t.Logf("[%s] %q\n    pob:  %s\n    ours: %s", r.class, r.line, r.pob, r.ours)
		}
	}
	if os.Getenv("MCD_ONLY") == "" && agree < mpModCacheAgreeFloor {
		t.Errorf("agreement %d fell below the floor %d", agree, mpModCacheAgreeFloor)
	}
	if out := os.Getenv("MCD_OUT"); out != "" {
		mpcWriteRanked(t, out, rows)
	}
	if out := os.Getenv("MCD_EXTRA_OUT"); out != "" {
		counts, _ := mpcCorpusLineCounts(t)
		var extra []string
		for ln := range counts {
			if ln != "" && !keys[ln] {
				extra = append(extra, ln)
			}
		}
		sort.Strings(extra)
		if err := os.WriteFile(out, []byte(strings.Join(extra, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s: %d corpus lines ModCache does not carry", out, len(extra))
	}
}

var (
	mpcItemPrefixRe = regexp.MustCompile(`^(\{[^}]*\})+`)
	mpcNodesRe      = regexp.MustCompile(`\bnodes="([0-9,]*)"`)
	mpcItemRe       = regexp.MustCompile(`(?s)<Item\b[^>]*>(.*?)</Item>`)
)

// mpcCorpusLineCounts counts, per line, how many corpus builds carry it on an item or an allocated tree node
// (presence, not activation).
func mpcCorpusLineCounts(t *testing.T) (map[string]int, int) {
	t.Helper()
	dir := filepath.Join("..", "testdata", "corpus-poe2-0.5")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("NOT-RUN: no PoE2 corpus at %s: %v", dir, err)
	}
	pobRoot := mpcPob2Root(t)
	raw, err := os.ReadFile(filepath.Join(pobRoot, "src", "TreeData", "0_5", "tree.json"))
	if err != nil {
		t.Skipf("NOT-RUN: no PoE2 tree.json: %v", err)
	}
	var tree struct {
		Nodes map[string]struct {
			Stats []string `json:"stats"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(raw, &tree); err != nil {
		t.Fatalf("REFUSING: tree.json: %v", err)
	}
	counts := map[string]int{}
	builds := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		builds++
		seen := map[string]bool{}
		for _, m := range mpcItemRe.FindAllStringSubmatch(string(data), -1) {
			mods := false
			for _, ln := range strings.Split(html.UnescapeString(m[1]), "\n") {
				ln = strings.TrimSpace(ln)
				if strings.HasPrefix(ln, "Implicits: ") {
					mods = true
					continue
				}
				if mods {
					seen[strings.TrimSpace(mpcItemPrefixRe.ReplaceAllString(ln, ""))] = true
				}
			}
		}
		for _, m := range mpcNodesRe.FindAllStringSubmatch(string(data), -1) {
			for _, id := range strings.Split(m[1], ",") {
				for _, st := range tree.Nodes[id].Stats {
					for _, ln := range strings.Split(st, "\n") {
						seen[ln] = true
					}
				}
			}
		}
		for ln := range seen {
			counts[ln]++
		}
	}
	if builds == 0 {
		t.Fatalf("REFUSING: %s holds no build XML", dir)
	}
	return counts, builds
}

func mpcWriteRanked(t *testing.T, out string, rows []mpcRow) {
	t.Helper()
	counts := map[string]int{}
	builds := 0
	if os.Getenv("CORPUS") == "true" {
		counts, builds = mpcCorpusLineCounts(t)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if counts[rows[i].line] != counts[rows[j].line] {
			return counts[rows[i].line] > counts[rows[j].line]
		}
		return rows[i].line < rows[j].line
	})
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fmt.Fprintf(f, "builds\tclass\tline\tpob\tours\n")
	carried := 0
	for _, r := range rows {
		if counts[r.line] > 0 {
			carried++
		}
		fmt.Fprintf(f, "%d\t%s\t%q\t%s\t%s\n", counts[r.line], r.class, r.line, r.pob, r.ours)
	}
	t.Logf("wrote %s: %d differing lines, %d of them on a corpus item or tree node (%d builds)", out, len(rows), carried, builds)
}
