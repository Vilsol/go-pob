package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

var (
	sbvLuaModFields = []string{"name", "type", "value", "flags", "keywordFlags", "tags"}
	sbvGoModFields  = []string{"name", "type", "value", "flags", "kw", "tags", "source"}
)

const sbvUsage = "usage: skill_base_mods_verify.py <lua.json> <go.json>"

func sbvLuaTag(t map[string]any, skillTypes map[string]any) (any, error) {
	tt := oeStr(t["type"])
	switch tt {
	case "GlobalEffect":
		et, err := sbvNeed(t, "effectType", "GlobalEffect")
		if err != nil {
			return nil, err
		}
		return pyTuple{"GlobalEffect", []any{et}, sbvGet(t, "effectName", ""),
			oeTruthy(t["unscalable"])}, nil
	case "Multiplier":
		v, err := sbvNeed(t, "var", "Multiplier")
		if err != nil {
			return nil, err
		}
		actor := ""
		if oeTruthy(t["actor"]) {
			actor = oeStr(t["actor"])
		}
		return pyTuple{"Multiplier", []any{v}, sbvNum(sbvGet(t, "base", json0)),
			sbvNum(sbvGet(t, "div", json1)), sbvNum(t["limit"]), t["limitVar"], actor}, nil
	case "MultiplierThreshold":
		v, err := sbvNeed(t, "var", "MultiplierThreshold")
		if err != nil {
			return nil, err
		}
		return pyTuple{"MultiplierThreshold", v, sbvNum(t["threshold"])}, nil
	case "PerStat":
		stats := sbvStrList(t["statList"])
		if len(stats) == 0 {
			one, err := sbvNeed(t, "stat", "PerStat")
			if err != nil {
				return nil, err
			}
			stats = []any{one}
		}
		return pyTuple{"PerStat", stats, sbvNum(sbvGet(t, "div", json1)),
			sbvNum(sbvGet(t, "base", json0))}, nil
	case "PercentStat":
		st, serr := sbvNeed(t, "stat", "PercentStat")
		if serr != nil {
			return nil, serr
		}
		return pyTuple{"PercentStat", st, sbvNum(sbvGet(t, "percent", json0)),
			sbvGet(t, "percentVar", "")}, nil
	case "Condition":
		vars := sbvStrList(t["varList"])
		if len(vars) == 0 {
			one, err := sbvNeed(t, "var", "Condition")
			if err != nil {
				return nil, err
			}
			vars = []any{one}
		}
		return pyTuple{"Condition", vars, oeTruthy(t["neg"])}, nil
	case "ActorCondition":
		actor, err := sbvNeed(t, "actor", "ActorCondition")
		if err != nil {
			return nil, err
		}
		vars := sbvStrList(t["varList"])
		if len(vars) == 0 {
			one, err := sbvNeed(t, "var", "ActorCondition")
			if err != nil {
				return nil, err
			}
			vars = []any{one}
		}
		return pyTuple{"ActorCondition", actor, vars, oeTruthy(t["neg"])}, nil
	case "SkillType":
		var name any
		if _, ok := t["skillType"]; ok {
			n, ok := gemsInt(t["skillType"])
			if !ok {
				return nil, &sbvError{fmt.Sprintf("SkillType %s is not a number",
					pyRepr(oeStr(t["skillType"])))}
			}
			name = skillTypes[fmt.Sprintf("%d", n)]
		}
		return pyTuple{"SkillType", name}, nil
	case "SkillPart":
		parts, err := sbvParts(t, "skillPartList", "skillPart", "SkillPart", true)
		if err != nil {
			return nil, err
		}
		return pyTuple{"SkillPart", parts, oeTruthy(t["neg"])}, nil
	}
	return pyTuple{"UNKNOWN", tagTypeOrNil(t)}, nil
}

