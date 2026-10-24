package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const sbvPy = "oracle/skill_base_mods_verify.py"

const vcPy = "oracle/verify_common.py"

func sbvLuaDump() string {
	return `{
  "skillCount": 3,
  "modCount": 8,
  "skillTypeNames": {"7": "Trappable", "1": "Attack"},
  "skills": {
    "AlphaSkill": {
      "mods": [
        {"name": "Damage", "type": "INC", "value": {"__number": 25}, "flags": 257,
         "keywordFlags": 65536,
         "tags": [{"__table": true, "type": {"__string": "GlobalEffect"},
                   "effectType": {"__string": "Buff"},
                   "effectName": {"__string": "Fortify"},
                   "unscalable": {"__bool": true}}]},
        {"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
         "value": {"__table": true, "key": {"__string": "showAverage"},
                   "value": {"__bool": true}}},
        {"name": "Buff", "type": "FLAG", "value": {"__bool": true}, "flags": 0,
         "keywordFlags": 0, "tags": []}
      ]
    },
    "BetaSkill": {
      "mods": [
        {"name": "Speed", "type": "MORE", "value": {"__number": 10}, "flags": 0,
         "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "Multiplier"},
                   "var": {"__string": "Frenzy"}, "base": {"__number": 2},
                   "div": {"__number": 5}, "limit": {"__number": 40},
                   "limitVar": {"__string": "MaxFrenzy"},
                   "actor": {"__string": "parent"}},
                  {"__table": true, "type": {"__string": "MultiplierThreshold"},
                   "var": {"__string": "Rage"}, "threshold": {"__number": 10}}]},
        {"name": "Life", "type": "BASE", "value": {"__number": 40}, "flags": 0,
         "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "PerStat"},
                   "stat": {"__string": "Str"}, "div": {"__number": 5},
                   "base": {"__number": 2}},
                  {"__table": true, "type": {"__string": "Condition"},
                   "varList": {"__table": true, "1": {"__string": "A"},
                               "2": {"__string": "B"}},
                   "neg": {"__bool": true}},
                  {"__table": true, "type": {"__string": "ActorCondition"},
                   "actor": {"__string": "enemy"}, "var": {"__string": "Shocked"}}]},
        {"name": "Armour", "type": "INC", "value": {"__number": 15}, "flags": 0,
         "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "SkillType"},
                   "skillType": {"__number": 7}},
                  {"__table": true, "type": {"__string": "SkillPart"},
                   "skillPartList": {"__table": true, "1": {"__number": 1},
                                     "2": {"__number": 2}}},
                  {"__table": true, "type": {"__string": "Skill"},
                   "skillPartList": {"__table": true, "1": {"__number": 1}}}]}
      ]
    },
    "GammaSkill": {
      "mods": [
        {"name": "Evasion", "type": "INC", "value": {"__number": 5}, "flags": 0,
         "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "SkillPart"},
                   "skillPart": {"__number": 3}, "neg": {"__bool": false}},
                  {"__table": true, "type": {"__string": "SkillType"}},
                  {"__table": true, "type": {"__string": "NoSuchTagType"}}]},
        {"name": "Mana", "type": "BASE", "value": {"__number": 7}, "flags": 0,
         "keywordFlags": 0, "tags": []}
      ]
    }
  }
}`
}

