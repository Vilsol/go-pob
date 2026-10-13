package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const (
	ssgPy    = "oracle/skill_stat_map_global_emit.py"
	ssgOutGo = "skill_stat_map.go"
)

func ssgDump(withDupes bool) string {
	dupes := "[]"
	if withDupes {
		dupes = `["repeated_key_one", "repeated_key_two"]`
	}
	return `{
  "pobVersion": "2.67.2-fixture",
  "entryCount": 15,
  "modCount": 8,
  "duplicateKeys": ` + dupes + `,
  "skillTypeNames": {"7": "Trappable"},
  "modFlagBits": {"Attack": 1, "Spell": 2, "Melee": 256, "Area": 512, "SourceMask": 1536},
  "keywordFlagBits": {"Aura": 1, "Physical": 16, "Fire": 32},
  "order": ["zeta_stat", "alpha_stat", "commented_stat", "empty_stat", "perstat_list_stat",
            "scalars_stat", "blank_01", "blank_02", "blank_03", "blank_04", "blank_05", "blank_06", "blank_07", "blank_08", "blank_09"],
  "entries": {
    "zeta_stat": {
      "scalars": [],
      "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 10}, "flags": 257,
                "keywordFlags": 32, "tags": []}]
    },
    "alpha_stat": {
      "scalars": [],
      "mods": [{"name": "SkillData", "type": "LIST", "flags": 0, "keywordFlags": 0, "tags": [],
                "value": {"__table": true, "key": {"__string": "showAverage"},
                          "value": {"__bool": true}}}]
    },
    "commented_stat": {
      "scalars": [],
      "comments": ["-- Section: one", "-- Section: two"],
      "mods": [{"name": "Life", "type": "BASE", "value": {"__number": 5}, "flags": 0,
                "keywordFlags": 0,
                "tags": [{"__table": true, "type": {"__string": "SkillType"},
                          "skillType": {"__number": 7}}]}]
    },
    "empty_stat": {
      "scalars": [],
      "mods": [{"name": "Damage", "type": "NOSUCHTYPE", "value": {"__number": 1}, "flags": 0,
                "keywordFlags": 0, "tags": []}]
    },
    "perstat_list_stat": {
      "scalars": [],
      "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 0,
                "keywordFlags": 0,
                "tags": [{"__table": true, "type": {"__string": "PerStat"},
                          "statList": {"__table": true, "1": {"__string": "Str"},
                                       "2": {"__string": "Dex"}},
                          "div": {"__number": 10}}]}]
    },
    "scalars_stat": {
      "scalars": {"value": {"__number": 1}, "mult": {"__number": 2},
                  "div": {"__number": 3}, "base": {"__number": 4}},
      "mods": [{"name": "Armour", "type": "MORE", "value": {"__number": 2}, "flags": 0,
                "keywordFlags": 0, "tags": []},
               {"name": "Evasion", "type": "MORE", "value": {"__number": 3}, "flags": 0,
                "keywordFlags": 0, "tags": []}]
    },
    "blank_01": {"scalars": [], "mods": []},
    "blank_02": {"scalars": [], "mods": []},
    "blank_03": {"scalars": [], "mods": []},
    "blank_04": {"scalars": [], "mods": []},
    "blank_05": {"scalars": [], "mods": []},
    "blank_06": {"scalars": [], "mods": []},
    "blank_07": {"scalars": [], "mods": []},
    "blank_08": {"scalars": [], "mods": []},
    "blank_09": {"scalars": [], "mods": []}
  }
}`
}

func ssgCleanDump() string {
	return `{
  "pobVersion": "2.67.2-fixture",
  "entryCount": 1,
  "modCount": 1,
  "duplicateKeys": [],
  "skillTypeNames": {"7": "Trappable"},
  "modFlagBits": {"Attack": 1, "Spell": 2},
  "keywordFlagBits": {"Aura": 1},
  "order": ["only_stat"],
  "entries": {
    "only_stat": {
      "scalars": [],
      "mods": [{"name": "Damage", "type": "INC", "value": {"__number": 1}, "flags": 0,
                "keywordFlags": 0, "tags": []}]
    }
  }
}`
}

func ssgSandbox(t *testing.T, repo, dump string) func() (string, error) {
	t.Helper()
	src := map[string]string{}
	for _, rel := range []string{mcPy, ssmPy, ssgPy} {
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

func ssgPut(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func ssgSpec(bin string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool:         "skill-stat-map-global-emit",
		PythonScript: ssgPy,
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"skill-stat-map-global-emit"}, args...),
		Stderr:       difftest.StderrExact,
		Files:        []string{ssgOutGo},
		Library:      []string{mcPy, ssmPy},
		RefusalSites: 2,
	}
}

