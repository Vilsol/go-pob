package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
)

var mdEntryFields = []string{"id", "name"}

const mdUsage = "usage: minion_data_emit.py <dump.json> <out.go> <minions|spectres>"

type mdRefusalError struct{ msg string }

func (e *mdRefusalError) Error() string { return e.msg }

type mdUnportableError struct{ msg string }

func (e *mdUnportableError) Error() string { return e.msg }

func mdRefuse(msg string) int {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	return 3
}

func mdGoStr(s string) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return `""`
	}
	return strings.TrimRight(b.String(), "\n")
}

func mdGoNum(v any) string {
	f, ok := oeFloat(v)
	if !ok {
		return "0"
	}
	if f == float64(int64(f)) && (f < 1e15 && f > -1e15) {
		return fmt.Sprintf("%d", int64(f))
	}
	return pyFloat(f)
}

type mdRun struct {
	fatal            error
	unportable       *pyCounter
	unportableDetail map[string][]string
	defaulted        []string
}

func (r *mdRun) drop(reason, where string) {
	r.unportable.add(reason)
	r.unportableDetail[reason] = append(r.unportableDetail[reason], where)
}

func mdStr(m map[string]any, key string) string { return oeStr(mdGet(m, key, "")) }

func mdGet(m map[string]any, key string, def any) any {
	if v, ok := m[key]; ok && v != nil {
		return v
	}
	return def
}

func mdStrList(v any) []string {
	a, _ := v.([]any)
	out := make([]string, 0, len(a))
	for _, x := range a {
		out = append(out, oeStr(x))
	}
	return out
}

var mdModFields = []string{"valueKind", "modType", "name", "flags", "keywordFlags", "tags"}

func mdNeed(d map[string]any, field, where string) (any, error) {
	v, ok := d[field]
	if !ok {
		return nil, &mdRefusalError{fmt.Sprintf("%s: the mod has no %s", where, pyRepr(field))}
	}
	return v, nil
}

func (r *mdRun) emitMod(m map[string]any, where string, nested bool) (string, error) {
	for _, f := range mdModFields {
		if _, ok := m[f]; !ok {
			return "", &mdRefusalError{fmt.Sprintf("%s: the mod has no %s", where, pyRepr(f))}
		}
	}
	kind := oeStr(m["valueKind"])
	var expr string
	switch kind {
	case "number":
		if oeStr(m["modType"]) == "FLAG" {
			return r.finish("mod.NewFlag("+mdGoStr(oeStr(m["name"]))+", true)", m, where, nested)
		}
		modType, ok := mdModTypes[oeStr(m["modType"])]
		if !ok {
			return "", &mdRefusalError{fmt.Sprintf("%s: mod type %s has no Go constant",
				where, pyRepr(oeStr(m["modType"])))}
		}
		num, nerr := mdNeed(m, "number", where)
		if nerr != nil {
			return "", nerr
		}
		expr = fmt.Sprintf("mod.NewFloat(%s, %s, %s)",
			mdGoStr(oeStr(m["name"])), modType, mdGoNum(num))
	case "boolean":
		if oeStr(m["modType"]) != "FLAG" {
			return "", &mdRefusalError{fmt.Sprintf("%s: boolean value on a %s mod, expected FLAG",
				where, pyRepr(oeStr(m["modType"])))}
		}
		bv, berr := mdNeed(m, "boolean", where)
		if berr != nil {
			return "", berr
		}
		b := "false"
		if oeTruthy(bv) {
			b = "true"
		}
		expr = fmt.Sprintf("mod.NewFlag(%s, %s)", mdGoStr(oeStr(m["name"])), b)
	case "skillData":
		dk, dkerr := mdNeed(m, "dataKey", where)
		if dkerr != nil {
			return "", dkerr
		}
		goKey, ok := mdSkillDataKeys[oeStr(dk)]
		if !ok {
			return "", &mdRefusalError{fmt.Sprintf("%s: SkillData key %s is not in SKILL_DATA_KEYS. "+
				"Add it with the calculator/types.go field it feeds, or the mod is emitted under "+
				"a name nothing reads -- which is inert and silent.",
				where, pyRepr(oeStr(m["dataKey"])))}
		}
		dvt, dverr := mdNeed(m, "dataValueType", where)
		if dverr != nil {
			return "", dverr
		}
		value := ""
		if oeStr(dvt) == "boolean" {
			db, dberr := mdNeed(m, "dataBoolean", where)
			if dberr != nil {
				return "", dberr
			}
			if !oeTruthy(db) {
				return "", &mdRefusalError{fmt.Sprintf("%s: SkillData %s is false; 0 and absent are "+
					"indistinguishable in mod.SkillData.Value", where, pyRepr(oeStr(m["dataKey"])))}
			}
			value = "1"
		} else {
			dn, dnerr := mdNeed(m, "dataNumber", where)
			if dnerr != nil {
				return "", dnerr
			}
			value = mdGoNum(dn)
		}
		expr = fmt.Sprintf("skill(%s, %s)", mdGoStr(goKey), value)
	case "nestedMod":
		outer := oeStr(m["name"])
		goType, ok := mdNestedListTypes[outer]
		if !ok {
			return "", &mdUnportableError{fmt.Sprintf("nested-mod outer name %s has no Go list type "+
				"[go-absent:mod.%s]. mod/list_types.go models %s; adding one is a port, not a "+
				"mapping.", pyRepr(outer), outer, pyReprValue(mdSortedKeys(mdNestedListTypes)))}
		}
		nv, nverr := mdNeed(m, "nested", where)
		if nverr != nil {
			return "", nverr
		}
		inner, err := r.emitModAny(nv, where+" nested", true)
		if err != nil {
			return "", err
		}
		expr = fmt.Sprintf("mod.NewList(%s, mod.%s{Mod: %s})", mdGoStr(outer), goType, inner)
	default:
		return "", &mdRefusalError{fmt.Sprintf("%s: value kind %s has no Go form", where, pyRepr(kind))}
	}
	return r.finish(expr, m, where, nested)
}

