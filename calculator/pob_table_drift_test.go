package calculator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestPoBTableDrift is TestPoBDataTableDrift for package `calculator`.
func TestPoBTableDrift(t *testing.T) {
	pobRoot := os.Getenv("POB_DIR")
	if pobRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("NOT-RUN: cannot resolve $HOME to find the pinned PoB: %v", err)
		}
		pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
	}
	if st, err := os.Stat(filepath.Join(pobRoot, "src")); err != nil || !st.IsDir() {
		t.Skipf("NOT-RUN: no pinned PoB source at %s (set POB_DIR). This has NOT told you the "+
			"tables are current — it has told you it could not look.", pobRoot)
	}

	graded := 0
	for _, tc := range gatedPoBTables {
		want, err := luaTablePairs(filepath.Join(pobRoot, "src", tc.luaFile), tc.luaTable)
		if err != nil {
			t.Errorf("REFUSING %s: %v", tc.goVar, err)
			continue
		}
		got, opaque, err := goMapPairs(filepath.Join("..", "calculator", tc.goFile), tc.goVar)
		if err != nil {
			t.Errorf("REFUSING %s: %v", tc.goVar, err)
			continue
		}
		for _, k := range opaque {
			if !slices.Contains(tc.ignoreKeys, k) {
				t.Errorf("REFUSING %s[%q]: value shape this comparator cannot flatten. Add the key "+
					"to the pair's ignoreKeys with a reason, or move the pair to unsupportedTables.",
					tc.goVar, k)
			}
		}
		if len(want) < tc.minKeys {
			t.Errorf("REFUSING %s: parsed only %d rows from %s:%s, expected at least %d — the "+
				"Lua shape moved and this row is no longer being checked",
				tc.goVar, len(want), tc.luaFile, tc.luaTable, tc.minKeys)
			continue
		}
		if len(got) == 0 {
			t.Errorf("REFUSING %s: parsed 0 rows from calculator/%s", tc.goVar, tc.goFile)
			continue
		}
		graded++

		for _, k := range tc.ignoreKeys {
			delete(want, k)
			delete(got, k)
		}

		var problems []string
		for _, k := range sortedStringKeys(want) {
			g, ok := got[k]
			if !ok {
				problems = append(problems, fmt.Sprintf("PIN-ONLY  %q = %q", k, want[k]))
			} else if g != want[k] {
				problems = append(problems, fmt.Sprintf("VALUE     %q: ours %q, pin %q", k, g, want[k]))
			}
		}
		for _, k := range sortedStringKeys(got) {
			if _, ok := want[k]; !ok {
				problems = append(problems, fmt.Sprintf("OURS-ONLY %q = %q", k, got[k]))
			}
		}
		if len(problems) > 0 {
			t.Errorf("%s drifts from %s:%s (pin %d rows, ours %d):\n  %s",
				tc.goVar, tc.luaFile, tc.luaTable, len(want), len(got), strings.Join(problems, "\n  "))
		}
	}

	for _, tc := range gatedPoBTables {
		if reason, ok := unsupportedTables[tc.goVar]; ok {
			t.Errorf("%s is GATED and also declared UNSUPPORTED (%q). Delete the declaration.",
				tc.goVar, reason)
		}
	}

	if graded == 0 {
		t.Fatal("REFUSING: not one table was graded; this test measured nothing")
	}
	t.Logf("graded %d of %d declared tables; %d more are declared UNSUPPORTED with a reason",
		graded, len(gatedPoBTables), len(unsupportedTables))
}

type pobTablePair struct {
	goFile, goVar     string
	luaFile, luaTable string
	minKeys           int
	ignoreKeys        []string
}

