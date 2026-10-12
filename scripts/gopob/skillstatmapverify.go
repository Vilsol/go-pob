package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

const ssvUsage = "usage: skill_stat_map_verify.py <lua.json> <go.json>"

var (
	ssvLuaModFields = []string{"name", "type", "value", "flags", "keywordFlags", "tags"}
	ssvGoModFields  = []string{"name", "type", "value", "flags", "kw", "tags"}
)

var ssvListValueIsATable = map[string]bool{
	"MinionModifier": true, "SkillData": true,
	"ExtraMinionSkill": true, "SupportedGemProperty": true,
	"BeidatPactDamage": true, "GhorrPactDamage": true,
	"KtashPactDamage": true, "LyciaPactDamage": true,
}

func ssvNeedIn(d any, key, what string) (any, error) {
	m, _ := d.(map[string]any)
	if m == nil {
		return nil, &sbvError{fmt.Sprintf("%s tag has no %s", what, pyRepr(key))}
	}
	return sbvNeed(m, key, what)
}

func ssvRampPairs(ramp any, what string) (any, error) {
	a, ok := ramp.([]any)
	if !ok {
		return nil, &sbvError{fmt.Sprintf("%s ramp is not a list of pairs", what)}
	}
	out := make([]any, 0, len(a))
	for _, p := range a {
		pair, ok := p.([]any)
		if !ok || len(pair) != 2 {
			return nil, &sbvError{fmt.Sprintf("%s ramp is not a list of pairs", what)}
		}
		out = append(out, []any{sbvNum(pair[0]), sbvNum(pair[1])})
	}
	return out, nil
}

func ssvListOr(t map[string]any, listKey, oneKey, what string) (any, error) {
	if raw, ok := t[listKey]; ok && oeTruthy(raw) {
		return sbvStrList(raw), nil
	}
	one, err := sbvNeed(t, oneKey, what)
	if err != nil {
		return nil, err
	}
	return []any{one}, nil
}

