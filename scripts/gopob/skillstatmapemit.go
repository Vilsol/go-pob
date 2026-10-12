package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

const ssmUsage = "usage: skill_stat_map_emit.py <dump.json> <out.go> [data/raw/skill_stat_map.go]"

var (
	ssmGlobalEffectOK = ssmSet("type", "effectType", "effectName", "unscalable",
		"effectCond", "effectEnemyCond", "modCond",
		"effectStackVar", "effectStackLimit", "effectStackLimitVar",
		"applyNotPlayer", "applyMinions", "allowTotemBuff", "div", "limit")
	ssmMultiplierOK = ssmSet("type", "var", "limitVar", "div", "limit", "limitTotal",
		"actor", "limitActor", "base", "noFloor", "divVar", "limitStat", "limitNegTotal")
	ssmMultiplierThresholdOK = ssmSet("type", "var", "threshold", "thresholdVar", "upper", "actor")
	ssmPerStatOK             = ssmSet("type", "stat", "statList", "div", "limit", "limitVar",
		"limitTotal", "actor")
	ssmPercentStatOK   = ssmSet("type", "stat", "percent", "percentVar", "actor", "floor")
	ssmStatThresholdOK = ssmSet("type", "stat", "threshold", "thresholdStat", "upper")
	ssmSkillDataOK     = ssmSet("key", "value", "merge")
)

const ssmSkillFlagLine = "\t\t\tSkillFlag: %s,"

func ssmModLines(mods []string, indent string) []string {
	switch {
	case len(mods) == 1:
		return []string{indent + "Mods: []mod.Mod{" + mods[0] + "},"}
	case len(mods) > 1:
		out := []string{indent + "Mods: []mod.Mod{"}
		for _, m := range mods {
			out = append(out, indent+"\t"+m+",")
		}
		return append(out, indent+"},")
	}
	return []string{indent + "Mods: []mod.Mod{},"}
}

