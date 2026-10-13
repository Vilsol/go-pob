package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const oeCommonPy = "oracle/emit_common.py"

const oeOutGo = "out.go"

func oeMinionUsesDump() string {
	return `{
  "pobVersion": "2.67.2-fixture",
  "minionUses": {
    "SingleSlot": {"Weapon 1": true},
    "MultiSlot": {"Weapon 2": true, "Helmet": false, "Body Armour": true},
    "NoLocation": {"Gloves": true}
  },
  "minionHasItemSet": {"SingleSlot": true, "NoLocation": false},
  "locs": {
    "SingleSlot": "src/Data/Skills/act_dex.lua:1234",
    "MultiSlot": "src/Data/Skills/act_str.lua:99",
    "NoLocation": ""
  }
}`
}

func oeAddSkillTypesDump(withDupes bool) string {
	dupes := `{}`
	if withDupes {
		dupes = `{
    "SupportBrandSupport": [
      {"file": "src/Data/Skills/sup_int.lua", "line": 100, "text": "addSkillTypes = { SkillType.Brand },"},
      {"file": "src/Data/Skills/sup_int.lua", "line": 120, "text": "addSkillTypes = { SkillType.Brand },"}
    ]
  }`
	}
	return `{
  "pobVersion": "2.67.2-fixture",
  "order": ["SupportBrandSupport", "SupportSecond"],
  "entries": {
    "SupportBrandSupport": ["Brand"],
    "SupportSecond": ["Triggered", "Cooldown"]
  },
  "duplicates": ` + dupes + `
}`
}

func oeHasGlobalEffectDump() string {
	return `{
  "pobVersion": "2.67.2-fixture",
  "skillCount": 1481,
  "ids": ["Discipline", "Anger", "Discipline", "Zealotry", "Fixture<A&B>"]
}`
}

