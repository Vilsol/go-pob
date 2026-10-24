package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const ldCanonPy = `
import json, sys
sys.path.insert(0, "scripts")
import lua_data as L

def canon(v):
    if isinstance(v, L.Table):
        keys = sorted(v.hash.items(), key=lambda kv: (type(kv[0]).__name__, str(kv[0])))
        return {"t": [canon(x) for x in v.array],
                "h": [[canon(k), canon(w)] for k, w in keys]}
    if isinstance(v, L.Expr):
        return {"e": v.text, "c": v.callee,
                "a": None if v.args is None else [canon(a) for a in v.args]}
    if isinstance(v, bool):
        return {"b": v}
    if isinstance(v, int):
        return {"i": v}
    if isinstance(v, float):
        return {"f": repr(v)}
    if isinstance(v, str):
        return {"s": v}
    if v is None:
        return {"z": True}
    raise SystemExit("canon: unknown type " + type(v).__name__)

def dump(v):
    return json.dumps(canon(v), sort_keys=True, separators=(",", ":"), ensure_ascii=False)

mode, arg = sys.argv[1], sys.argv[2]
try:
    if mode == "value":
        v, end = L.parse_value(arg)
        print(dump(v)); print(end)
    elif mode.startswith("scantext:"):
        for key, tbl, line in L.scan_assignments(arg, mode.split(":", 1)[1]):
            print("%s\t%d\t%s" % (key, line, dump(tbl)))
    elif mode == "return-table":
        print(dump(L.parse_return_table(open(arg, encoding="utf-8").read())))
    else:
        text = open(arg, encoding="utf-8").read()
        for key, tbl, line in L.scan_assignments(text, mode):
            print("%s\t%d\t%s" % (key, line, dump(tbl)))
except L.LuaParseError as exc:
    print("LuaParseError: %s" % exc)
`

func ldCanonGo(v any) any {
	switch t := v.(type) {
	case *ldTable:
		arr := make([]any, 0, len(t.array))
		for _, x := range t.array {
			arr = append(arr, ldCanonGo(x))
		}
		pairs := make([]any, 0, len(t.hash))
		for _, k := range ldCanonSortedKeys(t.hash) {
			pairs = append(pairs, []any{ldCanonGo(k), ldCanonGo(t.hash[k])})
		}
		return map[string]any{"t": arr, "h": pairs}
	case *ldExpr:
		out := map[string]any{"e": t.text, "c": nil, "a": nil}
		if t.isCall {
			out["c"] = t.callee
			args := []any{}
			for _, a := range t.args {
				args = append(args, ldCanonGo(a))
			}
			out["a"] = args
		}
		return out
	case bool:
		return map[string]any{"b": t}
	case int:
		return map[string]any{"i": t}
	case float64:
		return map[string]any{"f": pyFloat(t)}
	case string:
		return map[string]any{"s": t}
	case nil:
		return map[string]any{"z": true}
	}
	return map[string]any{"?": "unknown"}
}

func ldCanonSortedKeys(m map[any]any) []any {
	type row struct {
		typ, str string
		key      any
	}
	rows := make([]row, 0, len(m))
	for k := range m {
		switch t := k.(type) {
		case int:
			rows = append(rows, row{"int", intToStr(t), k})
		case string:
			rows = append(rows, row{"str", t, k})
		case bool:
			s := "False"
			if t {
				s = "True"
			}
			rows = append(rows, row{"bool", s, k})
		case float64:
			rows = append(rows, row{"float", pyFloat(t), k})
		default:
			rows = append(rows, row{"other", "", k})
		}
	}
	sortRows(rows, func(a, b row) bool {
		if a.typ != b.typ {
			return a.typ < b.typ
		}
		return a.str < b.str
	})
	out := make([]any, len(rows))
	for i, r := range rows {
		out[i] = r.key
	}
	return out
}

func ldDumpGo(v any) string {
	var b strings.Builder
	ldEncode(ldCanonGo(v), &b)
	return b.String()
}