func ssmSet(keys ...string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

var ssmGlobalKey = regexp.MustCompile(`(?m)^\t"([^"]+)": \{$`)

type ssmRun struct {
	vocab        *mcVocab
	skillTypes   map[string]any
	omit         *pyCounter
	omitDetail   *pyListMap
	boolLiterals map[string]int
}

func ssmExtra(t map[string]any, ok map[string]bool) []string {
	bad := []string{}
	for k := range t {
		if !ok[k] {
			bad = append(bad, k)
		}
	}
	sort.Strings(bad)
	return bad
}

func (r *ssmRun) numericLiteral(v any, site string) string {
	if b, ok := v.(bool); ok {
		r.boolLiterals[site]++
		if b {
			return "1"
		}
		return "0"
	}
	if _, ok := oeFloat(v); ok {
		return mcGoNum(v)
	}
	return "0"
}

func (r *ssmRun) reportBoolLiterals() {
	if len(r.boolLiterals) == 0 {
		fmt.Println("boolean mod values: 0 (no site had a population this run)")
		return
	}
	sites := make([]string, 0, len(r.boolLiterals))
	for k := range r.boolLiterals {
		sites = append(sites, k)
	}
	sort.Strings(sites)
	for _, site := range sites {
		fmt.Printf("boolean mod values emitted as 1/0: %4d at %s\n", r.boolLiterals[site], site)
	}
}

func ssmStr(t map[string]any, key string) string { return mcGoStr(oeStr(t[key])) }

//nolint:gocyclo,maintidx // a transliteration of one Lua-shaped dispatch; splitting it would make
func (r *ssmRun) emitTag(tAny any) (string, error) {
	t, ok := tAny.(map[string]any)
	if !ok {
		return "", mcUnrep("non-dict tag: a bare %s in the tags array, from a skill() call "+
			"written as if it took flags positionally [pob:Modules/Data.lua:52]", pyTypeName(tAny))
	}

	for _, fv := range t {
		if mcHasTable(fv) {
			return "", mcUnrep("a tag field is a table [unverifiable: the tag shape is data-dependent]")
		}
	}

	switch oeStr(t["type"]) {
	case "GlobalEffect":
		if bad := ssmExtra(t, ssmGlobalEffectOK); len(bad) > 0 {
			return "", mcUnrep("GlobalEffect tag carries %s, which mod.GlobalEffectTag cannot "+
				"express [go:mod.GlobalEffectTag]", strings.Join(bad, "/"))
		}
		et, err := sbmReq(t, "effectType", "GlobalEffect")
		if err != nil {
			return "", err
		}
		s := "mod.GlobalEffect(" + mcGoStr(oeStr(et)) + ")"
		if oeTruthy(t["effectName"]) {
			s += ".Name(" + ssmStr(t, "effectName") + ")"
		}
		if oeTruthy(t["unscalable"]) {
			s += ".Unscalable(true)"
		}
		if oeTruthy(t["effectCond"]) {
			s += ".EffectCond(" + ssmStr(t, "effectCond") + ")"
		}
		if oeTruthy(t["effectEnemyCond"]) {
			s += ".EffectEnemyCond(" + ssmStr(t, "effectEnemyCond") + ")"
		}
		if oeTruthy(t["modCond"]) {
			s += ".ModCond(" + ssmStr(t, "modCond") + ")"
		}
		if oeTruthy(t["effectStackVar"]) {
			s += ".EffectStackVar(" + ssmStr(t, "effectStackVar") + ")"
		}
		if oeTruthy(t["effectStackLimit"]) {
			s += ".EffectStackLimit(" + mcGoNum(t["effectStackLimit"]) + ")"
		}
		if oeTruthy(t["effectStackLimitVar"]) {
			s += ".EffectStackLimitVar(" + ssmStr(t, "effectStackLimitVar") + ")"
		}
		if oeTruthy(t["applyNotPlayer"]) {
			s += ".ApplyNotPlayer(true)"
		}
		if oeTruthy(t["applyMinions"]) {
			s += ".ApplyMinions(true)"
		}
		if oeTruthy(t["allowTotemBuff"]) {
			s += ".AllowTotemBuff(true)"
		}
		if _, ok := t["div"]; ok {
			s += ".Div(" + mcGoNum(t["div"]) + ")"
		}
		if _, ok := t["limit"]; ok {
			s += ".Limit(" + mcGoNum(t["limit"]) + ")"
		}
		return s, nil

	case "Multiplier":
		if bad := ssmExtra(t, ssmMultiplierOK); len(bad) > 0 {
			return "", mcUnrep("Multiplier tag carries %s, which mod.MultiplierTag cannot "+
				"express [go:mod.MultiplierTag]", strings.Join(bad, "/"))
		}
		v, err := sbmReq(t, "var", "Multiplier")
		if err != nil {
			return "", err
		}
		s := "mod.Multiplier(" + mcGoStr(oeStr(v)) + ")"
		if _, ok := t["limitVar"]; ok {
			s = fmt.Sprintf("multiplierLimitVar(%s, %s)", mcGoStr(oeStr(v)), ssmStr(t, "limitVar"))
		}
		if x, ok := t["div"]; ok {
			s += ".Div(" + mcGoNum(x) + ")"
		}
		if x, ok := t["base"]; ok {
			s += ".Base(" + mcGoNum(x) + ")"
		}
		if x, ok := t["limit"]; ok {
			s += ".Limit(" + mcGoNum(x) + ")"
		}
		if oeTruthy(t["limitTotal"]) {
			s += ".LimitTotal(true)"
		}
		if _, ok := t["actor"]; ok {
			s += ".Actor(" + ssmStr(t, "actor") + ")"
		}
		if _, ok := t["limitActor"]; ok {
			s += ".LimitActor(" + ssmStr(t, "limitActor") + ")"
		}
		if _, ok := t["divVar"]; ok {
			s += ".DivVar(" + ssmStr(t, "divVar") + ")"
		}
		if _, ok := t["limitStat"]; ok {
			s += ".LimitStat(" + ssmStr(t, "limitStat") + ")"
		}
		if oeTruthy(t["noFloor"]) {
			s += ".NoFloor()"
		}
		if oeTruthy(t["limitNegTotal"]) {
			s += ".LimitNegTotal()"
		}
		return s, nil

	case "MultiplierThreshold":
		if bad := ssmExtra(t, ssmMultiplierThresholdOK); len(bad) > 0 {
			return "", mcUnrep("MultiplierThreshold tag carries %s, which "+
				"mod.MultiplierThresholdTag cannot express [go:mod.MultiplierThresholdTag]",
				strings.Join(bad, "/"))
		}
		v, err := sbmReq(t, "var", "MultiplierThreshold")
		if err != nil {
			return "", err
		}
		s := "mod.MultiplierThreshold(" + mcGoStr(oeStr(v)) + ")"
		switch {
		case ssmHas(t, "threshold"):
			s += ".Threshold(" + mcGoNum(t["threshold"]) + ")"
		case ssmHas(t, "thresholdVar"):
			s += ".ThresholdVar(" + ssmStr(t, "thresholdVar") + ")"
		default:
			return "", mcUnrep("MultiplierThreshold tag carries neither threshold nor " +
				"thresholdVar [pob:src/Classes/ModStore.lua:331]")
		}
		if oeTruthy(t["upper"]) {
			s += ".Upper(true)"
		}
		if _, ok := t["actor"]; ok {
			s += ".Actor(" + ssmStr(t, "actor") + ")"
		}
		return s, nil

	case "PerStat":
		if bad := ssmExtra(t, ssmPerStatOK); len(bad) > 0 {
			return "", mcUnrep("PerStat tag carries %s, which mod.PerStatTag cannot express "+
				"[go:mod.PerStatTag]", strings.Join(bad, "/"))
		}
		stats, err := ssmVarList(t, "statList", "stat", "PerStat")
		if err != nil {
			return "", err
		}
		div := any(json1)
		if d, ok := t["div"]; ok {
			div = d
		}
		s := fmt.Sprintf("mod.PerStat(%s, %s)", mcGoNum(div), strings.Join(stats, ", "))
		if _, ok := t["limitVar"]; ok {
			s += ".LimitVar(" + ssmStr(t, "limitVar") + ")"
		}
		if x, ok := t["limit"]; ok {
			s += ".Limit(" + mcGoNum(x) + ")"
		}
		if oeTruthy(t["limitTotal"]) {
			s += ".LimitTotal(true)"
		}
		if _, ok := t["actor"]; ok {
			s += ".Actor(" + ssmStr(t, "actor") + ")"
		}
		return s, nil

	case "PercentStat":
		if bad := ssmExtra(t, ssmPercentStatOK); len(bad) > 0 {
			return "", mcUnrep("PercentStat tag carries %s, which mod.PercentStatTag cannot "+
				"express [go:mod.PercentStatTag]", strings.Join(bad, "/"))
		}
		stat, err := sbmReq(t, "stat", "PercentStat")
		if err != nil {
			return "", err
		}
		var s string
		switch {
		case ssmHas(t, "percent"):
			s = fmt.Sprintf("mod.PercentStat(%s, %s)",
				mcGoStr(oeStr(stat)), mcGoNum(t["percent"]))
		case oeTruthy(t["percentVar"]):
			s = fmt.Sprintf("mod.PercentStatVar(%s, %s)",
				mcGoStr(oeStr(stat)), ssmStr(t, "percentVar"))
		default:
			return "", mcUnrep("PercentStat with neither percent nor percentVar " +
				"[go:mod.PercentStatVar]")
		}
		if _, ok := t["actor"]; ok {
			s += ".Actor(" + ssmStr(t, "actor") + ")"
		}
		if oeTruthy(t["floor"]) {
			s += ".Floor()"
		}
		return s, nil

	case "Condition":
		vars, err := ssmVarList(t, "varList", "var", "Condition")
		if err != nil {
			return "", err
		}
		s := "mod.Condition(" + strings.Join(vars, ", ") + ")"
		if oeTruthy(t["neg"]) {
			s += ".Neg(true)"
		}
		return s, nil

	case "ActorCondition":
		vars, err := ssmVarList(t, "varList", "var", "ActorCondition")
		if err != nil {
			return "", err
		}
		actor, err := sbmReq(t, "actor", "ActorCondition")
		if err != nil {
			return "", err
		}
		s := fmt.Sprintf("mod.ActorCondition(%s, %s)",
			mcGoStr(oeStr(actor)), strings.Join(vars, ", "))
		if oeTruthy(t["neg"]) {
			s += ".Neg(true)"
		}
		return s, nil

	case "SkillType":
		//go-only: THE PYTHON ORIGINAL RAISED Unrepresentable HERE AND THIS ARM IS NEW.
		if l, ok := t["skillTypeList"].([]any); ok {
			if len(l) == 0 {
				return "", mcUnrep("SkillType tag carries an EMPTY skillTypeList " +
					"[pob:src/Classes/ModStore.lua:820-837]")
			}
			names := make([]string, 0, len(l))
			for _, x := range l {
				n, ok := gemsInt(x)
				if !ok {
					return "", mcUnrep("skillTypeList entry %s is not a number "+
						"[pob:src/Data/Global.lua:157-278]", pyRepr(oeStr(x)))
				}
				name, ok := r.skillTypes[fmt.Sprintf("%d", n)]
				if !ok || name == nil {
					return "", mcUnrep("skillTypeList entry %s has no name in Data/Global.lua "+
						"[pob:src/Data/Global.lua:157-278]", oeStr(x))
				}
				names = append(names, mcGoStr(oeStr(name)))
			}
			s := "mod.SkillType(" + strings.Join(names, ", ") + ")"
			if oeTruthy(t["neg"]) {
				s += ".Neg(true)"
			}
			return s, nil
		}
		var s string
		if !ssmHas(t, "skillType") {
			s = "mod.SkillType()"
		} else {
			n, ok := gemsInt(t["skillType"])
			if !ok {
				return "", mcUnrep("SkillType %s is not a number "+
					"[pob:src/Data/Global.lua:157-278]", pyRepr(oeStr(t["skillType"])))
			}
			name, ok := r.skillTypes[fmt.Sprintf("%d", n)]
			if !ok || name == nil {
				return "", mcUnrep("SkillType %s has no name in Data/Global.lua "+
					"[pob:src/Data/Global.lua:157-278]", oeStr(t["skillType"]))
			}
			s = "mod.SkillType(" + mcGoStr(oeStr(name)) + ")"
		}
		if oeTruthy(t["neg"]) {
			s += ".Neg(true)"
		}
		return s, nil

	case "SkillName":
		n, err := sbmReq(t, "skillName", "SkillName")
		if err != nil {
			return "", err
		}
		s := "mod.SkillName(" + mcGoStr(oeStr(n)) + ")"
		if oeTruthy(t["neg"]) {
			s += ".Neg(true)"
		}
		if oeTruthy(t["includeTransfigured"]) {
			s += ".Transfigured(true)"
		}
		if oeTruthy(t["summonSkill"]) {
			s += ".SummonSkill(true)"
		}
		return s, nil

	case "SkillId":
		if oeTruthy(t["neg"]) {
			return "", mcUnrep("SkillId tag carries neg, which the SkillId case does not " +
				"evaluate [pob:src/Classes/ModStore.lua:525-528]")
		}
		id, err := sbmReq(t, "skillId", "SkillId")
		if err != nil {
			return "", err
		}
		return "mod.SkillId(" + mcGoStr(oeStr(id)) + ")", nil

	case "SkillPart":
		if oeTruthy(t["neg"]) {
			return "", mcUnrep("SkillPart tag carries neg, which mod.SkillPartTag exposes as a " +
				"field with no setter [go-absent:SkillPartTag) Neg(]")
		}
		if spl, ok := t["skillPartList"].([]any); ok {
			parts := make([]string, 0, len(spl))
			for _, p := range spl {
				parts = append(parts, mcGoNum(p))
			}
			return "mod.SkillPartList(" + strings.Join(parts, ", ") + ")", nil
		}
		sp, err := sbmReq(t, "skillPart", "SkillPart")
		if err != nil {
			return "", err
		}
		return "mod.SkillPart(" + mcGoNum(sp) + ")", nil

	case "StatThreshold":
		if bad := ssmExtra(t, ssmStatThresholdOK); len(bad) > 0 {
			return "", mcUnrep("StatThreshold tag carries %s, which mod.StatThresholdTag cannot "+
				"express [go:mod.StatThresholdTag]", strings.Join(bad, "/"))
		}
		stat, err := sbmReq(t, "stat", "StatThreshold")
		if err != nil {
			return "", err
		}
		th, err := sbmReq(t, "threshold", "StatThreshold")
		if err != nil {
			return "", err
		}
		s := fmt.Sprintf("mod.StatThreshold(%s, %s)", mcGoStr(oeStr(stat)), mcGoNum(th))
		if oeTruthy(t["upper"]) {
			s += ".Upper(true)"
		}
		return s, nil

	case "DistanceRamp":
		ramp, err := sbmReq(t, "ramp", "DistanceRamp")
		if err != nil {
			return "", err
		}
		rows, _ := ramp.([]any)
		pairs := make([]string, 0, len(rows))
		for _, row := range rows {
			pair, ok := row.([]any)
			if !ok || len(pair) != 2 {
				return "", mcUnrep("DistanceRamp ramp is not a list of pairs [unverifiable: the " +
					"ramp shape is data-dependent]")
			}
			pairs = append(pairs, fmt.Sprintf("{%s, %s}", mcGoNum(pair[0]), mcGoNum(pair[1])))
		}
		return "mod.DistanceRamp([][]float64{" + strings.Join(pairs, ", ") + "})", nil

	case "MeleeProximity":
		ramp, err := sbmReq(t, "ramp", "MeleeProximity")
		if err != nil {
			return "", err
		}
		rows, _ := ramp.([]any)
		vals := make([]string, 0, len(rows))
		for _, x := range rows {
			vals = append(vals, mcGoNum(x))
		}
		return "mod.MeleeProximity([]int{" + strings.Join(vals, ", ") + "})", nil

	case "ModFlagOr":
		mf, err := sbmReq(t, "modFlags", "ModFlagOr")
		if err != nil {
			return "", err
		}
		f, ok := mcBits(mf, r.vocab.mflag)
		if !ok {
			return "", mcUnrep("ModFlagOr flags %s are not a union of named MFlag bits "+
				"[unverifiable: the flag value is data-dependent; MFLAG_BITS in this file is the "+
				"named set]", oeStr(mf))
		}
		return "mod.ModFlagOr(" + f + ")", nil

	case "Limit":
		if ssmHas(t, "limit") {
			return "mod.Limit(" + mcGoNum(t["limit"]) + ")", nil
		}
		lv, err := sbmReq(t, "limitVar", "Limit")
		if err != nil {
			return "", err
		}
		return "mod.LimitVar(" + mcGoStr(oeStr(lv)) + ")", nil

	case "MonsterTag":
		var names []string
		if lst, ok := t["monsterTagList"].([]any); ok {
			for _, n := range lst {
				names = append(names, mcGoStr(oeStr(n)))
			}
		} else {
			mt, err := sbmReq(t, "monsterTag", "MonsterTag")
			if err != nil {
				return "", err
			}
			names = append(names, mcGoStr(oeStr(mt)))
		}
		s := "mod.MonsterTag(" + strings.Join(names, ", ") + ")"
		if oeTruthy(t["neg"]) {
			s += ".Neg(true)"
		}
		return s, nil
	}
	if tt, ok := t["type"]; ok {
		if _, isStr := tt.(string); !isStr {
			return "", mcUnrep("a tag's 'type' is not a string " +
				"[unverifiable: the tag shape is data-dependent]")
		}
	}
	return "", mcUnrep("unknown tag type %s [unverifiable: a new tag type appears only when PoB "+
		"adds one]", oeStr(t["type"]))
}

func ssmHas(t map[string]any, key string) bool {
	_, ok := t[key]
	return ok
}

func ssmVarList(t map[string]any, listKey, oneKey, what string) ([]string, error) {
	var items []any
	if l, ok := t[listKey].([]any); ok {
		items = l
	} else {
		one, err := sbmReq(t, oneKey, what)
		if err != nil {
			return nil, err
		}
		items = []any{one}
	}
	out := make([]string, 0, len(items))
	for _, x := range items {
		out = append(out, mcGoStr(oeStr(x)))
	}
	return out, nil
}

func ssmNestedMod(v map[string]any) (map[string]any, error) {
	tags := []any{}
	for i := 1; ; i++ {
		x, ok := v[fmt.Sprintf("%d", i)]
		if !ok {
			break
		}
		tags = append(tags, x)
	}
	name, err := sbmReq(v, "name", "nested mod")
	if err != nil {
		return nil, err
	}
	mtype, err := sbmReq(v, "type", "nested mod")
	if err != nil {
		return nil, err
	}
	flags := any(json0)
	if f, ok := v["flags"]; ok {
		flags = f
	}
	kw := any(json0)
	if k, ok := v["keywordFlags"]; ok {
		kw = k
	}
	return map[string]any{
		"name": name, "type": mtype, "value": v["value"],
		"flags": flags, "keywordFlags": kw, "tags": tags,
	}, nil
}

func (r *ssmRun) emitMod(m map[string]any) (string, error) {
	if err := mcReqMod(m); err != nil {
		return "", err
	}
	if _, ok := oeFloat(m["flags"]); !ok || pyTypeName(m["flags"]) == "bool" {
		return "", mcUnrep("mod.flags holds a tag table (upstream arity bug), so the mod has no " +
			"tags and a nonsense flag set [pob:src/Data/Skills/act_int.lua:2621]")
	}
	if !ssmHas(m, "name") || !ssmHas(m, "type") {
		if extra := mcExtraKeys(m); len(extra) > 0 {
			return "", mcUnrep("nested mod group carrying the per-mod scalar %s, which raw.StatMap "+
				"cannot express: its Div/Mult are per-ENTRY and PoB writes one per GROUP "+
				"[go:raw.StatMap]", pyRepr(strings.Join(extra, "/")))
		}
		return "", mcUnrep("nested mod: name/type live inside tags[0], which mod/ has no " +
			"constructor for [go-absent:mod.NestedMod]")
	}
	if extra := mcExtraKeys(m); len(extra) > 0 {
		return "", mcUnrep("mod carries %s beyond the known mod fields, which this emitter neither "+
			"models nor declares [go:mod.Mod]", pyRepr(strings.Join(extra, "/")))
	}
	name, mtype := oeStr(m["name"]), oeStr(m["type"])
	val, err := mcUnwrap(m["value"])
	if err != nil {
		return "", err
	}
	if vm, isMap := val.(map[string]any); isMap {
		for fk, fv := range vm {
			if fk != "mod" && mcHasTable(fv) {
				return "", mcUnrep("a mod value field is a table [unverifiable: the value shape is data-dependent]")
			}
		}
	}

	var base string
	switch {
	case mtype == "LIST" && name == "SkillData":
		vm, ok := val.(map[string]any)
		if !ok {
			return "", mcUnrep("SkillData LIST value is not a table [unverifiable: the value " +
				"shape is data-dependent]")
		}
		if bad := ssmExtra(vm, ssmSkillDataOK); len(bad) > 0 {
			return "", mcUnrep("SkillData LIST value carries %s, which mod.SkillData cannot "+
				"express [go:mod.SkillData]", strings.Join(bad, "/"))
		}
		lit := r.numericLiteral(vm["value"], "SkillData")
		key, err := sbmReq(vm, "key", "SkillData")
		if err != nil {
			return "", err
		}
		if oeTruthy(vm["merge"]) {
			base = fmt.Sprintf("mod.NewList(\"SkillData\", mod.SkillData{Key: %s, Value: %s, "+
				"Merge: %s})", mcGoStr(oeStr(key)), lit, mcGoStr(oeStr(vm["merge"])))
		} else {
			base = fmt.Sprintf("skill(%s, %s)", mcGoStr(oeStr(key)), lit)
		}
	case mtype == "LIST" && name == "ExtraMinionSkill":
		vm, _ := val.(map[string]any)
		if vm == nil {
			return "", mcUnrep("ExtraMinionSkill tag has no %s "+
				"[unverifiable: the value shape is data-dependent]", pyRepr("skillId"))
		}
		id, err := sbmReq(vm, "skillId", "ExtraMinionSkill")
		if err != nil {
			return "", err
		}
		base = "mod.NewList(\"ExtraMinionSkill\", mod.ExtraMinionSkill{SkillID: " +
			mcGoStr(oeStr(id)) + "})"
	case mtype == "LIST" && name == "MinionModifier":
		vm, ok := val.(map[string]any)
		if !ok || !ssmHas(vm, "mod") {
			return "", mcUnrep("MinionModifier value carries no nested `mod` table " +
				"[unverifiable: the value shape is data-dependent]")
		}
		inner, ok := vm["mod"].(map[string]any)
		if !ok {
			return "", mcUnrep("MinionModifier value carries no nested `mod` table " +
				"[unverifiable: the value shape is data-dependent]")
		}
		reshaped, err := ssmNestedMod(inner)
		if err != nil {
			return "", err
		}
		nested, err := r.emitMod(reshaped)
		if err != nil {
			return "", err
		}
		base = "mod.NewList(\"MinionModifier\", mod.MinionModifier{Mod: " + nested + "})"
	case mtype == "LIST" && strings.HasSuffix(name, "PactDamage"):
		vm, ok := val.(map[string]any)
		if !ok || !ssmHas(vm, "mod") {
			return "", mcUnrep("%s value carries no nested `mod` table "+
				"[unverifiable: the value shape is data-dependent]", name)
		}
		inner, ok := vm["mod"].(map[string]any)
		if !ok {
			return "", mcUnrep("%s value carries no nested `mod` table "+
				"[unverifiable: the value shape is data-dependent]", name)
		}
		reshaped, err := ssmNestedMod(inner)
		if err != nil {
			return "", err
		}
		nested, err := r.emitMod(reshaped)
		if err != nil {
			return "", err
		}
		base = "mod.NewList(" + mcGoStr(name) + ", mod.PactDamage{Mod: " + nested + "})"
	case mtype == "LIST" && name == "SupportedGemProperty":
		vm, _ := val.(map[string]any)
		if vm == nil {
			return "", mcUnrep("SupportedGemProperty tag has no %s "+
				"[unverifiable: the value shape is data-dependent]", pyRepr("keyword"))
		}
		kw, err := sbmReq(vm, "keyword", "SupportedGemProperty")
		if err != nil {
			return "", err
		}
		key, err := sbmReq(vm, "key", "SupportedGemProperty")
		if err != nil {
			return "", err
		}
		base = fmt.Sprintf("mod.NewList(\"SupportedGemProperty\", mod.SupportedGemProperty{"+
			"Keyword: %s, Key: %s, Value: 0})", mcGoStr(oeStr(kw)), mcGoStr(oeStr(key)))
	case mtype == "LIST":
		return "", mcUnrep("unhandled LIST mod %s [unverifiable: the mod name is data-dependent]",
			name)
	case mtype == "FLAG":
		base = "mod.NewFlag(" + mcGoStr(name) + ", true)"
	default:
		goType, ok := mcModType[mtype]
		if !ok {
			return "", mcUnrep("unhandled mod type %s [unverifiable: the mod type is "+
				"data-dependent]", mtype)
		}
		base = fmt.Sprintf("mod.NewFloat(%s, %s, %s)", mcGoStr(name), goType,
			r.numericLiteral(val, goType))
	}

	if oeTruthy(m["flags"]) {
		f, ok := mcBits(m["flags"], r.vocab.mflag)
		if !ok {
			return "", mcUnrep("flags %s are not a union of named MFlag bits [unverifiable: the "+
				"flag value is data-dependent; MFLAG_BITS in this file is the named set]",
				oeStr(m["flags"]))
		}
		base += ".Flag(" + f + ")"
	}
	if oeTruthy(m["keywordFlags"]) {
		k, ok := mcBits(m["keywordFlags"], r.vocab.kw)
		if !ok {
			return "", mcUnrep("keywordFlags %s are not a union of named KeywordFlag bits "+
				"[unverifiable: the flag value is data-dependent; KW_BITS in this file is the "+
				"named set]", oeStr(m["keywordFlags"]))
		}
		base += ".KeywordFlag(" + k + ")"
	}
	rawTags, _ := m["tags"].([]any)
	tags := make([]string, 0, len(rawTags))
	for _, tAny := range rawTags {
		u, err := mcUnwrap(tAny)
		if err != nil {
			return "", err
		}
		got, err := r.emitTag(u)
		if err != nil {
			return "", err
		}
		tags = append(tags, got)
	}
	if len(tags) > 0 {
		base += ".Tag(" + strings.Join(tags, ", ") + ")"
	}
	return base, nil
}

var json0 = mustNumber("0")

func (r *ssmRun) emitMods(sid, stat string, rawMods []any, keptMods *int) ([]string, error) {
	mods := []string{}
	for _, mAny := range rawMods {
		m, ok := mAny.(map[string]any)
		if !ok {
			r.omit.add("a mod is not a table [unverifiable: the dump shape is data-dependent]")
			continue
		}
		got, err := r.emitMod(m)
		if err == nil {
			mods = append(mods, got)
			*keptMods++
			continue
		}
		var u *mcUnrepresentableError
		if !errorsAs(err, &u) {
			return nil, err
		}
		r.omit.add(u.reason)
		modName := "<nameless>"
		if n, ok := m["name"]; ok {
			modName = oeStr(n)
		}
		r.omitDetail.add(u.reason, fmt.Sprintf("%s/%s/%s", sid, stat, modName))
	}
	return mods, nil
}

func (r *ssmRun) emitScalars(sid, stat, indent string, scalars map[string]any, allowSkillFlag bool) ([]string, error) {
	out := []string{}
	for _, kv := range [][2]string{{"value", "Value"}, {"mult", "Mult"},
		{"div", "Div"}, {"base", "Base"}} {
		if x, ok := scalars[kv[0]]; ok {
			u, err := mcUnwrap(x)
			if err != nil {
				return nil, errors.New(err.Error()) //nolint:wrapcheck // dropping the TYPE is the point; see above
			}
			out = append(out, fmt.Sprintf("%s%s: utils.Ptr(float64(%s)),", indent, kv[1], mcGoNum(u)))
		}
	}
	if x, ok := scalars["skillFlag"]; ok && allowSkillFlag {
		u, err := mcUnwrap(x)
		if err != nil {
			return nil, errors.New(err.Error()) //nolint:wrapcheck // flattened, not wrapped — see the numeric loop above
		}
		out = append(out, fmt.Sprintf(ssmSkillFlagLine, mcGoStr(oeStr(u))))
	}
	for _, k := range oeSortedKeys(scalars) {
		if strings.HasPrefix(k, "__") {
			continue
		}
		switch k {
		case "value", "mult", "div", "base":
		case "skillFlag":
			if !allowSkillFlag {
				return nil, fmt.Errorf("skill %s stat %s carries statMap scalar 'skillFlag' inside a "+
					"GROUP, which PoB never reads: CalcActiveSkill.lua:344 tests map.skillFlag on "+
					"the ENTRY only. Model it or declare it; emitting it as an entry flag would "+
					"invent a mapping upstream does not have.", pyRepr(sid), pyRepr(stat))
			}
		default:
			//go-only: the python original had NO such guard. It copied value/mult/div/base out
			return nil, fmt.Errorf(
				"skill %s stat %s carries statMap scalar %s, which this emitter neither "+
					"models nor declares. Model it or declare it; emitting anyway is how "+
					"skillFlag was lost on every entry that had one.",
				pyRepr(sid), pyRepr(stat), pyRepr(k))
		}
	}
	return out, nil
}

func skillStatMapEmit(args []string) int {
	if len(args) != 2 && len(args) != 3 {
		fmt.Fprintln(os.Stderr, ssmUsage)
		return 1
	}
	src, dst := args[0], args[1]
	globalMap := strings.Replace(dst, "skill_stat_maps.go", "skill_stat_map.go", 1)
	if len(args) == 3 {
		globalMap = args[2]
	}

	d, reason := pyLoadJSONObject(src)
	if reason != "" {
		if strings.HasSuffix(reason, "is not a JSON object") {
			return mcRefuse(fmt.Sprintf("%s is not a JSON object", src))
		}
		return mcRefuse(strings.Replace(reason, src, fmt.Sprintf("the dump (%s)", src), 1))
	}
	for _, field := range []string{"skillTypeNames", "skills", "skillCount", "entryCount", "modCount", "groupCount"} {
		if !ssmHas(d, field) {
			return mcRefuse(fmt.Sprintf("the dump carries no %s. Re-run oracle/skill_stat_map.lua.",
				pyRepr(field)))
		}
	}
	skills, ok := d["skills"].(map[string]any)
	if !ok {
		return mcRefuse("the dump's 'skills' is not a JSON object")
	}
	vocab, vreason := mcUseFlagTables(d)
	if vreason != "" {
		return mcRefuse(vreason)
	}
	skillTypes, _ := d["skillTypeNames"].(map[string]any)

	raw, err := os.ReadFile(globalMap) //nolint:gosec // the caller's argument, as in the original
	if err != nil {
		return mcRefuse(fmt.Sprintf("the global map (%s) is not readable: %s -- pass it as the "+
			"third argument when the output is not written beside it", globalMap, errnoText(err)))
	}
	globalKeys := map[string]bool{}
	for _, m := range ssmGlobalKey.FindAllStringSubmatch(string(raw), -1) {
		globalKeys[m[1]] = true
	}

	r := &ssmRun{
		vocab: vocab, skillTypes: skillTypes,
		omit: newPyCounter(), omitDetail: newPyListMap(), boolLiterals: map[string]int{},
	}

	body := []string{}
	keptMods, keptEntries, keptSkills := 0, 0, 0
	emptyEntries, emptyShadowing := 0, 0
	emptyLost, emptyLostShadowing := 0, 0
	for _, sid := range oeSortedKeys(skills) {
		s, ok := skills[sid].(map[string]any)
		if !ok {
			return mcRefuse(fmt.Sprintf("skill %s has no 'entries' object", pyRepr(sid)))
		}
		entries, ok := s["entries"].(map[string]any)
		if !ok {
			return mcRefuse(fmt.Sprintf("skill %s has no 'entries' object", pyRepr(sid)))
		}
		lines := []string{}
		for _, stat := range oeSortedKeys(entries) {
			e, ok := entries[stat].(map[string]any)
			if !ok {
				return mcRefuse(fmt.Sprintf("skill %s stat %s has no 'mods' list",
					pyRepr(sid), pyRepr(stat)))
			}
			rawMods, ok := e["mods"].([]any)
			if !ok {
				return mcRefuse(fmt.Sprintf("skill %s stat %s has no 'mods' list",
					pyRepr(sid), pyRepr(stat)))
			}
			mods, err := r.emitMods(sid, stat, rawMods, &keptMods)
			if err != nil {
				return mcRefuse(err.Error())
			}
			scalars, _ := e["scalars"].(map[string]any)

			rawGroups, _ := e["groups"].([]any)
			groupLines := []string{}
			groupMods := 0
			for _, gAny := range rawGroups {
				g, ok := gAny.(map[string]any)
				if !ok {
					//go-only: the GROUP form reached neither original. oracle/skill_stat_map.lua only began
					return mcRefuse(fmt.Sprintf("skill %s stat %s has a group that is not a table",
						pyRepr(sid), pyRepr(stat)))
				}
				gRaw, _ := g["mods"].([]any)
				gMods, err := r.emitMods(sid, stat, gRaw, &keptMods)
				if err != nil {
					return mcRefuse(err.Error())
				}
				gScalars, _ := g["scalars"].(map[string]any)
				if len(gMods) == 0 {
					continue
				}
				gScalarLines, err := r.emitScalars(sid, stat, "\t\t\t\t\t", gScalars, false)
				if err != nil {
					return mcRefuse(err.Error())
				}
				groupMods += len(gMods)
				groupLines = append(groupLines, "\t\t\t\t{")
				groupLines = append(groupLines, ssmModLines(gMods, "\t\t\t\t\t")...)
				groupLines = append(groupLines, gScalarLines...)
				groupLines = append(groupLines, "\t\t\t\t},")
			}

			if len(mods) == 0 && groupMods == 0 {
				switch {
				case len(rawMods) == 0 && len(rawGroups) == 0:
					emptyEntries++
					if globalKeys[stat] {
						emptyShadowing++
					}
				default:
					emptyLost++
					if globalKeys[stat] {
						emptyLostShadowing++
					}
				}
			}
			bodyLines := []string{"\t\t" + mcGoStr(stat) + ": {"}
			if len(mods) > 0 || len(groupLines) == 0 {
				bodyLines = append(bodyLines, ssmModLines(mods, "\t\t\t")...)
			}
			if len(groupLines) > 0 {
				bodyLines = append(bodyLines, "\t\t\tGroups: []*StatMap{")
				bodyLines = append(bodyLines, groupLines...)
				bodyLines = append(bodyLines, "\t\t\t},")
			}
			scalarLines, err := r.emitScalars(sid, stat, "\t\t\t", scalars, true)
			if err != nil {
				return mcRefuse(err.Error())
			}
			bodyLines = append(bodyLines, scalarLines...)
			bodyLines = append(bodyLines, "\t\t},")
			lines = append(lines, bodyLines...)
			keptEntries++
		}
		keptSkills++
		loc := "src/Data/Skills"
		if oeTruthy(s["loc"]) {
			loc = oeStr(s["loc"])
		}
		body = append(body, "\t// "+loc, "\t"+mcGoStr(sid)+": {")
		body = append(body, lines...)
		body = append(body, "\t},")
	}
	_ = keptSkills

	omittedTotal := r.omit.total()
	headerOmissions := []string{}
	for _, reason := range r.omit.mostCommon() {
		headerOmissions = append(headerOmissions,
			fmt.Sprintf("//   - %d %s", r.omit.counts[reason], reason))
	}

	out := fmt.Sprintf(ssmHeader, emptyEntries, emptyShadowing, emptyLost, emptyLostShadowing,
		oeStr(d["skillCount"]), oeStr(d["entryCount"]), oeStr(d["modCount"]),
		keptMods, omittedTotal,
		strings.Join(headerOmissions, "\n")) + strings.Join(body, "\n") + "\n}\n"
	if err := os.WriteFile(dst, []byte(out), 0o644); err != nil { //nolint:gosec // the original's mode
		return mcRefuse(fmt.Sprintf("%s is not writable: %s", dst, errnoText(err)))
	}

	fmt.Printf("skills %s  entries %s  mods %s -> kept %d, omitted %d\n",
		oeStr(d["skillCount"]), oeStr(d["entryCount"]), oeStr(d["modCount"]), keptMods, omittedTotal)
	fmt.Printf("entries PoB declares with no mods: %d (of which %d shadow a global-map stat)\n",
		emptyEntries, emptyShadowing)
	fmt.Printf("entries emptied by omission: %d (of which %d suppress a real global entry)\n",
		emptyLost, emptyLostShadowing)
	r.reportBoolLiterals()
	for _, reason := range r.omit.mostCommon() {
		fmt.Printf("  %4d  %s\n", r.omit.counts[reason], reason)
		for _, x := range r.omitDetail.head(reason, 3) {
			fmt.Printf("          e.g. %s\n", x)
		}
	}
	return 0
}
