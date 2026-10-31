package poe2minions

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/storage"
)

// minionCanon is testdata/minions_dump.lua's canon: tables by sorted key, numbers first, %.14g numbers.
func minionCanon(v any, key string) string {
	switch x := v.(type) {
	case nil:
		return "nil"
	case float64:
		return fmt.Sprintf("%.14g", x)
	case int:
		return fmt.Sprintf("%.14g", float64(x))
	case string:
		if key == "skillType" {
			if id, ok := poe2.SkillType[x]; ok {
				return fmt.Sprintf("%.14g", float64(id))
			}
		}
		return `"` + x + `"`
	case bool:
		return strconv.FormatBool(x)
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = strconv.Itoa(i+1) + "=" + minionCanon(e, "")
		}
		return "{" + strings.Join(parts, ",") + "}"
	case []string:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = strconv.Itoa(i+1) + "=" + minionCanon(e, "")
		}
		return "{" + strings.Join(parts, ",") + "}"
	case map[string]bool:
		m := make(map[string]any, len(x))
		for k, b := range x {
			m[k] = b
		}
		return minionCanon(m, "")
	case map[string]any:
		var parts []string
		if tags, ok := x["tags"].([]any); ok && x["name"] != nil {
			for i, t := range tags {
				parts = append(parts, strconv.Itoa(i+1)+"="+minionCanon(t, ""))
			}
		}
		keys := make([]string, 0, len(x))
		for k := range x {
			if k == "tags" && x["name"] != nil {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if x[k] == nil {
				continue
			}
			parts = append(parts, k+"="+minionCanon(x[k], k))
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	return fmt.Sprintf("?%T", v)
}

func minionsPoB(t *testing.T) map[[2]string]string {
	t.Helper()
	if os.Getenv("OPERATOR") != "true" {
		t.Skip("SKIP: runs the PoB-PoE2 pin under luajit and reads go-pob-data; set OPERATOR=true")
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
	probe, err := filepath.Abs(filepath.Join("testdata", "minions_dump.lua"))
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
	rows := map[[2]string]string{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 1<<20), 1<<22)
	for sc.Scan() {
		f := strings.SplitN(sc.Text(), "\t", 3)
		if len(f) == 3 {
			rows[[2]string{f[0], f[1]}] = f[2]
		}
	}
	if len(rows) == 0 {
		t.Fatalf("the probe printed no rows\n%s", stderr.String())
	}
	return rows
}

func minionsOurs(t *testing.T, minions Minions) map[[2]string]string {
	t.Helper()
	rows := map[[2]string]string{}
	for key, m := range minions {
		rows[[2]string{key, "name"}] = m.Name
		rows[[2]string{key, "limit"}] = m.Limit
		rows[[2]string{key, "extraFlags"}] = minionCanon(m.ExtraFlags, "")
		if m.ExtraFlags == nil {
			rows[[2]string{key, "extraFlags"}] = "{}"
		}
		rows[[2]string{key, "skillList"}] = minionCanon(m.SkillList, "")
		rows[[2]string{key, "accuracy"}] = minionCanon(m.Accuracy, "")
		for i, mod := range m.ModList {
			rows[[2]string{key, "mod" + strconv.Itoa(i+1)}] = minionCanon(mod, "")
		}
		rows[[2]string{key, "mods"}] = strconv.Itoa(len(m.ModList))
	}
	return rows
}

// The pin exported its minion files from 0.5.5, so a field the live patch changed disagrees: these keys
// and fields are that patch drift, and the test fails once one agrees.
var minionPatchDrift = map[[2]string]bool{}

// TestMinionsMatchPoB compares data.minions built from go-pob-data and the templates with the pin's
// loaded data.minions, field by field.
func TestMinionsMatchPoB(t *testing.T) {
	want := minionsPoB(t)
	minions, err := Load(context.Background(), "0.5", assetcache.AssetLoaderWrapper{Storage: storage.Get()})
	if err != nil {
		t.Fatal(err)
	}
	got := minionsOurs(t, minions)
	keys := map[[2]string]bool{}
	for k := range want {
		keys[k] = true
	}
	for k := range got {
		keys[k] = true
	}
	sorted := make([][2]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Slice(sorted, func(a, b int) bool {
		if sorted[a][0] != sorted[b][0] {
			return sorted[a][0] < sorted[b][0]
		}
		return sorted[a][1] < sorted[b][1]
	})
	entries := map[string]bool{}
	bad := 0
	for _, k := range sorted {
		entries[k[0]] = true
		w, wok := want[k]
		g, gok := got[k]
		agree := wok && gok && w == g
		if minionPatchDrift[k] {
			if agree {
				t.Errorf("%s %s now agrees; drop it from minionPatchDrift", k[0], k[1])
			}
			continue
		}
		if !agree {
			bad++
			if bad <= 30 {
				t.Errorf("%s %s: pob %q (%v), ours %q (%v)", k[0], k[1], w, wok, g, gok)
			}
		}
	}
	if bad > 30 {
		t.Errorf("%d disagreements in all", bad)
	}
	if len(entries) < 600 {
		t.Fatalf("REFUSING: %d entries compared, expected over 600", len(entries))
	}
	t.Logf("%d entries, %d rows compared", len(entries), len(sorted))
}