var gatedPoBTables = []pobTablePair{
	{"mod_parser.go", "suffixTypes", "Modules/ModParser.lua", "suffixTypes", 30, nil},
	{"mod_parser.go", "dmgTypes", "Modules/ModParser.lua", "dmgTypes", 5, nil},
	{"calc_perform.go", "cursePriority", "Modules/Data.lua", "cursePriority", 20, nil},
	{"mod_parser.go", "flagTypes", "Modules/ModParser.lua", "flagTypes", 30,
		[]string{"hexproof", `hindered,? with (%d+)%% reduced movement speed`,
			`hindered, with (\d+)\% reduced movement speed`}},
	{"mod_parser.go", "penTypes", "Modules/ModParser.lua", "penTypes", 6, nil},
}

var unsupportedTables = map[string]string{
	"flaskBuffs": "gated separately by TestFlaskBaseBuffsMatchThePin — the Lua side is nested " +
		"under itemBases[...].flask.buff rather than a flat table",
	"formList": "keys are Lua PATTERNS upstream and Go REGEXES here (`%d` vs `\\d`); a literal " +
		"key diff compares two different languages and would report every row as drifting",
	"regenTypes": "PoB BUILDS it at runtime (`appendMod(resourceTypes, \"Regen\")`, " +
		"ModParser.lua:5991) rather than declaring it, so there is no literal table to read",
	"conquerorList": "values are nested {id, type} tables and PoB writes `id` as a NUMBER where " +
		"ours is a string. Swept by hand 2026-08-16: CLEAN on all 28 rows, types and ids",
	"costDivisors": "PoB's src/Data/Costs.lua is a LIST of records keyed by index, not by " +
		"resource name. Swept by hand 2026-08-16: CLEAN, 14 rows",
	"isElemental": "declared inline as `{ Fire = true, ... }` with bare identifiers rather than " +
		"quoted keys. Swept by hand 2026-08-16: CLEAN, 3 rows",
	"socketLimits": "PoB stores socketLimit PER BASE in src/Data/Bases/*.lua, not per class. " +
		"Swept exhaustively by the operator 2026-08-16: CLEAN, all 18 classes uniform upstream",
	"weaponTypes": "compared against data.weaponTypeInfo (Data.lua:550) by the operator " +
		"2026-08-16: differs only by Fishing Rod / None upstream and Rune Dagger / Warstaff here",
	"modNameList": "the parser GRAMMAR, not a data transcription; thousands of rows, and the Go side deliberately restructures it",
	"modFlagList": "grammar, as modNameList",
	"preFlagList": "grammar, as modNameList",
	"modTagList":  "grammar, as modNameList",
	"specialModList": "grammar, as modNameList — and half its values are Go funcs with no literal " +
		"counterpart",
	"skillFuncs":                     "a registry of Go funcs keyed by granted-effect id; there is no Lua table to diff, only per-skill callbacks",
	"configurations":                 "a registry of Go apply-funcs; ConfigOptions.lua is a LIST of records",
	"itemTagSpecial":                 "go-native: no PoB counterpart (see the comment on the var)",
	"itemTagSpecialExclusionPattern": "go-native, as itemTagSpecial",
	"pobOnlyGrantedEffects":          "go-native shim for effects PoB declares in Data/Skills/*.lua as full skill records",
	"socketColourNames":              "go-native: PoB composes the multiplier name inline at Item.lua:1010-1015 rather than holding a table",
	"validRarities":                  "go-native: PoB tests rarity with a string comparison chain",
	"damageTypeDotKeywordFlags":      "go-native: PoB derives the keyword flag inline",
	"legionAttributeNodes":           "transcribes node IDS, not a named Lua table; gated by the timeless-jewel tests",
	"timelessJewelTypeByConqueror":   "derived from conquerorList, which is swept above",
	"itemClassTypes":                 "NOT YET SWEPT — src/Export/Scripts/bases.lua builds the class map procedurally",
	"itemClassSubTypes":              "NOT YET SWEPT — as itemClassTypes",
	"itemBaseAliases":                "NOT YET SWEPT — go-pob-data id aliases, declared per port_manifest",
	"itemGrantedSupportNames":        "NOT YET SWEPT — the Lua side is two lines of src/Data/Skills/other.lua",
	"skillFlagIDAliases":             "NOT YET SWEPT",
	"skillStatKeyAliases":            "NOT YET SWEPT",
	"unsupportedModList":             "go-native: lines this port deliberately refuses",
	"bowTransformUsingConditions":    "go-native helper set",
	"jewelOtherFuncs":                "registries of Go funcs; no literal Lua table",
	"jewelSelfFuncs":                 "registries of Go funcs; no literal Lua table",
	"jewelSelfUnallocFuncs":          "registries of Go funcs; no literal Lua table",
	"jewelThresholdFuncs":            "registries of Go funcs; no literal Lua table",
	"configIndexDefaults":            "go-native: default selections, not a PoB table",
}