func (r *mdRun) emitModAny(v any, where string, nested bool) (string, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", &mdRefusalError{fmt.Sprintf("%s: the mod is not a table", where)}
	}
	return r.emitMod(m, where, nested)
}

func mdSortedKeys(m map[string]string) []any {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	any_ := make([]any, len(out))
	for i, s := range out {
		any_[i] = s
	}
	return any_
}

func (r *mdRun) finish(expr string, m map[string]any, where string, nested bool) (string, error) {
	for _, f := range mdStrList(m["flags"]) {
		c, ok := mdModFlags[f]
		if !ok {
			return "", &mdRefusalError{fmt.Sprintf("%s: ModFlag %s has no Go constant in MOD_FLAGS",
				where, pyRepr(f))}
		}
		expr += ".Flag(" + c + ")"
	}
	for _, f := range mdStrList(m["keywordFlags"]) {
		c, ok := mdKeywordFlags[f]
		if !ok {
			return "", &mdRefusalError{fmt.Sprintf(
				"%s: KeywordFlag %s has no Go constant in KEYWORD_FLAGS", where, pyRepr(f))}
		}
		expr += ".KeywordFlag(" + c + ")"
	}
	rawTags, _ := m["tags"].([]any)
	for _, tAny := range rawTags {
		tag, ok := tAny.(map[string]any)
		if !ok {
			return "", &mdRefusalError{fmt.Sprintf("%s: a tag is not a table", where)}
		}
		ttv, tterr := mdNeed(tag, "tagType", where)
		if tterr != nil {
			return "", tterr
		}
		tt := oeStr(ttv)
		switch tt {
		case "Condition":
			v, err := mdNeed(tag, "var", where)
			if err != nil {
				return "", err
			}
			expr += ".Tag(mod.Condition(" + mdGoStr(oeStr(v)) + "))"
		case "GlobalEffect":
			et, err := mdNeed(tag, "effectType", where)
			if err != nil {
				return "", err
			}
			g := "mod.GlobalEffect(" + mdGoStr(oeStr(et)) + ")"
			if oeTruthy(tag["effectName"]) {
				g += ".Name(" + mdGoStr(mdStr(tag, "effectName")) + ")"
			}
			if oeTruthy(tag["unscalable"]) {
				g += ".Unscalable(true)"
			}
			expr += ".Tag(" + g + ")"
		case "Multiplier":
			v, err := mdNeed(tag, "var", where)
			if err != nil {
				return "", err
			}
			mt := "mod.Multiplier(" + mdGoStr(oeStr(v)) + ")"
			if oeTruthy(tag["actor"]) {
				mt += ".Actor(" + mdGoStr(mdStr(tag, "actor")) + ")"
			}
			expr += ".Tag(" + mt + ")"
		case "PerStat":
			st, err := mdNeed(tag, "stat", where)
			if err != nil {
				return "", err
			}
			expr += fmt.Sprintf(".Tag(mod.PerStat(%s, %s))",
				mdGoNum(mdGet(tag, "div", json.Number("1"))), mdGoStr(oeStr(st)))
		case "MonsterTag":
			var names []string
			if lst, ok := tag["monsterTagList"].([]any); ok {
				for _, n := range lst {
					names = append(names, mdGoStr(oeStr(n)))
				}
			} else {
				v, err := mdNeed(tag, "monsterTag", where)
				if err != nil {
					return "", err
				}
				names = append(names, mdGoStr(oeStr(v)))
			}
			mt := "mod.MonsterTag(" + strings.Join(names, ", ") + ")"
			if oeTruthy(tag["neg"]) {
				mt += ".Neg(true)"
			}
			expr += ".Tag(" + mt + ")"
		default:
			return "", &mdUnportableError{fmt.Sprintf("tag type %s has no Go form [go-absent:mod.%sTag]",
				pyRepr(tt), tt)}
		}
	}
	if nested {
		return expr, nil
	}
	if !oeTruthy(m["source"]) {
		return "", &mdRefusalError{fmt.Sprintf("%s: mod has no source. src/Modules/Data.lua:1031-1033 "+
			"stamps \"Minion:<name>\" on every one; an unstamped mod means the dumper read the "+
			"table before Data.lua finished.", where)}
	}
	return expr + ".Source(" + mdGoStr(oeStr(m["source"])) + ")", nil
}

