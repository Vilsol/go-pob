package poe2skills

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// dirCache serves go-pob-data's json tables from a directory of <Table>.json.gz files.
type dirCache string

func (d dirCache) path(key string) string {
	return filepath.Join(string(d), filepath.Base(key))
}

func (d dirCache) Get(key string) ([]byte, error) {
	b, err := os.ReadFile(d.path(key))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", key, err)
	}
	return b, nil
}
func (d dirCache) Set(string, []byte) error { return nil }
func (d dirCache) Exists(key string) bool {
	_, err := os.Stat(d.path(key))
	return err == nil
}

// testSkills builds data.skills from POE2_SKILL_TABLES, a directory holding go-pob-data's
// <Table>.json.gz files.
func testSkills(t *testing.T) *Skills {
	t.Helper()
	dir := os.Getenv("POE2_SKILL_TABLES")
	if dir == "" {
		t.Skip("SKIP: set POE2_SKILL_TABLES to a directory of go-pob-data's PoE2 json tables")
	}
	skills, err := Load(context.Background(), "0.5", dirCache(dir))
	if err != nil {
		t.Fatal(err)
	}
	return skills
}

func luaNum(v float64) string {
	return strconv.FormatFloat(v, 'g', 17, 64)
}

func luaKey(k any) string {
	switch x := k.(type) {
	case int:
		return "[" + strconv.Itoa(x) + "]"
	case string:
		return `["` + x + `"]`
	}
	panic(fmt.Sprintf("key %T", k))
}

// rawName is a skill type printed by name, unquoted, as the dump prints an enum value.
type rawName string

func flattenTo(out map[string]string, path string, v any) {
	switch x := v.(type) {
	case nil:
	case float64:
		out[path] = luaNum(x)
	case int:
		out[path] = strconv.Itoa(x)
	case string:
		out[path] = `"` + strings.ReplaceAll(x, "\n", `\n`) + `"`
	case rawName:
		out[path] = string(x)
	case bool:
		out[path] = strconv.FormatBool(x)
	case []any:
		if len(x) == 0 {
			out[path] = "{}"
		}
		for i, e := range x {
			flattenTo(out, path+luaKey(i+1), e)
		}
	case map[string]any:
		if len(x) == 0 {
			out[path] = "{}"
		}
		for k, e := range x {
			flattenTo(out, path+luaKey(k), e)
		}
	default:
		panic(fmt.Sprintf("value %T", v))
	}
}

func names(list []string) []any {
	out := make([]any, len(list))
	for i, n := range list {
		out[i] = rawName(n)
	}
	return out
}

func strs(list []string) []any {
	out := make([]any, len(list))
	for i, n := range list {
		out[i] = n
	}
	return out
}

func floats(list []float64) []any {
	out := make([]any, len(list))
	for i, n := range list {
		out[i] = n
	}
	return out
}

func set(m map[string]bool) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

// luaValue is a mod JSON value as the dump prints the Lua table: tags at [1..n], skill types by name.
func luaValue(v any, field string) any {
	switch x := v.(type) {
	case map[string]any:
		if n, ok := x["$num"].(string); ok && len(x) == 1 {
			f, _ := strconv.ParseFloat(strings.TrimPrefix(n, "+"), 64)
			return f
		}
		out := map[string]any{}
		for k, e := range x {
			if k == "tags" {
				continue
			}
			out[k] = luaValue(e, k)
		}
		if tags, ok := x["tags"].([]any); ok && len(tags) > 0 {
			return withList(out, tags)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			if field == "skillTypeList" {
				out[i] = rawName(e.(string))
			} else {
				out[i] = luaValue(e, "")
			}
		}
		return out
	case string:
		if field == "skillType" {
			return rawName(x)
		}
	}
	return v
}

// mixed is a Lua table with both an array part and named fields.
type mixed struct {
	named map[string]any
	list  []any
}

func withList(named map[string]any, list []any) mixed {
	out := make([]any, len(list))
	for i, e := range list {
		out[i] = luaValue(e, "")
	}
	return mixed{named: named, list: out}
}