var luaScalarRowRe = regexp.MustCompile(`\["([^"]*)"\]\s*=\s*("(?:[^"]*)"|-?[0-9.]+|true|false)`)

func luaTablePairs(path, name string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	src := string(raw)
	loc := regexp.MustCompile(`(?m)^(?:local |data\.)` + regexp.QuoteMeta(name) + `\s*=\s*\{`).FindStringIndex(src)
	if loc == nil {
		return nil, fmt.Errorf("no table %q in %s", name, path)
	}
	depth, j := 1, loc[1]
	for j < len(src) && depth > 0 {
		switch src[j] {
		case '{':
			depth++
		case '}':
			depth--
		}
		j++
	}
	body := regexp.MustCompile(`--[^\n]*`).ReplaceAllString(src[loc[1]:j-1], "")
	out := map[string]string{}
	for _, m := range luaScalarRowRe.FindAllStringSubmatch(body, -1) {
		out[m[1]] = strings.Trim(m[2], `"`)
	}
	return out, nil
}

func goMapPairs(path, name string) (map[string]string, []string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot parse %s: %w", path, err)
	}
	var lit *ast.CompositeLit
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || vs.Names[0].Name != name || len(vs.Values) != 1 {
			return true
		}
		if cl, ok := vs.Values[0].(*ast.CompositeLit); ok {
			lit = cl
		}
		return false
	})
	if lit == nil {
		return nil, nil, fmt.Errorf("no package-level map var %q in %s", name, path)
	}
	out := map[string]string{}
	var opaque []string
	for _, e := range lit.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			return nil, nil, fmt.Errorf("%s: unexpected non key/value element", name)
		}
		k, ok := kv.Key.(*ast.BasicLit)
		if !ok {
			return nil, nil, fmt.Errorf("%s: non-literal key", name)
		}
		key, err := strconv.Unquote(k.Value)
		if err != nil {
			key = strings.Trim(k.Value, "`\"")
		}
		switch v := kv.Value.(type) {
		case *ast.BasicLit:
			val, err := strconv.Unquote(v.Value)
			if err != nil {
				val = strings.Trim(v.Value, "`\"")
			}
			out[key] = val
		case *ast.CompositeLit:
			names, ok := singleModName(v)
			if !ok {
				opaque = append(opaque, key)
				continue
			}
			out[key] = names
		default:
			return nil, nil, fmt.Errorf("%s[%q]: unexpected value expression %T", name, key, kv.Value)
		}
	}
	return out, opaque, nil
}

func singleModName(cl *ast.CompositeLit) (string, bool) {
	if len(cl.Elts) != 1 {
		return "", false
	}
	kv, ok := cl.Elts[0].(*ast.KeyValueExpr)
	if !ok {
		return "", false
	}
	if id, ok := kv.Key.(*ast.Ident); !ok || id.Name != "names" {
		return "", false
	}
	list, ok := kv.Value.(*ast.CompositeLit)
	if !ok || len(list.Elts) != 1 {
		return "", false
	}
	lit, ok := list.Elts[0].(*ast.BasicLit)
	if !ok {
		return "", false
	}
	val, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return val, true
}

func sortedStringKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