func oeSandbox(t *testing.T, repo string, emitter, dump string) func() (string, error) {
	t.Helper()
	src := map[string]string{}
	for _, rel := range []string{oeCommonPy, emitter} {
		body := difftest.ReadOriginal(t, repo, rel)
		if body == nil {
			continue
		}
		src[rel] = string(body)
	}
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{"dump.json": dump}
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

const (
	oeMinionUsesPy      = "oracle/skill_minion_uses_emit.py"
	oeAddSkillTypesPy   = "oracle/skill_add_skill_types_emit.py"
	oeHasGlobalEffectPy = "oracle/skill_has_global_effect_emit.py"
	oeMinionListsPy     = "oracle/skill_minion_lists_emit.py"
	oeHiddenLevelsPy    = "oracle/skill_hidden_levels_emit.py"
	oeTattooPy          = "oracle/tattoo_passives_emit.py"
	oeWeaponTypesPy     = "oracle/skill_weapon_types_emit.py"
	oePantheonsPy       = "oracle/pantheons_emit.py"
	oeClusterJewelsPy   = "oracle/cluster_jewels_emit.py"
	oeLegionPy          = "oracle/legion_passives_emit.py"
	oeBossSkillsPy      = "oracle/boss_skills_emit.py"
)

var oeSpecs = map[string]difftest.Spec{
	"skill-minion-uses-emit": {
		Tool: "skill-minion-uses-emit", PythonScript: oeMinionUsesPy,
		Stderr: difftest.StderrExact, Files: []string{oeOutGo},
		Library: []string{oeCommonPy},
	},
	"skill-add-skill-types-emit": {
		Tool: "skill-add-skill-types-emit", PythonScript: oeAddSkillTypesPy,
		Stderr: difftest.StderrExact, Files: []string{oeOutGo}, Library: []string{oeCommonPy},
	},
	"skill-has-global-effect-emit": {
		Tool: "skill-has-global-effect-emit", PythonScript: oeHasGlobalEffectPy,
		Stderr: difftest.StderrExact, Files: []string{oeOutGo}, Library: []string{oeCommonPy},
	},
	"skill-minion-lists-emit": {
		Tool: "skill-minion-lists-emit", PythonScript: oeMinionListsPy,
		Stderr: difftest.StderrExact, Files: []string{oeOutGo}, Library: []string{oeCommonPy},
	},
	"skill-hidden-levels-emit": {
		Tool: "skill-hidden-levels-emit", PythonScript: oeHiddenLevelsPy,
		Stderr: difftest.StderrExact, Files: []string{oeOutGo}, Library: []string{oeCommonPy},
	},
	"tattoo-passives-emit": {
		Tool: "tattoo-passives-emit", PythonScript: oeTattooPy,
		Stderr: difftest.StderrExact, Files: []string{oeOutGo}, Library: []string{oeCommonPy},
	},
	"skill-flags-emit": {
		Tool: "skill-flags-emit", PythonScript: oeSkillFlagsPy,
		Stderr: difftest.StderrExact, Files: []string{oeOutGo}, Library: []string{oeCommonPy},
	},
	"gems-emit": {
		Tool: "gems-emit", PythonScript: oeGemsPy,
		Stderr: difftest.StderrExact, Files: []string{oeOutGo}, Library: []string{oeCommonPy},
	},
	"skill-weapon-types-emit": {
		Tool: "skill-weapon-types-emit", PythonScript: oeWeaponTypesPy,
		Stderr: difftest.StderrExact, Library: []string{oeCommonPy},
	},
	"pantheons-emit": {
		Tool: "pantheons-emit", PythonScript: oePantheonsPy,
		Stderr: difftest.StderrExact, Files: []string{oeOutGo}, Library: []string{oeCommonPy},
	},
	"cluster-jewels-emit": {
		Tool: "cluster-jewels-emit", PythonScript: oeClusterJewelsPy,
		Stderr: difftest.StderrExact, Files: []string{oeOutGo}, Library: []string{oeCommonPy},
	},
	"legion-passives-emit": {
		Tool: "legion-passives-emit", PythonScript: oeLegionPy,
		Stderr: difftest.StderrExact, Files: []string{oeOutGo}, Library: []string{oeCommonPy},
	},
	"boss-skills-emit": {
		Tool: "boss-skills-emit", PythonScript: oeBossSkillsPy,
		Stderr: difftest.StderrExact, Files: []string{oeOutGo}, Library: []string{oeCommonPy},
	},
}

func oeSpec(bin, sub string, args ...string) difftest.Spec {
	s := oeSpecs[sub]
	s.PythonArgs = args
	s.GoBin = bin
	s.GoArgs = append([]string{sub}, args...)
	return s
}

func oeSharedFixtures() []difftest.Fixture {
	return []difftest.Fixture{
		{Name: "happy"},
		{
			Name:    "the-dump-is-missing",
			Refusal: "dump.json is not readable",
			Break:   func(sb string) error { return os.Remove(filepath.Join(sb, "dump.json")) },
		},
		{
			Name:    "the-dump-is-not-json",
			Refusal: "dump.json is not valid JSON",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte("{not json"), 0o600)
			},
		},
		{
			Name:    "the-dump-is-a-json-array",
			Refusal: "dump.json is not a JSON object",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte("[]"), 0o600)
			},
		},
		{
			Name:    "the-dump-is-not-utf8",
			Refusal: "dump.json is not valid JSON",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte("{\"a\": \"\xff\xfe\"}"), 0o600)
			},
		},
		{
			Name:    "the-output-is-not-writable",
			Refusal: "out.go is not writable",
			Break: func(sb string) error {
				return os.Mkdir(filepath.Join(sb, oeOutGo), 0o750)
			},
		},
	}
}