func TestSkillStatMapGlobalEmit(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssgSandbox(t, repo, ssgDump(true))
	pySrc := difftest.ReadOriginal(t, repo, ssgPy)

	spec := ssgSpec(bin, "dump.json", ssgOutGo)
	spec.Fixtures = []difftest.Fixture{
		{Name: "happy"},

		{Name: "no-args", Refusal: "usage: skill_stat_map_global_emit.py <dump.json> <out.go>",
			Break: func(string) error { return nil }},
		{Name: "three-args", Refusal: "usage: skill_stat_map_global_emit.py <dump.json> <out.go>",
			Break: func(string) error { return nil }},

		{Name: "dump-missing", Refusal: "REFUSING: the dump (dump.json) is not readable: No such file or directory",
			Break: func(sb string) error { return os.Remove(filepath.Join(sb, "dump.json")) }},
		{Name: "dump-not-json", Refusal: "REFUSING: the dump (dump.json) is not valid JSON",
			Break: func(sb string) error { return ssgPut(sb, "dump.json", "{ nope") }},
		{Name: "dump-not-an-object", Refusal: "REFUSING: dump.json is not a JSON object",
			Break: func(sb string) error { return ssgPut(sb, "dump.json", "[]") }},
		{Name: "dump-missing-a-count", Refusal: "REFUSING: the dump carries no 'modCount'",
			Break: func(sb string) error {
				return ssgPut(sb, "dump.json", `{"entryCount": 1, "order": ["a"],
					"entries": {"a": {"mods": [], "scalars": []}}, "skillTypeNames": {}}`)
			}},
		{Name: "entries-is-an-array", Refusal: "REFUSING: the dump's 'entries' is not a JSON object",
			Break: func(sb string) error {
				return ssgPut(sb, "dump.json", `{"entryCount": 1, "modCount": 1, "order": ["a"],
					"entries": [], "skillTypeNames": {}}`)
			}},
		{Name: "dump-has-no-order", Refusal: "REFUSING: the dump has no 'order'",
			Break: func(sb string) error {
				return ssgPut(sb, "dump.json", `{"entryCount": 1, "modCount": 1, "order": [],
					"entries": {"a": {"mods": [], "scalars": []}}, "skillTypeNames": {"7": "T"}}`)
			}},
		{Name: "order-and-entries-disagree",
			Refusal: "REFUSING: the dump's order (2) and entries (1) disagree",
			Break: func(sb string) error {
				return ssgPut(sb, "dump.json", `{"entryCount": 1, "modCount": 1,
					"order": ["a", "b"], "entries": {"a": {"mods": [], "scalars": []}},
					"skillTypeNames": {"7": "T"}, "modFlagBits": {"Attack": 1},
					"keywordFlagBits": {"Aura": 1}}`)
			}},
		{Name: "order-names-a-missing-entry",
			Refusal: "REFUSING: the dump's 'order' names 'b', which 'entries' does not carry",
			Break: func(sb string) error {
				return ssgPut(sb, "dump.json", `{"entryCount": 2, "modCount": 1,
					"order": ["a", "b"],
					"entries": {"a": {"mods": [], "scalars": []}, "c": {"mods": [], "scalars": []}},
					"skillTypeNames": {"7": "T"}, "modFlagBits": {"Attack": 1},
					"keywordFlagBits": {"Aura": 1}}`)
			}},
		{Name: "an-entry-has-no-mods", Refusal: "REFUSING: entry 'a' has no 'mods' list",
			Break: func(sb string) error {
				return ssgPut(sb, "dump.json", `{"entryCount": 1, "modCount": 1, "order": ["a"],
					"entries": {"a": {"scalars": []}}, "skillTypeNames": {"7": "T"},
					"modFlagBits": {"Attack": 1}, "keywordFlagBits": {"Aura": 1}}`)
			}},
		{Name: "dump-has-no-modFlagBits", Refusal: "REFUSING: the dump carries no 'modFlagBits'",
			Break: func(sb string) error {
				return ssgPut(sb, "dump.json", `{"entryCount": 1, "modCount": 1, "order": ["a"],
					"entries": {"a": {"mods": [], "scalars": []}}, "skillTypeNames": {"7": "T"},
					"keywordFlagBits": {"Aura": 1}}`)
			}},

		{Name: "output-is-a-directory",
			Refusal: "REFUSING: skill_stat_map.go is not writable: Is a directory",
			Break:   func(sb string) error { return os.Mkdir(filepath.Join(sb, ssgOutGo), 0o750) }},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			switch f.Name {
			case "no-args":
				s.PythonArgs = nil
			case "three-args":
				s.PythonArgs = []string{"dump.json", ssgOutGo, "extra"}
			}
			if f.Name != "happy" {
				s.Files = nil
			}
			s.GoArgs = append([]string{"skill-stat-map-global-emit"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestSkillStatMapGlobalEmitWithoutDuplicateKeys: THE EMPTY-LIST HALF OF duplicateKeys.
func TestSkillStatMapGlobalEmitWithoutDuplicateKeys(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssgSandbox(t, repo, ssgDump(false))
	s := ssgSpec(bin, "dump.json", ssgOutGo)
	difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "no-duplicate-keys"}, mk)
}

// TestSkillStatMapGlobalEmitWithNoOmissions: THE "(none)" FALLBACK.
func TestSkillStatMapGlobalEmitWithNoOmissions(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssgSandbox(t, repo, ssgCleanDump())
	s := ssgSpec(bin, "dump.json", ssgOutGo)
	difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "no-omissions"}, mk)
}

func TestSkillStatMapGlobalEmitOrderIsStableAcrossRuns(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ssgSandbox(t, repo, ssgDump(true))
	s := ssgSpec(bin, "dump.json", ssgOutGo)
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprintf("run%d", i), func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "happy"}, mk)
		})
	}
}
