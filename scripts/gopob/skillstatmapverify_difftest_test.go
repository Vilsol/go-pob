package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const ssvPy = "oracle/skill_stat_map_verify.py"

func ssvLuaDump() string {
	return `{"skillCount": 2, "entryCount": 2, "modCount": 3, "skillTypeNames": {"1": "Attack"},
	 "skills": {
	   "AlphaSkill": {"entries": {"damage": {"scalars": {"mult": {"__number": 2}}, "mods": [
	     {"name": "Damage", "type": "INC", "value": {"__number": 25}, "flags": 0, "keywordFlags": 0,
	      "tags": [{"__table": true, "type": {"__string": "Multiplier"}, "var": {"__string": "X"}}]},
	     {"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
	      "value": {"__table": true, "key": {"__string": "showAverage"}, "value": {"__bool": true}}}]}}},
	   "BetaSkill": {"entries": {"speed": {"scalars": {}, "mods": [
	     {"name": "Speed", "type": "MORE", "value": {"__number": 10}, "flags": 0, "keywordFlags": 0,
	      "tags": []}]}}}}}`
}

func ssvGoDump() string {
	return `{"AlphaSkill": {"damage": {"value": null, "mult": 2, "div": null, "base": null, "mods": [
	   {"name": "Damage", "type": "INC", "value": 25, "flags": 0, "kw": 0,
	    "tags": [{"type": "Multiplier", "vars": ["X"], "div": 1, "base": 0, "limit": null,
	              "limitVar": null, "limitTotal": false, "actor": ""}]},
	   {"name": "SkillData", "type": "LIST", "value": {"key": "showAverage", "value": 1},
	    "flags": 0, "kw": 0, "tags": []}]}},
	 "BetaSkill": {"speed": {"value": null, "mult": null, "div": null, "base": null, "mods": [
	   {"name": "Speed", "type": "MORE", "value": 10, "flags": 0, "kw": 0, "tags": []}]}}}`
}

