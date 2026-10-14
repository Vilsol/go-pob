package mod

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var scalingTagCtors = []string{"mod.Multiplier(", "mod.PerStat(", "mod.PercentStat("}

var listPayloadProducers = []string{"../calculator/*.go", "../data/*.go", "../data/raw/*.go", "../moddb/*.go"}

// TestListPayloadsImplementScalableValue MEASURES, rather than declares, which LIST payloads a value-scaling tag can reach.
func TestListPayloadsImplementScalableValue(t *testing.T) {
	implements := map[string]string{
		"GemProperty":    "Poet's Pen: +1 Level per 25 Player Levels (Multiplier)",
		"SkillData":      "ModStore.lua's `value.value` branch",
		"ExtraAura":      "Master of Metal: added damage per Impale (Multiplier, actor=enemy)",
		"MinionModifier": "Triad Grip: conversion per socket colour (Multiplier)",
	}

	declared := payloadTypesInListTypes(t)

	reachable, sites := scanOuterScalingTags(t)
	for _, name := range sortedKeys(reachable) {
		if _, ok := implements[name]; ok {
			continue
		}
		t.Errorf("payload %s is reached by a value-scaling tag on the OUTER LIST mod but does "+
			"NOT implement ScalableValue — the tag is evaluated and its result DISCARDED.\n"+
			"Producer site(s):\n\t%s\n"+
			"Implement ScaleValue on it (see MinionModifier for the `value.mod` shape and "+
			"SkillData for the `value.value` shape) and add it to `implements` here.",
			name, strings.Join(sites[name], "\n\t"))
	}

	live := payloadInstances(t)
	for name, why := range implements {
		if !declared[name] {
			t.Errorf("%s is declared to implement ScalableValue but is not a type in list_types.go", name)
			continue
		}
		v, ok := live[name]
		if !ok {
			t.Errorf("%s is declared to implement ScalableValue but has no instance here", name)
			continue
		}
		if _, ok := v.(ScalableValue); !ok {
			t.Errorf("%s NO LONGER implements ScalableValue. Its value-scaling tags are now "+
				"silently discarded. Population it was implemented for: %s", name, why)
		}
	}
}

// TestNestedModPayloadsImplementScalableNestedMod is the SECOND operation a nested-mod payload has to answer for.
func TestNestedModPayloadsImplementScalableNestedMod(t *testing.T) {
	live := payloadInstances(t)

	nested := nestedModPayloadTypes(t)
	if len(nested) < 5 {
		t.Fatalf("found only %d payloads carrying a nested Mod (%v) — expected the whole "+
			"EnemyModifier / ExtraAura / MinionModifier family; the parse has probably broken "+
			"and this test would pass vacuously", len(nested), nested)
	}

	for _, name := range nested {
		if _, ok := live[name].(ScalableNestedMod); !ok {
			t.Errorf("payload %s carries a nested Mod but does NOT implement ScalableNestedMod "+
				"— ModStore.lua:66's `scaledMod.value.mod` branch is a duck-typed field probe, "+
				"so ScaleAddList must scale it. Without the method moddb/mod_list.go's "+
				"scaleModClone copies the payload through and the nested value keeps its "+
				"UNSCALED number, which every census in this repo reads as a match.\n"+
				"Implement it as a one-liner over scaleNestedModValue, the way EnemyModifier "+
				"does.", name)
		}
	}
}

const modFieldType = "Mod"

func nestedModPayloadTypes(t *testing.T) []string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "list_types.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	out := make([]string, 0, 16)
	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, f := range st.Fields.List {
			if id, ok := f.Type.(*ast.Ident); ok && id.Name == modFieldType {
				out = append(out, ts.Name.Name)
				return true
			}
		}
		return true
	})
	sort.Strings(out)
	return out
}

func payloadInstances(t *testing.T) map[string]Cloneable {
	t.Helper()

	instances := []Cloneable{
		AffectedByAuraMod{}, AffectedByCurseMod{}, ArmourData{}, ConquerorType{},
		EnemyModifier{}, ExtraAura{}, ExtraAuraEffect{}, ExtraCurse{}, ExtraMinionSkill{},
		ExtraSkill{}, ExtraSkillMod{}, ExtraSkillStat{}, ExtraSupport{}, GemProperty{},
		GrantReservedPoolAsAura{}, GrantedAscendancyNode{}, GroupProperty{}, ImpossibleEscapeKeystones{},
		JewelData{}, LegionJewel{}, LinkedSupport{}, MinionModifier{}, PactDamage{},
		SelfDamage{}, SkillData{}, SocketProperty{}, SupportedGemProperty{},
		WeaponData{},
	}

	live := make(map[string]Cloneable, len(instances))
	for _, v := range instances {
		live[reflect.TypeOf(v).Name()] = v
	}

	for name := range payloadTypesInListTypes(t) {
		if _, ok := live[name]; !ok {
			t.Errorf("list_types.go declares payload %s but this registry has no instance of "+
				"it, so NEITHER ScalableValue NOR ScalableNestedMod is checked on it. Add "+
				"%s{} to `instances`.", name, name)
		}
	}
	return live
}