func sbvGoTag(t map[string]any) (any, error) {
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
	switch oeStr(t["type"]) {
	case "GlobalEffect":
		return out(pyTuple{"GlobalEffect", sbvStrList(need("effects", "GlobalEffect")),
			need("name", "GlobalEffect"), need("unscalable", "GlobalEffect")})
	case "Multiplier":
		vars := sbvStrList(need("vars", "Multiplier"))
		base, div := sbvNum(need("base", "Multiplier")), sbvNum(need("div", "Multiplier"))
		limit, limitVar := sbvNum(need("limit", "Multiplier")), need("limitVar", "Multiplier")
		actorRaw := need("actor", "Multiplier")
		actor := ""
		if oeTruthy(actorRaw) {
			actor = oeStr(actorRaw)
		}
		return out(pyTuple{"Multiplier", vars, base, div, limit, limitVar, actor})
	case "MultiplierThreshold":
		return out(pyTuple{"MultiplierThreshold", need("var", "MultiplierThreshold"),
			sbvNum(need("threshold", "MultiplierThreshold"))})
	case "PerStat":
		return out(pyTuple{"PerStat", sbvStrList(need("stats", "PerStat")),
			sbvNum(need("div", "PerStat")), sbvNum(need("base", "PerStat"))})
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
		switch {
		case len(st) == 1:
			return pyTuple{"SkillType", st[0]}, nil
		case len(st) == 0:
			return pyTuple{"SkillType", nil}, nil
		default:
			return pyTuple{"SkillType", pyTuple(st)}, nil
		}
	case "SkillPart":
		parts, perr := sbvParts(t, "partList", "part", "SkillPart", false)
		if perr != nil {
			return nil, perr
		}
		return out(pyTuple{"SkillPart", parts, need("neg", "SkillPart")})
	}
	return pyTuple{"UNKNOWN", tagTypeOrNil(t)}, nil
}

func sbvLuaMod(m map[string]any, skillTypes map[string]any) (string, error) {
	if err := sbvNeedMod(m, sbvLuaModFields, "Lua"); err != nil {
		return "", err
	}
	val, err := mcUnwrap(m["value"])
	if err != nil {
		return "", err
	}
	mtype, name := oeStr(m["type"]), oeStr(m["name"])
	var value any
	switch {
	case mtype == "LIST" && name == "SkillData":
		vm, _ := val.(map[string]any)
		if vm == nil {
			return "", &sbvError{"SkillData mod value is not a table"}
		}
		key, ok := vm["key"].(string)
		if !ok || key == "" {
			return "", &sbvError{"SkillData mod has no 'key' string"}
		}
		value = pyTuple{"skilldata", strings.ToUpper(key[:1]) + key[1:],
			sbvFloatOrZero(vm["value"], true)}
	case mtype == "LIST":
		value = pyTuple{"list", name}
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
		u, err := mcUnwrap(tAny)
		if err != nil {
			return "", err
		}
		tm, _ := u.(map[string]any)
		if tm == nil {
			return "", &sbvError{"a tag is not a table"}
		}
		if oeStr(tm["type"]) == "Skill" {
			continue
		}
		got, err := sbvLuaTag(tm, skillTypes)
		if err != nil {
			return "", err
		}
		tags = append(tags, got)
	}
	canonical := pyTuple{name, mtype, value, flags, kw, tags}
	if err := sbvNoTables(canonical); err != nil {
		return "", err
	}
	return pyReprValue(canonical), nil
}

func sbvGoMod(m map[string]any) (string, error) {
	var value any
	switch v := m["value"].(type) {
	case map[string]any:
		key, kerr := sbvNeed(v, "key", "SkillData")
		if kerr != nil {
			return "", kerr
		}
		val, verr := sbvNeed(v, "value", "SkillData")
		if verr != nil {
			return "", verr
		}
		value = pyTuple{"skilldata", key, sbvFloatOrZero(val, false)}
	case bool:
		value = pyTuple{"flag", v}
	default:
		value = pyTuple{"float", sbvFloatOrZero(m["value"], false)}
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
		got, err := sbvGoTag(tm)
		if err != nil {
			return "", err
		}
		tags = append(tags, got)
	}
	canonical := pyTuple{oeStr(m["name"]), oeStr(m["type"]), value, flags, kw, tags}
	if err := sbvNoTables(canonical); err != nil {
		return "", err
	}
	return pyReprValue(canonical), nil
}