func oeRunFixtures(t *testing.T, s difftest.Spec, extra []difftest.Fixture,
	mk func() (string, error)) {
	t.Helper()
	s.RefusalSites = 1
	s.Fixtures = append(oeSharedFixtures(), extra...)
	difftest.AssertRefusalSiteCount(t, s, oeReadPy(t, s.PythonScript))
	for _, f := range s.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

func oeReadPy(t *testing.T, rel string) []byte {
	t.Helper()
	return difftest.ReadOriginal(t, repoRoot(t), rel)
}

func TestOracleSkillMinionUsesEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeMinionUsesPy, oeMinionUsesDump())
	s := oeSpec(bin, "skill-minion-uses-emit", "dump.json", oeOutGo)
	oeRunFixtures(t, s, []difftest.Fixture{
		{
			Name:    "no-minionUses",
			Refusal: "the dump has no minionUses entries",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"minionUses": {}, "minionHasItemSet": {"a": true}}`), 0o600)
			},
		},
		{
			Name:    "no-minionHasItemSet",
			Refusal: "the dump has no minionHasItemSet entries",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"minionUses": {"a": {"Weapon 1": true}}, "minionHasItemSet": {}}`), 0o600)
			},
		},
	}, mk)
}

func TestOracleSkillMinionUsesEmitUsage(t *testing.T) {
	oeUsageSweep(t, oeMinionUsesPy, "skill-minion-uses-emit", oeMinionUsesDump())
}

// TestOracleSkillMinionUsesEmitNonBoolFlag: A FLAG THAT IS NOT A BOOL, WHICH NO REAL DUMP CARRIES.
func TestOracleSkillMinionUsesEmitNonBoolFlag(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeMinionUsesPy, `{
  "pobVersion": "2.67.2-fixture",
  "minionUses": {"NumericFlag": {"Ring 1": 1}, "StringFlag": {"Ring 2": "True", "Amulet": true}},
  "minionHasItemSet": {"NumericFlag": 0},
  "locs": {}
}`)
	difftest.RunFixture(t, t.Context(),
		oeSpec(bin, "skill-minion-uses-emit", "dump.json", oeOutGo),
		difftest.Fixture{Name: "a-flag-that-is-not-a-bool"}, mk)
}

func TestOracleSkillAddSkillTypesEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeAddSkillTypesPy, oeAddSkillTypesDump(true))
	s := oeSpec(bin, "skill-add-skill-types-emit", "dump.json", oeOutGo)
	oeRunFixtures(t, s, []difftest.Fixture{
		{
			Name:    "no-supports",
			Refusal: "the dump has no supports",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"order": [], "entries": {}}`), 0o600)
			},
		},
		{
			Name:    "a-support-declares-an-empty-list",
			Refusal: "SupportSecond has an empty addSkillTypes",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{
  "order": ["SupportBrandSupport", "SupportSecond"],
  "entries": {"SupportBrandSupport": ["Brand"], "SupportSecond": []}
}`), 0o600)
			},
		},
	}, mk)
}

// TestOracleSkillAddSkillTypesEmitWithoutDuplicates: THE NO-DUPLICATES BRANCH IS THE ONE THE REAL DUMP CANNOT REACH TODAY.
func TestOracleSkillAddSkillTypesEmitWithoutDuplicates(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeAddSkillTypesPy, oeAddSkillTypesDump(false))
	difftest.RunFixture(t, t.Context(),
		oeSpec(bin, "skill-add-skill-types-emit", "dump.json", oeOutGo),
		difftest.Fixture{Name: "no-duplicate-declarations"}, mk)
}

func TestOracleSkillAddSkillTypesEmitUsage(t *testing.T) {
	oeUsageSweep(t, oeAddSkillTypesPy, "skill-add-skill-types-emit", oeAddSkillTypesDump(true))
}

func TestOracleSkillHasGlobalEffectEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeHasGlobalEffectPy, oeHasGlobalEffectDump())
	s := oeSpec(bin, "skill-has-global-effect-emit", "dump.json", oeOutGo)
	oeRunFixtures(t, s, []difftest.Fixture{
		{
			Name:    "no-ids",
			Refusal: "the dump carries no flagged ids",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"ids": []}`), 0o600)
			},
		},
		{
			Name:    "a-non-string-id",
			Refusal: "the dump carries a non-string or empty id",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"ids": ["Discipline", 7]}`), 0o600)
			},
		},
		{
			Name:    "an-empty-string-id",
			Refusal: "the dump carries a non-string or empty id",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"ids": ["Discipline", ""]}`), 0o600)
			},
		},
		{
			Name:    "ids-is-a-string",
			Refusal: "the dump carries a non-string or empty id",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"ids": "Discipline"}`), 0o600)
			},
		},
	}, mk)
}

func TestOracleSkillHasGlobalEffectEmitUsage(t *testing.T) {
	oeUsageSweep(t, oeHasGlobalEffectPy, "skill-has-global-effect-emit", oeHasGlobalEffectDump())
}

func oeUsageSweep(t *testing.T, py, sub, dump string) {
	t.Helper()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, py, dump)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no-arguments", nil},
		{"one-argument", []string{"dump.json"}},
		{"three-arguments", []string{"dump.json", oeOutGo, "extra"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := oeSpec(bin, sub, tc.args...)
			s.Files = nil
			difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: tc.name}, mk)
		})
	}
}

func oeMinionListsDump() string {
	return `{
  "pobVersion": "2.67.2-fixture",
  "minionList": {
    "RaiseSpectre": [],
    "SummonSkeleton": ["RaisedSkeleton", "RaisedSkeletonWarrior"],
    "NoLocation": ["SomeMinion"]
  },
  "addMinionList": {"SupportSummonGhostOnKill": ["RaisedGhost"]},
  "locs": {
    "RaiseSpectre": "src/Data/Skills/spectre.lua:12",
    "SummonSkeleton": "src/Data/Skills/act_int.lua:400",
    "NoLocation": ""
  }
}`
}

func oeHiddenLevelsDump() string {
	return `{
  "pobVersion": "2.67.2-fixture",
  "order": ["HiddenA", "HiddenB"],
  "entries": {
    "HiddenA": {
      "note": "first line\nsecond line",
      "skillTypes": ["Spell", "Aura"],
      "stats": ["stat_one", "stat_two"],
      "levels": {"1": [20, 20], "9": [21, 21], "10": [22, 1234567.5]}
    },
    "HiddenB": {
      "stats": ["only_stat"],
      "levels": {"1": [1]}
    }
  }
}`
}

func oeTattooDump() string {
	return `{
  "pobVersion": "2.67.2-fixture",
  "scanned": 381,
  "nodes": {
    "Tattoo of the Tawhoa Shaman": {"id": "n1", "dn": "Tattoo of the Tawhoa Shaman",
      "sd": ["+10 to Strength"], "targetType": "Small Strength", "targetValue": "", "MinimumConnected": 0,
      "legacy": false, "icon": "Art/a.png", "activeEffectImage": "Art/b.png"},
    "Tattoo of the Ngamahu Shaman": {"id": "n2", "dn": "Tattoo of the Ngamahu Shaman",
      "sd": [], "isKeystone": true, "targetType": "Keystone", "targetValue": "", "MinimumConnected": 0,
      "icon": "", "activeEffectImage": "Art/c.png"},
    "Tattoo with no sd": {"id": "n3", "dn": "Tattoo with no sd", "targetType": "Notable",
      "targetValue": "+30 to Strength", "MinimumConnected": 7, "legacy": true,
      "reminderText": ["Requires at least 7 adjacent Passive Skills to be allocated"]}
  }
}`
}

func oeWeaponTypesDump(unknownName bool) string {
	name := "Bow"
	if unknownName {
		name = "Bows"
	}
	return `{
  "skills": {
    "ToxicRain": {"weaponTypes": ["` + name + `"], "support": false,
      "loc": "src/Data/Skills/act_dex.lua:77"},
    "SupportThing": {"weaponTypes": ["Claw", "Dagger"], "support": true},
    "NoRestriction": {"support": false}
  }
}`
}

func TestOracleSkillMinionListsEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeMinionListsPy, oeMinionListsDump())
	s := oeSpec(bin, "skill-minion-lists-emit", "dump.json", oeOutGo)
	oeRunFixtures(t, s, []difftest.Fixture{
		{
			Name:    "no-minionList",
			Refusal: "the dump has no minionList entries",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"minionList": {}, "addMinionList": {"a": ["b"]}}`), 0o600)
			},
		},
		{
			Name:    "no-addMinionList",
			Refusal: "the dump has no addMinionList entries",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"minionList": {"a": ["b"]}, "addMinionList": {}}`), 0o600)
			},
		},
	}, mk)
}

func TestOracleSkillMinionListsEmitUsage(t *testing.T) {
	oeUsageSweep(t, oeMinionListsPy, "skill-minion-lists-emit", oeMinionListsDump())
}

func TestOracleSkillHiddenLevelsEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeHiddenLevelsPy, oeHiddenLevelsDump())
	s := oeSpec(bin, "skill-hidden-levels-emit", "dump.json", oeOutGo)
	oeRunFixtures(t, s, []difftest.Fixture{
		{
			Name:    "no-entries",
			Refusal: "the dump has no entries",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"order": [], "entries": {}}`), 0o600)
			},
		},
		{
			Name:    "an-entry-with-no-stats",
			Refusal: "HiddenB came back with no stats or no levels",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{
  "order": ["HiddenA", "HiddenB"],
  "entries": {
    "HiddenA": {"stats": ["s"], "levels": {"1": [1]}},
    "HiddenB": {"stats": [], "levels": {"1": [1]}}
  }
}`), 0o600)
			},
		},
	}, mk)
}

func TestOracleSkillHiddenLevelsEmitUsage(t *testing.T) {
	oeUsageSweep(t, oeHiddenLevelsPy, "skill-hidden-levels-emit", oeHiddenLevelsDump())
}

func TestOracleTattooPassivesEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeTattooPy, oeTattooDump())
	s := oeSpec(bin, "tattoo-passives-emit", "dump.json", oeOutGo)
	oeRunFixtures(t, s, []difftest.Fixture{
		{
			Name:    "no-tattoo-rows",
			Refusal: "the dump carries no tattoo rows",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"nodes": {}}`), 0o600)
			},
		},
		{
			Name:    "a-row-with-no-targetType",
			Refusal: "has no targetType",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"nodes": {
  "Tattoo of the Tawhoa Shaman": {"id": "n1", "dn": "Tattoo of the Tawhoa Shaman"}}}`), 0o600)
			},
		},
		{
			Name:    "the-corpus-probe-row-is-absent",
			Refusal: "is absent from the dump; it is referenced by the corpus",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"nodes": {"Other": {"id": "n1", "dn": "Other"}}}`), 0o600)
			},
		},
		{
			Name:    "a-row-with-no-id",
			Refusal: "has no id",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"nodes": {
  "Tattoo of the Tawhoa Shaman": {"dn": "Tattoo of the Tawhoa Shaman"}}}`), 0o600)
			},
		},
		{
			Name:    "a-row-with-no-dn",
			Refusal: "has no `dn`, which is the key the build file references it by",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"nodes": {
  "Tattoo of the Tawhoa Shaman": {"id": "n1"}}}`), 0o600)
			},
		},
		{
			Name:    "a-non-string-stat-line",
			Refusal: "has a non-string stat line",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"nodes": {
  "Tattoo of the Tawhoa Shaman": {"id": "n1", "dn": "Tattoo of the Tawhoa Shaman",
    "sd": ["ok", 7]}}}`), 0o600)
			},
		},
	}, mk)
}

func TestOracleTattooPassivesEmitUsage(t *testing.T) {
	oeUsageSweep(t, oeTattooPy, "tattoo-passives-emit", oeTattooDump())
}

