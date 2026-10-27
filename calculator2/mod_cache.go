package calculator2

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/mod"
)

// ModSource is modLib.parseMod: a line's mods and its unparsed remainder ("" when PoB's extra is nil).
// A nil slice is PoB's nil list (the line was not understood).
type ModSource func(line string) ([]mod.Mod, string)

type shippedCacheEntry struct {
	mods  []mod.Mod
	extra string
	err   error
}

var (
	shippedCacheMu      sync.Mutex
	shippedCacheEntries = map[string]*shippedCacheEntry{}
)

// ModCache answers a line from PoB-PoE2's shipped Data/ModCache.lua. A line missing from the cache, or
// one whose mods carry a tag or payload the mod package cannot express (ModCacheUnsupported), reads
// as not understood: (nil, line).
//
//pob2:src/Modules/ModParser.lua:7435-7454
func ModCache(line string) ([]mod.Mod, string) {
	e := shippedCacheLookup(line)
	if e == nil || e.err != nil {
		return nil, line
	}
	if e.mods == nil {
		return nil, e.extra
	}
	return mod.CloneSlice(e.mods), e.extra
}

// ModCacheUnsupported says why a cached line cannot be converted; nil when it converts or is absent.
func ModCacheUnsupported(line string) error {
	if e := shippedCacheLookup(line); e != nil {
		return e.err
	}
	return nil
}

func shippedCacheLookup(line string) *shippedCacheEntry {
	shippedCacheMu.Lock()
	defer shippedCacheMu.Unlock()
	if e, ok := shippedCacheEntries[line]; ok {
		return e
	}
	row, ok := poe2.ModCacheByLine(line)
	if !ok {
		return nil
	}
	e := &shippedCacheEntry{extra: row.Extra}
	if row.Mods != "" {
		var raw []map[string]any
		if err := json.Unmarshal([]byte(row.Mods), &raw); err != nil {
			e.err = err
		} else {
			e.mods = make([]mod.Mod, 0, len(raw))
			for _, r := range raw {
				m, err := cacheMod(r)
				if err != nil {
					e.err, e.mods = err, nil
					break
				}
				e.mods = append(e.mods, m)
			}
		}
	}
	shippedCacheEntries[line] = e
	return e
}

func cacheNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case map[string]any:
		switch n["$num"] {
		case "+Inf":
			return posInf, true
		case "-Inf":
			return negInf, true
		}
	}
	return 0, false
}

func cacheMod(r map[string]any) (mod.Mod, error) {
	name, _ := r["name"].(string)
	typ, _ := r["type"].(string)
	value := r["value"]
	var m mod.Mod
	switch {
	case typ == "FLAG":
		// Flag() reads the value for truth, and every Lua value but false and nil is true.
		b, isBool := value.(bool)
		m = mod.NewFlag(name, !isBool || b)
		if n, ok := cacheArithNumber(value); ok {
			m.Value().SetFloat(n)
		}
	case typ == "LIST":
		v, err := cacheListValue(name, value)
		if err != nil {
			return nil, err
		}
		m = mod.NewList(name, v)
	default:
		if n, ok := cacheArithNumber(value); ok {
			m = mod.NewFloat(name, mod.Type(typ), n)
		} else if value == nil {
			// A skill file's statMap leaves the value nil; mergeStatSet sets it from the stat.
			m = mod.NewFloat(name, mod.Type(typ), 0)
		} else {
			v, err := cacheTableValue(value)
			if err != nil {
				return nil, err
			}
			m = mod.NewFloat(name, mod.Type(typ), 0).SetValue(mod.NewModValueList(v))
		}
	}
	flags, _ := cacheNumber(r["flags"])
	keywordFlags, _ := cacheNumber(r["keywordFlags"])
	m.Flag(mod.MFlag(flags)).KeywordFlag(mod.KeywordFlag(keywordFlags))
	if s, ok := r["source"].(string); ok {
		m.Source(mod.Source(s))
	}
	tags, _ := r["tags"].([]any)
	for _, raw := range tags {
		t, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: a tag that is not a table", name)
		}
		tag, err := cacheTag(t)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		m.Tag(tag)
	}
	return m, nil
}

func cacheTableValue(v any) (any, error) {
	switch x := v.(type) {
	case map[string]any:
		out := LuaTable{}
		for k, val := range x {
			if k == "mod" {
				if inner, ok := val.(map[string]any); ok {
					m, err := cacheMod(inner)
					if err != nil {
						return nil, err
					}
					out[k] = m
					continue
				}
			}
			conv, err := cacheTableValue(val)
			if err != nil {
				return nil, err
			}
			out[k] = conv
		}
		return out, nil
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			conv, err := cacheTableValue(val)
			if err != nil {
				return nil, err
			}
			out[i] = conv
		}
		return out, nil
	}
	if n, ok := cacheNumber(v); ok {
		return n, nil
	}
	return v, nil
}

