package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const (
	ssmPy        = "oracle/skill_stat_map_emit.py"
	ssmGlobalGo  = "skill_stat_map.go"
	ssmOutGo     = "skill_stat_maps.go"
	ssmGlobalSrc = `package raw

var SkillStatMap = map[string]*StatMap{
	"shadowed_empty_stat": {
		Mods: []mod.Mod{},
	},
	"other_global_stat": {
		Mods: []mod.Mod{},
	},
	"declared_empty_stat": {
		Mods: []mod.Mod{},
	},
}
`
)

func ssmDump() string {
	return `{
  "skillCount": 5,
  "entryCount": 29,
  "modCount": 32,
  "groupCount": 5,
  "skillTypeNames": {"1": "Attack", "7": "Trappable"},
  "modFlagBits": {"Attack": 1, "Spell": 2, "Hit": 4, "Melee": 256, "Area": 512,
                  "Projectile": 1024, "SourceMask": 1536, "Weapon": 8192},
  "keywordFlagBits": {"Aura": 1, "Curse": 2, "Physical": 16, "Fire": 32, "Attack": 65536,
                      "MatchAll": 1073741824},
  "skills": {
    "AlphaSkill": {
      "loc": "src/Data/Skills/act_str.lua:100",
      "entries": {
        "energy_shield_delay_-%": {
          "scalars": {"__table": true, "value": {"__number": 1}},
          "mods": [{"name": "EnergyShieldRechargeFaster", "type": "INC",
                    "value": {"__number": 1}, "flags": 0, "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "GlobalEffect"},
                              "effectType": {"__string": "Aura"},
                              "effectName": {"__string": "Discipline"},
                              "unscalable": {"__bool": true},
                              "effectCond": {"__string": "HaveDiscipline"},
                              "effectEnemyCond": {"__string": "Shocked"},
                              "modCond": {"__string": "Focused"},
                              "effectStackVar": {"__string": "Rage"},
                              "effectStackLimit": {"__number": 5},
                              "effectStackLimitVar": {"__string": "MaxRage"},
                              "applyNotPlayer": {"__bool": true},
                              "applyMinions": {"__bool": true}}]}]
        },
        "multiplier_stat": {
          "scalars": {"__table": true, "mult": {"__number": 2}, "div": {"__number": 3},
                      "base": {"__number": 4}},
          "mods": [{"name": "Damage", "type": "MORE", "value": {"__number": 5}, "flags": 257,
                    "keywordFlags": 65536,
                    "tags": [{"__table": true, "type": {"__string": "Multiplier"},
                              "var": {"__string": "Virulence"}, "limitVar": {"__string": "MaxV"},
                              "div": {"__number": 2}, "limit": {"__number": 40},
                              "limitTotal": {"__bool": true}, "actor": {"__string": "parent"},
                              "limitActor": {"__string": "enemy"}}]}]
        },
        "multiplier_unmodelled": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "Multiplier"},
                              "var": {"__string": "X"}, "noFloor": {"__bool": true}}]}]
        },
        "multiplier_unrepresentable": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "Multiplier"},
                              "var": {"__string": "X"}, "nosuchkey": {"__bool": true}}]}]
        },
        "threshold_stat": {
          "scalars": [],
          "mods": [{"name": "Life", "type": "BASE", "value": {"__number": 10}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "MultiplierThreshold"},
                              "var": {"__string": "Rage"}, "thresholdVar": {"__string": "MaxRage"},
                              "upper": {"__bool": true}, "actor": {"__string": "parent"}}]}]
        },
        "globaleffect_unmodelled": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "GlobalEffect"},
                              "effectType": {"__string": "Buff"},
                              "nosuchkey": {"__bool": true}}]}]
        },
        "threshold_unmodelled": {
          "scalars": [],
          "mods": [{"name": "Life", "type": "BASE", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "MultiplierThreshold"},
                              "var": {"__string": "Rage"},
                              "varList": {"__table": true, "1": {"__string": "A"}}}]}]
        },
        "threshold_zero": {
          "scalars": [],
          "mods": [{"name": "Life", "type": "BASE", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "MultiplierThreshold"},
                              "var": {"__string": "Rage"}, "threshold": {"__number": 0},
                              "thresholdVar": {"__string": "Fallback"}}]}]
        },
        "nameless_mod": {
          "scalars": [],
          "mods": [{"flags": 0, "keywordFlags": 0,
                    "tags": [{"__table": true, "name": {"__string": "DamageTaken"},
                              "type": {"__string": "MORE"}}]}]
        },
        "threshold_neither": {
          "scalars": [],
          "mods": [{"name": "Life", "type": "BASE", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "MultiplierThreshold"},
                              "var": {"__string": "Rage"}}]}]
        }
      }
    },
    "BetaSkill": {
      "loc": "src/Data/Skills/act_dex.lua:200",
      "entries": {
        "perstat_list": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "PerStat"},
                              "statList": {"__table": true, "1": {"__string": "Str"},
                                           "2": {"__string": "Dex"}},
                              "div": {"__number": 10}, "limitVar": {"__string": "Cap"},
                              "limit": {"__number": 50}, "limitTotal": {"__bool": true},
                              "actor": {"__string": "parent"}}]}]
        },
        "percentstat_literal": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "BASE", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "PercentStat"},
                              "stat": {"__string": "Mana"}, "percent": {"__number": 25}}]}]
        },
        "percentstat_var": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "BASE", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "PercentStat"},
                              "stat": {"__string": "Mana"},
                              "percentVar": {"__string": "ArcaneCloakConsumed"}}]}]
        },
        "condition_list": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "Condition"},
                              "varList": {"__table": true, "1": {"__string": "A"},
                                          "2": {"__string": "B"}},
                              "neg": {"__bool": true}},
                             {"__table": true, "type": {"__string": "ActorCondition"},
                              "actor": {"__string": "enemy"}, "var": {"__string": "Shocked"}}]}]
        },
        "skilltype_named": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "MAX", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "SkillType"},
                              "skillType": {"__number": 7}, "neg": {"__bool": true}}]}]
        },
        "skilltype_absent": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "MAX", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "SkillType"},
                              "neg": {"__bool": true}}]}]
        }
      }
    },
    "GammaSkill": {
      "entries": {
        "misc_tags": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "SkillName"},
                              "skillName": {"__string": "Cyclone"}, "neg": {"__bool": true}},
                             {"__table": true, "type": {"__string": "SkillId"},
                              "skillId": {"__string": "CycloneId"}},
                             {"__table": true, "type": {"__string": "SkillPart"},
                              "skillPartList": {"__table": true, "1": {"__number": 1},
                                                "2": {"__number": 2}}},
                             {"__table": true, "type": {"__string": "StatThreshold"},
                              "stat": {"__string": "Str"}, "threshold": {"__number": 100},
                              "upper": {"__bool": true}}]}]
        },
        "ramp_tags": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "MORE", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "DistanceRamp"},
                              "ramp": {"__table": true,
                                       "1": {"__table": true, "1": {"__number": 0},
                                             "2": {"__number": 1}},
                                       "2": {"__table": true, "1": {"__number": 40},
                                             "2": {"__number": 0}}}},
                             {"__table": true, "type": {"__string": "ModFlagOr"},
                              "modFlags": {"__number": 258}},
                             {"__table": true, "type": {"__string": "Limit"},
                              "limitVar": {"__string": "Cap"}}]}]
        },
        "meleeproximity": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "MORE", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "MeleeProximity"},
                              "ramp": {"__table": true, "1": {"__number": 10},
                                       "2": {"__number": 20}}}]}]
        },
        "nondict_tag": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0, "tags": [{"__number": 2}]}]
        },
        "unknown_tag": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0,
                    "tags": [{"__table": true, "type": {"__string": "NoSuchTagType"}}]}]
        }
      }
    },
    "DeltaSkill": {
      "loc": "src/Data/Skills/other.lua:400",
      "entries": {
        "skilldata_true": {
          "scalars": [],
          "mods": [{"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
                    "value": {"__table": true, "key": {"__string": "showAverage"},
                              "value": {"__bool": true}}}]
        },
        "skilldata_false": {
          "scalars": [],
          "mods": [{"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
                    "value": {"__table": true, "key": {"__string": "dotIsSpell"},
                              "value": {"__bool": false}}}]
        },
        "skilldata_absent": {
          "scalars": [],
          "mods": [{"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
                    "value": {"__table": true, "key": {"__string": "levelPlaceholder"}}}]
        },
        "skilldata_merge": {
          "scalars": [],
          "mods": [{"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
                    "value": {"__table": true, "key": {"__string": "chaosDot"},
                              "value": {"__number": 5}, "merge": {"__string": "MAX"}}}]
        },
        "skilldata_unmodelled": {
          "scalars": [],
          "mods": [{"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
                    "value": {"__table": true, "key": {"__string": "k"},
                              "value": {"__number": 1}, "nosuchfield": {"__number": 1}}}]
        },
        "list_mods": {
          "scalars": [],
          "mods": [{"name": "ExtraMinionSkill", "type": "LIST", "flags": 0, "keywordFlags": 0,
                    "tags": [],
                    "value": {"__table": true, "skillId": {"__string": "MinionInstability"}}},
                   {"name": "SupportedGemProperty", "type": "LIST", "flags": 0, "keywordFlags": 0,
                    "tags": [],
                    "value": {"__table": true, "keyword": {"__string": "aura"},
                              "key": {"__string": "level"}}},
                   {"name": "MinionModifier", "type": "LIST", "flags": 0, "keywordFlags": 0,
                    "tags": [],
                    "value": {"__table": true, "mod": {"__table": true,
                              "name": {"__string": "DamageTaken"}, "type": {"__string": "MORE"},
                              "1": {"__table": true, "type": {"__string": "GlobalEffect"},
                                    "effectType": {"__string": "Buff"}}}}}]
        },
        "shadowed_empty_stat": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "NOSUCHTYPE", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0, "tags": []}]
        },
        "unshadowed_empty_stat": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "NOSUCHTYPE", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0, "tags": []}]
        },
        "unnamed_bit": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 4096,
                    "keywordFlags": 0, "tags": []}]
        },
        "declared_empty_stat": {
          "scalars": [],
          "mods": []
        },
        "declared_empty_unshadowed": {
          "scalars": [],
          "mods": []
        },
        "nameless_mod_with_extra_key": {
          "scalars": [],
          "mods": [{"value": {"__other": "nil"}, "flags": 0, "keywordFlags": 0,
                    "extraKeys": ["div"],
                    "tags": [{"__table": true, "name": {"__string": "LifeRegen"},
                              "type": {"__string": "BASE"}, "flags": {"__number": 0},
                              "keywordFlags": {"__number": 0}}]}]
        },
        "named_mod_with_extra_key": {
          "scalars": [],
          "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0, "tags": [], "extraKeys": ["nosuchmodkey"]}]
        }
      }
    },
    "EpsilonSkill": {
      "loc": "src/Data/Skills/act_str.lua:500",
      "entries": {
        "two_groups_two_scalars": {
          "scalars": [],
          "mods": [],
          "groups": [
            {"scalars": {"__table": true, "div": {"__number": 60}},
             "mods": [{"name": "LifeRegen", "type": "BASE", "value": {"__number": 1},
                       "flags": 0, "keywordFlags": 0, "tags": []}]},
            {"scalars": {"__table": true, "div": {"__number": 400}, "base": {"__number": 2}},
             "mods": [{"name": "ManaRegen", "type": "BASE", "value": {"__number": 1},
                       "flags": 0, "keywordFlags": 0, "tags": []}]}
          ]
        },
        "bare_mod_beside_a_group": {
          "scalars": {"__table": true, "mult": {"__number": 7}},
          "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 0,
                    "keywordFlags": 0, "tags": []}],
          "groups": [
            {"scalars": {"__table": true, "mult": {"__number": -1}},
             "mods": [{"name": "DamageTaken", "type": "MORE", "value": {"__number": 1},
                       "flags": 0, "keywordFlags": 0, "tags": []}]}
          ]
        },
        "group_with_no_scalar": {
          "scalars": [],
          "mods": [],
          "groups": [
            {"scalars": [],
             "mods": [{"name": "Life", "type": "BASE", "value": {"__number": 1}, "flags": 0,
                       "keywordFlags": 0, "tags": []}]}
          ]
        },
        "group_wholly_omitted": {
          "scalars": [],
          "mods": [],
          "groups": [
            {"scalars": {"__table": true, "div": {"__number": 5}},
             "mods": [{"name": "Damage", "type": "NOSUCHTYPE", "value": {"__number": 1},
                       "flags": 0, "keywordFlags": 0, "tags": []}]}
          ]
        }
      }
    }
  }
}`
}

