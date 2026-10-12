package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func dmRepoRoot() string {
	cmd := exec.CommandContext(context.Background(), "git", "rev-parse", "--show-toplevel")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			dmRefuse(fmt.Sprintf("git rev-parse --show-toplevel failed: exit status %d",
				ee.ExitCode()))
		}
		dmRefuse("git rev-parse --show-toplevel failed: " + errnoText(err))
	}
	return strings.TrimSpace(string(out))
}

func dmRead(pob, rel string) (string, error) {
	path := pyPathJoin(pob, rel)
	st, err := os.Stat(path) //nolint:gosec // POB_DIR is the operator's, as in the original
	if err != nil || st.IsDir() {
		return "", dmMissing("%s is not in %s", rel, pob)
	}
	return dmReadText(path, "a PoB source file"), nil
}

func dmSkillFiles(pob string) ([]string, error) {
	files, err := filepath.Glob(pyPathJoin(pob, dmSkillGlob))
	if err != nil || len(files) == 0 {
		return nil, dmMissing("%s matched nothing in %s", dmSkillGlob, pob)
	}
	sort.Strings(files)
	return files, nil
}

type dmSkill struct {
	table *ldTable
	src   string
	line  int
}

type dmSkillSet struct {
	order []string
	byKey map[string]dmSkill
}

var dmSkillsCache = map[string]*dmSkillSet{}

func dmAllSkills(pob string) (*dmSkillSet, error) {
	if cached, ok := dmSkillsCache[pob]; ok {
		return cached, nil
	}
	files, err := dmSkillFiles(pob)
	if err != nil {
		return nil, err
	}
	out := &dmSkillSet{byKey: map[string]dmSkill{}}
	for _, path := range files {
		text := dmReadText(path, "a PoB skills file")
		rows, err := ldScanAssignments(text, "skills")
		if err != nil {
			return nil, err
		}
		rel, relErr := filepath.Rel(pob, path)
		if relErr != nil {
			rel = path
		}
		for _, r := range rows {
			key, _ := r.key.(string)
			if _, seen := out.byKey[key]; !seen {
				out.order = append(out.order, key)
			}
			out.byKey[key] = dmSkill{table: r.table, src: rel, line: r.line}
		}
	}
	if len(out.byKey) == 0 {
		return nil, dmMissing("no skills[...] assignments found -- wrong layout?")
	}
	dmSkillsCache[pob] = out
	return out, nil
}

func dmSkillField(pob, field string, transform func(any) (any, error)) (map[string]any, error) {
	skills, err := dmAllSkills(pob)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, key := range skills.order {
		sk := skills.byKey[key]
		v, ok := sk.table.hash[field]
		if !ok {
			continue
		}
		row, err := transform(v)
		if err != nil {
			return nil, err
		}
		out[key] = row
	}
	return out, nil
}

func dmAsTable(value any) (*ldTable, error) {
	t, ok := value.(*ldTable)
	if !ok {
		return nil, ldFail("expected a table, got %s", pyReprValue(value))
	}
	return t, nil
}

func dmFlagSet(value any) (any, error) {
	t, err := dmAsTable(value)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, k := range t.keys() {
		if b, ok := t.hash[k].(bool); ok && b {
			if ks, ok := k.(string); ok {
				names = append(names, ks)
			}
		}
	}
	sort.Strings(names)
	return dmStrList(names), nil
}

