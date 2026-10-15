package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

const sbmUsage = "usage: skill_base_mods_emit.py <dump.json> <out.go> [calculator/types.go]"

var sbmGlobalEffectOK = map[string]bool{
	"type": true, "effectType": true, "effectName": true, "unscalable": true,
	"effectCond": true, "effectEnemyCond": true, "modCond": true,
}

var (
	sbmMultiplierOK = map[string]bool{
		"type": true, "var": true, "limitVar": true, "div": true, "limit": true,
		"limitTotal": true, "actor": true, "limitActor": true, "base": true,
		"noFloor": true, "divVar": true, "limitStat": true, "limitNegTotal": true,
	}
	sbmMultiplierThresholdOK = map[string]bool{
		"type": true, "var": true, "threshold": true, "thresholdVar": true,
		"upper": true, "actor": true,
	}
	sbmPerStatOK = map[string]bool{
		"type": true, "stat": true, "statList": true, "div": true, "base": true,
		"limit": true, "limitVar": true, "limitTotal": true, "actor": true,
	}
	sbmPercentStatOK = map[string]bool{
		"type": true, "stat": true, "percent": true, "percentVar": true,
		"actor": true, "floor": true,
	}
)

func sbmExtra(t map[string]any, ok map[string]bool) []string {
	bad := []string{}
	for k := range t {
		if !ok[k] {
			bad = append(bad, k)
		}
	}
	sort.Strings(bad)
	return bad
}

var sbmField = regexp.MustCompile(`(?m)^\t([\p{L}\p{N}_]+)` + pySpaceClass)

var sbmStringField = regexp.MustCompile(`(?m)^\t([\p{L}\p{N}_]+)` + pySpaceClass + `+string$`)

func sbmReq(d map[string]any, key, what string) (any, error) {
	v, ok := d[key]
	if !ok {
		return nil, mcUnrep("%s tag has no %s", what, pyRepr(key))
	}
	return v, nil
}

func sbmReqStr(d map[string]any, key, what string) (string, error) {
	v, err := sbmReq(d, key, what)
	if err != nil {
		return "", err
	}
	return oeStr(v), nil
}