// cacheFields reads a table's fields, refusing any it was not told about: a field nobody reads would
// drop a conjunct of PoB's own condition.
type cacheFields struct {
	t    map[string]any
	used map[string]bool
	what string
}

func newCacheFields(t map[string]any, what string) *cacheFields {
	return &cacheFields{t: t, used: map[string]bool{"type": true}, what: what}
}

func (f *cacheFields) has(k string) bool {
	_, ok := f.t[k]
	return ok
}

func (f *cacheFields) str(k string) string {
	f.used[k] = true
	s, _ := f.t[k].(string)
	return s
}

func (f *cacheFields) num(k string) (float64, bool) {
	f.used[k] = true
	return cacheArithNumber(f.t[k])
}

// cacheArithNumber reads a number used in arithmetic: Lua coerces a numeric string there, and PoB-PoE2's
// ModCache stores some tag numbers and mod values as strings.
func cacheArithNumber(v any) (float64, bool) {
	if s, ok := v.(string); ok {
		if n, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return n, true
		}
	}
	return cacheNumber(v)
}

// truthy is Lua truth: present and not false.
func (f *cacheFields) truthy(k string) bool {
	f.used[k] = true
	v, ok := f.t[k]
	if !ok || v == nil {
		return false
	}
	b, isBool := v.(bool)
	return !isBool || b
}

func (f *cacheFields) strs(one, list string) []string {
	f.used[one], f.used[list] = true, true
	if l, ok := f.t[list].([]any); ok {
		out := make([]string, 0, len(l))
		for _, v := range l {
			s, _ := v.(string)
			out = append(out, s)
		}
		return out
	}
	if s, ok := f.t[one].(string); ok {
		return []string{s}
	}
	return nil
}