func ldEncode(v any, b *strings.Builder) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sortStringsAsc(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			ldEncodeString(k, b)
			b.WriteByte(':')
			ldEncode(t[k], b)
		}
		b.WriteByte('}')
	case []any:
		b.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			ldEncode(e, b)
		}
		b.WriteByte(']')
	case string:
		ldEncodeString(t, b)
	case bool:
		if t {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case int:
		b.WriteString(intToStr(t))
	case nil:
		b.WriteString("null")
	}
}

func ldEncodeString(s string, b *strings.Builder) {
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		b.WriteString(`""`)
		return
	}
	b.WriteString(strings.TrimSuffix(buf.String(), "\n"))
}

const ldFrozenFile = "testdata/luadata-cpython.json"

const ldRecordEnv = "LUADATA_RECORD"

var (
	ldFrozenOnce sync.Once
	ldFrozenMap  map[string]string
	errLdFrozen  error
	ldRecordMu   sync.Mutex
	ldRecorded   = map[string]string{}
)

func ldFrozen(t *testing.T, repo string) map[string]string {
	t.Helper()
	ldFrozenOnce.Do(func() {
		b, err := os.ReadFile(filepath.Join(repo, "scripts", "gopob", ldFrozenFile))
		if err != nil {
			errLdFrozen = err
			return
		}
		errLdFrozen = json.Unmarshal(b, &ldFrozenMap)
	})
	if errLdFrozen != nil && !ldRecording() {
		t.Fatalf("the frozen CPython answers are unreadable: %v\n"+
			"  Record them with %s=1 while scripts/lua_data.py still exists; they cannot be\n"+
			"  reconstructed from the Go parser, which would compare it against itself.",
			errLdFrozen, ldRecordEnv)
	}
	return ldFrozenMap
}

func ldRecording() bool { return os.Getenv(ldRecordEnv) != "" }

func ldRunPy(t *testing.T, repo, mode, arg string) string {
	t.Helper()
	key := mode + "\x00" + arg
	frozen, haveFrozen := ldFrozen(t, repo)[key]

	if difftest.OriginalsDeleted(repo) {
		if !haveFrozen {
			t.Fatalf("no frozen CPython answer for %s, and scripts/lua_data.py is gone — this case "+
				"was added after the freeze and has no oracle. Assert it against a hand-derived "+
				"expectation or delete it; do not compare the parser against itself.", mode)
		}
		return frozen
	}

	cmd := exec.CommandContext(t.Context(), "python3", "-c", ldCanonPy, mode, arg)
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python3 (%s %s): %v", mode, arg, err)
	}
	if ldRecording() {
		ldRecordMu.Lock()
		ldRecorded[key] = string(out)
		ldRecordMu.Unlock()
		return string(out)
	}
	if haveFrozen && frozen != string(out) {
		t.Fatalf("the frozen CPython answer for %s has drifted from scripts/lua_data.py\n"+
			"--- frozen ---\n%s\n--- now ---\n%s\n  Re-record with %s=1 in a commit that does "+
			"nothing else.", mode, frozen, out, ldRecordEnv)
	}
	return string(out)
}