func ssmSandbox(t *testing.T, repo, dump string) func() (string, error) {
	t.Helper()
	src := map[string]string{}
	for _, rel := range []string{mcPy, ssmPy} {
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
		files := map[string]string{"dump.json": dump, ssmGlobalGo: ssmGlobalSrc}
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

func ssmPut(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

const ssmMinimal = `{"skillCount": 1, "entryCount": 1, "modCount": 1, "groupCount": 0, "skillTypeNames": {},
  "modFlagBits": {"Attack": 1}, "keywordFlagBits": {"Aura": 1},
  "skills": {"A": {"entries": {"s": {"scalars": [], "mods": [
    {"name": "Life", "type": "BASE", "value": {"__number": 1}, "flags": 0,
     "keywordFlags": 0, "tags": []}]}}}}}`

func TestSkillStatMapEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssmSandbox(t, repo, ssmDump())
	pySrc := difftest.ReadOriginal(t, repo, ssmPy)

	base := []string{"dump.json", ssmOutGo, ssmGlobalGo}
	spec := difftest.Spec{
		Tool:         "skill-stat-maps-emit",
		PythonScript: ssmPy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"skill-stat-maps-emit"}, base...),
		Stderr:       difftest.StderrExact,
		Files:        []string{ssmOutGo},
		Library:      []string{mcPy},
		RefusalSites: 2,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "usage: skill_stat_map_emit.py <dump.json> <out.go>",
				Break: func(string) error { return nil }},
			{Name: "four-args", Refusal: "usage: skill_stat_map_emit.py <dump.json> <out.go>",
				Break: func(string) error { return nil }},

			{Name: "dump-missing", Refusal: "REFUSING: the dump (dump.json) is not readable: No such file or directory",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "dump.json")) }},
			{Name: "dump-not-json", Refusal: "REFUSING: the dump (dump.json) is not valid JSON",
				Break: func(sb string) error { return ssmPut(sb, "dump.json", "{ nope") }},
			{Name: "dump-not-an-object", Refusal: "REFUSING: dump.json is not a JSON object",
				Break: func(sb string) error { return ssmPut(sb, "dump.json", "[]") }},
			{Name: "dump-missing-a-field", Refusal: "REFUSING: the dump carries no 'entryCount'",
				Break: func(sb string) error {
					return ssmPut(sb, "dump.json", `{"skillCount": 1, "modCount": 1,
						"skillTypeNames": {}, "skills": {}}`)
				}},
			{Name: "skills-is-an-array", Refusal: "REFUSING: the dump's 'skills' is not a JSON object",
				Break: func(sb string) error {
					return ssmPut(sb, "dump.json", `{"skillCount": 1, "entryCount": 1, "modCount": 1, "groupCount": 0,
						"skillTypeNames": {}, "skills": []}`)
				}},
			{Name: "dump-has-no-modFlagBits", Refusal: "REFUSING: the dump carries no 'modFlagBits'",
				Break: func(sb string) error {
					return ssmPut(sb, "dump.json", `{"skillCount": 1, "entryCount": 1, "modCount": 1, "groupCount": 0,
						"skillTypeNames": {}, "skills": {}, "keywordFlagBits": {"Aura": 1}}`)
				}},
			{Name: "a-skill-has-no-entries", Refusal: "REFUSING: skill 'A' has no 'entries' object",
				Break: func(sb string) error {
					return ssmPut(sb, "dump.json", `{"skillCount": 1, "entryCount": 1, "modCount": 1, "groupCount": 0,
						"skillTypeNames": {}, "modFlagBits": {"Attack": 1},
						"keywordFlagBits": {"Aura": 1}, "skills": {"A": {}}}`)
				}},
			{Name: "dump-predates-groups", Refusal: "REFUSING: the dump carries no 'groupCount'",
				Break: func(sb string) error {
					return ssmPut(sb, "dump.json", `{"skillCount": 1, "entryCount": 1, "modCount": 1,
						"skillTypeNames": {}, "modFlagBits": {"Attack": 1},
						"keywordFlagBits": {"Aura": 1}, "skills": {}}`)
				}},
			{Name: "group-carries-a-skillflag",
				Refusal: "REFUSING: skill 'A' stat 's' carries statMap scalar 'skillFlag' inside a GROUP",
				Break: func(sb string) error {
					return ssmPut(sb, "dump.json", `{"skillCount": 1, "entryCount": 1, "modCount": 1,
						"groupCount": 1, "skillTypeNames": {}, "modFlagBits": {"Attack": 1},
						"keywordFlagBits": {"Aura": 1},
						"skills": {"A": {"entries": {"s": {"scalars": [], "mods": [], "groups": [
						  {"scalars": {"__table": true, "skillFlag": {"__string": "arrow"}},
						   "mods": [{"name": "Life", "type": "BASE", "value": {"__number": 1},
						             "flags": 0, "keywordFlags": 0, "tags": []}]}]}}}}}`)
				}},

			{Name: "an-entry-has-no-mods", Refusal: "REFUSING: skill 'A' stat 's' has no 'mods' list",
				Break: func(sb string) error {
					return ssmPut(sb, "dump.json", `{"skillCount": 1, "entryCount": 1, "modCount": 1, "groupCount": 0,
						"skillTypeNames": {}, "modFlagBits": {"Attack": 1},
						"keywordFlagBits": {"Aura": 1},
						"skills": {"A": {"entries": {"s": {"scalars": []}}}}}`)
				}},

			{Name: "global-map-missing",
				Refusal: "REFUSING: the global map (skill_stat_map.go) is not readable: No such file or directory",
				Break:   func(sb string) error { return os.Remove(filepath.Join(sb, ssmGlobalGo)) }},

			{Name: "output-is-a-directory",
				Refusal: "REFUSING: skill_stat_maps.go is not writable: Is a directory",
				Break: func(sb string) error {
					if err := ssmPut(sb, "dump.json", ssmMinimal); err != nil {
						return err
					}
					return os.Mkdir(filepath.Join(sb, ssmOutGo), 0o750)
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
				s.PythonArgs = []string{"dump.json", ssmOutGo, ssmGlobalGo, "extra"}
			}
			if f.Name != "happy" {
				s.Files = nil
			}
			s.GoArgs = append([]string{"skill-stat-maps-emit"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}

	difftest.AssertKeyDeletionAgrees(t, t.Context(), spec, []string{"dump.json"}, mk)
}

func TestSkillStatMapEmitTallyOrderIsStableAcrossRuns(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssmSandbox(t, repo, ssmDump())
	base := []string{"dump.json", ssmOutGo, ssmGlobalGo}
	spec := difftest.Spec{
		Tool: "skill-stat-maps-emit", PythonScript: ssmPy, PythonArgs: base,
		GoBin: bin, GoArgs: append([]string{"skill-stat-maps-emit"}, base...),
		Stderr: difftest.StderrExact, Files: []string{ssmOutGo},
		Library: []string{mcPy}, RefusalSites: 2,
	}
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprintf("run%d", i), func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), spec, difftest.Fixture{Name: "happy"}, mk)
		})
	}
}

func TestSkillStatMapEmitPythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := ssmSandbox(t, repo, ssmDump())
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool: "skill-stat-maps-emit", PythonScript: ssmPy,
		PythonArgs: []string{"dump.json", ssmOutGo, ssmGlobalGo},
		Files:      []string{ssmOutGo},
	}, mk, []string{"0", "12345", "999983"})
}