// TestParseModOnlyMovesTagsOntoExtraAura pins the ONE producer path the static scan above cannot see.
func TestParseModOnlyMovesTagsOntoExtraAura(t *testing.T) {
	src, err := os.ReadFile("../calculator/mod_parser.go")
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(string(src), "\n")
	movers := make([]string, 0, 1)
	for i, ln := range lines {
		if !strings.Contains(ln, "ClearTags()") {
			continue
		}
		window := strings.Join(lines[i:min(i+4, len(lines))], "\n")
		m := regexp.MustCompile(`mod\.NewList\("(\w+)"`).FindStringSubmatch(window)
		if m == nil {
			t.Errorf("mod_parser.go:%d clears a mod's tags but no LIST rewrap follows within "+
				"3 lines — the tags may be moving somewhere this test cannot see", i+1)
			continue
		}
		movers = append(movers, m[1])
	}

	sort.Strings(movers)
	if len(movers) != 1 || movers[0] != "ExtraAura" {
		t.Errorf("parseMod moves tags onto %v; expected exactly [ExtraAura] "+
			"(ModParser.lua:4190-4197). A new entry here is a LIST payload that can now be "+
			"reached by a value-scaling tag through the GENERIC parser path, which the static "+
			"scan in TestListPayloadsImplementScalableValue cannot see. It needs ScaleValue.",
			movers)
	}
}

var listTypeRe = regexp.MustCompile(`(?m)^type (\w+) struct`)

func payloadTypesInListTypes(t *testing.T) map[string]bool {
	t.Helper()
	src, err := os.ReadFile("list_types.go")
	if err != nil {
		t.Fatal(err)
	}
	found := listTypeRe.FindAllStringSubmatch(string(src), -1)
	if len(found) == 0 {
		t.Fatal("parsed no payload types out of list_types.go — has the file moved?")
	}
	out := make(map[string]bool, len(found))
	for _, m := range found {
		out[m[1]] = true
	}
	return out
}

var listModOpenRe = regexp.MustCompile(`MOD\(\s*"(\w+)"\s*,\s*"LIST"|mod\.NewList\(\s*"(\w+)"`)

var payloadTypeRe = regexp.MustCompile(`mod\.(\w+)\{`)

func scanOuterScalingTags(t *testing.T) (map[string]bool, map[string][]string) {
	t.Helper()

	var files []string
	for _, pattern := range listPayloadProducers {
		matched, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range matched {
			if !strings.HasSuffix(f, "_test.go") {
				files = append(files, f)
			}
		}
	}
	if len(files) < 10 {
		t.Fatalf("found only %d producer files for %v — the scan would pass vacuously, which is "+
			"the one outcome this test must never have", len(files), listPayloadProducers)
	}

	reachable := map[string]bool{}
	sites := map[string][]string{}
	total := 0
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src := string(raw)
		for _, loc := range listModOpenRe.FindAllStringIndex(src, -1) {
			total++
			stmt := listModStatement(src, loc[0])

			name := ""
			if m := listModOpenRe.FindStringSubmatch(stmt); m != nil {
				name = m[1] + m[2]
			}
			outer := blankBracedLiterals(stmt)
			payload := name
			if m := payloadTypeRe.FindStringSubmatch(stmt); m != nil {
				payload = m[1]
			}

			for _, ctor := range scalingTagCtors {
				if strings.Contains(outer, ctor) {
					reachable[payload] = true
					line := strings.Count(src[:loc[0]], "\n") + 1
					sites[payload] = append(sites[payload], fmt.Sprintf("%s:%d", f, line))
					break
				}
			}
		}
	}
	if total < 100 {
		t.Fatalf("scanned only %d LIST-mod sites — expected hundreds; the constructors have "+
			"probably been renamed and this test is no longer measuring anything", total)
	}
	return reachable, sites
}

func listModStatement(src string, start int) string {
	i := strings.IndexByte(src[start:], '(')
	if i < 0 {
		return src[start:]
	}
	end := spanBalanced(src, start+i, '(', ')')
	for end < len(src) && src[end] == '.' {
		q := strings.IndexByte(src[end:], '(')
		if q < 0 || strings.Contains(src[end:end+q], "\n") {
			break
		}
		end = spanBalanced(src, end+q, '(', ')')
	}
	return src[start:end]
}

func blankBracedLiterals(stmt string) string {
	var b strings.Builder
	for i := 0; i < len(stmt); {
		j := strings.IndexByte(stmt[i:], '{')
		if j < 0 {
			b.WriteString(stmt[i:])
			break
		}
		b.WriteString(stmt[i : i+j])
		i = spanBalanced(stmt, i+j, '{', '}')
	}
	return b.String()
}

func spanBalanced(s string, i int, open, closed byte) int {
	depth := 0
	for i < len(s) {
		switch s[i] {
		case open:
			depth++
		case closed:
			depth--
			if depth == 0 {
				return i + 1
			}
		case '"':
			i++
			for i < len(s) && s[i] != '"' {
				if s[i] == '\\' {
					i++
				}
				i++
			}
		}
		i++
	}
	return len(s)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