func ssvLuaTag(t map[string]any, skillTypes map[string]any) (any, error) {
	var err error
	need := func(key, what string) any {
		if err != nil {
			return nil
		}
		var v any
		v, err = sbvNeed(t, key, what)
		return v
	}
	out := func(v any) (any, error) {
		if err != nil {
			return nil, err
		}
		return v, nil
	}
	switch oeStr(sbvGet(t, "type", nil)) {
	case "GlobalEffect":
		return out(pyTuple{"GlobalEffect", []any{need("effectType", "GlobalEffect")},
			sbvGet(t, "effectName", ""), oeTruthy(t["unscalable"])})
	case "Multiplier":
		return out(pyTuple{"Multiplier", []any{need("var", "Multiplier")},
			sbvNum(sbvGet(t, "div", 1.0)), sbvNum(sbvGet(t, "base", 0.0)), sbvNum(t["limit"]),
			t["limitVar"], oeTruthy(t["limitTotal"]), sbvGet(t, "actor", "")})
	case "MultiplierThreshold":
		return out(pyTuple{"MultiplierThreshold", need("var", "MultiplierThreshold"),
			sbvNum(t["threshold"]), oeTruthy(t["upper"]), sbvGet(t, "actor", "")})
	case "PerStat":
		stats, serr := ssvListOr(t, "statList", "stat", "PerStat")
		if serr != nil {
			return nil, serr
		}
		return out(pyTuple{"PerStat", stats, sbvNum(sbvGet(t, "div", 1.0)),
			sbvNum(sbvGet(t, "base", 0.0)), sbvNum(t["limit"]), t["limitVar"],
			oeTruthy(t["limitTotal"]), sbvGet(t, "actor", "")})
	case "PercentStat":
		return out(pyTuple{"PercentStat", need("stat", "PercentStat"),
			sbvNum(sbvGet(t, "percent", 0.0)), sbvGet(t, "percentVar", "")})
	case "Condition":
		vars, verr := ssvListOr(t, "varList", "var", "Condition")
		if verr != nil {
			return nil, verr
		}
		return out(pyTuple{"Condition", vars, oeTruthy(t["neg"])})
	case "ActorCondition":
		actor := need("actor", "ActorCondition")
		vars, verr := ssvListOr(t, "varList", "var", "ActorCondition")
		if verr != nil {
			return nil, verr
		}
		return out(pyTuple{"ActorCondition", actor, vars, oeTruthy(t["neg"])})
	case "SkillType":
		raw, ok := t["skillType"]
		if !ok {
			return pyTuple{"SkillType", nil, oeTruthy(t["neg"])}, nil
		}
		n, ok := gemsInt(raw)
		if !ok || pyTypeName(raw) == "bool" {
			return nil, &sbvError{fmt.Sprintf("SkillType %s is not a number", pyRepr(oeStr(raw)))}
		}
		return pyTuple{"SkillType", skillTypes[strconv.Itoa(n)], oeTruthy(t["neg"])}, nil
	case "SkillName":
		return out(pyTuple{"SkillName", []any{need("skillName", "SkillName")}, oeTruthy(t["neg"])})
	case "SkillPart":
		parts, perr := sbvParts(t, "skillPartList", "skillPart", "SkillPart", true)
		if perr != nil {
			return nil, perr
		}
		return pyTuple{"SkillPart", parts, oeTruthy(t["neg"])}, nil
	case "StatThreshold":
		return out(pyTuple{"StatThreshold", need("stat", "StatThreshold"),
			sbvNum(need("threshold", "StatThreshold")), oeTruthy(t["upper"])})
	case "DistanceRamp":
		raw := need("ramp", "DistanceRamp")
		if err != nil {
			return nil, err
		}
		pairs, rerr := ssvRampPairs(raw, "DistanceRamp")
		if rerr != nil {
			return nil, rerr
		}
		return pyTuple{"DistanceRamp", pairs}, nil
	case "MeleeProximity":
		raw := need("ramp", "MeleeProximity")
		if err != nil {
			return nil, err
		}
		ints, ierr := ssvIntList(raw, "MeleeProximity tag has a non-numeric ramp entry")
		if ierr != nil {
			return nil, ierr
		}
		return pyTuple{"MeleeProximity", ints}, nil
	case "ModFlagOr":
		raw := need("modFlags", "ModFlagOr")
		if err != nil {
			return nil, err
		}
		n, ok := gemsInt(raw)
		if !ok || pyTypeName(raw) == "bool" {
			return nil, &sbvError{"ModFlagOr tag has non-numeric modFlags"}
		}
		return pyTuple{"ModFlagOr", n}, nil
	case "SkillId":
		return out(pyTuple{"SkillId", need("skillId", "SkillId")})
	}
	return pyTuple{"UNKNOWN", tagTypeOrNil(t)}, nil
}

func ssvIntList(raw any, msg string) (any, error) {
	a, ok := raw.([]any)
	if !ok {
		return nil, &sbvError{msg}
	}
	out := make([]any, 0, len(a))
	for _, x := range a {
		n, ok := gemsInt(x)
		if !ok || pyTypeName(x) == "bool" {
			return nil, &sbvError{msg}
		}
		out = append(out, n)
	}
	return out, nil
}