func skillBaseModsVerify(args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, sbvUsage)
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
	matched := 0

	onlyGo := []string{}
	both := []string{}
	for k := range goT {
		if _, ok := luaSkills[k]; ok {
			both = append(both, k)
		} else {
			onlyGo = append(onlyGo, k)
		}
	}
	sort.Strings(onlyGo)
	sort.Strings(both)
	for _, sid := range onlyGo {
		errors = append(errors, fmt.Sprintf("skill %s is in the Go table but not in the Lua", sid))
	}

	for _, sid := range both {
		luaSkill, _ := luaSkills[sid].(map[string]any)
		rawMods, modsOK := luaSkill["mods"].([]any)
		if luaSkill == nil || !modsOK {
			return sbvRefuse(fmt.Sprintf("skill %s has no 'mods' list in the Lua dump", pyRepr(sid)))
		}
		pool := []string{}
		for _, mAny := range rawMods {
			m, ok := mAny.(map[string]any)
			if !ok {
				return sbvRefuse(fmt.Sprintf("skill %s has a mod that is not a table", pyRepr(sid)))
			}
			key, err := sbvLuaMod(m, skillTypes)
			if err != nil {
				return sbvRefuse(err.Error())
			}
			pool = append(pool, key)
		}
		goMods, _ := goT[sid].([]any)
		for _, gmAny := range goMods {
			gm, ok := gmAny.(map[string]any)
			if !ok {
				return sbvRefuse(fmt.Sprintf("skill %s has a Go mod that is not a table", pyRepr(sid)))
			}
			if err := sbvNeedMod(gm, sbvGoModFields, "Go"); err != nil {
				return sbvRefuse(err.Error())
			}
			if oeStr(gm["source"]) != "Skill:"+sid {
				errors = append(errors, fmt.Sprintf("%s: mod %s carries source %s",
					sid, oeStr(gm["name"]), oeStr(gm["source"])))
			}
			key, err := sbvGoMod(gm)
			if err != nil {
				return sbvRefuse(err.Error())
			}
			idx := -1
			for i, p := range pool {
				if p == key {
					idx = i
					break
				}
			}
			if idx >= 0 {
				pool = append(pool[:idx], pool[idx+1:]...)
				matched++
				continue
			}
			left := strings.Join(pool, "\n            ")
			if left == "" {
				left = "<none left>"
			}
			errors = append(errors, fmt.Sprintf(
				"%s: emitted mod has no Lua counterpart:\n      go  = %s\n      lua = %s",
				sid, key, left))
		}
	}

	goMods := 0
	for _, v := range goT {
		a, _ := v.([]any)
		goMods += len(a)
	}
	for _, field := range []string{"skillCount", "modCount"} {
		if _, ok := lua[field]; !ok {
			return sbvRefuse(fmt.Sprintf("the Lua dump carries no %s", pyRepr(field)))
		}
	}
	fmt.Printf("lua: %s skills, %s mods\n", oeStr(lua["skillCount"]), oeStr(lua["modCount"]))
	fmt.Printf("go : %d skills, %d mods\n", len(goT), goMods)
	fmt.Printf("matched 1-to-1 against the Lua: %d\n", matched)
	if len(errors) > 0 {
		fmt.Printf("\nDIFF: %d problem(s)\n", len(errors))
		head := errors
		if len(head) > 40 {
			head = head[:40]
		}
		for _, e := range head {
			fmt.Println("  " + e)
		}
		return 1
	}
	fmt.Println("DIFF: none - every emitted mod, value, flag, tag and source matches the Lua")
	return 0
}