var ldCases = []string{
	`{ }`,
	`{ 1, 2, 3 }`,
	`{ a = 1, b = 2 }`,
	`{ a = 1, a = 2 }`,
	`{ [1] = { 20, 20 }, [2] = { 21 } }`,
	`{ ["quoted key"] = true }`,
	`{ x = 1.5, y = -2, z = 1e3, w = 2.0 }`,
	`{ t = true, f = false, n = nil }`,
	`{ s = "plain", e = "esc\ttab\nnl", q = "say \"hi\"", a = 'single' }`,
	`{ long = [[a long
string]] }`,
	`{ -- a line comment
	  a = 1, --[[ a block comment ]] b = 2 }`,
	`{ call = mod("Damage", "BASE", 1) }`,
	`{ nested = mod("A", "LIST", { mod("B", "BASE", 2) }) }`,
	`{ ident = SkillType.Spell }`,
	`{ arith = 1 / 3 }`,
	`{ neg = -0.5 }`,
	`{ alias = skills.ExplosiveTrap.parts }`,
	`{ f = function(a) if a then return { 1 } end end, parts = { { name = "P" } } }`,
	`{ f = function(a) do return 1 end end, after = 2 }`,
	`{ cmp = function(a) if a == 1 then return 2 end end, after = 3 }`,
	`{ colon = obj:method(1) }`,
	`{ deep = { { { 1 } } } }`,
	`{ trailing = 1, }`,
	`{ mixed = { 1, a = 2, 3 } }`,
	"{ [\"kéy\"] : 1 }",
	"{ [\"plain\"] : 1 }",
	`{ unterminated = "abc`,
	`{ unterminated`,
	`{ f = function(a) return 1 }`,
	`{ , }`,
	`[[unterminated long`,
	``,
}

func TestLuaDataMatchesCPython(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	for _, src := range ldCases {
		t.Run(ldCaseName(src), func(t *testing.T) {
			want := ldRunPy(t, repo, "value", src)
			v, end, err := ldParseValue(src, 0)
			var got string
			if err != nil {
				got = "LuaParseError: " + err.Error() + "\n"
			} else {
				got = ldDumpGo(v) + "\n" + intToStr(end) + "\n"
			}
			if got != want {
				t.Fatalf("source %q\n--- python ---\n%s\n--- go ---\n%s", src, want, got)
			}
		})
	}
}

var ldScanCases = []struct{ name, prefix, text string }{
	{"two-plain-entries", "skills",
		"\nskills[\"A\"] = { x = 1 }\nskills[\"B\"] = { y = 2 }\n"},
	{"unterminated-table-not-the-invariant", "skills",
		"\nskills[\"A\"] = { x = { \n\nskills[\"B\"] = { y = 1 }\n"},
	{"long-string-runs-past-the-next", "skills",
		"\nskills[\"A\"] = { x = [[\nskills[\"B\"] = { y = 1 }\n]] }\nskills[\"C\"] = { z = 3 }\n"},
	{"block-comment-runs-past-the-next", "skills",
		"\nskills[\"A\"] = { x = 1, --[[\nskills[\"B\"] = { y = 1 }\n]] z = 2 }\nskills[\"C\"] = { z = 3 }\n"},
	{"function-body-does-not-truncate", "skills",
		"\nskills[\"A\"] = {\n f = function(a) if a then return {1} end end,\n parts = { { name = \"P\" } },\n}\n"},
	{"not-a-table", "skills", "\nskills[\"A\"] = 7\n"},
	{"no-anchor-at-column-0", "skills", "  skills[\"A\"] = { x = 1 }\n"},
	{"bracket-then-no-equals", "skills", "\nskills[\"A\"] : { x = 1 }\n"},
	{"nonascii-before-the-entry", "skills",
		"-- héllo\nskills[\"A\"] = { x = 1 }\nskills[\"B\"] = { y = 2 }\n"},
}

func TestLuaDataScanAssignmentsSynthetic(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	for _, tc := range ldScanCases {
		t.Run(tc.name, func(t *testing.T) {
			want := ldRunPy(t, repo, "scantext:"+tc.prefix, tc.text)
			rows, err := ldScanAssignments(tc.text, tc.prefix)
			var got strings.Builder
			if err != nil {
				got.WriteString("LuaParseError: " + err.Error() + "\n")
			} else {
				for _, r := range rows {
					key, _ := r.key.(string)
					got.WriteString(key + "\t" + intToStr(r.line) + "\t" + ldDumpGo(r.table) + "\n")
				}
			}
			if got.String() != want {
				t.Fatalf("%s\n--- python ---\n%s\n--- go ---\n%s", tc.name, want, got.String())
			}
		})
	}
}