func ssvGoTag(t map[string]any) (any, error) {
	var err error
	need := func(key, what string) any {
		if err != nil {
			return nil
		}
		var v any
		v, err = sbvNeed(t, key, what)
		return v
	}
	out := func(v any) (any, error) {
		if err != nil {
			return nil, err
		}
		return v, nil
	}
	switch oeStr(sbvGet(t, "type", nil)) {
	case "GlobalEffect":
		return out(pyTuple{"GlobalEffect", sbvStrList(need("effects", "GlobalEffect")),
			need("name", "GlobalEffect"), need("unscalable", "GlobalEffect")})
	case "Multiplier":
		return out(pyTuple{"Multiplier", sbvStrList(need("vars", "Multiplier")),
			sbvNum(need("div", "Multiplier")), sbvNum(need("base", "Multiplier")),
			sbvNum(need("limit", "Multiplier")), need("limitVar", "Multiplier"),
			need("limitTotal", "Multiplier"), need("actor", "Multiplier")})
	case "MultiplierThreshold":
		return out(pyTuple{"MultiplierThreshold", need("var", "MultiplierThreshold"),
			sbvNum(need("threshold", "MultiplierThreshold")), need("upper", "MultiplierThreshold"),
			need("actor", "MultiplierThreshold")})
	case "PerStat":
		return out(pyTuple{"PerStat", sbvStrList(need("stats", "PerStat")),
			sbvNum(need("div", "PerStat")), sbvNum(need("base", "PerStat")),
			sbvNum(need("limit", "PerStat")), need("limitVar", "PerStat"),
			need("limitTotal", "PerStat"), need("actor", "PerStat")})
	case "PercentStat":
		return out(pyTuple{"PercentStat", need("stat", "PercentStat"),
			sbvNum(need("percent", "PercentStat")), sbvGet(t, "percentVar", "")})
	case "Condition":
		return out(pyTuple{"Condition", sbvStrList(need("vars", "Condition")),
			need("neg", "Condition")})
	case "ActorCondition":
		return out(pyTuple{"ActorCondition", need("actor", "ActorCondition"),
			sbvStrList(need("vars", "ActorCondition")), need("neg", "ActorCondition")})
	case "SkillType":
		st := sbvStrList(t["skillTypes"])
		var v any
		switch {
		case len(st) == 1:
			v = st[0]
		case len(st) == 0:
			v = nil
		default:
			v = pyTuple(st)
		}
		return out(pyTuple{"SkillType", v, oeTruthy(need("neg", "SkillType"))})
	case "SkillName":
		return out(pyTuple{"SkillName", sbvStrList(need("names", "SkillName")),
			need("neg", "SkillName")})
	case "SkillPart":
		parts, perr := sbvParts(t, "partList", "part", "SkillPart", false)
		if perr != nil {
			return nil, perr
		}
		return out(pyTuple{"SkillPart", parts, need("neg", "SkillPart")})
	case "StatThreshold":
		return out(pyTuple{"StatThreshold", need("stat", "StatThreshold"),
			sbvNum(need("threshold", "StatThreshold")), need("upper", "StatThreshold")})
	case "DistanceRamp":
		raw := need("ramp", "DistanceRamp")
		if err != nil {
			return nil, err
		}
		pairs, rerr := ssvRampPairs(raw, "DistanceRamp")
		if rerr != nil {
			return nil, rerr
		}
		return pyTuple{"DistanceRamp", pairs}, nil
	case "MeleeProximity":
		raw := need("ramp", "MeleeProximity")
		if err != nil {
			return nil, err
		}
		ints, ierr := ssvIntList(raw, "MeleeProximity tag has a non-numeric ramp entry")
		if ierr != nil {
			return nil, ierr
		}
		return pyTuple{"MeleeProximity", ints}, nil
	case "ModFlagOr":
		raw := need("modFlags", "ModFlagOr")
		if err != nil {
			return nil, err
		}
		n, ok := gemsInt(raw)
		if !ok || pyTypeName(raw) == "bool" {
			return nil, &sbvError{"ModFlagOr tag has non-numeric modFlags"}
		}
		return pyTuple{"ModFlagOr", n}, nil
	case "SkillId":
		return out(pyTuple{"SkillId", need("skillId", "SkillId")})
	}
	return pyTuple{"UNKNOWN", tagTypeOrNil(t)}, nil
}