func flattenAny(out map[string]string, path string, v any) {
	if m, ok := v.(mixed); ok {
		if len(m.named) == 0 && len(m.list) == 0 {
			out[path] = "{}"
		}
		for k, e := range m.named {
			flattenAny(out, path+luaKey(k), e)
		}
		for i, e := range m.list {
			flattenAny(out, path+luaKey(i+1), e)
		}
		return
	}
	switch x := v.(type) {
	case map[string]any:
		if len(x) == 0 {
			out[path] = "{}"
		}
		for k, e := range x {
			flattenAny(out, path+luaKey(k), e)
		}
	case []any:
		if len(x) == 0 {
			out[path] = "{}"
		}
		for i, e := range x {
			flattenAny(out, path+luaKey(i+1), e)
		}
	default:
		flattenTo(out, path, v)
	}
}

func modTree(m Mod) any {
	return luaValue(m, "")
}

func scalars(t map[string]any, value, mult, div, base float64, hv, hm, hd, hb bool) {
	if hv {
		t["value"] = value
	}
	if hm {
		t["mult"] = mult
	}
	if hd {
		t["div"] = div
	}
	if hb {
		t["base"] = base
	}
}

func statMapTree(m *StatMap) any {
	named := map[string]any{}
	scalars(named, m.Value, m.Mult, m.Div, m.Base, m.HasValue, m.HasMult, m.HasDiv, m.HasBase)
	if m.SkillFlag != "" {
		named["skillFlag"] = m.SkillFlag
	}
	var list []any
	for _, it := range m.Items {
		if it.Mod != nil {
			list = append(list, modTree(it.Mod))
			continue
		}
		g := map[string]any{}
		scalars(g, it.Value, it.Mult, it.Div, it.Base, it.HasValue, it.HasMult, it.HasDiv, it.HasBase)
		if it.GroupSource != "" {
			g["source"] = it.GroupSource
		}
		mods := make([]any, len(it.Group))
		for j, gm := range it.Group {
			mods[j] = modTree(gm)
		}
		list = append(list, mixed{named: g, list: mods})
	}
	if len(list) == 0 && len(named) == 0 {
		return map[string]any{}
	}
	return mixed{named: named, list: list}
}

