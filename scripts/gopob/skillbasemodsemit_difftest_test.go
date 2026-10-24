package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const (
	sbmPy       = "oracle/skill_base_mods_emit.py"
	mcPy        = "oracle/mod_common.py"
	sbmTypesGo  = "types.go"
	sbmOutGo    = "out.go"
	sbmTypesSrc = `package calculator

// A trimmed stand-in for the real struct. Only the FIELD NAMES matter: utils.Set resolves a
// SkillData mod's key by field name, so a key with no field here is dropped at runtime.
type SkillData struct {
	CritChance          float64
	CannotBeEvaded      bool
	ShowAverage         bool
	DebuffTertiary      float64
}

type Other struct {
	Ignored float64
}
`
)

func sbmDump() string {
	return `{
  "skillCount": 9,
  "modCount": 18,
  "skillTypeNames": {"1": "Attack", "7": "Trappable"},
  "modFlagBits": {
    "Attack": 1, "Spell": 2, "Hit": 4, "Dot": 8, "Cast": 16,
    "Melee": 256, "Area": 512, "Projectile": 1024, "SourceMask": 1536,
    "Ailment": 2048, "MeleeHit": 4096, "Weapon": 8192,
    "Axe": 65536, "Bow": 131072, "Claw": 262144, "Dagger": 524288,
    "Mace": 1048576, "Staff": 2097152, "Sword": 4194304, "Wand": 8388608,
    "Unarmed": 16777216, "Fishing": 33554432, "GhostMask": 32,
    "WeaponMelee": 67108864, "WeaponRanged": 134217728,
    "Weapon1H": 268435456, "Weapon2H": 536870912, "WeaponMask": 788987904
  },
  "keywordFlagBits": {
    "Aura": 1, "Curse": 2, "Warcry": 4, "Movement": 8, "Physical": 16,
    "Fire": 32, "Cold": 64, "Lightning": 128, "Chaos": 256, "Vaal": 512,
    "Bow": 1024, "Arrow": 2048, "Trap": 4096, "Mine": 8192, "Totem": 16384,
    "Minion": 32768, "Attack": 65536, "Spell": 131072, "Hit": 262144,
    "MatchAll": 1073741824
  },
  "skills": {
    "AlphaSkill": {
      "loc": "Data/Skills/act_str.lua:100",
      "mods": [
        {"name": "Damage", "type": "INC", "value": {"__number": 25}, "flags": 257,
         "keywordFlags": 65536, "tags": []},
        {"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
         "value": {"__table": true, "key": {"__string": "critChance"}, "value": {"__number": 6.5}}},
        {"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
         "value": {"__table": true, "key": {"__string": "cannotBeEvaded"}, "value": {"__bool": true}}}
      ]
    },
    "BetaSkill": {
      "loc": "Data/Skills/act_dex.lua:200",
      "mods": [
        {"name": "Speed", "type": "MORE", "value": {"__number": 10}, "flags": 0, "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "Multiplier"},
                   "var": {"__string": "Frenzy"}, "limit": {"__number": 3},
                   "actor": {"__string": "parent"}}]},
        {"name": "Life", "type": "BASE", "value": {"__number": 40}, "flags": 0, "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "PerStat"},
                   "stat": {"__string": "Str"}, "div": {"__number": 5},
                   "base": {"__number": 2}}]},
        {"name": "Armour", "type": "INC", "value": {"__number": 15}, "flags": 0, "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "Condition"},
                   "var": {"__string": "Blocking"}, "neg": {"__bool": true}}]}
      ]
    },
    "GammaSkill": {
      "loc": "Data/Skills/act_int.lua:300",
      "mods": [
        {"name": "Damage", "type": "MAX", "value": {"__number": 3}, "flags": 0, "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "SkillType"},
                   "skillType": {"__number": 7}}]},
        {"name": "AreaOfEffect", "type": "INC", "value": {"__number": 20}, "flags": 0,
         "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "SkillPart"},
                   "skillPartList": {"__table": true, "1": {"__number": 1}, "2": {"__number": 2}}}]},
        {"name": "Buff", "type": "FLAG", "value": {"__bool": true}, "flags": 0, "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "GlobalEffect"},
                   "effectType": {"__string": "Buff"}, "effectName": {"__string": "Fortify"}}]}
      ]
    },
    "DeltaSkill": {
      "loc": "Data/Skills/other.lua:400",
      "mods": [
        {"name": "MinionDamage", "type": "INC", "value": {"__number": 30}, "flags": 0,
         "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "ActorCondition"},
                   "actor": {"__string": "enemy"}, "var": {"__string": "Shocked"}}]},
        {"name": "ExtraMinionSkill", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
         "value": {"__table": true, "skillId": {"__string": "MinionInstability"}}},
        {"name": "HitTime", "type": "BASE", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "Skill"},
                   "skillPartList": {"__table": true, "1": {"__number": 1}, "2": {"__number": 2}}}]}
      ]
    },
    "NoLoc": {
      "mods": [
        {"name": "Life", "type": "BASE", "value": {"__number": 5}, "flags": 0, "keywordFlags": 0,
         "tags": [{"__table": true, "type": {"__string": "MultiplierThreshold"},
                   "var": {"__string": "Rage"}, "threshold": {"__number": 10},
                   "upper": {"__bool": true}}]}
      ]
    },
    "AllOmitted": {
      "loc": "Data/Skills/other.lua:500",
      "mods": [
        {"name": "MinionModifier", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
         "value": {"__table": true}},
        {"name": "Unknown", "type": "WEIRD", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []}
      ]
    },
    "OmitTieA": {
      "loc": "Data/Skills/other.lua:600",
      "mods": [
        {"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
         "value": {"__table": true, "key": {"__string": "radiusLabel"},
                   "value": {"__string": "Radius"}}},
        {"name": "Kept", "type": "BASE", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0,
         "tags": []}
      ]
    },
    "OmitTieB": {
      "loc": "Data/Skills/other.lua:700",
      "mods": [
        {"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
         "value": {"__table": true, "key": {"__string": "noSuchField"}, "value": {"__number": 1}}},
        {"name": "Kept", "type": "BASE", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0,
         "tags": []}
      ]
    },
    "UnnamedBit": {
      "loc": "Data/Skills/other.lua:800",
      "mods": [
        {"name": "Damage", "type": "INC", "value": {"__number": 5}, "flags": 1073741824,
         "keywordFlags": 0, "tags": []},
        {"name": "Kept", "type": "BASE", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0,
         "tags": []}
      ]
    },
    "EmptyLoc": {
      "loc": "",
      "mods": [
        {"name": "Kept", "type": "BASE", "value": {"__number": 2}, "flags": 0, "keywordFlags": 0,
         "tags": []}
      ]
    },
    "MaskBits": {
      "loc": "Data/Skills/other.lua:900",
      "mods": [
        {"name": "AreaDamage", "type": "INC", "value": {"__number": 7}, "flags": 512,
         "keywordFlags": 0, "tags": []},
        {"name": "PhysKeyword", "type": "INC", "value": {"__number": 3}, "flags": 0,
         "keywordFlags": 16, "tags": []},
        {"name": "GhostBit", "type": "INC", "value": {"__number": 9}, "flags": 32,
         "keywordFlags": 0, "tags": []}
      ]
    },
    "ZHeavy": {
      "loc": "Data/Skills/other.lua:1100",
      "mods": [
        {"name": "H1", "type": "HEAVY", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0, "tags": []},
        {"name": "H2", "type": "HEAVY", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0, "tags": []},
        {"name": "H3", "type": "HEAVY", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0, "tags": []},
        {"name": "H4", "type": "HEAVY", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0, "tags": []},
        {"name": "M1", "type": "MID", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0, "tags": []},
        {"name": "M2", "type": "MID", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0, "tags": []},
        {"name": "Kept", "type": "BASE", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0,
         "tags": []}
      ]
    },
    "TieFarm": {
      "loc": "Data/Skills/other.lua:1000",
      "mods": [
        {"name": "M01", "type": "T01", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M02", "type": "T02", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M03", "type": "T03", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M04", "type": "T04", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M05", "type": "T05", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M06", "type": "T06", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M07", "type": "T07", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M08", "type": "T08", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M09", "type": "T09", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M10", "type": "T10", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M11", "type": "T11", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M12", "type": "T12", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M13", "type": "T13", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "M14", "type": "T14", "value": {"__number": 1}, "flags": 0,
         "keywordFlags": 0, "tags": []},
        {"name": "Kept", "type": "BASE", "value": {"__number": 1}, "flags": 0, "keywordFlags": 0,
         "tags": []}
      ]
    }
  }
}`
}