func ssvIsNested(m map[string]any) bool {
	_, hasName := m["name"]
	_, hasType := m["type"]
	return !hasName || !hasType
}

func ssvLuaNested(v any, skillTypes map[string]any) (string, error) {
	vm, _ := v.(map[string]any)
	if vm == nil {
		return "", &sbvError{"a MinionModifier's nested mod is not a table"}
	}
	tags := []any{}
	for i := 1; ; i++ {
		x, ok := vm[strconv.Itoa(i)]
		if !ok {
			break
		}
		tags = append(tags, x)
	}
	if err := sbvNeedMod(vm, []string{"name", "type"}, "nested"); err != nil {
		return "", err
	}
	return ssvLuaMod(map[string]any{
		"name": vm["name"], "type": vm["type"], "value": vm["value"],
		"flags": sbvGet(vm, "flags", 0.0), "keywordFlags": sbvGet(vm, "keywordFlags", 0.0),
		"tags": tags,
	}, skillTypes)
}

func ssvLuaMod(m map[string]any, skillTypes map[string]any) (string, error) {
	if err := sbvNeedMod(m, ssvLuaModFields, "Lua"); err != nil {
		return "", err
	}
	val, err := mcUnwrap(m["value"])
	if err != nil {
		return "", err
	}
	mtype, name := oeStr(m["type"]), oeStr(m["name"])
	var value any
	switch {
	case mtype == "LIST":
		if _, isTable := val.(map[string]any); !isTable && ssvListValueIsATable[name] {
			return "", &sbvError{fmt.Sprintf("%s mod value is not a table", name)}
		}
		switch {
		case name == "MinionModifier", strings.HasSuffix(name, "PactDamage"):
			inner, ierr := ssvNeedIn(val, "mod", name)
			if ierr != nil {
				return "", ierr
			}
			nested, nerr := ssvLuaNested(inner, skillTypes)
			if nerr != nil {
				return "", nerr
			}
			value = pyTuple{"minionmodifier", ssvRendered(nested)}
		case name == "SkillData":
			key, kerr := ssvNeedIn(val, "key", "SkillData")
			if kerr != nil {
				return "", kerr
			}
			vm, _ := val.(map[string]any)
			value = pyTuple{"skilldata", key, sbvFloatOrZero(vm["value"], true)}
		case name == "ExtraMinionSkill":
			id, ierr := ssvNeedIn(val, "skillId", "ExtraMinionSkill")
			if ierr != nil {
				return "", ierr
			}
			value = pyTuple{"extraminionskill", id}
		case name == "SupportedGemProperty":
			kw, kerr := ssvNeedIn(val, "keyword", "SupportedGemProperty")
			if kerr != nil {
				return "", kerr
			}
			key, kerr2 := ssvNeedIn(val, "key", "SupportedGemProperty")
			if kerr2 != nil {
				return "", kerr2
			}
			vm, _ := val.(map[string]any)
			value = pyTuple{"supportedgemproperty", kw, key, sbvFloatOrZero(vm["value"], true)}
		default:
			value = pyTuple{"list", name}
		}
	case mtype == "FLAG":
		value = pyTuple{"flag", true}
	default:
		value = pyTuple{"float", sbvFloatOrZero(val, false)}
	}

	flags := -1
	if n, ok := gemsInt(m["flags"]); ok && pyTypeName(m["flags"]) != "bool" {
		flags = n
	}
	kw, _ := gemsInt(m["keywordFlags"])

	rawTags, _ := m["tags"].([]any)
	tags := pyTuple{}
	for _, tAny := range rawTags {
		u, uerr := mcUnwrap(tAny)
		if uerr != nil {
			return "", uerr
		}
		tm, _ := u.(map[string]any)
		if tm == nil {
			return "", &sbvError{"a tag is not a table"}
		}
		got, terr := ssvLuaTag(tm, skillTypes)
		if terr != nil {
			return "", terr
		}
		tags = append(tags, got)
	}
	canonical := pyTuple{name, mtype, value, flags, kw, tags}
	if err := sbvNoTables(canonical); err != nil {
		return "", err
	}
	return pyReprValue(canonical), nil
}