// goRows is the same `path -> value` view of one built effect as skills_dump.lua prints of PoB's.
func goRows(ge *GrantedEffect) map[string]string {
	out := map[string]string{}
	t := map[string]any{"id": ge.ID, "name": ge.Name, "modSource": ge.ModSource}
	o := ge.Overlay
	flags := map[string]bool{
		"hidden": ge.Hidden, "fromItem": o.FromItem, "fromTree": o.FromTree, "hideFromSideBar": o.HideFromSideBar,
		"grantsThornsDamage": o.GrantsThornsDamage, "legacy": o.Legacy, "minionHasItemSet": o.MinionHasItemSet,
		"support": ge.Support, "isLineage": ge.IsLineage, "isTrigger": ge.IsTrigger, "supportGemsOnly": ge.SupportGemsOnly,
		"ignoreMinionTypes": ge.IgnoreMinionTypes, "cannotBeSupported": ge.CannotBeSupported, "hasGlobalEffect": ge.HasGlobalEffect,
	}
	for k, v := range flags {
		if v {
			t[k] = true
		}
	}
	if o.PreDamageFunc {
		out[`["preDamageFunc"]`] = "function"
	}
	if o.MinionList != nil {
		t["minionList"] = strs(o.MinionList)
	}
	for k, list := range map[string][]string{"addFlags": o.AddFlags, "minionUses": o.MinionUses} {
		if list != nil {
			m := map[string]any{}
			for _, f := range list {
				m[f] = true
			}
			t[k] = m
		}
	}
	if ge.HasBaseTypeName {
		t["baseTypeName"] = ge.BaseTypeName
	}
	if ge.Description != "" {
		t["description"] = ge.Description
	}
	if ge.HasIcon {
		t["icon"] = ge.Icon
	}
	if ge.Color != 0 {
		t["color"] = ge.Color
	}
	if ge.Support {
		t["requireSkillTypes"] = names(ge.RequireSkillTypes)
		t["addSkillTypes"] = names(ge.AddSkillTypes)
		t["excludeSkillTypes"] = names(ge.ExcludeSkillTypes)
	}
	if ge.GemFamily != nil {
		t["gemFamily"] = strs(ge.GemFamily)
	}
	if ge.FlavourText != nil {
		t["flavourText"] = strs(ge.FlavourText)
	}
	if ge.WeaponTypes != nil {
		t["weaponTypes"] = set(ge.WeaponTypes)
	}
	if ge.SkillTypes != nil {
		t["skillTypes"] = set(ge.SkillTypes)
	}
	if ge.MinionSkillTypes != nil {
		t["minionSkillTypes"] = set(ge.MinionSkillTypes)
	}
	if ge.HasSkillTotemID {
		t["skillTotemId"] = ge.SkillTotemID
	}
	if ge.HasCastTime {
		t["castTime"] = ge.CastTime
	}
	for k, qs := range map[string][]QualityStat{"qualityStats": ge.QualityStats, "altQualityStats": ge.AltQualityStats} {
		if qs == nil {
			continue
		}
		list := make([]any, len(qs))
		for i, q := range qs {
			list[i] = []any{q.Stat, q.Value, floats(q.StatSets)}
		}
		t[k] = list
	}
	if ge.Levels != nil {
		levels := map[string]any{}
		for n, l := range ge.Levels {
			lt := map[string]any{}
			for k, v := range l.Extra {
				lt[k] = v
			}
			if len(l.Cost) > 0 {
				cost := map[string]any{}
				for k, v := range l.Cost {
					cost[k] = v
				}
				lt["cost"] = cost
			}
			levels[strconv.Itoa(n)] = lt
		}
		flattenLevels(out, `["levels"]`, levels)
	}
	sets := make([]any, len(ge.StatSets))
	for i, s := range ge.StatSets {
		st := map[string]any{}
		if s.HasLabel {
			st["label"] = s.Label
		}
		if s.HasBaseEffectiveness {
			st["baseEffectiveness"] = s.BaseEffectiveness
		}
		if s.HasIncrementalEffectiveness {
			st["incrementalEffectiveness"] = s.IncrementalEffectiveness
		}
		if s.HasDamageIncrementalEffect {
			st["damageIncrementalEffectiveness"] = s.DamageIncrementalEffectiveness
		}
		if s.HasStatDescriptionScope {
			st["statDescriptionScope"] = s.StatDescriptionScope
		}
		if s.BaseFlags != nil {
			st["baseFlags"] = set(s.BaseFlags)
		}
		if s.BaseMods != nil {
			mods := make([]any, len(s.BaseMods))
			for j, m := range s.BaseMods {
				mods[j] = modTree(m)
			}
			st["baseMods"] = mods
		}
		if s.ConstantStats != nil {
			cs := make([]any, len(s.ConstantStats))
			for j, c := range s.ConstantStats {
				cs[j] = []any{c.Stat, c.Value}
			}
			st["constantStats"] = cs
		}
		if s.HasStats {
			st["stats"] = strs(s.Stats)
		}
		if len(s.NotMinionStat) > 0 {
			st["notMinionStat"] = strs(s.NotMinionStat)
		}
		sets[i] = st
		if s.Levels != nil {
			levels := map[string]any{}
			for n, l := range s.Levels {
				lt := map[string]any{}
				for k, v := range l.Extra {
					lt[k] = v
				}
				if l.StatInterpolation != nil {
					lt["statInterpolation"] = floats(l.StatInterpolation)
				}
				if l.HasActorLevel {
					lt["actorLevel"] = l.ActorLevel
				}
				levels[strconv.Itoa(n)] = mixed{named: lt, list: floats(l.Values)}
			}
			flattenLevels(out, `["statSets"]`+luaKey(i+1)+`["levels"]`, levels)
		}
	}
	t["statSets"] = sets
	for k, v := range t {
		flattenAny(out, luaKey(k), v)
	}
	out["hasGlobalEffect"] = strconv.FormatBool(ge.HasGlobalEffect)
	for i, s := range ge.StatSets {
		var stats []string
		stats = append(stats, s.Stats...)
		for _, c := range s.ConstantStats {
			stats = append(stats, c.Stat)
		}
		for _, stat := range stats {
			s.StatMap(stat)
		}
		for stat, m := range s.statMap {
			flattenAny(out, `["statSets"]`+luaKey(i+1)+`["statMap"]`+luaKey(stat), statMapTree(m))
		}
	}
	out["hasGlobalEffectAfterStatMap"] = strconv.FormatBool(ge.HasGlobalEffect)
	return out
}