func sbvGoDump(matching bool) string {
	showAverage := "1"
	if !matching {
		showAverage = "0"
	}
	source := `"Skill:GammaSkill"`
	if !matching {
		source = `"Skill:WrongSkill"`
	}
	damage := "25"
	if !matching {
		damage = "26"
	}
	return `{
  "AlphaSkill": [
    {"name": "Damage", "type": "INC", "source": "Skill:AlphaSkill", "flags": 257, "kw": 65536,
     "value": ` + damage + `,
     "tags": [{"type": "GlobalEffect", "effects": ["Buff"], "name": "Fortify",
               "unscalable": true}]},
    {"name": "SkillData", "type": "LIST", "source": "Skill:AlphaSkill", "flags": 0, "kw": 0,
     "value": {"key": "ShowAverage", "value": ` + showAverage + `}, "tags": []},
    {"name": "Buff", "type": "FLAG", "source": "Skill:AlphaSkill", "flags": 0, "kw": 0,
     "value": true, "tags": []}
  ],
  "BetaSkill": [
    {"name": "Speed", "type": "MORE", "source": "Skill:BetaSkill", "flags": 0, "kw": 0,
     "value": 10,
     "tags": [{"type": "Multiplier", "vars": ["Frenzy"], "base": 2, "div": 5, "limit": 40,
               "limitVar": "MaxFrenzy", "actor": "parent"},
              {"type": "MultiplierThreshold", "var": "Rage", "threshold": 10}]},
    {"name": "Life", "type": "BASE", "source": "Skill:BetaSkill", "flags": 0, "kw": 0,
     "value": 40,
     "tags": [{"type": "PerStat", "stats": ["Str"], "div": 5, "base": 2},
              {"type": "Condition", "vars": ["A", "B"], "neg": true},
              {"type": "ActorCondition", "actor": "enemy", "vars": ["Shocked"], "neg": false}]},
    {"name": "Armour", "type": "INC", "source": "Skill:BetaSkill", "flags": 0, "kw": 0,
     "value": 15,
     "tags": [{"type": "SkillType", "skillTypes": ["Trappable"]},
              {"type": "SkillPart", "part": 0, "partList": [1, 2], "neg": false}]}
  ],
  "GammaSkill": [
    {"name": "Evasion", "type": "INC", "source": ` + source + `, "flags": 0, "kw": 0,
     "value": 5,
     "tags": [{"type": "SkillPart", "part": 3, "partList": null, "neg": false},
              {"type": "SkillType", "skillTypes": []},
              {"type": "NoSuchTagType"}]},
    {"name": "Mana", "type": "BASE", "source": "Skill:GammaSkill", "flags": 0, "kw": 0,
     "value": 7, "tags": []}
  ]
}`
}