func sbmSandbox(t *testing.T, repo, dump string) func() (string, error) {
	t.Helper()
	src := map[string]string{}
	for _, rel := range []string{mcPy, sbmPy} {
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
		files := map[string]string{"dump.json": dump, sbmTypesGo: sbmTypesSrc}
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

func sbmPut(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

const sbmMinimal = `{"skillCount": 1, "modCount": 1, "skillTypeNames": {},
  "modFlagBits": {"Attack": 1}, "keywordFlagBits": {"Aura": 1},
  "skills": {"A": {"mods": [{"name": "Life", "type": "BASE", "value": {"__number": 1},
                            "flags": 0, "keywordFlags": 0, "tags": []}]}}}`

func TestSkillBaseModsEmit(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := sbmSandbox(t, repo, sbmDump())
	pySrc := difftest.ReadOriginal(t, repo, sbmPy)

	base := []string{"dump.json", sbmOutGo, sbmTypesGo}
	spec := difftest.Spec{
		Tool:         "skill-base-mods-emit",
		PythonScript: sbmPy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"skill-base-mods-emit"}, base...),
		Stderr:       difftest.StderrExact,
		Files:        []string{sbmOutGo},
		Library:      []string{mcPy},
		RefusalSites: 2,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "usage: skill_base_mods_emit.py <dump.json> <out.go>",
				Break: func(string) error { return nil }},
			{Name: "four-args", Refusal: "usage: skill_base_mods_emit.py <dump.json> <out.go>",
				Break: func(string) error { return nil }},

			{Name: "dump-missing", Refusal: "REFUSING: the dump (dump.json) is not readable: No such file or directory",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "dump.json")) }},
			{Name: "dump-not-json", Refusal: "REFUSING: the dump (dump.json) is not valid JSON",
				Break: func(sb string) error { return sbmPut(sb, "dump.json", "{ nope") }},
			{Name: "dump-not-an-object", Refusal: "REFUSING: dump.json is not a JSON object",
				Break: func(sb string) error { return sbmPut(sb, "dump.json", "[]") }},
			{Name: "dump-missing-a-field", Refusal: "REFUSING: the dump carries no 'modCount'",
				Break: func(sb string) error {
					return sbmPut(sb, "dump.json", `{"skillCount": 1, "skillTypeNames": {}, "skills": {}}`)
				}},
			{Name: "skills-is-an-array", Refusal: "REFUSING: the dump's 'skills' is not a JSON object",
				Break: func(sb string) error {
					return sbmPut(sb, "dump.json", `{"skillCount": 1, "modCount": 1,
						"skillTypeNames": {}, "skills": []}`)
				}},

			{Name: "dump-has-no-modFlagBits",
				Refusal: "REFUSING: the dump carries no 'modFlagBits'",
				Break: func(sb string) error {
					return sbmPut(sb, "dump.json", `{"skillCount": 1, "modCount": 1,
						"skillTypeNames": {}, "skills": {}, "keywordFlagBits": {"Aura": 1}}`)
				}},
			{Name: "dump-has-no-keywordFlagBits",
				Refusal: "REFUSING: the dump carries no 'keywordFlagBits'",
				Break: func(sb string) error {
					return sbmPut(sb, "dump.json", `{"skillCount": 1, "modCount": 1,
						"skillTypeNames": {}, "skills": {}, "modFlagBits": {"Attack": 1}}`)
				}},

			{Name: "types-go-missing",
				Refusal: "REFUSING: calculator/types.go (types.go) is not readable: No such file or directory",
				Break:   func(sb string) error { return os.Remove(filepath.Join(sb, sbmTypesGo)) }},
			{Name: "types-go-has-no-SkillData",
				Refusal: "declares no `type SkillData struct {`",
				Break:   func(sb string) error { return sbmPut(sb, sbmTypesGo, "package calculator\n") }},
			{Name: "SkillData-has-no-fields",
				Refusal: "SkillData has no fields; every skill() entry would be dropped",
				Break: func(sb string) error {
					return sbmPut(sb, sbmTypesGo, "package calculator\n\ntype SkillData struct {\n}\n")
				}},

			{Name: "output-is-a-directory",
				Refusal: "REFUSING: out.go is not writable: Is a directory",
				Break: func(sb string) error {
					if err := sbmPut(sb, "dump.json", sbmMinimal); err != nil {
						return err
					}
					return os.Mkdir(filepath.Join(sb, sbmOutGo), 0o750)
				}},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			switch f.Name {
			case "no-args":
				s.PythonArgs = nil
			case "four-args":
				s.PythonArgs = []string{"dump.json", sbmOutGo, sbmTypesGo, "extra"}
			}
			if f.Name != "happy" {
				s.Files = nil
			}
			s.GoArgs = append([]string{"skill-base-mods-emit"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}

	difftest.AssertKeyDeletionAgrees(t, t.Context(), spec, []string{"dump.json"}, mk)
}

// TestSkillBaseModsEmitTallyOrderIsStableAcrossRuns: REPEATED: the omission tally is a Counter and the header prints most_common().
func TestSkillBaseModsEmitTallyOrderIsStableAcrossRuns(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := sbmSandbox(t, repo, sbmDump())
	base := []string{"dump.json", sbmOutGo, sbmTypesGo}
	spec := difftest.Spec{
		Tool: "skill-base-mods-emit", PythonScript: sbmPy, PythonArgs: base,
		GoBin: bin, GoArgs: append([]string{"skill-base-mods-emit"}, base...),
		Stderr: difftest.StderrExact, Files: []string{sbmOutGo},
		Library: []string{mcPy}, RefusalSites: 2,
	}
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprintf("run%d", i), func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), spec, difftest.Fixture{Name: "happy"}, mk)
		})
	}
}

func TestSkillBaseModsEmitPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := sbmSandbox(t, repo, sbmDump())
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool: "skill-base-mods-emit", PythonScript: sbmPy,
		PythonArgs: []string{"dump.json", sbmOutGo, sbmTypesGo},
		Files:      []string{sbmOutGo},
	}, mk, []string{"0", "12345", "999983"})
}