// The pinned files never had a frozen CPython answer; these expectations are read off the pin with grep, so a pin
// bump that moves them is re-derived there, never from this parser.
func TestLuaDataScanAssignmentsMatchesCPython(t *testing.T) {
	t.Parallel()
	pob := ldPinDir(t)
	for _, tc := range []struct {
		rel, prefix         string
		rows                int
		firstKey, firstName string
		firstLine           int
		lastKey             string
		lastLine            int
	}{
		{"src/Data/Minions.lua", "minions", 64, "RaisedZombie", "Raised Zombie", 11, "ShamblingUndead", 1927},
		{"src/Data/Spectres.lua", "minions", 268, "Metadata/Monsters/Axis/AxisCaster", "Blackguard Mage", 12,
			"Metadata/Monsters/VaalMonsters/VaalOverseer", 7207},
		{"src/Data/Skills/act_dex.lua", "skills", 198, "AlchemistsMark", "Alchemist's Mark", 8, "QuickstepHardMode", 18446},
		{"src/Data/Skills/act_int.lua", "skills", 229, "Arc", "Arc", 8, "Zealotry", 21403},
		{"src/Data/Skills/act_str.lua", "skills", 142, "Absolution", "Absolution", 8, "WarBanner", 13088},
		{"src/Data/Skills/sup_dex.lua", "skills", 82, "SupportAddedColdDamage", "Added Cold Damage", 8,
			"SupportWitheringTouch", 4991},
		{"src/Data/Skills/other.lua", "skills", 166, "RepeatingShockwave", "Abberath's Fury", 8, "EnemyExplode", 6187},
	} {
		t.Run(filepath.Base(tc.rel), func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(pob, tc.rel))
			if err != nil {
				t.Fatalf("%s: %v", tc.rel, err)
			}
			rows, err := ldScanAssignments(string(body), tc.prefix)
			if err != nil {
				t.Fatalf("%s: %v", tc.rel, err)
			}
			if len(rows) != tc.rows {
				t.Fatalf("%s: %d rows, the pin has %d", tc.rel, len(rows), tc.rows)
			}
			first, last := rows[0], rows[len(rows)-1]
			if first.key != tc.firstKey || first.line != tc.firstLine || last.key != tc.lastKey || last.line != tc.lastLine {
				t.Fatalf("%s: rows run %v@%d .. %v@%d, the pin has %s@%d .. %s@%d", tc.rel,
					first.key, first.line, last.key, last.line, tc.firstKey, tc.firstLine, tc.lastKey, tc.lastLine)
			}
			if name := first.table.hash["name"]; name != tc.firstName {
				t.Fatalf("%s: %s.name = %v, the pin has %q", tc.rel, tc.firstKey, name, tc.firstName)
			}
		})
	}
}

func TestLuaDataReturnTableMatchesCPython(t *testing.T) {
	t.Parallel()
	pob := ldPinDir(t)
	for _, tc := range []struct {
		rel         string
		keys        int
		first, last string
		alsoHas     []string
	}{
		{"src/Data/Gems.lua", 822, "Metadata/Items/Gems/SkillGemFireball", "Metadata/Items/Gems/SkillGemPactOfLycia", nil},
		// 716 entries, two keys assigned twice: a Lua table keeps 714.
		{"src/Data/SkillStatMap.lua", 714, "base_skill_effect_duration", "pact_empower_limitation_specifier_for_stat_description",
			[]string{"mine_throwing_speed_+%_per_frenzy_charge", "triggered_by_kinetic_instability_support"}},
	} {
		t.Run(filepath.Base(tc.rel), func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(pob, tc.rel))
			if err != nil {
				t.Fatalf("%s: %v", tc.rel, err)
			}
			tbl, err := ldParseReturnTable(string(body))
			if err != nil {
				t.Fatalf("%s: %v", tc.rel, err)
			}
			keys := tbl.keys()
			if len(keys) != tc.keys || len(tbl.array) != 0 {
				t.Fatalf("%s: %d keys and %d array items, the pin has %d keys", tc.rel, len(keys), len(tbl.array), tc.keys)
			}
			if keys[0] != tc.first || keys[len(keys)-1] != tc.last {
				t.Fatalf("%s: keys run %v .. %v, the pin has %s .. %s", tc.rel, keys[0], keys[len(keys)-1], tc.first, tc.last)
			}
			for _, k := range tc.alsoHas {
				if _, ok := tbl.hash[k]; !ok {
					t.Fatalf("%s: no key %s", tc.rel, k)
				}
			}
		})
	}
}