// TestOracleSkillWeaponTypesEmit is the pipe-fed member of the family.
func TestOracleSkillWeaponTypesEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeWeaponTypesPy, "{}")
	for _, tc := range []struct {
		name    string
		stdin   string
		refusal string
	}{
		{"happy", oeWeaponTypesDump(false), ""},
		{"not-json", "{not json", "the dump on stdin is not valid JSON"},
		{"a-json-array", "[]", "the dump on stdin is not a JSON object"},
		{"no-skills-object", `{"other": 1}`, "the dump on stdin has no `skills` object"},
		{"no-skill-declares-weaponTypes", `{"skills": {"A": {"support": false}}}`,
			"no skill in the dump carries weaponTypes"},
		{"a-weapon-type-outside-PoBs-own-set", oeWeaponTypesDump(true),
			"weaponTypes names not in PoB's weaponTypeInfo: ['Bows']"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := oeSpec(bin, "skill-weapon-types-emit")
			s.Stdin = []byte(tc.stdin)
			difftest.RunFixture(t, t.Context(), s,
				difftest.Fixture{Name: tc.name, Refusal: tc.refusal}, mk)
		})
	}
}

func oePantheonsDump() string {
	return `{
  "pobVersion": "2.67.2-fixture",
  "gods": {
    "TheBrineKing": {"isMajorGod": true, "souls": [
      {"name": "Soul of the Brine King", "mods": ["You cannot be Stunned if you have been Stunned"]},
      {"name": "Soul of a Capturable", "mods": []}
    ]},
    "Minor God 1": {"isMajorGod": false, "souls": []},
    "NoSoulsKey": {"isMajorGod": false}
  }
}`
}

func oeClusterJewelsDump() string {
	return `{
  "pobVersion": "2.67.2-fixture",
  "jewels": {
    "Large": {"size": "Large", "sizeIndex": 3, "minNodes": 5, "maxNodes": 6,
      "smallIndicies": [1, 2, 3], "notableIndicies": [4], "socketIndicies": [6],
      "totalIndicies": 12,
      "skills": {
        "AreaDamage": {"name": "Area Damage", "icon": "icon.png", "masteryIcon": "m.png",
          "tag": "area", "stats": ["10% increased Area Damage"], "enchant": ["Added Small Passive"]},
        "Bare": {"name": "Bare", "icon": "", "tag": "", "stats": [], "enchant": []}
      }},
    "Small": {"size": "Small", "sizeIndex": 1, "minNodes": 2, "maxNodes": 3,
      "smallIndicies": [], "totalIndicies": 6, "skills": {}}
  },
  "notableSortOrder": {"Zeal": 2, "Ambidexterity": 1},
  "keystones": ["Brand Loyalty", "Fettle"],
  "orbitOffsets": {"25134": {"0": 0, "1": 11, "2": 11}, "10643": {"0": 4}}
}`
}

func oeLegionDump() string {
	return `{
  "pobVersion": "2.67.2-fixture",
  "scanned": 249,
  "named": {
    "templar_devotion_node": {"id": "templar_devotion_node", "dn": "Devotion",
      "sd": ["+10 to Devotion"], "rolls": {"1": {"key": "devotion", "fmt": "d", "index": 1,
        "min": 10, "max": 10}}},
    "eternal_small_blank": {"id": "eternal_small_blank", "dn": "Nothing"}
  },
  "keystones": {
    "karui_keystone_2": {"id": "karui_keystone_2", "dn": "Strength of Blood",
      "sd": ["Fixed line", "Rolled line", "Trailing fixed line"],
      "rolls": {"2": {"key": "k", "fmt": "", "index": 1, "min": 1.5, "max": 2}}},
    "vaal_keystone_1_v2": {"id": "vaal_keystone_1_v2", "dn": "Chaotic Disposition",
      "sd": ["A", "B"], "rolls": [{"key": "a", "index": 1, "min": 0, "max": 1},
                                  {"key": "b", "fmt": "f", "index": 2, "min": 2, "max": 3}]}
  },
  "nodes": {"n1": {"id": "n1", "dn": "Node One", "sd": ["line"]}},
  "additions": {"a1": {"id": "a1", "dn": "Addition One"}}
}`
}