// flattenLevels keys levels by number, as PoB's levels[n] are.
func flattenLevels(out map[string]string, path string, levels map[string]any) {
	if len(levels) == 0 {
		out[path] = "{}"
	}
	for n, l := range levels {
		i, _ := strconv.Atoi(n)
		flattenAny(out, path+luaKey(i), l)
	}
}

// pobRows runs skills_dump.lua over the PoB-PoE2 pin.
func pobRows(t *testing.T) (map[string]map[string]string, int) {
	t.Helper()
	if f := os.Getenv("POE2_SKILL_DUMP"); f != "" {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		return parseDump(t, raw)
	}
	return parseDump(t, runPoBLua(t, "skills_dump.lua"))
}

// runPoBLua runs a testdata script inside the PoB-PoE2 pin under luajit and returns its stdout.
func runPoBLua(t *testing.T, script string) []byte {
	t.Helper()
	if os.Getenv("OPERATOR") != "true" {
		t.Skip("SKIP: runs the PoB-PoE2 pin under luajit; set OPERATOR=true")
	}
	common, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		t.Skipf("SKIP: not in a git checkout: %v", err)
	}
	main := filepath.Dir(strings.TrimSpace(string(common)))
	pob := os.Getenv("POB2_DIR")
	if pob == "" {
		pob = filepath.Join(main, "..", "PathOfBuilding-PoE2")
	}
	luajit := filepath.Join(main, ".luajit-det", "bin", "luajit")
	for _, p := range []string{filepath.Join(pob, "src", "HeadlessWrapper.lua"), luajit} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("SKIP: %s is absent", p)
		}
	}
	probe, err := filepath.Abs(filepath.Join("testdata", script))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(context.Background(), "setarch", "-R", luajit, probe)
	cmd.Dir = filepath.Join(pob, "src")
	cmd.Env = append(os.Environ(),
		"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
		"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	raw, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", script, err, stderr.String())
	}
	return raw
}

func parseDump(t *testing.T, raw []byte) (map[string]map[string]string, int) {
	t.Helper()
	rows := map[string]map[string]string{}
	count := -1
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		f := strings.SplitN(sc.Text(), "\t", 3)
		if len(f) != 3 {
			continue
		}
		if f[0] == "#count" {
			count, _ = strconv.Atoi(f[2])
			continue
		}
		if rows[f[0]] == nil {
			rows[f[0]] = map[string]string{}
		}
		rows[f[0]][f[1]] = f[2]
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if count <= 0 || len(rows) != count {
		t.Fatalf("the dump printed %d effects but counted %d; it stopped early", len(rows), count)
	}
	return rows, count
}

var familyIndex = regexp.MustCompile(`\[[0-9]+\]`)

// family is a path with its indices and stat map stat names dropped.
func family(path string) string {
	f := familyIndex.ReplaceAllString(path, "[]")
	if i := strings.Index(f, `["statMap"]`); i >= 0 {
		rest := f[i+len(`["statMap"]`):]
		if j := strings.Index(rest, `"]`); j >= 0 {
			f = f[:i] + `["statMap"][stat]` + rest[j+2:]
		}
	}
	return f
}

type familyTally struct{ agree, disagree, pobOnly, oursOnly int }

func diffRows(pob, ours map[string]map[string]string, only map[string]bool) (map[string]*familyTally, []string, map[string]bool) {
	fams := map[string]*familyTally{}
	tally := func(path string) *familyTally {
		f := family(path)
		if fams[f] == nil {
			fams[f] = &familyTally{}
		}
		return fams[f]
	}
	var bad []string
	badIDs := map[string]bool{}
	for id, prow := range pob {
		if only != nil && !only[id] {
			continue
		}
		orow := ours[id]
		for path, pv := range prow {
			ov, ok := orow[path]
			switch {
			case !ok:
				tally(path).pobOnly++
				bad = append(bad, fmt.Sprintf("%s %s: PoB %s, ours absent", id, path, pv))
				badIDs[id] = true
			case ov != pv:
				tally(path).disagree++
				bad = append(bad, fmt.Sprintf("%s %s: PoB %s, ours %s", id, path, pv, ov))
				badIDs[id] = true
			default:
				tally(path).agree++
			}
		}
		for path, ov := range orow {
			if _, ok := prow[path]; !ok {
				tally(path).oursOnly++
				bad = append(bad, fmt.Sprintf("%s %s: PoB absent, ours %s", id, path, ov))
				badIDs[id] = true
			}
		}
	}
	sort.Strings(bad)
	return fams, bad, badIDs
}