func sbmEmitTag(t map[string]any, skillTypes map[string]any, v *mcVocab) (string, error) {
	for _, fv := range t {
		if mcHasTable(fv) {
			return "", mcUnrep("a tag field is a table")
		}
	}
	tt := oeStr(t["type"])
	switch tt {
	case "GlobalEffect":
		bad := []string{}
		for k := range t {
			if !sbmGlobalEffectOK[k] {
				bad = append(bad, k)
			}
		}
		sort.Strings(bad)
		if len(bad) > 0 {
			return "", mcUnrep("GlobalEffect tag carries %s, which mod.GlobalEffectTag cannot express",
				strings.Join(bad, "/"))
		}
		et, err := sbmReqStr(t, "effectType", "GlobalEffect")
		if err != nil {
			return "", err
		}
		s := "mod.GlobalEffect(" + mcGoStr(et) + ")"
		if oeTruthy(t["effectName"]) {
			s += ".Name(" + mcGoStr(oeStr(t["effectName"])) + ")"
		}
		if oeTruthy(t["unscalable"]) {
			s += ".Unscalable(true)"
		}
		if oeTruthy(t["effectCond"]) {
			s += ".EffectCond(" + mcGoStr(oeStr(t["effectCond"])) + ")"
		}
		if oeTruthy(t["effectEnemyCond"]) {
			s += ".EffectEnemyCond(" + mcGoStr(oeStr(t["effectEnemyCond"])) + ")"
		}
		if oeTruthy(t["modCond"]) {
			s += ".ModCond(" + mcGoStr(oeStr(t["modCond"])) + ")"
		}
		return s, nil

	case "Multiplier":
		if bad := sbmExtra(t, sbmMultiplierOK); len(bad) > 0 {
			return "", mcUnrep("Multiplier tag carries %s, which mod.MultiplierTag cannot express",
				strings.Join(bad, "/"))
		}
		name, err := sbmReqStr(t, "var", "Multiplier")
		if err != nil {
			return "", err
		}
		s := "mod.Multiplier(" + mcGoStr(name) + ")"
		if lv, ok := t["limitVar"]; ok {
			s = fmt.Sprintf("multiplierLimitVar(%s, %s)", mcGoStr(name), mcGoStr(oeStr(lv)))
		}
		for _, kv := range [][2]string{{"div", ".Div(%s)"}, {"base", ".Base(%s)"}, {"limit", ".Limit(%s)"}} {
			if x, ok := t[kv[0]]; ok {
				s += fmt.Sprintf(kv[1], mcGoNum(x))
			}
		}
		if oeTruthy(t["limitTotal"]) {
			s += ".LimitTotal(true)"
		}
		if a, ok := t["actor"]; ok {
			s += ".Actor(" + mcGoStr(oeStr(a)) + ")"
		}
		if a, ok := t["limitActor"]; ok {
			s += ".LimitActor(" + mcGoStr(oeStr(a)) + ")"
		}
		if x, ok := t["divVar"]; ok {
			s += ".DivVar(" + mcGoStr(oeStr(x)) + ")"
		}
		if x, ok := t["limitStat"]; ok {
			s += ".LimitStat(" + mcGoStr(oeStr(x)) + ")"
		}
		if oeTruthy(t["noFloor"]) {
			s += ".NoFloor()"
		}
		if oeTruthy(t["limitNegTotal"]) {
			s += ".LimitNegTotal()"
		}
		return s, nil

	case "MultiplierThreshold":
		if bad := sbmExtra(t, sbmMultiplierThresholdOK); len(bad) > 0 {
			return "", mcUnrep("MultiplierThreshold tag carries %s, which "+
				"mod.MultiplierThresholdTag cannot express", strings.Join(bad, "/"))
		}
		name, err := sbmReqStr(t, "var", "MultiplierThreshold")
		if err != nil {
			return "", err
		}
		s := "mod.MultiplierThreshold(" + mcGoStr(name) + ")"
		switch {
		case ssmHas(t, "threshold"):
			s += ".Threshold(" + mcGoNum(t["threshold"]) + ")"
		case ssmHas(t, "thresholdVar"):
			s += ".ThresholdVar(" + mcGoStr(oeStr(t["thresholdVar"])) + ")"
		default:
			return "", mcUnrep("MultiplierThreshold tag carries neither threshold nor " +
				"thresholdVar [pob:src/Classes/ModStore.lua:434]")
		}
		if oeTruthy(t["upper"]) {
			s += ".Upper(true)"
		}
		if a, ok := t["actor"]; ok {
			s += ".Actor(" + mcGoStr(oeStr(a)) + ")"
		}
		return s, nil

	case "PerStat":
		if bad := sbmExtra(t, sbmPerStatOK); len(bad) > 0 {
			return "", mcUnrep("PerStat tag carries %s, which mod.PerStatTag cannot express",
				strings.Join(bad, "/"))
		}
		var statList []any
		if sl, ok := t["statList"].([]any); ok {
			statList = sl
		} else {
			one, err := sbmReq(t, "stat", "PerStat")
			if err != nil {
				return "", err
			}
			statList = []any{one}
		}
		stats := make([]string, 0, len(statList))
		for _, x := range statList {
			stats = append(stats, mcGoStr(oeStr(x)))
		}
		div := any(json1)
		if d, ok := t["div"]; ok {
			div = d
		}
		var s string
		if b, ok := t["base"]; ok {
			s = fmt.Sprintf("perStatBase(%s, %s, %s)", mcGoNum(div), mcGoNum(b), strings.Join(stats, ", "))
		} else {
			s = fmt.Sprintf("mod.PerStat(%s, %s)", mcGoNum(div), strings.Join(stats, ", "))
		}
		if lv, ok := t["limitVar"]; ok {
			s += ".LimitVar(" + mcGoStr(oeStr(lv)) + ")"
		}
		if l, ok := t["limit"]; ok {
			s += ".Limit(" + mcGoNum(l) + ")"
		}
		if oeTruthy(t["limitTotal"]) {
			s += ".LimitTotal(true)"
		}
		if a, ok := t["actor"]; ok {
			s += ".Actor(" + mcGoStr(oeStr(a)) + ")"
		}
		return s, nil

	case "PercentStat":
		if bad := sbmExtra(t, sbmPercentStatOK); len(bad) > 0 {
			return "", mcUnrep("PercentStat tag carries %s, which mod.PercentStatTag cannot express",
				strings.Join(bad, "/"))
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
				mcGoStr(oeStr(stat)), mcGoStr(oeStr(t["percentVar"])))
		default:
			return "", mcUnrep("PercentStat with neither percent nor percentVar " +
				"[go:mod.PercentStatVar]")
		}
		if _, ok := t["actor"]; ok {
			s += ".Actor(" + mcGoStr(oeStr(t["actor"])) + ")"
		}
		if oeTruthy(t["floor"]) {
			s += ".Floor()"
		}
		return s, nil

	case "Condition", "ActorCondition":
		var varList []any
		if vl, ok := t["varList"].([]any); ok {
			varList = vl
		} else {
			one, err := sbmReq(t, "var", tt)
			if err != nil {
				return "", err
			}
			varList = []any{one}
		}
		names := make([]string, 0, len(varList))
		for _, x := range varList {
			names = append(names, mcGoStr(oeStr(x)))
		}
		var s string
		if tt == "Condition" {
			s = "mod.Condition(" + strings.Join(names, ", ") + ")"
		} else {
			actor, err := sbmReqStr(t, "actor", "ActorCondition")
			if err != nil {
				return "", err
			}
			s = fmt.Sprintf("mod.ActorCondition(%s, %s)", mcGoStr(actor), strings.Join(names, ", "))
		}
		if oeTruthy(t["neg"]) {
			s += ".Neg(true)"
		}
		return s, nil

	case "SkillType":
		st, ok := t["skillType"]
		if !ok {
			return "", mcUnrep("SkillType tag has no skillType (upstream typo resolved to nil)")
		}
		n, ok := gemsInt(st)
		if !ok {
			return "", mcUnrep("SkillType %s has no name in Data/Global.lua", oeStr(st))
		}
		name, ok := skillTypes[fmt.Sprintf("%d", n)]
		if !ok || name == nil {
			return "", mcUnrep("SkillType %s has no name in Data/Global.lua", oeStr(st))
		}
		s := "mod.SkillType(" + mcGoStr(oeStr(name)) + ")"
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

	case "SkillPart":
		if oeTruthy(t["neg"]) {
			return "", mcUnrep("SkillPart tag carries neg, which mod.SkillPartTag exposes as a " +
				"field with no setter")
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

	case "Skill":
		return "", nil
	}
	if tt, ok := t["type"]; ok {
		if _, isStr := tt.(string); !isStr {
			return "", mcUnrep("a tag's 'type' is not a string")
		}
	}
	return "", mcUnrep("unknown tag type %s", oeStr(t["type"]))
}