func ssvSandbox(t *testing.T, repo string, luaDump, goDump string) func() (string, error) {
	t.Helper()
	src := map[string]string{}
	for _, rel := range []string{mcPy, vcPy, ssvPy} {
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
		for rel, b := range src {
			files[rel] = b
		}
		for rel, b := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(b), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func ssvPut(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func ssvSpec(bin string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool: "skill-stat-map-verify", PythonScript: ssvPy, PythonArgs: args,
		GoBin: bin, GoArgs: append([]string{"skill-stat-map-verify"}, args...),
		Stderr: difftest.StderrExact, Library: []string{mcPy, vcPy},
		RefusalSites: 2,
	}
}

func TestSkillStatMapVerify(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssvSandbox(t, repo, ssvLuaDump(), ssvGoDump())
	pySrc := difftest.ReadOriginal(t, repo, ssvPy)

	spec := ssvSpec(bin, "lua.json", "go.json")
	spec.Fixtures = []difftest.Fixture{
		{Name: "happy", Passes: true},

		{Name: "no-args", Refusal: "usage: skill_stat_map_verify.py <lua.json> <go.json>",
			Break: func(string) error { return nil }},
		{Name: "three-args", Refusal: "usage: skill_stat_map_verify.py <lua.json> <go.json>",
			Break: func(string) error { return nil }},

		{Name: "lua-dump-missing",
			Refusal: "REFUSING: the Lua dump (lua.json) is not readable: No such file or directory",
			Break:   func(sb string) error { return os.Remove(filepath.Join(sb, "lua.json")) }},
		{Name: "go-dump-missing",
			Refusal: "REFUSING: the Go dump (go.json) is not readable: No such file or directory",
			Break:   func(sb string) error { return os.Remove(filepath.Join(sb, "go.json")) }},
		{Name: "lua-dump-not-json", Refusal: "REFUSING: the Lua dump (lua.json) is not valid JSON",
			Break: func(sb string) error { return ssvPut(sb, "lua.json", "{ nope") }},

		{Name: "lua-not-an-object",
			Refusal: "REFUSING: the Lua dump is not a JSON object",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `[]`)
			}},
		{Name: "go-not-an-object",
			Refusal: "REFUSING: the Go dump is not a JSON object",
			Break: func(sb string) error {
				return ssvPut(sb, "go.json", `[]`)
			}},
		{Name: "lua-has-no-skills",
			Refusal: "REFUSING: the Lua dump carries no 'skills'",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillTypeNames": {},
					"skillCount": 0,
					"entryCount": 0,
					"modCount": 0
					}`)
			}},
		{Name: "lua-skills-is-an-array",
			Refusal: "REFUSING: the Lua dump's 'skills' is not a JSON object",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillTypeNames": {},
					"skills": [],
					"skillCount": 0,
					"entryCount": 0,
					"modCount": 0
					}`)
			}},
		{Name: "a-tag-is-missing-a-required-field",
			Refusal: "REFUSING: Multiplier tag has no 'var'",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					{
					"name": "Damage",
					"type": "INC",
					"value": {
					"__number": 25
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": [
					{
					"__table": true,
					"type": {
					"__string": "Multiplier"
					}
					}
					]
					}
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "a-lua-mod-has-no-flags",
			Refusal: "REFUSING: a Lua mod has no 'flags'",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					{
					"name": "D",
					"type": "INC",
					"value": {
					"__number": 1
					},
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "skilltype-is-not-a-number",
			Refusal: "REFUSING: SkillType 'nope' is not a number",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					{
					"name": "Damage",
					"type": "INC",
					"value": {
					"__number": 25
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": [
					{
					"__table": true,
					"type": {
					"__string": "SkillType"
					},
					"skillType": {
					"__string": "nope"
					}
					}
					]
					}
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "skillpart-is-not-a-number",
			Refusal: "REFUSING: SkillPart tag has a non-numeric part",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					{
					"name": "Damage",
					"type": "INC",
					"value": {
					"__number": 25
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": [
					{
					"__table": true,
					"type": {
					"__string": "SkillPart"
					},
					"skillPart": {
					"__string": "nope"
					}
					}
					]
					}
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "meleeproximity-ramp-is-not-numeric",
			Refusal: "REFUSING: MeleeProximity tag has a non-numeric ramp entry",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					{
					"name": "Damage",
					"type": "INC",
					"value": {
					"__number": 25
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": [
					{
					"__table": true,
					"type": {
					"__string": "MeleeProximity"
					},
					"ramp": {
					"__table": true,
					"1": {
					"__string": "x"
					}
					}
					}
					]
					}
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "modflagor-is-not-numeric",
			Refusal: "REFUSING: ModFlagOr tag has non-numeric modFlags",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					{
					"name": "Damage",
					"type": "INC",
					"value": {
					"__number": 25
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": [
					{
					"__table": true,
					"type": {
					"__string": "ModFlagOr"
					},
					"modFlags": {
					"__string": "x"
					}
					}
					]
					}
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "distanceramp-is-not-pairs",
			Refusal: "REFUSING: DistanceRamp ramp is not a list of pairs",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					{
					"name": "Damage",
					"type": "INC",
					"value": {
					"__number": 25
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": [
					{
					"__table": true,
					"type": {
					"__string": "DistanceRamp"
					},
					"ramp": {
					"__table": true,
					"1": {
					"__number": 5
					}
					}
					}
					]
					}
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "skilldata-value-is-not-a-table",
			Refusal: "REFUSING: SkillData mod value is not a table",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					{
					"name": "SkillData",
					"type": "LIST",
					"value": {
					"__number": 5
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "minionmodifier-has-no-inner-mod",
			Refusal: "REFUSING: MinionModifier tag has no 'mod'",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					{
					"name": "MinionModifier",
					"type": "LIST",
					"value": {
					"__table": true,
					"x": {
					"__number": 1
					}
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "a-lua-tag-is-not-a-table",
			Refusal: "REFUSING: a tag is not a table",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					{
					"name": "Damage",
					"type": "INC",
					"value": {
					"__number": 25
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": [
					{
					"__number": 7
					}
					]
					}
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "a-lua-mod-is-not-a-table",
			Refusal: "REFUSING: AlphaSkill/damage: a Lua mod is not a table",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					7
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "a-go-mod-is-not-a-table",
			Refusal: "REFUSING: AlphaSkill/damage: a Go mod is not a table",
			Break: func(sb string) error {
				return ssvPut(sb, "go.json", `{
					"AlphaSkill": {
					"damage": {
					"value": null,
					"mult": 2,
					"div": null,
					"base": null,
					"mods": [
					7
					]
					}
					}
					}`)
			}},
		{Name: "a-go-mod-has-no-kw",
			Refusal: "REFUSING: a Go mod has no 'kw'",
			Break: func(sb string) error {
				return ssvPut(sb, "go.json", `{
					"AlphaSkill": {
					"damage": {
					"value": null,
					"mult": 2,
					"div": null,
					"base": null,
					"mods": [
					{
					"name": "D",
					"type": "INC",
					"value": 1,
					"flags": 0,
					"tags": []
					}
					]
					}
					}
					}`)
			}},
		{Name: "a-go-tag-is-missing-a-field",
			Refusal: "REFUSING: Multiplier tag has no 'limitTotal'",
			Break: func(sb string) error {
				return ssvPut(sb, "go.json", `{
					"AlphaSkill": {
					"damage": {
					"value": null,
					"mult": 2,
					"div": null,
					"base": null,
					"mods": [
					{
					"name": "D",
					"type": "INC",
					"value": 1,
					"flags": 0,
					"kw": 0,
					"tags": [
					{
					"type": "Multiplier",
					"vars": [
					"X"
					],
					"div": 1,
					"base": 0,
					"limit": null,
					"limitVar": null
					}
					]
					}
					]
					}
					}
					}`)
			}},
		{Name: "lua-skill-has-no-entries",
			Refusal: "REFUSING: skill 'AlphaSkill' has no 'entries' table in the Lua dump",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"mods": []
					}
					}
					}`)
			}},
		{Name: "go-skill-is-not-a-table",
			Refusal: "REFUSING: skill 'AlphaSkill' is not a table in the Go dump",
			Break: func(sb string) error {
				return ssvPut(sb, "go.json", `{
					"AlphaSkill": 7
					}`)
			}},
		{Name: "lua-entry-has-no-scalars",
			Refusal: "REFUSING: AlphaSkill/damage: the Lua entry has no 'scalars'",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"mods": []
					}
					}
					}
					}
					}`)
			}},
		{Name: "lua-entry-has-no-mods",
			Refusal: "REFUSING: AlphaSkill/damage: the Lua entry has no 'mods' list",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {}
					}
					}
					}
					}
					}`)
			}},
		{Name: "go-entry-has-no-mult",
			Refusal: "REFUSING: AlphaSkill/damage: the Go entry has no 'mult'",
			Break: func(sb string) error {
				return ssvPut(sb, "go.json", `{
					"AlphaSkill": {
					"damage": {
					"value": null,
					"div": null,
					"base": null,
					"mods": []
					}
					}
					}`)
			}},
		{Name: "lua-has-no-modcount",
			Refusal: "REFUSING: the Lua dump carries no 'modCount'",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					{
					"name": "Damage",
					"type": "INC",
					"value": {
					"__number": 25
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": [
					{
					"__table": true,
					"type": {
					"__string": "Multiplier"
					},
					"var": {
					"__string": "X"
					}
					}
					]
					},
					{
					"name": "SkillData",
					"type": "LIST",
					"flags": 0,
					"keywordFlags": 0,
					"tags": [],
					"value": {
					"__table": true,
					"key": {
					"__string": "showAverage"
					},
					"value": {
					"__bool": true
					}
					}
					}
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "non-contiguous-digit-keys",
			Refusal: "REFUSING: a table with all-digit keys is not the array 1..2",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					"skillCount": 2,
					"entryCount": 2,
					"modCount": 3,
					"skillTypeNames": {
					"1": "Attack"
					},
					"skills": {
					"AlphaSkill": {
					"entries": {
					"damage": {
					"scalars": {
					"mult": {
					"__number": 2
					}
					},
					"mods": [
					{
					"name": "Damage",
					"type": "INC",
					"value": {
					"__table": true,
					"1": {
					"__number": 1
					},
					"3": {
					"__number": 2
					}
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					},
					"BetaSkill": {
					"entries": {
					"speed": {
					"scalars": {},
					"mods": [
					{
					"name": "Speed",
					"type": "MORE",
					"value": {
					"__number": 10
					},
					"flags": 0,
					"keywordFlags": 0,
					"tags": []
					}
					]
					}
					}
					}
					}
					}`)
			}},
		{Name: "distanceramp-pair-is-not-a-pair",
			Refusal: "REFUSING: DistanceRamp ramp is not a list of pairs",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					 "skillCount": 2,
					 "entryCount": 2,
					 "modCount": 3,
					 "skillTypeNames": {
					  "1": "Attack"
					 },
					 "skills": {
					  "AlphaSkill": {
					   "entries": {
					    "damage": {
					     "scalars": {
					      "mult": {
					       "__number": 2
					      }
					     },
					     "mods": [
					      {
					       "name": "D",
					       "type": "INC",
					       "value": {
					        "__number": 1
					       },
					       "flags": 0,
					       "keywordFlags": 0,
					       "tags": [
					        {
					         "__table": true,
					         "type": {
					          "__string": "DistanceRamp"
					         },
					         "ramp": {
					          "__table": true,
					          "1": {
					           "__table": true,
					           "1": {
					            "__number": 1
					           },
					           "2": {
					            "__number": 2
					           },
					           "3": {
					            "__number": 3
					           }
					          }
					         }
					        }
					       ]
					      }
					     ]
					    }
					   }
					  },
					  "BetaSkill": {
					   "entries": {
					    "speed": {
					     "scalars": {},
					     "mods": [
					      {
					       "name": "Speed",
					       "type": "MORE",
					       "value": {
					        "__number": 10
					       },
					       "flags": 0,
					       "keywordFlags": 0,
					       "tags": []
					      }
					     ]
					    }
					   }
					  }
					 }
					}`)
			}},
		{Name: "statthreshold-missing-its-FIRST-field",
			Refusal: "REFUSING: StatThreshold tag has no 'stat'",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					 "skillCount": 2,
					 "entryCount": 2,
					 "modCount": 3,
					 "skillTypeNames": {
					  "1": "Attack"
					 },
					 "skills": {
					  "AlphaSkill": {
					   "entries": {
					    "damage": {
					     "scalars": {
					      "mult": {
					       "__number": 2
					      }
					     },
					     "mods": [
					      {
					       "name": "D",
					       "type": "INC",
					       "value": {
					        "__number": 1
					       },
					       "flags": 0,
					       "keywordFlags": 0,
					       "tags": [
					        {
					         "__table": true,
					         "type": {
					          "__string": "StatThreshold"
					         },
					         "threshold": {
					          "__number": 20
					         }
					        }
					       ]
					      }
					     ]
					    }
					   }
					  },
					  "BetaSkill": {
					   "entries": {
					    "speed": {
					     "scalars": {},
					     "mods": [
					      {
					       "name": "Speed",
					       "type": "MORE",
					       "value": {
					        "__number": 10
					       },
					       "flags": 0,
					       "keywordFlags": 0,
					       "tags": []
					      }
					     ]
					    }
					   }
					  }
					 }
					}`)
			}},
		{Name: "a-nested-mod-is-not-a-table",
			Refusal: "REFUSING: a MinionModifier's nested mod is not a table",
			Break: func(sb string) error {
				return ssvPut(sb, "lua.json", `{
					 "skillCount": 2,
					 "entryCount": 2,
					 "modCount": 3,
					 "skillTypeNames": {
					  "1": "Attack"
					 },
					 "skills": {
					  "AlphaSkill": {
					   "entries": {
					    "damage": {
					     "scalars": {
					      "mult": {
					       "__number": 2
					      }
					     },
					     "mods": [
					      {
					       "name": "MinionModifier",
					       "type": "LIST",
					       "flags": 0,
					       "keywordFlags": 0,
					       "tags": [],
					       "value": {
					        "__table": true,
					        "mod": {
					         "__number": 5
					        }
					       }
					      }
					     ]
					    }
					   }
					  },
					  "BetaSkill": {
					   "entries": {
					    "speed": {
					     "scalars": {},
					     "mods": [
					      {
					       "name": "Speed",
					       "type": "MORE",
					       "value": {
					        "__number": 10
					       },
					       "flags": 0,
					       "keywordFlags": 0,
					       "tags": []
					      }
					     ]
					    }
					   }
					  }
					 }
					}`)
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
			s.GoArgs = append([]string{"skill-stat-map-verify"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestSkillStatMapVerifyReportsFindings: THE FINDING PATH IS A SEPARATE OUTCOME FROM THE REFUSAL PATH AND NEEDS ITS OWN FIXTURE.
func TestSkillStatMapVerifyReportsFindings(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssvSandbox(t, repo, `{"skillCount": 1, "entryCount": 1, "modCount": 6, "skillTypeNames": {"1": "Attack", "7": "Trap"},
		 "skills": {"AlphaSkill": {"entries": {"damage": {"scalars":
		   {"value": {"__number": 3}, "mult": {"__number": 2}, "div": {"__number": 4}, "base": {"__number": 1}},
		   "mods": [
		    {"name": "MinionModifier", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
		     "value": {"__table": true, "mod": {"__table": true, "name": {"__string": "Damage"},
		       "type": {"__string": "INC"}, "value": {"__number": 9}, "flags": {"__number": 2},
		       "1": {"__table": true, "type": {"__string": "SkillId"}, "skillId": {"__string": "Zap"}}}}},
		    {"name": "SupportedGemProperty", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
		     "value": {"__table": true, "keyword": {"__string": "aura"}, "key": {"__string": "level"},
		               "value": {"__number": 2}}},
		    {"name": "ExtraMinionSkill", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
		     "value": {"__table": true, "skillId": {"__string": "Golem"}}},
		    {"name": "Ramps", "type": "BASE", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0,
		     "tags": [{"__table": true, "type": {"__string": "DistanceRamp"},
		               "ramp": {"__table": true, "1": {"__table": true, "1": {"__number": 0}, "2": {"__number": 1}},
		                        "2": {"__table": true, "1": {"__number": 40}, "2": {"__number": 0.5}}}},
		              {"__table": true, "type": {"__string": "MeleeProximity"},
		               "ramp": {"__table": true, "1": {"__number": 0}, "2": {"__number": 10}}}]},
		    {"name": "Flags", "type": "MORE", "value": {"__number": 5}, "flags": 0, "keywordFlags": 0,
		     "tags": [{"__table": true, "type": {"__string": "ModFlagOr"}, "modFlags": {"__number": 3}},
		              {"__table": true, "type": {"__string": "SkillName"}, "skillName": {"__string": "Fire"}},
		              {"__table": true, "type": {"__string": "StatThreshold"}, "stat": {"__string": "Str"},
		               "threshold": {"__number": 20}, "upper": {"__bool": true}},
		              {"__table": true, "type": {"__string": "PercentStat"}, "stat": {"__string": "Life"},
		               "percentVar": {"__string": "V"}},
		              {"__table": true, "type": {"__string": "ActorCondition"}, "actor": {"__string": "enemy"},
		               "varList": {"__table": true, "1": {"__string": "A"}, "2": {"__string": "B"}}},
		              {"__table": true, "type": {"__string": "MultiplierThreshold"}, "var": {"__string": "M"},
		               "threshold": {"__bool": true}},
		              {"__table": true, "type": {"__string": "GlobalEffect"}, "effectType": {"__string": "Buff"}},
		              {"__table": true, "type": {"__string": "SkillType"}, "skillType": {"__number": 7}},
		              {"__table": true, "type": {"__string": "NoSuchTagType"}}]},
		    {"type": "LIST", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0, "tags": []}
		   ]}}}}}`, `{"AlphaSkill": {"damage": {"value": 3, "mult": 2, "div": 4, "base": 1, "mods": [
		  {"name": "MinionModifier", "type": "LIST", "flags": 0, "kw": 0, "tags": [],
		   "value": {"mod": {"name": "Damage", "type": "INC", "value": 9, "flags": 2, "kw": 0,
		     "tags": [{"type": "SkillId", "skillId": "Zap"}]}}},
		  {"name": "SupportedGemProperty", "type": "LIST", "flags": 0, "kw": 0, "tags": [],
		   "value": {"keyword": "aura", "propKey": "level", "value": 2}},
		  {"name": "ExtraMinionSkill", "type": "LIST", "flags": 0, "kw": 0, "tags": [],
		   "value": {"skillId": "Golem"}},
		  {"name": "Ramps", "type": "BASE", "value": 1, "flags": 0, "kw": 0,
		   "tags": [{"type": "DistanceRamp", "ramp": [[0, 1], [40, 0.5]]},
		            {"type": "MeleeProximity", "ramp": [0, 10]}]},
		  {"name": "Flags", "type": "MORE", "value": 99, "flags": 0, "kw": 0,
		   "tags": [{"type": "ModFlagOr", "modFlags": 3},
		            {"type": "SkillName", "names": ["Fire"], "neg": false},
		            {"type": "StatThreshold", "stat": "Str", "threshold": 20, "upper": true},
		            {"type": "PercentStat", "stat": "Life", "percent": 0, "percentVar": "V"},
		            {"type": "ActorCondition", "actor": "enemy", "vars": ["A", "B"], "neg": false},
		            {"type": "MultiplierThreshold", "var": "M", "threshold": 1, "upper": false, "actor": ""},
		            {"type": "GlobalEffect", "effects": ["Buff"], "name": "", "unscalable": false},
		            {"type": "SkillType", "skillTypes": ["Trap"], "neg": false},
		            {"type": "NoSuchTagType"}]}
		]}}}`)
	s := ssvSpec(bin, "lua.json", "go.json")
	difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "every-tag-type"}, mk)
}

// TestSkillStatMapVerifyCapsTheErrorList: THE 40-ERROR CAP.
func TestSkillStatMapVerifyCapsTheErrorList(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssvSandbox(t, repo,
		`{"skillCount": 1, "entryCount": 1, "modCount": 0, "skillTypeNames": {},
		 "skills": {"A": {"entries": {"s": {"scalars": {}, "mods": []}}}}}`,
		`{"A": {"s": {"value": null, "mult": null, "div": null, "base": null, "mods": [
		     {"name": "M01", "type": "INC", "flags": 0, "kw": 0, "value": 1, "tags": []},
		     {"name": "M02", "type": "INC", "flags": 0, "kw": 0, "value": 2, "tags": []},
		     {"name": "M03", "type": "INC", "flags": 0, "kw": 0, "value": 3, "tags": []},
		     {"name": "M04", "type": "INC", "flags": 0, "kw": 0, "value": 4, "tags": []},
		     {"name": "M05", "type": "INC", "flags": 0, "kw": 0, "value": 5, "tags": []},
		     {"name": "M06", "type": "INC", "flags": 0, "kw": 0, "value": 6, "tags": []},
		     {"name": "M07", "type": "INC", "flags": 0, "kw": 0, "value": 7, "tags": []},
		     {"name": "M08", "type": "INC", "flags": 0, "kw": 0, "value": 8, "tags": []},
		     {"name": "M09", "type": "INC", "flags": 0, "kw": 0, "value": 9, "tags": []},
		     {"name": "M10", "type": "INC", "flags": 0, "kw": 0, "value": 10, "tags": []},
		     {"name": "M11", "type": "INC", "flags": 0, "kw": 0, "value": 11, "tags": []},
		     {"name": "M12", "type": "INC", "flags": 0, "kw": 0, "value": 12, "tags": []},
		     {"name": "M13", "type": "INC", "flags": 0, "kw": 0, "value": 13, "tags": []},
		     {"name": "M14", "type": "INC", "flags": 0, "kw": 0, "value": 14, "tags": []},
		     {"name": "M15", "type": "INC", "flags": 0, "kw": 0, "value": 15, "tags": []},
		     {"name": "M16", "type": "INC", "flags": 0, "kw": 0, "value": 16, "tags": []},
		     {"name": "M17", "type": "INC", "flags": 0, "kw": 0, "value": 17, "tags": []},
		     {"name": "M18", "type": "INC", "flags": 0, "kw": 0, "value": 18, "tags": []},
		     {"name": "M19", "type": "INC", "flags": 0, "kw": 0, "value": 19, "tags": []},
		     {"name": "M20", "type": "INC", "flags": 0, "kw": 0, "value": 20, "tags": []},
		     {"name": "M21", "type": "INC", "flags": 0, "kw": 0, "value": 21, "tags": []},
		     {"name": "M22", "type": "INC", "flags": 0, "kw": 0, "value": 22, "tags": []},
		     {"name": "M23", "type": "INC", "flags": 0, "kw": 0, "value": 23, "tags": []},
		     {"name": "M24", "type": "INC", "flags": 0, "kw": 0, "value": 24, "tags": []},
		     {"name": "M25", "type": "INC", "flags": 0, "kw": 0, "value": 25, "tags": []},
		     {"name": "M26", "type": "INC", "flags": 0, "kw": 0, "value": 26, "tags": []},
		     {"name": "M27", "type": "INC", "flags": 0, "kw": 0, "value": 27, "tags": []},
		     {"name": "M28", "type": "INC", "flags": 0, "kw": 0, "value": 28, "tags": []},
		     {"name": "M29", "type": "INC", "flags": 0, "kw": 0, "value": 29, "tags": []},
		     {"name": "M30", "type": "INC", "flags": 0, "kw": 0, "value": 30, "tags": []},
		     {"name": "M31", "type": "INC", "flags": 0, "kw": 0, "value": 31, "tags": []},
		     {"name": "M32", "type": "INC", "flags": 0, "kw": 0, "value": 32, "tags": []},
		     {"name": "M33", "type": "INC", "flags": 0, "kw": 0, "value": 33, "tags": []},
		     {"name": "M34", "type": "INC", "flags": 0, "kw": 0, "value": 34, "tags": []},
		     {"name": "M35", "type": "INC", "flags": 0, "kw": 0, "value": 35, "tags": []},
		     {"name": "M36", "type": "INC", "flags": 0, "kw": 0, "value": 36, "tags": []},
		     {"name": "M37", "type": "INC", "flags": 0, "kw": 0, "value": 37, "tags": []},
		     {"name": "M38", "type": "INC", "flags": 0, "kw": 0, "value": 38, "tags": []},
		     {"name": "M39", "type": "INC", "flags": 0, "kw": 0, "value": 39, "tags": []},
		     {"name": "M40", "type": "INC", "flags": 0, "kw": 0, "value": 40, "tags": []},
		     {"name": "M41", "type": "INC", "flags": 0, "kw": 0, "value": 41, "tags": []},
		     {"name": "M42", "type": "INC", "flags": 0, "kw": 0, "value": 42, "tags": []},
		     {"name": "M43", "type": "INC", "flags": 0, "kw": 0, "value": 43, "tags": []},
		     {"name": "M44", "type": "INC", "flags": 0, "kw": 0, "value": 44, "tags": []},
		     {"name": "M45", "type": "INC", "flags": 0, "kw": 0, "value": 45, "tags": []}
		]}}}`)
	s := ssvSpec(bin, "lua.json", "go.json")
	difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "forty-five-findings"}, mk)
}