func sbvSandbox(t *testing.T, repo string, luaDump, goDump string) func() (string, error) {
	t.Helper()
	src := map[string]string{}
	for _, rel := range []string{mcPy, vcPy, sbvPy} {
		b := difftest.ReadOriginal(t, repo, rel)
		if b == nil {
			continue
		}
		src[rel] = string(b)
	}
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{"lua.json": luaDump, "go.json": goDump}
		for rel, body := range src {
			files[rel] = body
		}
		for rel, body := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(body), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func sbvPut(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func sbvSpec(bin string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool: "skill-base-mods-verify", PythonScript: sbvPy, PythonArgs: args,
		GoBin: bin, GoArgs: append([]string{"skill-base-mods-verify"}, args...),
		Stderr: difftest.StderrExact, Library: []string{mcPy, vcPy},
		RefusalSites: 2,
	}
}

func TestSkillBaseModsVerify(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := sbvSandbox(t, repo, sbvLuaDump(), sbvGoDump(true))
	pySrc := difftest.ReadOriginal(t, repo, sbvPy)

	spec := sbvSpec(bin, "lua.json", "go.json")
	spec.Fixtures = []difftest.Fixture{
		{Name: "happy", Passes: true},

		{Name: "no-args", Refusal: "usage: skill_base_mods_verify.py <lua.json> <go.json>",
			Break: func(string) error { return nil }},
		{Name: "three-args", Refusal: "usage: skill_base_mods_verify.py <lua.json> <go.json>",
			Break: func(string) error { return nil }},

		{Name: "lua-dump-missing",
			Refusal: "REFUSING: the Lua dump (lua.json) is not readable: No such file or directory",
			Break:   func(sb string) error { return os.Remove(filepath.Join(sb, "lua.json")) }},
		{Name: "go-dump-missing",
			Refusal: "REFUSING: the Go dump (go.json) is not readable: No such file or directory",
			Break:   func(sb string) error { return os.Remove(filepath.Join(sb, "go.json")) }},
		{Name: "lua-dump-not-json", Refusal: "REFUSING: the Lua dump (lua.json) is not valid JSON",
			Break: func(sb string) error { return sbvPut(sb, "lua.json", "{ nope") }},
		{Name: "lua-dump-not-an-object", Refusal: "REFUSING: the Lua dump is not a JSON object",
			Break: func(sb string) error { return sbvPut(sb, "lua.json", "[]") }},
		{Name: "go-dump-not-an-object", Refusal: "REFUSING: the Go dump is not a JSON object",
			Break: func(sb string) error { return sbvPut(sb, "go.json", "[]") }},
		{Name: "lua-dump-has-no-skills", Refusal: "REFUSING: the Lua dump carries no 'skills'",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 0, "modCount": 0}`)
			}},
		{Name: "lua-skills-is-an-array", Refusal: "REFUSING: the Lua dump's 'skills' is not a JSON object",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skills": [],
					"skillCount": 0, "modCount": 0}`)
			}},
		{Name: "a-tag-is-missing-a-required-field",
			Refusal: "REFUSING: Multiplier tag has no 'var'",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 1,
					"modCount": 1, "skills": {"AlphaSkill": {"mods": [
					{"name": "D", "type": "INC", "value": {"__number": 1}, "flags": 0,
					 "keywordFlags": 0,
					 "tags": [{"__table": true, "type": {"__string": "Multiplier"}}]}]}}}`)
			}},
		{Name: "a-table-with-non-contiguous-digit-keys",
			Refusal: "REFUSING: a table with all-digit keys is not the array 1..2; dkjson " +
				"writes a contiguous run for a Lua array, so this dump has been edited or truncated",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 1,
					"modCount": 1, "skills": {"AlphaSkill": {"mods": [
					{"name": "D", "type": "INC", "flags": 0, "keywordFlags": 0, "tags": [],
					 "value": {"__table": true, "1": {"__number": 1}, "3": {"__number": 2}}}]}}}`)
			}},
		{Name: "skilltype-is-not-a-number",
			Refusal: "REFUSING: SkillType 'nope' is not a number",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 1, "modCount": 1,
					"skills": {"AlphaSkill": {"mods": [{"name": "D", "type": "INC", "flags": 0,
					 "keywordFlags": 0, "value": {"__number": 1}, "tags": [{"__table": true, "type": {"__string": "SkillType"}, "skillType": {"__string": "nope"}}]}]}}}`)
			}},
		{Name: "skillpart-is-not-a-number",
			Refusal: "REFUSING: SkillPart tag has a non-numeric part",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 1, "modCount": 1,
					"skills": {"AlphaSkill": {"mods": [{"name": "D", "type": "INC", "flags": 0,
					 "keywordFlags": 0, "value": {"__number": 1}, "tags": [{"__table": true, "type": {"__string": "SkillPart"}, "skillPart": {"__string": "nope"}}]}]}}}`)
			}},
		{Name: "skilldata-value-is-not-a-table",
			Refusal: "REFUSING: SkillData mod value is not a table",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 1, "modCount": 1,
					"skills": {"AlphaSkill": {"mods": [{"name": "SkillData", "type": "LIST", "flags": 0,
					 "keywordFlags": 0, "value": {"__number": 5}, "tags": []}]}}}`)
			}},
		{Name: "skilldata-has-no-key",
			Refusal: "REFUSING: SkillData mod has no 'key' string",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 1, "modCount": 1,
					"skills": {"AlphaSkill": {"mods": [{"name": "SkillData", "type": "LIST", "flags": 0,
					 "keywordFlags": 0, "value": {"__table": true, "value": {"__number": 5}}, "tags": []}]}}}`)
			}},
		{Name: "a-lua-tag-is-not-a-table",
			Refusal: "REFUSING: a tag is not a table",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 1, "modCount": 1,
					"skills": {"AlphaSkill": {"mods": [{"name": "D", "type": "INC", "flags": 0,
					 "keywordFlags": 0, "value": {"__number": 1}, "tags": [{"__number": 7}]}]}}}`)
			}},
		{Name: "a-go-tag-is-not-a-table",
			Refusal: "REFUSING: a tag is not a table",
			Break: func(sb string) error {
				return sbvPut(sb, "go.json", `{"AlphaSkill": [{"name": "D", "type": "INC",
					"source": "Skill:AlphaSkill", "flags": 0, "kw": 0, "value": 1, "tags": [7]}]}`)
			}},
		{Name: "a-lua-mod-is-not-a-table",
			Refusal: "REFUSING: skill 'AlphaSkill' has a mod that is not a table",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 1, "modCount": 1, "skills": {"AlphaSkill": {"mods": [7]}}}`)
			}},
		{Name: "a-go-mod-is-not-a-table",
			Refusal: "REFUSING: skill 'AlphaSkill' has a Go mod that is not a table",
			Break: func(sb string) error {
				return sbvPut(sb, "go.json", `{"AlphaSkill": [7]}`)
			}},
		{Name: "present-but-empty-skillpartlist",
			Passes: true,
			Break: func(sb string) error {
				if err := sbvPut(sb, "go.json", `{"AlphaSkill": []}`); err != nil {
					return err
				}
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 1, "modCount": 1,
					"skills": {"AlphaSkill": {"mods": [{"name": "D", "type": "INC", "flags": 0,
					 "keywordFlags": 0, "value": {"__number": 1}, "tags": [{"__table": true, "type": {"__string": "SkillPart"}, "skillPartList": {"__table": true}}]}]}}}`)
			}},
		{Name: "a-lua-mod-has-no-name",
			Refusal: "REFUSING: a Lua mod has no 'name'",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 1,
					"modCount": 1, "skills": {"AlphaSkill": {"mods": [{"type": "INC", "flags": 0,
					 "keywordFlags": 0, "value": {"__number": 1}, "tags": []}]}}}`)
			}},
		{Name: "a-lua-mod-has-no-flags",
			Refusal: "REFUSING: a Lua mod has no 'flags'",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 1,
					"modCount": 1, "skills": {"AlphaSkill": {"mods": [{"name": "D", "type": "INC",
					 "value": {"__number": 1}, "tags": []}]}}}`)
			}},
		{Name: "a-go-mod-has-no-source",
			Refusal: "REFUSING: a Go mod has no 'source'",
			Break: func(sb string) error {
				return sbvPut(sb, "go.json", `{"AlphaSkill": [{"name": "D", "type": "INC",
					"flags": 0, "kw": 0, "value": 1, "tags": []}]}`)
			}},
		{Name: "a-go-tag-is-missing-a-later-field",
			Refusal: "REFUSING: Multiplier tag has no 'limitVar'",
			Break: func(sb string) error {
				return sbvPut(sb, "go.json", `{"AlphaSkill": [{"name": "D", "type": "INC",
					"source": "Skill:AlphaSkill", "flags": 0, "kw": 0, "value": 1,
					"tags": [{"type": "Multiplier", "vars": ["X"], "base": 0, "div": 1,
					 "limit": null}]}]}`)
			}},
		{Name: "a-go-tag-is-missing-its-FIRST-field",
			Refusal: "REFUSING: Multiplier tag has no 'vars'",
			Break: func(sb string) error {
				return sbvPut(sb, "go.json", `{"AlphaSkill": [{"name": "D", "type": "INC",
					"source": "Skill:AlphaSkill", "flags": 0, "kw": 0, "value": 1,
					"tags": [{"type": "Multiplier", "base": 0, "div": 1, "limit": null,
					 "limitVar": null, "actor": ""}]}]}`)
			}},
		{Name: "lua-dump-has-no-skillcount",
			Refusal: "REFUSING: the Lua dump carries no 'skillCount'",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "modCount": 0,
					"skills": {"AlphaSkill": {"mods": []}}}`)
			}},
		{Name: "a-lua-skill-has-no-mods-list",
			Refusal: "REFUSING: skill 'AlphaSkill' has no 'mods' list in the Lua dump",
			Break: func(sb string) error {
				return sbvPut(sb, "lua.json", `{"skillTypeNames": {}, "skillCount": 1,
					"modCount": 0, "skills": {"AlphaSkill": {}}}`)
			}},
		{Name: "a-go-skilldata-value-has-no-key",
			Refusal: "REFUSING: SkillData tag has no 'key'",
			Break: func(sb string) error {
				return sbvPut(sb, "go.json", `{"AlphaSkill": [{"name": "SkillData", "type": "LIST",
					"source": "Skill:AlphaSkill", "flags": 0, "kw": 0, "tags": [],
					"value": {"value": 1}}]}`)
			}},
		{Name: "a-go-skilldata-value-has-no-value",
			Refusal: "REFUSING: SkillData tag has no 'value'",
			Break: func(sb string) error {
				return sbvPut(sb, "go.json", `{"AlphaSkill": [{"name": "SkillData", "type": "LIST",
					"source": "Skill:AlphaSkill", "flags": 0, "kw": 0, "tags": [],
					"value": {"key": "ShowAverage"}}]}`)
			}},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			switch f.Name {
			case "no-args":
				s.PythonArgs = nil
			case "three-args":
				s.PythonArgs = []string{"lua.json", "go.json", "extra"}
			}
			s.GoArgs = append([]string{"skill-base-mods-verify"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestSkillBaseModsVerifyReportsFindings: THE FINDING PATH IS A SEPARATE OUTCOME FROM THE REFUSAL PATH AND NEEDS ITS OWN FIXTURE.
func TestSkillBaseModsVerifyReportsFindings(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := sbvSandbox(t, repo, sbvLuaDump(), sbvGoDump(false))
	s := sbvSpec(bin, "lua.json", "go.json")
	difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "mismatched"}, mk)
}