var json1 = mustNumber("1")

func sbmEmitMod(m map[string]any, sid string, fields, stringFields map[string]bool, v *mcVocab) (string, []any, error) {
	if err := mcReqMod(m); err != nil {
		return "", nil, err
	}
	for _, field := range []string{"flags", "name", "type", "value", "keywordFlags", "tags"} {
		if _, ok := m[field]; !ok {
			return "", nil, mcUnrep("mod has no %s", pyRepr(field))
		}
	}
	if _, ok := oeFloat(m["flags"]); !ok || pyTypeName(m["flags"]) == "bool" {
		return "", nil, mcUnrep("mod.flags holds a tag table (upstream arity bug), so the mod " +
			"has no tags and a nonsense flag set")
	}
	if extra := mcExtraKeys(m); len(extra) > 0 {
		return "", nil, mcUnrep("mod carries %s beyond the known mod fields, which this emitter "+
			"neither models nor declares", pyRepr(strings.Join(extra, "/")))
	}
	name, mtype := oeStr(m["name"]), oeStr(m["type"])
	val, err := mcUnwrap(m["value"])
	if err != nil {
		return "", nil, err
	}
	if vm, isMap := val.(map[string]any); isMap {
		for fk, fv := range vm {
			if fk != "mod" && mcHasTable(fv) {
				return "", nil, mcUnrep("a mod value field is a table")
			}
		}
	}

	var base string
	switch {
	case mtype == "LIST" && name == "SkillData":
		vm, ok := val.(map[string]any)
		if !ok {
			return "", nil, mcUnrep("SkillData mod value is not a table")
		}
		x := vm["value"]
		rawKey, err := sbmReq(vm, "key", "SkillData")
		if err != nil {
			return "", nil, err
		}
		ks, ok := rawKey.(string)
		if !ok || ks == "" {
			return "", nil, mcUnrep("skill() entry whose key is not a non-empty string")
		}
		key := strings.ToUpper(ks[:1]) + ks[1:]
		if str, isStr := x.(string); isStr {
			if !stringFields[key] {
				return "", nil, mcUnrep("string-valued skill() entries (triggerSource, " +
					"corpseMonsterVariety) whose key has\n//     no string field on calculator.SkillData")
			}
			base = fmt.Sprintf("mod.NewList(\"SkillData\", mod.SkillData{Key: %s, String: %s})",
				mcGoStr(key), mcGoStr(str))
			break
		}
		if !fields[key] {
			return "", nil, mcUnrep("skill() entry whose key has no field on "+
				"calculator.SkillData\n//     (%s), which utils.Set drops as an invalid key "+
				"rather than storing", ks)
		}
		lit := "0"
		if _, isNum := oeFloat(x); isNum && pyTypeName(x) != "bool" {
			lit = mcGoNum(x)
		} else if oeTruthy(x) {
			lit = "1"
		}
		base = fmt.Sprintf("mod.NewList(\"SkillData\", mod.SkillData{Key: %s, Value: %s})",
			mcGoStr(key), lit)
	case mtype == "LIST" && name == "ExtraMinionSkill":
		vm, _ := val.(map[string]any)
		if vm == nil {
			return "", nil, mcUnrep("ExtraMinionSkill tag has no %s", pyRepr("skillId"))
		}
		id, err := sbmReqStr(vm, "skillId", "ExtraMinionSkill")
		if err != nil {
			return "", nil, err
		}
		base = "mod.NewList(\"ExtraMinionSkill\", mod.ExtraMinionSkill{SkillID: " + mcGoStr(id) + "})"
	case mtype == "LIST" && name == "MinionModifier":
		vm, _ := val.(map[string]any)
		nested, _ := vm["mod"].(map[string]any)
		if nested == nil {
			return "", nil, mcUnrep("MinionModifier value carries no nested mod table")
		}
		filled := make(map[string]any, len(nested)+3)
		for k, x := range nested {
			filled[k] = x
		}
		for field, def := range map[string]any{"flags": mustNumber("0"), "keywordFlags": mustNumber("0"), "tags": []any{}} {
			if _, ok := filled[field]; !ok {
				filled[field] = def
			}
		}
		nbase, ntags, err := sbmEmitMod(filled, sid, fields, stringFields, v)
		if err != nil {
			return "", nil, err
		}
		if len(ntags) > 0 {
			return "", nil, mcUnrep("MinionModifier's nested mod carries tags, which this arm does not emit")
		}
		base = "mod.NewList(\"MinionModifier\", mod.MinionModifier{Mod: " + nbase + "})"
	case mtype == "LIST" && name == "EnemyModifier":
		vm, _ := val.(map[string]any)
		nested, _ := vm["mod"].(map[string]any)
		if nested == nil {
			return "", nil, mcUnrep("EnemyModifier value carries no nested mod table")
		}
		filled := make(map[string]any, len(nested)+3)
		for k, x := range nested {
			filled[k] = x
		}
		for field, def := range map[string]any{"flags": mustNumber("0"), "keywordFlags": mustNumber("0"), "tags": []any{}} {
			if _, ok := filled[field]; !ok {
				filled[field] = def
			}
		}
		nbase, ntags, err := sbmEmitMod(filled, sid, fields, stringFields, v)
		if err != nil {
			return "", nil, err
		}
		if len(ntags) > 0 {
			return "", nil, mcUnrep("EnemyModifier's nested mod carries tags, which this arm does not emit")
		}
		base = "mod.NewList(\"EnemyModifier\", mod.EnemyModifier{Mod: " + nbase + "})"
	case mtype == "LIST":
		return "", nil, mcUnrep("unhandled LIST mod %s", name)
	case mtype == "FLAG":
		base = "mod.NewFlag(" + mcGoStr(name) + ", true)"
	default:
		goType, ok := mcModType[mtype]
		if !ok {
			return "", nil, mcUnrep("unhandled mod type %s", mtype)
		}
		lit := "0"
		if _, isNum := oeFloat(val); isNum && pyTypeName(val) != "bool" {
			lit = mcGoNum(val)
		}
		base = fmt.Sprintf("mod.NewFloat(%s, %s, %s)", mcGoStr(name), goType, lit)
	}

	if oeTruthy(m["flags"]) {
		f, ok := mcBits(m["flags"], v.mflag)
		if !ok {
			return "", nil, mcUnrep("flags %s are not a union of named MFlag bits", oeStr(m["flags"]))
		}
		base += ".Flag(" + f + ")"
	}
	if oeTruthy(m["keywordFlags"]) {
		k, ok := mcBits(m["keywordFlags"], v.kw)
		if !ok {
			return "", nil, mcUnrep("keywordFlags %s are not a union of named KeywordFlag bits",
				oeStr(m["keywordFlags"]))
		}
		base += ".KeywordFlag(" + k + ")"
	}
	tags, _ := m["tags"].([]any)
	return base, tags, nil
}

