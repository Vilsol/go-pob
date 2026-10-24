package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const oeSkillFlagsPy = "oracle/skill_flags_emit.py"

func oeSkillFlagsDump() string {
	return `{
  "pobVersion": "2.67.2-fixture",
  "baseFlagNames": ["attack", "spell", "melee", "projectile", "area", "duration",
                    "trap", "mine", "totem", "brand"],
  "addFlagNames": ["trap", "mine", "totem"],
  "skills": {
    "k1": {"id": "AncestralCall", "name": "Ancestral Call", "loc": "src/Data/Skills/act_str.lua:10",
           "baseFlags": ["attack", "melee"], "color": 1},
    "k2": {"id": "Focus", "name": "Focus", "loc": "src/Data/Skills/other.lua:20",
           "baseFlags": [], "hidden": true, "color": 0},
    "k3": {"id": "PetrifiedBlood", "name": "Petrified Blood", "loc": "src/Data/Skills/act_int.lua:30",
           "baseFlags": [], "hidden": true, "fromTree": true},
    "k4": {"id": "SupportTrap", "name": "Trap Support", "loc": "src/Data/Skills/sup_dex.lua:40",
           "addFlags": ["trap"], "support": true, "color": 2},
    "k5": {"id": "SummonVoidSphere", "name": "Summon Volatile Anomaly",
           "loc": "src/Data/Skills/glove.lua:50", "baseFlags": ["spell", "area"],
           "fromItem": true, "hidden": true, "color": 4},
    "k6": {"id": "Angle<Br>&Co", "name": "Naïve Wördmark 💩",
           "loc": "src/Data/Skills/other.lua:60", "baseFlags": ["duration"],
           "addFlags": ["mine"], "fromTree": true}
  }
}`
}

func TestOracleSkillFlagsEmit(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := oeSandbox(t, repo, oeSkillFlagsPy, oeSkillFlagsDump())
	s := oeSpec(bin, "skill-flags-emit", "dump.json", oeOutGo)
	oeRunFixtures(t, s, []difftest.Fixture{
		{
			Name:    "no-skills",
			Refusal: "the dump has no skills. Re-run oracle/skill_flags.lua.",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"skills": {}}`), 0o600)
			},
		},
		{
			Name:    "skills-is-an-array",
			Refusal: "the dump's 'skills' is not a JSON object",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"skills": [{"id": "A"}]}`), 0o600)
			},
		},
		{
			Name:    "a-skill-has-no-id",
			Refusal: "the dump has a skill with no 'id'",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"),
					[]byte(`{"skills": {"k1": {"name": "No Id", "baseFlags": ["attack"]}}}`), 0o600)
			},
		},
		{
			Name:    "no-skill-carries-baseFlags",
			Refusal: "no skill carries baseFlags; the dump is structurally wrong",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"skills": {
					"k1": {"id": "A", "name": "A", "loc": "f.lua:1", "addFlags": ["trap"]}}}`), 0o600)
			},
		},
		{
			Name:    "no-skill-carries-addFlags",
			Refusal: "no skill carries addFlags; the dump is structurally wrong",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"skills": {
					"k1": {"id": "A", "name": "A", "loc": "f.lua:1", "baseFlags": ["attack"]}}}`), 0o600)
			},
		},
		{
			Name:    "a-row-has-no-loc",
			Refusal: "A: the dumper found no declaring line to cite",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"skills": {
					"k1": {"id": "A", "name": "A", "loc": "", "baseFlags": ["attack"]}}}`), 0o600)
			},
		},
		{
			Name:    "baseFlags-is-a-string",
			Refusal: "A: baseFlags is not a list of strings",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"skills": {
					"k1": {"id": "A", "name": "A", "loc": "f.lua:1", "baseFlags": "trap"}}}`), 0o600)
			},
		},
		{
			Name:    "a-flag-is-not-a-string",
			Refusal: "A: baseFlags is not a list of strings",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"skills": {
					"k1": {"id": "A", "name": "A", "loc": "f.lua:1", "baseFlags": ["attack", 7]}}}`), 0o600)
			},
		},
		{
			Name:    "a-row-has-no-name",
			Refusal: "A: has no 'name' string",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"skills": {
					"k1": {"id": "A", "loc": "f.lua:1", "baseFlags": ["attack"], "addFlags": ["trap"]}}}`),
					0o600)
			},
		},
		{
			Name:    "color-is-not-a-number",
			Refusal: "A: 'color' is not a number",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "dump.json"), []byte(`{"skills": {
					"k1": {"id": "A", "name": "A", "loc": "f.lua:1", "color": "R",
					       "baseFlags": ["attack"], "addFlags": ["trap"]}}}`), 0o600)
			},
		},
	}, mk)
}

func TestOracleSkillFlagsEmitUsage(t *testing.T) {
	t.Parallel()
	oeUsageSweep(t, oeSkillFlagsPy, "skill-flags-emit", oeSkillFlagsDump())
}