func (f *cacheFields) done() error {
	var extra []string
	for k := range f.t {
		if !f.used[k] {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		return fmt.Errorf("%s carries %s, which the mod package cannot express", f.what, strings.Join(extra, "/"))
	}
	return nil
}

func cacheTag(t map[string]any) (mod.Tag, error) {
	typ, _ := t["type"].(string)
	f := newCacheFields(t, typ+" tag")
	var tag mod.Tag
	switch typ {
	case "Condition":
		tag = mod.Condition(f.strs("var", "varList")...).Neg(f.truthy("neg"))
	case "ActorCondition":
		tag = mod.ActorCondition(f.str("actor"), f.strs("var", "varList")...).Neg(f.truthy("neg"))
	case "PerStat":
		div := 1.0
		if d, ok := f.num("div"); ok {
			div = d
		}
		p := mod.PerStat(div, f.strs("stat", "statList")...)
		if f.has("actor") {
			p.Actor(f.str("actor"))
		}
		if l, ok := f.num("limit"); ok {
			p.Limit(l)
		}
		p.LimitTotal(f.truthy("limitTotal"))
		if f.has("divVar") {
			p.DivVar(f.str("divVar"))
		}
		tag = p
	case "SkillType":
		tag = mod.SkillType(f.strs("skillType", "skillTypeList")...).Neg(f.truthy("neg"))
	case "Multiplier":
		m := mod.Multiplier(f.strs("var", "varList")...)
		if d, ok := f.num("div"); ok {
			m.Div(d)
		}
		if l, ok := f.num("limit"); ok {
			m.Limit(l)
		}
		m.LimitTotal(f.truthy("limitTotal"))
		if f.has("limitVar") {
			m.LimitVar(f.str("limitVar"))
		}
		if f.has("actor") {
			m.Actor(f.str("actor"))
		}
		if g, ok := f.num("globalLimit"); ok {
			m.GlobalLimit(g)
		}
		if f.has("globalLimitKey") {
			m.GlobalLimitKey(f.str("globalLimitKey"))
		}
		if b, ok := f.num("base"); ok {
			m.Base(b)
		}
		if f.has("divVar") {
			m.DivVar(f.str("divVar"))
		}
		if f.truthy("limitNegTotal") {
			m.LimitNegTotal()
		}
		if f.truthy("invert") {
			m.Invert()
		}
		// calcs.mergeSkillInstanceMods reads scalar from the skill's mod table; ModStore does not.
		f.used["scalar"] = true
		tag = m
	case "PercentStat":
		p, _ := f.num("percent")
		ps := mod.PercentStat(f.str("stat"), p)
		if f.has("statList") {
			ps = mod.PercentStatList(p, f.strs("", "statList")...)
		}
		if f.has("percentVar") {
			ps.PercentVar = f.str("percentVar")
		}
		if f.truthy("floor") {
			ps.Floor()
		}
		tag = ps
	case "StatThreshold":
		stat := f.str("stat")
		var st *mod.StatThresholdTag
		if th, ok := f.num("threshold"); ok {
			st = mod.StatThreshold(stat, th)
		} else {
			st = mod.StatThresholdStat(stat, f.str("thresholdStat"))
		}
		if f.has("statList") {
			st.Stats(f.strs("", "statList")...)
		}
		tag = st.Upper(f.truthy("upper"))
	case "MultiplierThreshold":
		mt := mod.MultiplierThreshold(f.str("var"))
		if th, ok := f.num("threshold"); ok {
			mt.Threshold(th)
		}
		mt.Upper(f.truthy("upper")).Equals(f.truthy("equals"))
		if f.has("thresholdVar") {
			mt.ThresholdVar(f.str("thresholdVar"))
		}
		if f.has("actor") {
			mt.Actor(f.str("actor"))
		}
		if f.has("thresholdActor") {
			mt.ThresholdActor(f.str("thresholdActor"))
		}
		// calcs.mergeSkillInstanceMods reads scalar from the skill's mod table; ModStore does not.
		f.used["scalar"] = true
		tag = mt
	case "SkillName":
		tag = mod.SkillName(f.strs("skillName", "skillNameList")...).Transfigured(f.truthy("includeTransfigured")).
			Neg(f.truthy("neg")).SummonSkill(f.truthy("summonSkill")).PartialMatch(f.truthy("partialMatch"))
	case "SlotName":
		tag = mod.SlotName(f.strs("slotName", "slotNameList")...)
	case "SocketedIn":
		s := mod.SocketedIn(f.str("slotName"))
		if f.has("keyword") {
			s.Keyword(f.str("keyword"))
		}
		if f.has("socketColor") {
			s.SocketColor(f.str("socketColor"))
		}
		if f.has("sockets") {
			switch v := t["sockets"].(type) {
			case string:
				if v != "all" {
					return nil, fmt.Errorf("SocketedIn sockets %q", v)
				}
				s.SocketsAll()
			case float64:
				s.SocketsCount(v)
			case []any:
				ids := make([]int, 0, len(v))
				for _, x := range v {
					n, _ := cacheNumber(x)
					ids = append(ids, int(n))
				}
				s.Sockets(ids...)
			default:
				return nil, fmt.Errorf("SocketedIn sockets of an unexpected shape")
			}
			f.used["sockets"] = true
		}
		tag = s
	case "SkillId":
		tag = mod.SkillId(f.str("skillId"))
	case "ItemCondition":
		ic := mod.ItemCondition(f.str("itemSlot"))
		if f.has("rarityCond") {
			ic.Rarity(f.str("rarityCond"))
		}
		tag = ic
	case "GlobalEffect":
		g := mod.GlobalEffect(f.str("effectType"))
		if f.has("effectName") {
			g.Name(f.str("effectName"))
		}
		g.EffectCond(f.str("effectCond")).EffectEnemyCond(f.str("effectEnemyCond")).ModCond(f.str("modCond"))
		g.EffectStackVar(f.str("effectStackVar")).EffectStackLimitVar(f.str("effectStackLimitVar"))
		if l, ok := f.num("effectStackLimit"); ok {
			g.EffectStackLimit(l)
		}
		g.ApplyNotPlayer(f.truthy("applyNotPlayer")).ApplyMinions(f.truthy("applyMinions")).AllowTotemBuff(f.truthy("allowTotemBuff"))
		tag = g.Unscalable(f.truthy("unscalable"))
	case "GemTag":
		tag = mod.GemTag(f.strs("gemTag", "gemTagList")...).Neg(f.truthy("neg"))
	case "SlotNumber":
		n, _ := f.num("num")
		tag = mod.SlotNumber(int(n))
	case "InSlot":
		n, _ := f.num("num")
		tag = mod.InSlot(int(n))
	case "BaseFlag":
		tag = mod.BaseFlag(f.str("baseFlag")).Neg(f.truthy("neg"))
	case "DistanceRamp":
		f.used["ramp"] = true
		var ramp [][]float64
		rows, _ := t["ramp"].([]any)
		for _, r := range rows {
			pair, _ := r.([]any)
			var out []float64
			for _, v := range pair {
				n, _ := cacheNumber(v)
				out = append(out, n)
			}
			ramp = append(ramp, out)
		}
		tag = mod.DistanceRamp(ramp)
	case "DisablesItem":
		tag = mod.DisablesItem(f.str("slotName"))
	case "KeywordFlagAnd":
		n, _ := f.num("keywordFlags")
		tag = mod.KeywordFlagAnd(mod.KeywordFlag(n))
	case "Global":
		tag = mod.Global()
	case "ModFlagOr":
		n, _ := f.num("modFlags")
		tag = mod.ModFlagOr(mod.MFlag(n))
	case "Limit":
		var l *mod.LimitTag
		if n, ok := f.num("limit"); ok {
			l = mod.Limit(n)
		} else {
			l = mod.LimitVar(f.str("limitVar"))
		}
		tag = l.Neg(f.truthy("neg"))
	default:
		return nil, fmt.Errorf("a %q tag, which the mod package has no tag for", typ)
	}
	if err := f.done(); err != nil {
		return nil, err
	}
	return tag, nil
}

func cacheNestedMod(f *cacheFields) (mod.Mod, error) {
	f.used["mod"] = true
	inner, ok := f.t["mod"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s without a mod", f.what)
	}
	return cacheMod(inner)
}

func cacheListValue(name string, value any) (any, error) {
	t, ok := value.(map[string]any)
	if !ok {
		return cacheTableValue(value)
	}
	f := newCacheFields(t, name+" payload")
	delete(f.used, "type")
	var out any
	switch name {
	case "GemProperty":
		v, _ := f.num("value")
		g := mod.GemProperty{Key: f.str("key"), Value: v}
		if f.has("keyword") {
			kw := f.str("keyword")
			g.Keyword = &kw
		}
		if f.has("keywordList") {
			g.KeywordList = f.strs("", "keywordList")
		}
		// keyOfScaledMod is always "value" here; GemProperty scales its Value.
		if f.str("keyOfScaledMod") != "value" {
			return nil, fmt.Errorf("GemProperty scales a key other than value")
		}
		g.KeyOfScaledMod = "value"
		if f.has("gemRequirements") {
			f.used["gemRequirements"] = true
			reqs, ok := t["gemRequirements"].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("GemProperty gemRequirements of an unexpected shape")
			}
			g.GemRequirements = make(map[string]float64, len(reqs))
			for k, v := range reqs {
				n, ok := cacheNumber(v)
				if !ok {
					return nil, fmt.Errorf("GemProperty gemRequirements.%s is not a number", k)
				}
				g.GemRequirements[k] = n
			}
		}
		out = g
	case "MinionModifier", "EnemyModifier", "ExtraSkillMod", "ExtraAuraEffect", "ExtraAura":
		m, err := cacheNestedMod(f)
		if err != nil {
			return nil, err
		}
		switch name {
		case "MinionModifier":
			out = mod.MinionModifier{Mod: m}
		case "EnemyModifier":
			out = mod.EnemyModifier{Mod: m}
		case "ExtraSkillMod":
			out = mod.ExtraSkillMod{Mod: m}
		case "ExtraAuraEffect":
			out = mod.ExtraAuraEffect{Mod: m}
		case "ExtraAura":
			// A tag the parser wrote into the payload's array part; ModStore never reads it.
			f.used["1"] = true
			out = mod.ExtraAura{Mod: m, OnlyAllies: f.truthy("onlyAllies")}
		}
	case "SkillData":
		f.used["value"] = true
		sd := mod.SkillData{Key: f.str("key")}
		if f.has("merge") {
			sd.Merge = f.str("merge")
		}
		switch v := t["value"].(type) {
		case string:
			sd.String = v
		case nil:
			// A skill file's statMap leaves the value nil; mergeStatSet sets it from the stat.
		case bool:
			// skill(key, true): the mod package keeps a SkillData truth as 1, as the PoE1 tables do.
			if v {
				sd.Value = 1
			}
		default:
			n, ok := cacheNumber(v)
			if !ok {
				return nil, fmt.Errorf("SkillData value of an unexpected shape")
			}
			sd.Value = n
		}
		out = sd
	case "ExtraSkill":
		lvl, _ := f.num("level")
		out = mod.ExtraSkill{SkillID: f.str("skillId"), SkillName: f.str("name"), Level: int(lvl),
			NoSupports: f.truthy("noSupports"), Triggered: f.truthy("triggered")}
	case "ExtraSupport":
		lvl, _ := f.num("level")
		out = mod.ExtraSupport{SkillID: f.str("skillId"), Level: int(lvl)}
	case "SupportedGemProperty":
		v, _ := f.num("value")
		out = mod.SupportedGemProperty{Keyword: f.str("keyword"), Key: f.str("key"), Value: int(v)}
	case "WeaponData":
		v, ok := f.num("value")
		out = mod.WeaponData{Key: f.str("key"), Value: v, Unset: !ok}
	case "ExplodeMod":
		v, _ := f.num("value")
		amount, _ := f.num("amount")
		out = ExplodeMod{Type: f.str("type"), Value: v, Amount: amount, KeyOfScaledMod: f.str("keyOfScaledMod")}
	case "ExtraMinionSkill":
		// A skill's stat map leaves the stat's number in value; createMinionSkills reads only skillId.
		f.used["value"] = true
		out = mod.ExtraMinionSkill{SkillID: f.str("skillId")}
	default:
		return cacheTableValue(value)
	}
	if err := f.done(); err != nil {
		return nil, err
	}
	return out, nil
}