func skillBaseModsEmit(args []string) int {
	if len(args) != 2 && len(args) != 3 {
		fmt.Fprintln(os.Stderr, sbmUsage)
		return 1
	}
	src, dst := args[0], args[1]
	d, reason := pyLoadJSONObject(src)
	if reason != "" {
		if strings.HasSuffix(reason, "is not a JSON object") {
			return mcRefuse(fmt.Sprintf("%s is not a JSON object", src))
		}
		return mcRefuse(strings.Replace(reason, src, fmt.Sprintf("the dump (%s)", src), 1))
	}
	for _, field := range []string{"skillTypeNames", "skills", "skillCount", "modCount"} {
		if _, ok := d[field]; !ok {
			return mcRefuse(fmt.Sprintf("the dump carries no %s. Re-run oracle/skill_base_mods.lua.",
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

	typesPath := strings.Replace(dst, "data/raw/skill_base_mods.go", "calculator/types.go", 1)
	if len(args) == 3 {
		typesPath = args[2]
	}
	raw, err := os.ReadFile(typesPath) //nolint:gosec // the caller's argument, as in the original
	if err != nil {
		return mcRefuse(fmt.Sprintf("calculator/types.go (%s) is not readable: %s",
			typesPath, errnoText(err)))
	}
	typesGo := string(raw)
	const marker = "type SkillData struct {"
	if !strings.Contains(typesGo, marker) {
		return mcRefuse(fmt.Sprintf("%s declares no `type SkillData struct {`; utils.Set resolves "+
			"a SkillData mod's key by FIELD NAME on it, so this emitter cannot tell a valid key "+
			"from an invalid one without it", typesPath))
	}
	body := typesGo[strings.Index(typesGo, marker)+len(marker):]
	if i := strings.Index(body, "\n}"); i >= 0 {
		body = body[:i]
	}
	fields := map[string]bool{}
	for _, m := range sbmField.FindAllStringSubmatch(body, -1) {
		fields[m[1]] = true
	}
	stringFields := map[string]bool{}
	for _, m := range sbmStringField.FindAllStringSubmatch(body, -1) {
		stringFields[m[1]] = true
	}
	if len(fields) == 0 {
		return mcRefuse(fmt.Sprintf("%s: SkillData has no fields; every skill() entry would be "+
			"dropped as an invalid key", typesPath))
	}

	omit := newPyCounter()
	omitDetail := newPyListMap()
	outBody := []string{}
	keptMods := 0
	for _, sid := range oeSortedKeys(skills) {
		s, _ := skills[sid].(map[string]any)
		mods, modsOK := s["mods"].([]any)
		if s == nil || !modsOK {
			return mcRefuse(fmt.Sprintf(
				"skill %s carries no 'mods' list. Re-run oracle/skill_base_mods.lua.", pyRepr(sid)))
		}
		lines := []string{}
		for _, mAny := range mods {
			m, ok := mAny.(map[string]any)
			if !ok {
				omit.add("mod is not a JSON object")
				omitDetail.add("mod is not a JSON object", sid+"/?")
				continue
			}
			base, tags, err := sbmEmitMod(m, sid, fields, stringFields, vocab)
			if err == nil {
				emitted := []string{}
				for _, tAny := range tags {
					u, uerr := mcUnwrap(tAny)
					if uerr != nil {
						err = uerr
						break
					}
					tm, _ := u.(map[string]any)
					if tm == nil {
						err = mcUnrep("unknown tag type %s", oeStr(nil))
						break
					}
					got, terr := sbmEmitTag(tm, skillTypes, vocab)
					if terr != nil {
						err = terr
						break
					}
					if got != "" {
						emitted = append(emitted, got)
					}
				}
				if err == nil {
					if len(emitted) > 0 {
						base += ".Tag(" + strings.Join(emitted, ", ") + ")"
					}
					base += ".Source(" + mcGoStr("Skill:"+sid) + ")"
					lines = append(lines, "\t\t"+base+",")
					keptMods++
					continue
				}
			}
			var u *mcUnrepresentableError
			if !errorsAs(err, &u) {
				return mcRefuse(err.Error())
			}
			omit.add(u.reason)
			omitDetail.add(u.reason, sid+"/"+oeStr(m["name"]))
		}
		if len(lines) == 0 {
			continue
		}
		loc := "src/Data/Skills"
		if oeTruthy(s["loc"]) {
			loc = "src/" + oeStr(s["loc"])
		}
		outBody = append(outBody, "\t// "+loc, "\t"+mcGoStr(sid)+": {")
		outBody = append(outBody, lines...)
		outBody = append(outBody, "\t},")
	}

	omittedTotal := omit.total()
	headerOmissions := []string{}
	for _, reason := range omit.mostCommon() {
		headerOmissions = append(headerOmissions,
			fmt.Sprintf("//   - %d %s", omit.counts[reason], reason))
	}

	out := fmt.Sprintf(sbmHeader, oeStr(d["skillCount"]), oeStr(d["modCount"]),
		keptMods, omittedTotal, strings.Join(headerOmissions, "\n")) +
		strings.Join(outBody, "\n") + "\n}\n" + sbmFooter
	if err := os.WriteFile(dst, []byte(out), 0o644); err != nil { //nolint:gosec // the original's mode
		return mcRefuse(fmt.Sprintf("%s is not writable: %s", dst, errnoText(err)))
	}

	fmt.Printf("skills %s  mods %s -> kept %d, omitted %d\n",
		oeStr(d["skillCount"]), oeStr(d["modCount"]), keptMods, omittedTotal)
	for _, reason := range omit.mostCommon() {
		fmt.Printf("  %4d  %s\n", omit.counts[reason], reason)
		for _, x := range omitDetail.head(reason, 3) {
			fmt.Printf("          e.g. %s\n", x)
		}
	}
	return 0
}