// TestSkillBaseModsVerifyBooleanFlags: NEITHER A REFUSAL NOR A SUCCESS.
func TestSkillBaseModsVerifyBooleanFlags(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := sbvSandbox(t, repo,
		`{"skillTypeNames": {}, "skillCount": 1, "modCount": 1, "skills": {"AlphaSkill": {"mods": [
			{"name": "D", "type": "INC", "flags": true, "keywordFlags": 0,
			 "value": {"__number": 1}, "tags": []}]}}}`,
		`{"AlphaSkill": [{"name": "D", "type": "INC", "source": "Skill:AlphaSkill",
			"flags": 1, "kw": 0, "value": 1, "tags": []}]}`)
	s := sbvSpec(bin, "lua.json", "go.json")
	difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "boolean-flags"}, mk)
}

// TestSkillBaseModsVerifyCapsTheErrorList: THE 40-ERROR CAP.
func TestSkillBaseModsVerifyCapsTheErrorList(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := sbvSandbox(t, repo,
		`{"skillCount": 1, "modCount": 0, "skillTypeNames": {}, "skills": {"A": {"mods": []}}}`,
		"{\n  \"A\": [\n"+`    {"name": "M01", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 1, "tags": []},
    {"name": "M02", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 2, "tags": []},
    {"name": "M03", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 3, "tags": []},
    {"name": "M04", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 4, "tags": []},
    {"name": "M05", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 5, "tags": []},
    {"name": "M06", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 6, "tags": []},
    {"name": "M07", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 7, "tags": []},
    {"name": "M08", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 8, "tags": []},
    {"name": "M09", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 9, "tags": []},
    {"name": "M10", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 10, "tags": []},
    {"name": "M11", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 11, "tags": []},
    {"name": "M12", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 12, "tags": []},
    {"name": "M13", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 13, "tags": []},
    {"name": "M14", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 14, "tags": []},
    {"name": "M15", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 15, "tags": []},
    {"name": "M16", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 16, "tags": []},
    {"name": "M17", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 17, "tags": []},
    {"name": "M18", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 18, "tags": []},
    {"name": "M19", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 19, "tags": []},
    {"name": "M20", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 20, "tags": []},
    {"name": "M21", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 21, "tags": []},
    {"name": "M22", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 22, "tags": []},
    {"name": "M23", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 23, "tags": []},
    {"name": "M24", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 24, "tags": []},
    {"name": "M25", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 25, "tags": []},
    {"name": "M26", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 26, "tags": []},
    {"name": "M27", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 27, "tags": []},
    {"name": "M28", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 28, "tags": []},
    {"name": "M29", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 29, "tags": []},
    {"name": "M30", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 30, "tags": []},
    {"name": "M31", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 31, "tags": []},
    {"name": "M32", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 32, "tags": []},
    {"name": "M33", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 33, "tags": []},
    {"name": "M34", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 34, "tags": []},
    {"name": "M35", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 35, "tags": []},
    {"name": "M36", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 36, "tags": []},
    {"name": "M37", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 37, "tags": []},
    {"name": "M38", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 38, "tags": []},
    {"name": "M39", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 39, "tags": []},
    {"name": "M40", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 40, "tags": []},
    {"name": "M41", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 41, "tags": []},
    {"name": "M42", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 42, "tags": []},
    {"name": "M43", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 43, "tags": []},
    {"name": "M44", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 44, "tags": []},
    {"name": "M45", "type": "INC", "source": "Skill:A", "flags": 0, "kw": 0, "value": 45, "tags": []}`+"\n  ]\n}")
	s := sbvSpec(bin, "lua.json", "go.json")
	difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "forty-five-findings"}, mk)
}

// TestSkillBaseModsVerifyExtraGoSkill: A skill the Go table carries and the Lua does not — its own arm, and sorted independently.
func TestSkillBaseModsVerifyExtraGoSkill(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := sbvSandbox(t, repo, `{"skillCount": 0, "modCount": 0, "skillTypeNames": {}, "skills": {}}`,
		`{"ZetaSkill": [], "AlphaSkill": []}`)
	s := sbvSpec(bin, "lua.json", "go.json")
	difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "extra-go-skills"}, mk)
}

// TestSkillBaseModsVerifyKeyDeletion: THE **REVERSE** DIRECTION OF THE ONE-SIDED-REFUSAL CLASS, WHICH NO LEXICAL CHECK CAN SEE.
func TestSkillBaseModsVerifyKeyDeletion(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := sbvSandbox(t, repo, sbvLuaDump(), sbvGoDump(true))
	difftest.AssertKeyDeletionAgrees(t, t.Context(), sbvSpec(bin, "lua.json", "go.json"),
		[]string{"lua.json", "go.json"}, mk)
}