func ssvGoMod(m map[string]any) (string, error) {
	if err := sbvNeedMod(m, ssvGoModFields, "Go"); err != nil {
		return "", err
	}
	v := m["value"]
	var value any
	if vm, ok := v.(map[string]any); ok {
		switch {
		case vm["mod"] != nil || hasKey(vm, "mod"):
			inner, _ := vm["mod"].(map[string]any)
			if inner == nil {
				return "", &sbvError{"a Go nested mod is not a table"}
			}
			nested, nerr := ssvGoMod(inner)
			if nerr != nil {
				return "", nerr
			}
			value = pyTuple{"minionmodifier", ssvRendered(nested)}
		case hasKey(vm, "keyword"):
			pk, perr := sbvNeed(vm, "propKey", "SupportedGemProperty")
			if perr != nil {
				return "", perr
			}
			pv, verr := sbvNeed(vm, "value", "SupportedGemProperty")
			if verr != nil {
				return "", verr
			}
			value = pyTuple{"supportedgemproperty", vm["keyword"], pk, ssvFloat(pv)}
		case hasKey(vm, "key"):
			pv, verr := sbvNeed(vm, "value", "SkillData")
			if verr != nil {
				return "", verr
			}
			value = pyTuple{"skilldata", vm["key"], ssvFloat(pv)}
		default:
			id, ierr := sbvNeed(vm, "skillId", "ExtraMinionSkill")
			if ierr != nil {
				return "", ierr
			}
			value = pyTuple{"extraminionskill", id}
		}
	} else if b, ok := v.(bool); ok {
		value = pyTuple{"flag", b}
	} else {
		value = pyTuple{"float", ssvFloat(v)}
	}

	flags, _ := gemsInt(m["flags"])
	kw, _ := gemsInt(m["kw"])
	rawTags, _ := m["tags"].([]any)
	tags := pyTuple{}
	for _, tAny := range rawTags {
		tm, _ := tAny.(map[string]any)
		if tm == nil {
			return "", &sbvError{"a tag is not a table"}
		}
		got, terr := ssvGoTag(tm)
		if terr != nil {
			return "", terr
		}
		tags = append(tags, got)
	}
	canonical := pyTuple{m["name"], m["type"], value, flags, kw, tags}
	if err := sbvNoTables(canonical); err != nil {
		return "", err
	}
	return pyReprValue(canonical), nil
}

func hasKey(m map[string]any, k string) bool {
	_, ok := m[k]
	return ok
}

func ssvFloat(v any) float64 {
	f, _ := oeFloat(v)
	return f
}

type ssvRendered string

func (r ssvRendered) pyReprSelf() string { return string(r) }