// TestSkillStatMapVerifyEmptySkillPartList: A PRESENT-BUT-EMPTY skillPartList, WHICH IS NOT A REFUSAL ON EITHER SIDE.
func TestSkillStatMapVerifyEmptySkillPartList(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssvSandbox(t, repo, `{
		 "skillCount": 2,
		 "entryCount": 2,
		 "modCount": 3,
		 "skillTypeNames": {
		  "1": "Attack"
		 },
		 "skills": {
		  "AlphaSkill": {
		   "entries": {
		    "damage": {
		     "scalars": {
		      "mult": {
		       "__number": 2
		      }
		     },
		     "mods": [
		      {
		       "name": "D",
		       "type": "INC",
		       "value": {
		        "__number": 1
		       },
		       "flags": 0,
		       "keywordFlags": 0,
		       "tags": [
		        {
		         "__table": true,
		         "type": {
		          "__string": "SkillPart"
		         },
		         "skillPartList": {
		          "__table": true
		         }
		        }
		       ]
		      }
		     ]
		    }
		   }
		  },
		  "BetaSkill": {
		   "entries": {
		    "speed": {
		     "scalars": {},
		     "mods": [
		      {
		       "name": "Speed",
		       "type": "MORE",
		       "value": {
		        "__number": 10
		       },
		       "flags": 0,
		       "keywordFlags": 0,
		       "tags": []
		      }
		     ]
		    }
		   }
		  }
		 }
		}`, ssvGoDump())
	difftest.RunFixture(t, t.Context(), ssvSpec(bin, "lua.json", "go.json"),
		difftest.Fixture{Name: "empty-skillpartlist"}, mk)
}