func dmStrList(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func dmStringList(value any) (any, error) {
	t, err := dmAsTable(value)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, v := range t.array {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out, nil
}

func dmSkillTypeList(value any) (any, error) {
	t, err := dmAsTable(value)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, v := range t.array {
		e, ok := v.(*ldExpr)
		if !ok || !strings.HasPrefix(e.text, "SkillType.") {
			return nil, ldFail("not a SkillType: %s", pyReprValue(v))
		}
		out = append(out, strings.SplitN(e.text, ".", 2)[1])
	}
	return out, nil
}

func dmNum(v any, def float64) (float64, error) {
	if v == nil {
		return def, nil
	}
	if _, isBool := v.(bool); isBool {
		return 0, ldFail("expected a number, got %s", pyReprValue(v))
	}
	switch t := v.(type) {
	case int:
		return float64(t), nil
	case float64:
		return t, nil
	}
	return 0, ldFail("expected a number, got %s", pyReprValue(v))
}

func dmModCallNames(value any) (any, error) {
	t, err := dmAsTable(value)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, v := range t.array {
		e, ok := v.(*ldExpr)
		if !ok || !e.isCall || e.callee == "" {
			return nil, ldFail("not a mod call: %s", pyReprValue(v))
		}
		callee := e.callee
		if i := strings.LastIndex(callee, "."); i >= 0 {
			callee = callee[i+1:]
		}
		switch callee {
		case "skill":
			out = append(out, "SkillData")
		case "mod", "flag":
			if len(e.args) == 0 {
				return nil, ldFail("%s() without a literal name: %s", callee, pyReprValue(v))
			}
			name, ok := e.args[0].(string)
			if !ok {
				return nil, ldFail("%s() without a literal name: %s", callee, pyReprValue(v))
			}
			out = append(out, name)
		default:
			return nil, ldFail("unknown mod constructor %s", pyRepr(callee))
		}
	}
	return out, nil
}

func dmStatMapRow(value any) (any, error) {
	t, err := dmAsTable(value)
	if err != nil {
		return nil, err
	}
	mods, err := dmModCallNames(value)
	if err != nil {
		return nil, err
	}
	row := map[string]any{"mods": mods}
	for _, key := range []string{"value", "mult", "div", "base"} {
		if v, ok := t.hash[key]; ok {
			n, err := dmNum(v, 0)
			if err != nil {
				return nil, err
			}
			row[key] = n
		}
	}
	return row, nil
}

func dmHashStr(t *ldTable, key string) string {
	if v, ok := t.hash[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func dmMinionRow(value any, forcedLimit string) (any, error) {
	t, err := dmAsTable(value)
	if err != nil {
		return nil, err
	}
	row := map[string]any{}
	if v, ok := t.hash["name"]; ok {
		row["name"] = v
	} else {
		row["name"] = ""
	}
	for _, f := range []struct {
		key string
		def float64
	}{
		{"life", 0}, {"energyShield", 0}, {"armour", 0}, {"fireResist", 0}, {"coldResist", 0},
		{"lightningResist", 0}, {"chaosResist", 0}, {"damage", 0}, {"damageSpread", 0},
		{"damageFixup", 0}, {"attackTime", 0}, {"attackRange", 0},
		{"accuracy", 1.0},
	} {
		n, err := dmNum(t.hash[f.key], f.def)
		if err != nil {
			return nil, err
		}
		row[f.key] = n
	}
	row["weaponType1"] = dmHashStr(t, "weaponType1")
	row["weaponType2"] = dmHashStr(t, "weaponType2")
	limit := forcedLimit
	if limit == "" {
		limit = dmHashStr(t, "limit")
	}
	row["limit"] = limit
	row["skillList"] = []any{}
	if v, ok := t.hash["skillList"]; ok && v != nil {
		l, err := dmStringList(v)
		if err != nil {
			return nil, err
		}
		row["skillList"] = l
	}
	row["modList"] = []any{}
	if v, ok := t.hash["modList"]; ok && v != nil {
		l, err := dmModCallNames(v)
		if err != nil {
			return nil, err
		}
		row["modList"] = l
	}
	return row, nil
}

func dmExtractMinionsFrom(pob, rel, forcedLimit string) (map[string]any, error) {
	text, err := dmRead(pob, rel)
	if err != nil {
		return nil, err
	}
	rows, err := ldScanAssignments(text, "minions")
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, r := range rows {
		key, _ := r.key.(string)
		row, err := dmMinionRow(r.table, forcedLimit)
		if err != nil {
			return nil, err
		}
		out[key] = row
	}
	return out, nil
}

func dmExtractPoBSkills(pob string) (map[string]any, error) {
	skills, err := dmAllSkills(pob)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, key := range skills.order {
		h := skills.byKey[key].table
		out[key] = map[string]any{
			"name":     dmHashStr(h, "name"),
			"hidden":   dmIsTrue(h, "hidden"),
			"fromItem": dmIsTrue(h, "fromItem"),
			"fromTree": dmIsTrue(h, "fromTree"),
			"support":  dmIsTrue(h, "support"),
		}
	}
	return out, nil
}

func dmIsTrue(t *ldTable, key string) bool {
	b, ok := t.hash[key].(bool)
	return ok && b
}

var dmGemRenameRE = regexp.MustCompile(`"([^"]+)":\s*"([^"]+)"`)

func dmGemIDRenames() (map[string]string, error) {
	path := pyPathJoin(pyPathJoin(dmRepoRoot(), "builds"), "parser.go")
	src := dmReadText(path, "our own builds/parser.go")
	start := strings.Index(src, "pobGemIDtoGameGemIDs = map[string]string{")
	if start < 0 {
		return nil, dmMissing("builds.pobGemIDtoGameGemIDs is gone from builds/parser.go")
	}
	end := strings.Index(src[start:], "\n}")
	if end < 0 {
		return nil, dmMissing("builds.pobGemIDtoGameGemIDs is gone from builds/parser.go")
	}
	body := src[start : start+end]
	out := map[string]string{}
	for _, m := range dmGemRenameRE.FindAllStringSubmatch(body, -1) {
		out[m[1]] = m[2]
	}
	if len(out) == 0 {
		return nil, dmMissing("builds.pobGemIDtoGameGemIDs parsed as empty")
	}
	return out, nil
}

func dmExtractGems(pob string) (map[string]any, error) {
	renames, err := dmGemIDRenames()
	if err != nil {
		return nil, err
	}
	text, err := dmRead(pob, "src/Data/Gems.lua")
	if err != nil {
		return nil, err
	}
	table, err := ldParseReturnTable(text)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, k := range table.keys() {
		et, ok := table.hash[k].(*ldTable)
		if !ok {
			return nil, ldFail("gems[%s] is not a table", pyReprValue(k))
		}
		tags := []any{}
		if v, ok := et.hash["tags"]; ok && v != nil {
			t, err := dmFlagSet(v)
			if err != nil {
				return nil, err
			}
			tags = t.([]any)
		}
		key, _ := k.(string)
		if r, ok := renames[key]; ok {
			key = r
		}
		out[key] = map[string]any{"name": dmHashStr(et, "name"), "tags": tags}
	}
	return out, nil
}

func dmExtractSkillStatMap(pob string) (map[string]any, error) {
	text, err := dmRead(pob, "src/Data/SkillStatMap.lua")
	if err != nil {
		return nil, err
	}
	table, err := ldParseReturnTable(text)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, k := range table.keys() {
		row, err := dmStatMapRow(table.hash[k])
		if err != nil {
			return nil, err
		}
		key, _ := k.(string)
		out[key] = row
	}
	return out, nil
}

func dmExtractSkillStatMaps(pob string) (map[string]any, error) {
	return dmSkillField(pob, "statMap", func(v any) (any, error) {
		t, err := dmAsTable(v)
		if err != nil {
			return nil, err
		}
		row := map[string]any{}
		for _, k := range t.keys() {
			r, err := dmStatMapRow(t.hash[k])
			if err != nil {
				return nil, err
			}
			key, _ := k.(string)
			row[key] = r
		}
		return row, nil
	})
}

func dmPartsRows(value any) (any, error) {
	t, err := dmAsTable(value)
	if err != nil {
		return nil, err
	}
	rows := []any{}
	for _, part := range t.array {
		pt, ok := part.(*ldTable)
		if !ok {
			return nil, ldFail("part is not a table: %s", pyReprValue(part))
		}
		flags := map[string]any{}
		for _, k := range pt.keys() {
			ks, isStr := k.(string)
			b, isBool := pt.hash[k].(bool)
			if isStr && ks != "name" && isBool {
				flags[ks] = b
			}
		}
		rows = append(rows, map[string]any{"name": dmHashStr(pt, "name"), "flags": flags})
	}
	return rows, nil
}

func dmExtractHiddenLevels(pob string) (map[string]any, error) {
	skills, err := dmAllSkills(pob)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, key := range skills.order {
		h := skills.byKey[key].table
		types := []any{}
		if tt, ok := h.hash["skillTypes"].(*ldTable); ok {
			var names []string
			for _, k := range tt.keys() {
				if b, isBool := tt.hash[k].(bool); !isBool || !b {
					continue
				}
				s := dmStrOf(k)
				if strings.HasPrefix(s, "SkillType.") {
					s = strings.SplitN(s, ".", 2)[1]
				}
				names = append(names, s)
			}
			sort.Strings(names)
			types = dmStrList(names)
		}
		stats := []any{}
		if st, ok := h.hash["stats"].(*ldTable); ok {
			l, err := dmStringList(st)
			if err != nil {
				return nil, err
			}
			stats = l.([]any)
		}
		levels := map[string]any{}
		if lt, ok := h.hash["levels"].(*ldTable); ok {
			for _, k := range lt.keys() {
				vt, isTable := lt.hash[k].(*ldTable)
				if !isTable {
					continue
				}
				vals := []any{}
				for _, x := range vt.array {
					if _, isStr := x.(string); isStr {
						continue
					}
					if _, isExpr := x.(*ldExpr); isExpr {
						continue
					}
					n, err := dmNum(x, 0)
					if err != nil {
						return nil, err
					}
					vals = append(vals, n)
				}
				levels[dmStrOf(k)] = vals
			}
		}
		out[key] = map[string]any{"skillTypes": types, "stats": stats, "levels": levels}
	}
	return out, nil
}

func dmStrOf(k any) string {
	switch t := k.(type) {
	case string:
		return t
	case int:
		return fmt.Sprintf("%d", t)
	}
	return pyReprValue(k)
}