func (r *mdRun) emitEntry(row map[string]any, luaFile string) []string {
	loc, _ := gemsInt(row["loc"])
	out := []string{
		fmt.Sprintf("\t// %s:%d", luaFile, loc),
		fmt.Sprintf("\t%s: {", mdGoStr(oeStr(row["id"]))),
		fmt.Sprintf("\t\tName: %s,", mdGoStr(oeStr(row["name"]))),
	}
	for _, kf := range mdNumeric {
		key, field := kf[0], kf[1]
		if v, ok := row[key]; ok && v != nil {
			out = append(out, fmt.Sprintf("\t\t%s: %s,", field, mdGoNum(v)))
		} else if key == "accuracy" {
			r.defaulted = append(r.defaulted, oeStr(row["id"]))
			out = append(out, "\t\tAccuracy: 1,")
		}
	}
	for _, kf := range mdStrings {
		if oeTruthy(row[kf[0]]) {
			out = append(out, fmt.Sprintf("\t\t%s: %s,", kf[1], mdGoStr(oeStr(row[kf[0]]))))
		}
	}
	for _, kf := range mdBools {
		if oeTruthy(row[kf[0]]) {
			out = append(out, fmt.Sprintf("\t\t%s: true,", kf[1]))
		}
	}
	if tags := mdStrList(row["monsterTags"]); len(tags) > 0 {
		out = append(out, "\t\tMonsterTags: []string{"+mdJoinStr(tags)+"},")
	}
	if skills := mdStrList(row["skillList"]); len(skills) > 0 {
		out = append(out, "\t\tSkillList: []string{"+mdJoinStr(skills)+"},")
	}
	mods, _ := row["modList"].([]any)
	if len(mods) > 0 {
		out = append(out, "\t\tModList: []mod.Mod{")
		for i, mAny := range mods {
			where := fmt.Sprintf("%s %s modList[%d]", luaFile, oeStr(row["id"]), i+1)
			s, err := r.emitModAny(mAny, where, false)
			if err != nil {
				var un *mdUnportableError
				if errorsAs(err, &un) {
					r.drop(un.msg, where)
					continue
				}
				r.fatal = err
				return out
			}
			out = append(out, "\t\t\t"+s+",")
		}
		out = append(out, "\t\t},")
	}
	return append(out, "\t},")
}

func mdJoinStr(xs []string) string {
	q := make([]string, len(xs))
	for i, s := range xs {
		q[i] = mdGoStr(s)
	}
	return strings.Join(q, ", ")
}