func skillStatMapVerify(args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, ssvUsage)
		return 1
	}
	lua, reason := sbvLoad(args[0], "the Lua dump")
	if reason != "" {
		return sbvRefuse(reason)
	}
	goT, reason := sbvLoad(args[1], "the Go dump")
	if reason != "" {
		return sbvRefuse(reason)
	}
	for _, field := range []string{"skillTypeNames", "skills"} {
		if _, ok := lua[field]; !ok {
			return sbvRefuse(fmt.Sprintf("the Lua dump carries no %s", pyRepr(field)))
		}
	}
	luaSkills, ok := lua["skills"].(map[string]any)
	if !ok {
		return sbvRefuse("the Lua dump's 'skills' is not a JSON object")
	}
	skillTypes, _ := lua["skillTypeNames"].(map[string]any)

	errors := []string{}
	matched, nestedTotal, groupTotal := 0, 0, 0

	onlyLua, onlyGo, both := []string{}, []string{}, []string{}
	for k := range luaSkills {
		if _, ok := goT[k]; ok {
			both = append(both, k)
		} else {
			onlyLua = append(onlyLua, k)
		}
	}
	for k := range goT {
		if _, ok := luaSkills[k]; !ok {
			onlyGo = append(onlyGo, k)
		}
	}
	sort.Strings(onlyLua)
	sort.Strings(onlyGo)
	sort.Strings(both)
	for _, sid := range onlyLua {
		errors = append(errors, fmt.Sprintf("skill %s missing from Go table", sid))
	}
	for _, sid := range onlyGo {
		errors = append(errors, fmt.Sprintf("skill %s is in the Go table but not in the Lua", sid))
	}

	for _, sid := range both {
		ls, _ := luaSkills[sid].(map[string]any)
		le, entriesOK := ls["entries"].(map[string]any)
		if ls == nil || !entriesOK {
			return sbvRefuse(fmt.Sprintf("skill %s has no 'entries' table in the Lua dump", pyRepr(sid)))
		}
		ge, ok := goT[sid].(map[string]any)
		if !ok {
			return sbvRefuse(fmt.Sprintf("skill %s is not a table in the Go dump", pyRepr(sid)))
		}
		sym, inter := []string{}, []string{}
		for k := range le {
			if _, ok := ge[k]; ok {
				inter = append(inter, k)
			} else {
				sym = append(sym, k)
			}
		}
		for k := range ge {
			if _, ok := le[k]; !ok {
				sym = append(sym, k)
			}
		}
		sort.Strings(sym)
		sort.Strings(inter)
		for _, stat := range sym {
			errors = append(errors, fmt.Sprintf("%s/%s: entry key present on only one side", sid, stat))
		}
		for _, stat := range inter {
			lent, lok := le[stat].(map[string]any)
			gent, gok := ge[stat].(map[string]any)
			for _, side := range []struct {
				what string
				e    map[string]any
				ok   bool
			}{{"Lua", lent, lok}, {"Go", gent, gok}} {
				if !side.ok {
					return sbvRefuse(fmt.Sprintf("%s/%s: the %s entry is not a table", sid, stat, side.what))
				}
				if _, ok := side.e["mods"].([]any); !ok {
					return sbvRefuse(fmt.Sprintf("%s/%s: the %s entry has no 'mods' list", sid, stat, side.what))
				}
			}
			if _, ok := lent["scalars"]; !ok {
				return sbvRefuse(fmt.Sprintf("%s/%s: the Lua entry has no 'scalars'", sid, stat))
			}
			lsc, _ := lent["scalars"].(map[string]any)
			for _, key := range []string{"value", "mult", "div", "base"} {
				var want any
				if raw, ok := lsc[key]; ok {
					u, uerr := mcUnwrap(raw)
					if uerr != nil {
						return sbvRefuse(uerr.Error())
					}
					want = sbvNum(u)
					if err := sbvNoTables(want); err != nil {
						return sbvRefuse(err.Error())
					}
				}
				gotRaw, ok := gent[key]
				if !ok {
					return sbvRefuse(fmt.Sprintf("%s/%s: the Go entry has no %s", sid, stat, pyRepr(key)))
				}
				got := sbvNum(gotRaw)
				if pyReprValue(want) != pyReprValue(got) {
					errors = append(errors, fmt.Sprintf("%s/%s: scalar %s lua=%s go=%s",
						sid, stat, key, ssvStr(want), ssvStr(got)))
				}
			}

			luaMods, _ := lent["mods"].([]any)
			pool := []string{}
			for _, mAny := range luaMods {
				m, ok := mAny.(map[string]any)
				if !ok {
					return sbvRefuse(fmt.Sprintf("%s/%s: a Lua mod is not a table", sid, stat))
				}
				if ssvIsNested(m) {
					nestedTotal++
					continue
				}
				key, err := ssvLuaMod(m, skillTypes)
				if err != nil {
					return sbvRefuse(err.Error())
				}
				pool = append(pool, key)
			}
			goMods, _ := gent["mods"].([]any)
			for _, gmAny := range goMods {
				gm, ok := gmAny.(map[string]any)
				if !ok {
					return sbvRefuse(fmt.Sprintf("%s/%s: a Go mod is not a table", sid, stat))
				}
				key, err := ssvGoMod(gm)
				if err != nil {
					return sbvRefuse(err.Error())
				}
				found := -1
				for i, p := range pool {
					if p == key {
						found = i
						break
					}
				}
				if found >= 0 {
					pool = append(pool[:found], pool[found+1:]...)
					matched++
				} else {
					left := "<none left>"
					if len(pool) > 0 {
						left = strings.Join(pool, "\n            ")
					}
					errors = append(errors, fmt.Sprintf(
						"%s/%s: emitted mod has no Lua counterpart:\n      go  = %s\n      lua = %s",
						sid, stat, key, left))
				}
			}

			luaGroups, _ := lent["groups"].([]any)
			goGroups, _ := gent["groups"].([]any)
			groupTotal += len(luaGroups)
			if len(luaGroups) != len(goGroups) {
				errors = append(errors, fmt.Sprintf("%s/%s: %d group(s) in the Lua, %d in the Go",
					sid, stat, len(luaGroups), len(goGroups)))
				continue
			}
			for gi := range luaGroups {
				lg, lgok := luaGroups[gi].(map[string]any)
				gg, ggok := goGroups[gi].(map[string]any)
				if !lgok || !ggok {
					//go-only: the GROUP form reached neither original. oracle/skill_stat_map.lua only began
					return sbvRefuse(fmt.Sprintf("%s/%s: group %d is not a table on both sides",
						sid, stat, gi))
				}
				lgsc, _ := lg["scalars"].(map[string]any)
				for _, key := range []string{"value", "mult", "div", "base"} {
					var want any
					if raw, ok := lgsc[key]; ok {
						u, uerr := mcUnwrap(raw)
						if uerr != nil {
							return sbvRefuse(uerr.Error())
						}
						want = sbvNum(u)
						if err := sbvNoTables(want); err != nil {
							return sbvRefuse(err.Error())
						}
					}
					gotRaw, ok := gg[key]
					if !ok {
						//go-only: the GROUP form reached neither original. oracle/skill_stat_map.lua only began
						return sbvRefuse(fmt.Sprintf("%s/%s: the Go group %d has no %s",
							sid, stat, gi, pyRepr(key)))
					}
					got := sbvNum(gotRaw)
					if pyReprValue(want) != pyReprValue(got) {
						errors = append(errors, fmt.Sprintf("%s/%s: group %d scalar %s lua=%s go=%s",
							sid, stat, gi, key, ssvStr(want), ssvStr(got)))
					}
				}
				gpool := []string{}
				lgMods, _ := lg["mods"].([]any)
				for _, mAny := range lgMods {
					m, ok := mAny.(map[string]any)
					if !ok {
						//go-only: the GROUP form reached neither original. oracle/skill_stat_map.lua only began
						return sbvRefuse(fmt.Sprintf("%s/%s: a Lua group mod is not a table", sid, stat))
					}
					if ssvIsNested(m) {
						nestedTotal++
						continue
					}
					key, err := ssvLuaMod(m, skillTypes)
					if err != nil {
						return sbvRefuse(err.Error())
					}
					gpool = append(gpool, key)
				}
				ggMods, _ := gg["mods"].([]any)
				for _, gmAny := range ggMods {
					gm, ok := gmAny.(map[string]any)
					if !ok {
						//go-only: the GROUP form reached neither original. oracle/skill_stat_map.lua only began
						return sbvRefuse(fmt.Sprintf("%s/%s: a Go group mod is not a table", sid, stat))
					}
					key, err := ssvGoMod(gm)
					if err != nil {
						return sbvRefuse(err.Error())
					}
					found := -1
					for i, p := range gpool {
						if p == key {
							found = i
							break
						}
					}
					if found >= 0 {
						gpool = append(gpool[:found], gpool[found+1:]...)
						matched++
					} else {
						left := "<none left>"
						if len(gpool) > 0 {
							left = strings.Join(gpool, "\n            ")
						}
						errors = append(errors, fmt.Sprintf(
							"%s/%s: emitted group mod has no Lua counterpart:\n      go  = %s\n      lua = %s",
							sid, stat, key, left))
					}
				}
			}
		}
	}

	for _, field := range []string{"skillCount", "entryCount", "modCount"} {
		if _, ok := lua[field]; !ok {
			return sbvRefuse(fmt.Sprintf("the Lua dump carries no %s", pyRepr(field)))
		}
	}
	fmt.Printf("lua: %d skills, %d entries, %d mods\n",
		ssvInt(lua["skillCount"]), ssvInt(lua["entryCount"]), ssvInt(lua["modCount"]))
	goSids := make([]string, 0, len(goT))
	for sid := range goT {
		goSids = append(goSids, sid)
	}
	sort.Strings(goSids)
	goEntries, goMods := 0, 0
	for _, sid := range goSids {
		v, ok := goT[sid].(map[string]any)
		if !ok {
			return sbvRefuse(fmt.Sprintf("skill %s is not a table in the Go dump", pyRepr(sid)))
		}
		stats := make([]string, 0, len(v))
		for stat := range v {
			stats = append(stats, stat)
		}
		sort.Strings(stats)
		for _, stat := range stats {
			e, ok := v[stat].(map[string]any)
			if !ok {
				return sbvRefuse(fmt.Sprintf("%s/%s: the Go entry has no 'mods' list", sid, stat))
			}
			mods, ok := e["mods"].([]any)
			if !ok {
				return sbvRefuse(fmt.Sprintf("%s/%s: the Go entry has no 'mods' list", sid, stat))
			}
			goMods += len(mods)
			if groups, ok := e["groups"].([]any); ok {
				for _, gAny := range groups {
					g, ok := gAny.(map[string]any)
					if !ok {
						//go-only: the GROUP form reached neither original. oracle/skill_stat_map.lua only began
						return sbvRefuse(fmt.Sprintf("%s/%s: a Go group is not a table", sid, stat))
					}
					gm, ok := g["mods"].([]any)
					if !ok {
						//go-only: the GROUP form reached neither original. oracle/skill_stat_map.lua only began
						return sbvRefuse(fmt.Sprintf("%s/%s: a Go group has no 'mods' list", sid, stat))
					}
					goMods += len(gm)
				}
			}
		}
		goEntries += len(v)
	}
	fmt.Printf("go : %d skills, %d entries, %d mods\n", len(goT), goEntries, goMods)
	fmt.Printf("matched 1-to-1 against the Lua: %d\n", matched)
	fmt.Printf("lua mods with no name/type (nested; mod/ has no constructor): %d\n", nestedTotal)
	fmt.Printf("lua statMap GROUPS compared (per-group scalars, CalcActiveSkill.lua:65-73): %d\n", groupTotal)
	if len(errors) > 0 {
		fmt.Printf("\nDIFF: %d problem(s)\n", len(errors))
		for i, e := range errors {
			if i >= 40 {
				break
			}
			fmt.Println("  " + e)
		}
		return 1
	}
	fmt.Println("DIFF: none - every emitted mod, entry key and scalar matches the Lua")
	return 0
}

func ssvStr(v any) string {
	if v == nil {
		return "None"
	}
	if f, ok := v.(float64); ok {
		return pyFloat(f)
	}
	return oeStr(v)
}

func ssvInt(v any) int {
	n, _ := gemsInt(v)
	return n
}