func ldCompareLines(t *testing.T, what, want, got string) {
	t.Helper()
	if want == got {
		return
	}
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wl) || i < len(gl); i++ {
		var a, b string
		if i < len(wl) {
			a = wl[i]
		}
		if i < len(gl) {
			b = gl[i]
		}
		if a == b {
			continue
		}
		ar, br := []rune(a), []rune(b)
		j := 0
		for j < len(ar) && j < len(br) && ar[j] == br[j] {
			j++
		}
		lo := j - 60
		if lo < 0 {
			lo = 0
		}
		t.Fatalf("%s: line %d of %d differs at rune %d\n--- python ---\n%s\n--- go ---\n%s",
			what, i+1, len(wl), j, ldWindow(ar, lo, j+120), ldWindow(br, lo, j+120))
	}
	t.Fatalf("%s: outputs differ but no line does (python %d lines, go %d)", what, len(wl), len(gl))
}

func ldWindow(r []rune, lo, hi int) string {
	if lo > len(r) {
		lo = len(r)
	}
	if hi > len(r) {
		hi = len(r)
	}
	return string(r[lo:hi])
}

func ldCaseName(src string) string {
	s := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\t' || r == '/' {
			return '_'
		}
		return r
	}, src)
	if len(s) > 40 {
		s = s[:40]
	}
	if s == "" {
		return "empty"
	}
	return s
}

func ldPinDir(t *testing.T) string {
	t.Helper()
	p := pobPinDir()
	if _, err := os.Stat(filepath.Join(p, "src")); err != nil {
		t.Skipf("no pinned PoB at %s", p)
	}
	return p
}

func intToStr(n int) string { return strconv.Itoa(n) }

func sortStringsAsc(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func sortRows[T any](rows []T, less func(a, b T) bool) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && less(rows[j], rows[j-1]); j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
}

// TestLuaDataFreezeIsComplete writes the frozen answers under LUADATA_RECORD=1 and otherwise asserts that every synthetic case has one.
func TestLuaDataFreezeIsComplete(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	if ldRecording() {
		ldRecordMu.Lock()
		defer ldRecordMu.Unlock()
		want := map[string]string{}
		for _, src := range ldCases {
			want["value\x00"+src] = ""
		}
		for _, tc := range ldScanCases {
			want["scantext:"+tc.prefix+"\x00"+tc.text] = ""
		}
		out := map[string]string{}
		for k := range want {
			v, ok := ldRecorded[k]
			if !ok {
				t.Fatalf("case %q produced no answer — this test ran before the cases did; it must "+
					"be the LAST test in the file", k)
			}
			out[k] = v
		}
		ldRecorded = out
		b, err := json.MarshalIndent(ldRecorded, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, "scripts", "gopob", ldFrozenFile),
			append(b, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Logf("recorded %d CPython answers", len(ldRecorded))
		return
	}
	frozen := ldFrozen(t, repo)
	missing := 0
	for _, src := range ldCases {
		if _, ok := frozen["value\x00"+src]; !ok {
			t.Errorf("no frozen CPython answer for value case %q", src)
			missing++
		}
	}
	for _, tc := range ldScanCases {
		if _, ok := frozen["scantext:"+tc.prefix+"\x00"+tc.text]; !ok {
			t.Errorf("no frozen CPython answer for scan case %q", tc.name)
			missing++
		}
	}
	if missing > 0 {
		t.Logf("re-record with %s=1 while scripts/lua_data.py still exists", ldRecordEnv)
	}
}