func oeBossSkillsDump() string {
	return `{
  "pobVersion": "2.67.2-fixture",
  "bossSkills": {
    "Atziri Flameblast": {
      "DamageType": "Fire",
      "DamageMultipliers": {"Fire": [1.5, 2], "Physical": [1, 1]},
      "DamagePenetrations": {"FirePen": 25, "ColdPen": null},
      "speed": 0.5, "critChance": 0, "earlierUber": true,
      "additionalStats": {"base": {"CannotBeBlocked": "flag", "EnemyDamage": 12},
                          "uber": {"EnemySpeed": 1.25}}
    },
    "Shaper Slam": {
      "DamageMultipliers": {"Physical": [2, 3]},
      "UberDamageMultiplier": 1.5,
      "additionalStats": {"uber": {"OnlyUber": "flag"}}
    },
    "Plain": {"DamageMultipliers": {"Chaos": [1, 1]}}
  }
}`
}

func TestOraclePantheonsEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oePantheonsPy, oePantheonsDump())
	oeRunFixtures(t, oeSpec(bin, "pantheons-emit", "dump.json", oeOutGo), []difftest.Fixture{
		{
			Name:    "no-gods",
			Refusal: "the dump has no 'gods'",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"gods": {}}`), 0o600)
			},
		},
		{
			Name:    "a-soul-with-no-name",
			Refusal: "a soul has no name; the dump is malformed",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"gods": {"G": {"souls": [{"mods": []}]}}}`), 0o600)
			},
		},
		{
			Name:    "a-non-string-mod-line",
			Refusal: "a mod line is not a string",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"gods": {"G": {"souls": [{"name": "S", "mods": ["ok", 7]}]}}}`), 0o600)
			},
		},
	}, mk)
}

func TestOraclePantheonsEmitUsage(t *testing.T) {
	oeUsageSweep(t, oePantheonsPy, "pantheons-emit", oePantheonsDump())
}

func TestOracleClusterJewelsEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeClusterJewelsPy, oeClusterJewelsDump())
	extra := make([]difftest.Fixture, 0, 4)
	for _, f := range []struct{ field, keep string }{
		{"jewels", `{"jewels": {}, "keystones": ["k"], "notableSortOrder": {"n": 1}}`},
		{"keystones", `{"jewels": {"J": {}}, "keystones": [], "notableSortOrder": {"n": 1}}`},
		{"notableSortOrder", `{"jewels": {"J": {}}, "keystones": ["k"], "notableSortOrder": {}}`},
		{"orbitOffsets", `{"jewels": {"J": {}}, "keystones": ["k"], "notableSortOrder": {"n": 1}, "orbitOffsets": {}}`},
	} {
		body := f.keep
		extra = append(extra, difftest.Fixture{
			Name:    "no-" + f.field,
			Refusal: "the dump has no '" + f.field + "'",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(body), 0o600)
			},
		})
	}
	oeRunFixtures(t, oeSpec(bin, "cluster-jewels-emit", "dump.json", oeOutGo), extra, mk)
}

func TestOracleClusterJewelsEmitUsage(t *testing.T) {
	oeUsageSweep(t, oeClusterJewelsPy, "cluster-jewels-emit", oeClusterJewelsDump())
}

func TestOracleLegionPassivesEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeLegionPy, oeLegionDump())
	oeRunFixtures(t, oeSpec(bin, "legion-passives-emit", "dump.json", oeOutGo), []difftest.Fixture{
		{
			Name:    "no-keystones",
			Refusal: "the dump carries no keystone nodes",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"keystones": {}}`), 0o600)
			},
		},
		{
			Name:    "a-named-Normal-branch-node-is-missing",
			Refusal: "the dump is missing the Normal-branch node 'eternal_small_blank'",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{
  "keystones": {"k": {"id": "k", "dn": "K"}},
  "named": {"templar_devotion_node": {"id": "t", "dn": "T"}}}`), 0o600)
			},
		},
		{
			Name:    "no-nodes-table",
			Refusal: "the dump carries no `nodes` table",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{
  "keystones": {"k": {"id": "k", "dn": "K"}},
  "named": {"templar_devotion_node": {"id": "t", "dn": "T"},
            "eternal_small_blank": {"id": "e", "dn": "E"}}}`), 0o600)
			},
		},
		{
			Name:    "no-additions-table",
			Refusal: "the dump carries no `additions` table",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{
  "keystones": {"k": {"id": "k", "dn": "K"}},
  "named": {"templar_devotion_node": {"id": "t", "dn": "T"},
            "eternal_small_blank": {"id": "e", "dn": "E"}},
  "nodes": {"n": {"id": "n", "dn": "N"}}}`), 0o600)
			},
		},
		{
			Name:    "a-node-with-no-dn",
			Refusal: "has no `dn`; PassiveTree.lua:553 builds the keystone mod from it",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{
  "keystones": {"k": {"id": "k"}},
  "named": {"templar_devotion_node": {"id": "t", "dn": "T"},
            "eternal_small_blank": {"id": "e", "dn": "E"}},
  "nodes": {"n": {"id": "n", "dn": "N"}}, "additions": {"a": {"id": "a", "dn": "A"}}}`), 0o600)
			},
		},
	}, mk)
}

