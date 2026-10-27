package poe2

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type overlayKey struct{ kind, key, field string }

func overlayCanon(v any) string {
	switch x := v.(type) {
	case nil:
		return "nil"
	case float64:
		return fmt.Sprintf("%.17g", x)
	case string:
		return `"` + x + `"`
	case bool:
		return strconv.FormatBool(x)
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = "[" + strconv.Itoa(i+1) + "]=" + overlayCanon(e)
		}
		return "{" + strings.Join(parts, ",") + "}"
	case map[string]any:
		if n, ok := x["$num"]; ok && len(x) == 1 {
			return n.(string)
		}
		if _, isMod := x["tags"]; isMod && x["name"] != nil {
			// a nested mod: PoB's table holds its tags at [1..n] beside the named fields
			parts := []string{}
			for i, t := range x["tags"].([]any) {
				parts = append(parts, "["+strconv.Itoa(i+1)+"]="+overlayCanon(t))
			}
			for _, k := range []string{"flags", "keywordFlags", "name", "type", "value"} {
				if x[k] != nil {
					parts = append(parts, k+"="+overlayCanon(x[k]))
				}
			}
			return "{" + strings.Join(parts, ",") + "}"
		}
		var nums []int
		var strs []string
		for k := range x {
			if n, err := strconv.Atoi(k); err == nil {
				nums = append(nums, n)
			} else {
				strs = append(strs, k)
			}
		}
		sort.Ints(nums)
		sort.Strings(strs)
		parts := []string{}
		for _, n := range nums {
			parts = append(parts, "["+strconv.Itoa(n)+"]="+overlayCanon(x[strconv.Itoa(n)]))
		}
		for _, k := range strs {
			parts = append(parts, k+"="+overlayCanon(x[k]))
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	panic(fmt.Sprintf("unexpected JSON value %T", v))
}

func overlayCanonMod(s string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		panic(err)
	}
	rawTags := m["tags"].([]any)
	tags := make([]string, 0, len(rawTags))
	for _, t := range rawTags {
		tags = append(tags, overlayCanon(t))
	}
	return m["name"].(string) + "|" + overlayCanon(m["type"]) + "|" + overlayCanon(m["value"]) + "|" + overlayCanon(m["flags"]) + "|" +
		overlayCanon(m["keywordFlags"]) + "|" + strings.Join(tags, ";")
}

func overlayScalars(value, mult, div, base float64, hasValue, hasMult, hasDiv, hasBase bool) string {
	f := func(v float64, has bool) string {
		if !has {
			return "nil"
		}
		return fmt.Sprintf("%.17g", v)
	}
	return "value=" + f(value, hasValue) + ",mult=" + f(mult, hasMult) + ",div=" + f(div, hasDiv) + ",base=" + f(base, hasBase)
}

func overlayCanonEntry(e StatMapEntry) string {
	items := make([]string, len(e.Items))
	for i, it := range e.Items {
		if it.Group == nil {
			items[i] = overlayCanonMod(it.Mod)
			continue
		}
		mods := make([]string, len(it.Group))
		for j, m := range it.Group {
			mods[j] = overlayCanonMod(m)
		}
		items[i] = "group(" + overlayScalars(it.Value, it.Mult, it.Div, it.Base, it.HasValue, it.HasMult, it.HasDiv, it.HasBase) + ")[" + strings.Join(mods, " ; ") + "]"
	}
	flag := "nil"
	if e.SkillFlag != "" {
		flag = `"` + e.SkillFlag + `"`
	}
	return overlayScalars(e.Value, e.Mult, e.Div, e.Base, e.HasValue, e.HasMult, e.HasDiv, e.HasBase) + ",skillFlag=" + flag + "|" + strings.Join(items, " ; ")
}

func overlayGoSide() (map[overlayKey]string, map[string]int) {
	m := map[overlayKey]string{}
	counts := map[string]int{}
	for enum, vals := range map[string]map[string]uint64{"ModFlag": ModFlag, "KeywordFlag": KeywordFlag} {
		for name, v := range vals {
			m[overlayKey{"enum", enum, name}] = fmt.Sprintf("%.17g", float64(v))
		}
		counts["enum:"+enum] = len(vals)
	}
	for name, v := range SkillType {
		m[overlayKey{"enum", "SkillType", name}] = strconv.Itoa(v)
	}
	counts["enum:SkillType"] = len(SkillType)
	for _, e := range SkillStatMap {
		m[overlayKey{"statmap", e.Stat, "entry"}] = overlayCanonEntry(e)
	}
	counts["statmap"] = len(SkillStatMap)
	for _, s := range SkillOverlays {
		for f, v := range map[string]bool{"hidden": s.Hidden, "fromItem": s.FromItem, "fromTree": s.FromTree, "hideFromSideBar": s.HideFromSideBar,
			"grantsThornsDamage": s.GrantsThornsDamage, "legacy": s.Legacy, "minionHasItemSet": s.MinionHasItemSet, "preDamageFunc": s.PreDamageFunc} {
			m[overlayKey{"skill", s.ID, f}] = strconv.FormatBool(v)
		}
		m[overlayKey{"skill", s.ID, "minionList"}] = strings.Join(s.MinionList, ",")
		m[overlayKey{"skill", s.ID, "addFlags"}] = strings.Join(s.AddFlags, ",")
		m[overlayKey{"skill", s.ID, "minionUses"}] = strings.Join(s.MinionUses, ",")
		m[overlayKey{"skill", s.ID, "sets"}] = strconv.Itoa(len(s.StatSets))
		for i, set := range s.StatSets {
			key := s.ID + "#" + strconv.Itoa(i+1)
			flags := "nil"
			if set.HasBaseFlags {
				flags = strings.Join(set.BaseFlags, ",")
			}
			m[overlayKey{"set", key, "baseFlags"}] = flags
			mods := "nil"
			if set.BaseMods != nil {
				canon := make([]string, len(set.BaseMods))
				for j, bm := range set.BaseMods {
					canon[j] = overlayCanonMod(bm)
				}
				mods = strings.Join(canon, " ; ")
			}
			m[overlayKey{"set", key, "baseMods"}] = mods
			stats := make([]string, len(set.StatMap))
			for j, e := range set.StatMap {
				stats[j] = e.Stat
				m[overlayKey{"set", key, "stat:" + e.Stat}] = overlayCanonEntry(e)
			}
			m[overlayKey{"set", key, "stats"}] = strings.Join(stats, ",")
		}
	}
	counts["skill"] = len(SkillOverlays)
	return m, counts
}