// TestSkillStatMapVerifyEntryKeysAreSorted: FOURTEEN ENTRY KEYS ON ONE SIDE ONLY, BECAUSE THE HAPPY FIXTURE HAS ZERO.
func TestSkillStatMapVerifyEntryKeysAreSorted(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssvSandbox(t, repo, `{
		 "skillCount": 2,
		 "entryCount": 2,
		 "modCount": 3,
		 "skillTypeNames": {
		  "1": "Attack"
		 },
		 "skills": {
		  "AlphaSkill": {
		   "entries": {
		    "zeta": {
		     "scalars": {},
		     "mods": []
		    },
		    "alpha": {
		     "scalars": {},
		     "mods": []
		    },
		    "mu": {
		     "scalars": {},
		     "mods": []
		    },
		    "beta": {
		     "scalars": {},
		     "mods": []
		    },
		    "omega": {
		     "scalars": {},
		     "mods": []
		    },
		    "gamma": {
		     "scalars": {},
		     "mods": []
		    },
		    "delta": {
		     "scalars": {},
		     "mods": []
		    },
		    "kappa": {
		     "scalars": {},
		     "mods": []
		    },
		    "iota": {
		     "scalars": {},
		     "mods": []
		    },
		    "theta": {
		     "scalars": {},
		     "mods": []
		    },
		    "lambda": {
		     "scalars": {},
		     "mods": []
		    },
		    "sigma": {
		     "scalars": {},
		     "mods": []
		    },
		    "nu": {
		     "scalars": {},
		     "mods": []
		    },
		    "xi": {
		     "scalars": {},
		     "mods": []
		    }
		   }
		  },
		  "BetaSkill": {
		   "entries": {
		    "speed": {
		     "scalars": {},
		     "mods": [
		      {
		       "name": "Speed",
		       "type": "MORE",
		       "value": {
		        "__number": 10
		       },
		       "flags": 0,
		       "keywordFlags": 0,
		       "tags": []
		      }
		     ]
		    }
		   }
		  }
		 }
		}`, ssvGoDump())
	difftest.RunFixture(t, t.Context(), ssvSpec(bin, "lua.json", "go.json"),
		difftest.Fixture{Name: "unsorted-entry-keys"}, mk)
}