func minionDataEmit(args []string) int {
	if len(args) != 3 {
		fmt.Fprintln(os.Stderr, mdUsage)
		return 2
	}
	table := args[2]
	spec, ok := mdTables[table]
	if !ok {
		return mdRefuse(fmt.Sprintf("unknown table %s; have %s",
			pyRepr(table), strings.Join(mdTableOrder, ", ")))
	}
	raw, rerr := os.ReadFile(args[0]) //nolint:gosec // the operator's dump path, as in the original
	if rerr != nil {
		return mdRefuse(fmt.Sprintf("the dump (%s) is not readable: %s", args[0], errnoText(rerr)))
	}
	if !utf8.Valid(raw) {
		return mdRefuse(fmt.Sprintf("the dump (%s) is not valid JSON", args[0]))
	}
	var top any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&top); err != nil {
		return mdRefuse(fmt.Sprintf("the dump (%s) is not valid JSON", args[0]))
	}
	dump, isObj := top.(map[string]any)
	if !isObj {
		return mdRefuse(fmt.Sprintf("the dump (%s) is not a JSON object", args[0]))
	}
	rowsAny, _ := dump[table].(map[string]any)
	if len(rowsAny) == 0 {
		//go-only: the python named oracle/minion_data.lua, retired for scripts/gamedatadump minion-data.
		return mdRefuse(fmt.Sprintf("the dump has no %s entries. Re-run `go run ./scripts/gamedatadump minion-data`.",
			pyRepr(table)))
	}

	r := &mdRun{unportable: newPyCounter(), unportableDetail: map[string][]string{}}

	rows := make([]map[string]any, 0, len(rowsAny))
	rowKeys := make([]string, 0, len(rowsAny))
	for _, k := range oeSortedKeys(rowsAny) {
		row, ok := rowsAny[k].(map[string]any)
		if !ok {
			return mdRefuse(fmt.Sprintf("%s: the entry has no 'id'", k))
		}
		rows = append(rows, row)
		rowKeys = append(rowKeys, k)
	}
	for i, row := range rows {
		for _, f := range mdEntryFields {
			if !oeTruthy(row[f]) {
				return mdRefuse(fmt.Sprintf("%s: the entry has no %s", rowKeys[i], pyRepr(f)))
			}
		}
		if !oeTruthy(row["loc"]) {
			return mdRefuse(fmt.Sprintf("%s: the dumper found no declaring line to cite",
				oeStr(row["id"])))
		}
		if _, ok := gemsInt(row["loc"]); !ok || pyTypeName(row["loc"]) == "bool" {
			return mdRefuse(fmt.Sprintf("%s: the declaring line %s is not a number",
				oeStr(row["id"]), pyReprValue(row["loc"])))
		}
	}
	sort.SliceStable(rows, func(a, b int) bool {
		x, _ := gemsInt(rows[a]["loc"])
		y, _ := gemsInt(rows[b]["loc"])
		return x < y
	})

	body := []string{}
	for _, row := range rows {
		body = append(body, r.emitEntry(row, spec.luaFile)...)
		if r.fatal != nil {
			return mdRefuse(r.fatal.Error())
		}
	}

	needsMod := false
	for _, row := range rows {
		if a, _ := row["modList"].([]any); len(a) > 0 {
			needsMod = true
			break
		}
	}
	generator := "spectre-data"
	if table == "minions" {
		generator = "minion-data"
	}
	head := []string{"//pob:" + spec.luaFile, "package raw", ""}
	if needsMod {
		head = append(head, `import "github.com/Vilsol/go-pob/mod"`, "")
	}
	head = append(head,
		fmt.Sprintf("// CODE GENERATED BY `gamedatadump minion-data` + `gopob minion-data-emit` FROM %s. DO NOT EDIT.",
			oeStr(mdGet(dump, "pobVersion", "unknown"))),
		fmt.Sprintf("// Regenerate with `task data-regen -- --only %s`.",
			generator),
		"//",
		`// Every ModList mod carries source "Minion:<name>", which`,
		"// src/Modules/Data.lua:1031-1033 stamps on after loading.",
		"")
	if table == "minions" {
		head = append(head, strings.TrimSpace(mdMinionInfoStruct), "")
	}
	head = append(head, spec.doc)

	lines := append(head, "var "+spec.goVar+" = map[string]*MinionInfo{")
	lines = append(lines, body...)
	lines = append(lines, "}")
	if spec.trailer != "" {
		lines = append(lines, strings.TrimRight(spec.trailer, " \t\n"))
	}
	lines = append(lines, "")

	if err := os.WriteFile(args[1], []byte(strings.Join(lines, "\n")), 0o644); err != nil { //nolint:gosec // the original's mode
		return mdRefuse(fmt.Sprintf("%s is not writable: %s", args[1], errnoText(err)))
	}

	nMods, nSkills := 0, 0
	for _, row := range rows {
		a, _ := row["modList"].([]any)
		nMods += len(a)
		nSkills += len(mdStrList(row["skillList"]))
	}
	tail := ""
	if len(r.defaulted) > 0 {
		tail = fmt.Sprintf(", %d accuracy defaulted to 1", len(r.defaulted))
	}
	fmt.Fprintf(os.Stderr, "wrote %s: %d %s, %d mods, %d skill-list entries%s\n",
		args[1], len(rows), table, nMods, nSkills, tail)

	if r.unportable.total() > 0 {
		fmt.Fprintf(os.Stderr, "  DROPPED %d mod(s) this emitter cannot represent:\n",
			r.unportable.total())
		for _, reason := range r.unportable.mostCommon() {
			fmt.Fprintf(os.Stderr, "    %4d  %s\n", r.unportable.counts[reason], reason)
			details := r.unportableDetail[reason]
			if len(details) > 2 {
				details = details[:2]
			}
			for _, d := range details {
				fmt.Fprintf(os.Stderr, "          e.g. %s\n", d)
			}
		}
	}
	return 0
}