func overlayDiff(pob map[overlayKey]string, pobCounts map[string]int, ours map[overlayKey]string, ourCounts map[string]int) []string {
	var bad []string
	for k, v := range pob {
		if o, ok := ours[k]; !ok || o != v {
			bad = append(bad, fmt.Sprintf("%s %s %s: PoB %s, ours %s (present %v)", k.kind, k.key, k.field, v, o, ok))
		}
	}
	for k, v := range ours {
		if _, ok := pob[k]; !ok {
			bad = append(bad, fmt.Sprintf("%s %s %s: PoB absent, ours %s", k.kind, k.key, k.field, v))
		}
	}
	for kind, n := range pobCounts {
		if ourCounts[kind] != n {
			bad = append(bad, fmt.Sprintf("%s rows: PoB %d, ours %d", kind, n, ourCounts[kind]))
		}
	}
	if len(pobCounts) != len(ourCounts) {
		bad = append(bad, fmt.Sprintf("count kinds: PoB %v, ours %v", pobCounts, ourCounts))
	}
	sort.Strings(bad)
	return bad
}

func overlayPoB(t *testing.T) (map[overlayKey]string, map[string]int) {
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
	probe, err := filepath.Abs(filepath.Join("testdata", "skill_overlay_check.lua"))
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
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("the probe failed: %v\n%s", err, stderr.String())
	}
	rows, counts := map[overlayKey]string{}, map[string]int{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 1<<20), 1<<22)
	for sc.Scan() {
		f := strings.SplitN(sc.Text(), "\t", 4)
		if len(f) != 4 {
			continue
		}
		v := strings.ReplaceAll(f[3], `\n`, "\n")
		switch f[0] {
		case "count":
			n, err := strconv.Atoi(v)
			if err != nil {
				t.Fatalf("bad count row %q", sc.Text())
			}
			counts[f[1]] = n
		case "enum", "statmap", "skill", "set":
			rows[overlayKey{f[0], f[1], f[2]}] = v
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(counts) != 5 || len(rows) == 0 {
		t.Fatalf("the probe printed %d count rows and %d value rows; it stopped early\n%s", len(counts), len(rows), stderr.String())
	}
	return rows, counts
}

// TestSkillOverlayMatchesPoBPoE2 grades every generated skill overlay row, global stat map entry and enum
// value against PoB-PoE2's own loaded data.skills, data.skillStatMap and Global.lua.
func TestSkillOverlayMatchesPoBPoE2(t *testing.T) {
	pob, pobCounts := overlayPoB(t)
	ours, ourCounts := overlayGoSide()
	bad := overlayDiff(pob, pobCounts, ours, ourCounts)
	t.Logf("%d PoB rows, %d disagreements; counts %v", len(pob), len(bad), pobCounts)
	for i, b := range bad {
		if i == 40 {
			t.Errorf("... and %d more", len(bad)-40)
			break
		}
		t.Error(b)
	}

	// The control: the same differential must see a changed divisor, a dropped stat map entry and an
	// invented flag.
	mut := map[overlayKey]string{}
	for k, v := range ours {
		mut[k] = v
	}
	mutCounts := map[string]int{}
	for k, v := range ourCounts {
		mutCounts[k] = v
	}
	dur := overlayKey{"statmap", "base_skill_effect_duration", "entry"}
	if _, ok := pob[dur]; !ok {
		t.Fatalf("the control row %v is not in PoB's output", dur)
	}
	mut[dur] = strings.Replace(mut[dur], "div=1000", "div=100", 1)
	delete(mut, overlayKey{"statmap", "base_secondary_skill_effect_duration", "entry"})
	mut[overlayKey{"skill", "ArcPlayer", "legacy"}] = "true"
	if got := overlayDiff(pob, pobCounts, mut, mutCounts); len(got) != 3 {
		t.Errorf("the must-mismatch control reports %d disagreements, want 3: %v", len(got), got)
	}
}