// TestSkillStatMapVerifyGoTotalsWalkTheWholeDump: A SKILL THE LUA DUMP NEVER MENTIONS.
func TestSkillStatMapVerifyGoTotalsWalkTheWholeDump(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssvSandbox(t, repo, `{"skillCount": 2, "entryCount": 2, "modCount": 3, "skillTypeNames": {"1": "Attack"},
		 "skills": {
		   "AlphaSkill": {"entries": {"damage": {"scalars": {"mult": {"__number": 2}}, "mods": [
		     {"name": "Damage", "type": "INC", "value": {"__number": 25}, "flags": 0, "keywordFlags": 0,
		      "tags": [{"__table": true, "type": {"__string": "Multiplier"}, "var": {"__string": "X"}}]},
		     {"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
		      "value": {"__table": true, "key": {"__string": "showAverage"}, "value": {"__bool": true}}}]}}},
		   "BetaSkill": {"entries": {"speed": {"scalars": {}, "mods": [
		     {"name": "Speed", "type": "MORE", "value": {"__number": 10}, "flags": 0, "keywordFlags": 0,
		      "tags": []}]}}}}}`, `{
		 "AlphaSkill": {
		  "damage": {
		   "value": null,
		   "mult": 2,
		   "div": null,
		   "base": null,
		   "mods": [
		    {
		     "name": "Damage",
		     "type": "INC",
		     "value": 25,
		     "flags": 0,
		     "kw": 0,
		     "tags": [
		      {
		       "type": "Multiplier",
		       "vars": [
		        "X"
		       ],
		       "div": 1,
		       "base": 0,
		       "limit": null,
		       "limitVar": null,
		       "limitTotal": false,
		       "actor": ""
		      }
		     ]
		    },
		    {
		     "name": "SkillData",
		     "type": "LIST",
		     "value": {
		      "key": "showAverage",
		      "value": 1
		     },
		     "flags": 0,
		     "kw": 0,
		     "tags": []
		    }
		   ]
		  }
		 },
		 "BetaSkill": {
		  "speed": {
		   "value": null,
		   "mult": null,
		   "div": null,
		   "base": null,
		   "mods": [
		    {
		     "name": "Speed",
		     "type": "MORE",
		     "value": 10,
		     "flags": 0,
		     "kw": 0,
		     "tags": []
		    }
		   ]
		  }
		 },
		 "ZetaSkill": {
		  "extra": {
		   "value": null,
		   "mult": null,
		   "div": null,
		   "base": null,
		   "mods": [
		    {
		     "name": "Z",
		     "type": "INC",
		     "value": 1,
		     "flags": 0,
		     "kw": 0,
		     "tags": []
		    }
		   ]
		  }
		 }
		}`)
	difftest.RunFixture(t, t.Context(), ssvSpec(bin, "lua.json", "go.json"),
		difftest.Fixture{Name: "go-dump-has-an-extra-skill"}, mk)
}