func TestOracleLegionPassivesEmitUsage(t *testing.T) {
	oeUsageSweep(t, oeLegionPy, "legion-passives-emit", oeLegionDump())
}

func TestOracleBossSkillsEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeBossSkillsPy, oeBossSkillsDump())
	oeRunFixtures(t, oeSpec(bin, "boss-skills-emit", "dump.json", oeOutGo), []difftest.Fixture{
		{
			Name:    "no-bossSkills",
			Refusal: "the dump has no 'bossSkills'",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"bossSkills": {}}`), 0o600)
			},
		},
		{
			Name:    "a-preset-with-no-DamageMultipliers",
			Refusal: "Bare has no DamageMultipliers; the dump is malformed",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"bossSkills": {"Bare": {}}}`), 0o600)
			},
		},
		{
			Name:    "a-DamageMultiplier-that-is-not-a-pair",
			Refusal: "DamageMultipliers.Fire is not a pair",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"bossSkills": {"A": {"DamageMultipliers": {"Fire": [1]}}}}`), 0o600)
			},
		},
		{
			Name:    "a-penetration-table-that-is-present-but-empty",
			Refusal: "DamagePenetrations is present but empty; the dump is malformed",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"bossSkills": {
  "A": {"DamageMultipliers": {"Fire": [1, 1]}, "DamagePenetrations": {}}}}`), 0o600)
			},
		},
		{
			Name:    "additionalStats-with-an-unmodelled-difficulty",
			Refusal: "additionalStats carries ['legacy']; only base and uber are modelled",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"bossSkills": {
  "A": {"DamageMultipliers": {"Fire": [1, 1]},
        "additionalStats": {"legacy": {"X": 1}}}}}`), 0o600)
			},
		},
		{
			Name:    "an-additionalStat-that-is-neither-a-number-nor-flag",
			Refusal: `is 'banana' -- neither a number nor "flag"`,
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"bossSkills": {
  "A": {"DamageMultipliers": {"Fire": [1, 1]},
        "additionalStats": {"base": {"X": "banana"}}}}}`), 0o600)
			},
		},
	}, mk)
}

func TestOracleBossSkillsEmitUsage(t *testing.T) {
	oeUsageSweep(t, oeBossSkillsPy, "boss-skills-emit", oeBossSkillsDump())
}