func logFamilies(t *testing.T, fams map[string]*familyTally) {
	keys := make([]string, 0, len(fams))
	for k := range fams {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f := fams[k]
		if f.disagree+f.pobOnly+f.oursOnly > 0 {
			t.Logf("family %s: agree %d, disagree %d, PoB only %d, ours only %d", k, f.agree, f.disagree, f.pobOnly, f.oursOnly)
		}
	}
}

// readIDs reads effect ids, one per line; a gem or game id (Metadata/...) stands for every effect its gems grant.
func readIDs(t *testing.T, skills *Skills, path string) map[string]bool {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, l := range strings.Fields(string(b)) {
		if !strings.HasPrefix(l, "Metadata/") {
			ids[l] = true
			continue
		}
		found := false
		for _, gem := range skills.gems {
			if gem.Data.ID != l && gem.Data.GameID != l {
				continue
			}
			found = true
			for _, ge := range gem.GrantedEffectList {
				if ge != nil {
					ids[ge.ID] = true
				}
			}
		}
		if !found {
			t.Errorf("gem %s is not in data.gems", l)
		}
	}
	return ids
}

// TestSkillsMatchPoBPoE2 grades every built effect against PoB-PoE2's loaded data.skills, path by path.
// Every effect agrees, those PoB writes by hand with no export template included. POE2_SKILL_IDS names a file of effect, gem or game ids to report apart (the ids a
// build corpus uses).
func TestSkillsMatchPoBPoE2(t *testing.T) {
	skills := testSkills(t)
	pob, count := pobRows(t)
	ours := map[string]map[string]string{}
	for _, id := range skills.IDs() {
		ge, _ := skills.Effect(id)
		ours[id] = goRows(ge)
	}
	if len(ours) != count {
		t.Errorf("built %d effects, PoB has %d", len(ours), count)
	}
	fams, bad, badIDs := diffRows(pob, ours, nil)
	logFamilies(t, fams)
	t.Logf("all: %d effects, %d disagree on some row; %d rows disagree", count, len(badIDs), len(bad))
	show := 0
	if n, err := strconv.Atoi(os.Getenv("SHOW")); err == nil {
		show = n
	}
	for i, b := range bad {
		if i == show {
			break
		}
		t.Log(b)
	}
	noTemplate := 0
	for id := range badIDs {
		ge, _ := skills.Effect(id)
		if ge == nil || !ge.Overlay.NoTemplate {
			t.Errorf("%s disagrees with PoB", id)
		} else {
			noTemplate++
		}
	}
	for _, b := range bad {
		id := b[:strings.IndexByte(b, ' ')]
		if ge, _ := skills.Effect(id); ge == nil || !ge.Overlay.NoTemplate {
			t.Error(b)
		}
	}
	const noTemplateCeiling = 0
	if noTemplate > noTemplateCeiling {
		t.Errorf("%d hand-written effects disagree, ceiling %d", noTemplate, noTemplateCeiling)
	}
	if f := os.Getenv("POE2_SKILL_IDS"); f != "" {
		only := readIDs(t, skills, f)
		_, sub, subIDs := diffRows(pob, ours, only)
		t.Logf("subset: %d ids, %d disagree on some row; %d rows disagree", len(only), len(subIDs), len(sub))
	}

	// The control: a changed number, a dropped row and an invented one must each be seen.
	id := "ArcPlayer"
	mut := map[string]map[string]string{id: {}}
	for k, v := range ours[id] {
		mut[id][k] = v
	}
	cast := `["castTime"]`
	if _, ok := pob[id][cast]; !ok {
		t.Fatalf("the control row %s %s is not in PoB's dump", id, cast)
	}
	mut[id][cast] = "1.2"
	delete(mut[id], `["statSets"][1]["stats"][1]`)
	mut[id][`["legacy"]`] = "true"
	if _, got, _ := diffRows(pob, mut, map[string]bool{id: true}); len(got) != 3 {
		t.Errorf("the must-mismatch control reports %d disagreements, want 3: %v", len(got), got)
	}
}