// TestSkillStatMapVerifyNestedModRepr: THE NESTED MOD MUST **MISMATCH** OR ITS REPR NEVER REACHES THE OUTPUT — and the mutation control is what proved it.
func TestSkillStatMapVerifyNestedModRepr(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssvSandbox(t, repo, `{"skillCount": 1, "entryCount": 1, "modCount": 6, "skillTypeNames": {"1": "Attack", "7": "Trap"},
		 "skills": {"AlphaSkill": {"entries": {"damage": {"scalars":
		   {"value": {"__number": 3}, "mult": {"__number": 2}, "div": {"__number": 4}, "base": {"__number": 1}},
		   "mods": [
		    {"name": "MinionModifier", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
		     "value": {"__table": true, "mod": {"__table": true, "name": {"__string": "Damage"},
		       "type": {"__string": "INC"}, "value": {"__number": 9}, "flags": {"__number": 2},
		       "1": {"__table": true, "type": {"__string": "SkillId"}, "skillId": {"__string": "Zap"}}}}},
		    {"name": "SupportedGemProperty", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
		     "value": {"__table": true, "keyword": {"__string": "aura"}, "key": {"__string": "level"},
		               "value": {"__number": 2}}},
		    {"name": "ExtraMinionSkill", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
		     "value": {"__table": true, "skillId": {"__string": "Golem"}}},
		    {"name": "Ramps", "type": "BASE", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0,
		     "tags": [{"__table": true, "type": {"__string": "DistanceRamp"},
		               "ramp": {"__table": true, "1": {"__table": true, "1": {"__number": 0}, "2": {"__number": 1}},
		                        "2": {"__table": true, "1": {"__number": 40}, "2": {"__number": 0.5}}}},
		              {"__table": true, "type": {"__string": "MeleeProximity"},
		               "ramp": {"__table": true, "1": {"__number": 0}, "2": {"__number": 10}}}]},
		    {"name": "Flags", "type": "MORE", "value": {"__number": 5}, "flags": 0, "keywordFlags": 0,
		     "tags": [{"__table": true, "type": {"__string": "ModFlagOr"}, "modFlags": {"__number": 3}},
		              {"__table": true, "type": {"__string": "SkillName"}, "skillName": {"__string": "Fire"}},
		              {"__table": true, "type": {"__string": "StatThreshold"}, "stat": {"__string": "Str"},
		               "threshold": {"__number": 20}, "upper": {"__bool": true}},
		              {"__table": true, "type": {"__string": "PercentStat"}, "stat": {"__string": "Life"},
		               "percentVar": {"__string": "V"}},
		              {"__table": true, "type": {"__string": "ActorCondition"}, "actor": {"__string": "enemy"},
		               "varList": {"__table": true, "1": {"__string": "A"}, "2": {"__string": "B"}}},
		              {"__table": true, "type": {"__string": "MultiplierThreshold"}, "var": {"__string": "M"},
		               "threshold": {"__number": 4}},
		              {"__table": true, "type": {"__string": "GlobalEffect"}, "effectType": {"__string": "Buff"}},
		              {"__table": true, "type": {"__string": "SkillType"}, "skillType": {"__number": 7}},
		              {"__table": true, "type": {"__string": "NoSuchTagType"}}]},
		    {"type": "LIST", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0, "tags": []}
		   ]}}}}}`, `{
		 "AlphaSkill": {
		  "damage": {
		   "value": 3,
		   "mult": 2,
		   "div": 4,
		   "base": 1,
		   "mods": [
		    {
		     "name": "MinionModifier",
		     "type": "LIST",
		     "flags": 0,
		     "kw": 0,
		     "tags": [],
		     "value": {
		      "mod": {
		       "name": "Damage",
		       "type": "INC",
		       "value": 77,
		       "flags": 2,
		       "kw": 0,
		       "tags": [
		        {
		         "type": "SkillId",
		         "skillId": "Zap"
		        }
		       ]
		      }
		     }
		    },
		    {
		     "name": "SupportedGemProperty",
		     "type": "LIST",
		     "flags": 0,
		     "kw": 0,
		     "tags": [],
		     "value": {
		      "keyword": "aura",
		      "propKey": "level",
		      "value": 2
		     }
		    },
		    {
		     "name": "ExtraMinionSkill",
		     "type": "LIST",
		     "flags": 0,
		     "kw": 0,
		     "tags": [],
		     "value": {
		      "skillId": "Golem"
		     }
		    },
		    {
		     "name": "Ramps",
		     "type": "BASE",
		     "value": 1,
		     "flags": 0,
		     "kw": 0,
		     "tags": [
		      {
		       "type": "DistanceRamp",
		       "ramp": [
		        [
		         0,
		         1
		        ],
		        [
		         40,
		         0.5
		        ]
		       ]
		      },
		      {
		       "type": "MeleeProximity",
		       "ramp": [
		        0,
		        10
		       ]
		      }
		     ]
		    },
		    {
		     "name": "Flags",
		     "type": "MORE",
		     "value": 5,
		     "flags": 0,
		     "kw": 0,
		     "tags": [
		      {
		       "type": "ModFlagOr",
		       "modFlags": 3
		      },
		      {
		       "type": "SkillName",
		       "names": [
		        "Fire"
		       ],
		       "neg": false
		      },
		      {
		       "type": "StatThreshold",
		       "stat": "Str",
		       "threshold": 20,
		       "upper": true
		      },
		      {
		       "type": "PercentStat",
		       "stat": "Life",
		       "percent": 0,
		       "percentVar": "V"
		      },
		      {
		       "type": "ActorCondition",
		       "actor": "enemy",
		       "vars": [
		        "A",
		        "B"
		       ],
		       "neg": false
		      },
		      {
		       "type": "MultiplierThreshold",
		       "var": "M",
		       "threshold": 4,
		       "upper": false,
		       "actor": ""
		      },
		      {
		       "type": "GlobalEffect",
		       "effects": [
		        "Buff"
		       ],
		       "name": "",
		       "unscalable": false
		      },
		      {
		       "type": "SkillType",
		       "skillTypes": [
		        "Trap"
		       ],
		       "neg": false
		      },
		      {
		       "type": "NoSuchTagType"
		      }
		     ]
		    }
		   ]
		  }
		 }
		}`)
	difftest.RunFixture(t, t.Context(), ssvSpec(bin, "lua.json", "go.json"),
		difftest.Fixture{Name: "nested-mod-mismatch"}, mk)
}

// TestSkillStatMapVerifyKeyDeletion: THE **REVERSE** DIRECTION OF THE ONE-SIDED-REFUSAL CLASS, WHICH NO LEXICAL CHECK CAN SEE.
func TestSkillStatMapVerifyKeyDeletion(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssvSandbox(t, repo, ssvLuaDump(), ssvGoDump())
	difftest.AssertKeyDeletionAgrees(t, t.Context(), ssvSpec(bin, "lua.json", "go.json"),
		[]string{"lua.json", "go.json"}, mk)
}
