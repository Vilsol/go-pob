package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const mdPy = "oracle/minion_data_emit.py"

func mdDump() string {
	return `{
	 "pobVersion": "2.67.2",
	 "minions": {
	  "AbsolutionTemplarJudgeVaal": {
	   "accuracy": 1,
	   "attackRange": 9,
	   "attackTime": 1.17,
	   "chaosResist": 20,
	   "coldResist": 40,
	   "damage": 2.8,
	   "damageSpread": 0.2,
	   "energyShield": 0.2,
	   "fireResist": 40,
	   "id": "AbsolutionTemplarJudgeVaal",
	   "life": 4,
	   "lightningResist": 40,
	   "limit": "ActiveSentinelOfAbsolutionLimit",
	   "loc": 888,
	   "modList": [
	    {
	     "flags": [],
	     "keywordFlags": [],
	     "modType": "MORE",
	     "name": "Damage",
	     "number": 300,
	     "source": "Minion:Ascended Sentinel of Absolution",
	     "tags": [],
	     "valueKind": "number"
	    },
	    {
	     "flags": [],
	     "keywordFlags": [],
	     "modType": "MORE",
	     "name": "DamageTaken",
	     "number": -70,
	     "source": "Minion:Ascended Sentinel of Absolution",
	     "tags": [],
	     "valueKind": "number"
	    }
	   ],
	   "monsterTags": [
	    "bludgeoning_weapon",
	    "caster",
	    "fast_movement",
	    "flesh_armour",
	    "has_one_hand_mace",
	    "has_one_handed_melee",
	    "human",
	    "humanoid",
	    "innocence_story",
	    "melee",
	    "not_dex",
	    "not_str",
	    "physical_affinity",
	    "ranged",
	    "templar"
	   ],
	   "name": "Ascended Sentinel of Absolution",
	   "skillList": [
	    "AbsolutionMinionVaal",
	    "AbsolutionMinionVaalCascade",
	    "GTVaalAbsolutionLarge",
	    "VaalAbsolutionDelayedBlast",
	    "GSVaalAbsolutionEmerge"
	   ]
	  },
	  "AbyssCurseBot": {
	   "accuracy": 1,
	   "attackRange": 6,
	   "attackTime": 1,
	   "chaosResist": 0,
	   "coldResist": 0,
	   "damage": 1,
	   "damageSpread": 0,
	   "fireResist": 0,
	   "id": "AbyssCurseBot",
	   "life": 1,
	   "lightningResist": 0,
	   "loc": 1161,
	   "modList": [
	    {
	     "flags": [],
	     "keywordFlags": [],
	     "modType": "OVERRIDE",
	     "name": "Life",
	     "number": 1,
	     "source": "Minion:Bone Feeder",
	     "tags": [],
	     "valueKind": "number"
	    },
	    {
	     "flags": [],
	     "keywordFlags": [],
	     "modType": "MORE",
	     "name": "DamageTaken",
	     "number": -100,
	     "source": "Minion:Bone Feeder",
	     "tags": [],
	     "valueKind": "number"
	    },
	    {
	     "flags": [],
	     "keywordFlags": [],
	     "modType": "FLAG",
	     "name": "StunImmune",
	     "number": 1,
	     "source": "Minion:Bone Feeder",
	     "tags": [],
	     "valueKind": "number"
	    },
	    {
	     "flags": [],
	     "keywordFlags": [],
	     "modType": "FLAG",
	     "name": "KnockbackImmune",
	     "number": 1,
	     "source": "Minion:Bone Feeder",
	     "tags": [],
	     "valueKind": "number"
	    },
	    {
	     "flags": [],
	     "keywordFlags": [],
	     "modType": "OVERRIDE",
	     "name": "EnduranceChargesMax",
	     "number": 0,
	     "source": "Minion:Bone Feeder",
	     "tags": [],
	     "valueKind": "number"
	    },
	    {
	     "flags": [],
	     "keywordFlags": [],
	     "modType": "OVERRIDE",
	     "name": "FrenzyChargesMax",
	     "number": 0,
	     "source": "Minion:Bone Feeder",
	     "tags": [],
	     "valueKind": "number"
	    },
	    {
	     "flags": [],
	     "keywordFlags": [],
	     "modType": "OVERRIDE",
	     "name": "PowerChargesMax",
	     "number": 0,
	     "source": "Minion:Bone Feeder",
	     "tags": [],
	     "valueKind": "number"
	    }
	   ],
	   "monsterTags": [
	    "construct",
	    "extra_extra_small_height",
	    "is_unarmed",
	    "non_attacking",
	    "stone_armour",
	    "very_fast_movement"
	   ],
	   "name": "Bone Feeder",
	   "skillList": [
	    "SkitterbotWait",
	    "AbyssalFiendEnfeebleAura",
	    "AbyssalFiendTemporalChainsAura",
	    "AbyssalFiendPunishmentAura"
	   ]
	  }
	 },
	 "spectres": {
	  "Metadata/Monsters/LeagueAzmeri/SpecialCorpses/RobotArgusMid": {
	   "accuracy": 1,
	   "armour": 1,
	   "attackRange": 21,
	   "attackTime": 1.785,
	   "baseDamageIgnoresAttackSpeed": true,
	   "chaosResist": 0,
	   "coldResist": 20,
	   "damage": 1.5,
	   "damageSpread": 0.2,
	   "fireResist": 20,
	   "id": "Metadata/Monsters/LeagueAzmeri/SpecialCorpses/RobotArgusMid",
	   "life": 3.75,
	   "lifeScaling": "AltLife2",
	   "lightningResist": 20,
	   "limit": "ActiveSpectreLimit",
	   "loc": 4428,
	   "modList": [
	    {
	     "flags": [
	      "Attack"
	     ],
	     "keywordFlags": [],
	     "modType": "INC",
	     "name": "Damage",
	     "number": 5,
	     "source": "Minion:Hulking Miscreation",
	     "tags": [
	      {
	       "div": 450,
	       "stat": "Armour",
	       "tagType": "PerStat"
	      }
	     ],
	     "valueKind": "number"
	    },
	    {
	     "flags": [],
	     "keywordFlags": [],
	     "modType": "LIST",
	     "name": "MinionModifier",
	     "nested": {
	      "flags": [],
	      "keywordFlags": [],
	      "modType": "INC",
	      "name": "Speed",
	      "number": 30,
	      "tags": [
	       {
	        "effectName": "HulkingMiscreation",
	        "effectType": "Buff",
	        "tagType": "GlobalEffect",
	        "unscalable": true
	       },
	       {
	        "monsterTag": "Construct",
	        "tagType": "MonsterTag"
	       }
	      ],
	      "valueKind": "number"
	     },
	     "source": "Minion:Hulking Miscreation",
	     "tags": [],
	     "valueKind": "nestedMod"
	    },
	    {
	     "flags": [],
	     "keywordFlags": [],
	     "modType": "LIST",
	     "name": "MinionModifier",
	     "nested": {
	      "flags": [],
	      "keywordFlags": [],
	      "modType": "INC",
	      "name": "Damage",
	      "number": 100,
	      "tags": [
	       {
	        "effectName": "HulkingMiscreation",
	        "effectType": "Buff",
	        "tagType": "GlobalEffect",
	        "unscalable": true
	       },
	       {
	        "monsterTag": "Construct",
	        "tagType": "MonsterTag"
	       }
	      ],
	      "valueKind": "number"
	     },
	     "source": "Minion:Hulking Miscreation",
	     "tags": [],
	     "valueKind": "nestedMod"
	    }
	   ],
	   "monsterTags": [
	    "animal_claw_weapon",
	    "construct",
	    "is_unarmed",
	    "melee",
	    "metal_armour",
	    "not_dex",
	    "not_int",
	    "physical_affinity",
	    "very_fast_movement"
	   ],
	   "name": "Hulking Miscreation",
	   "skillList": [
	    "AzmeriArgusMeleeAtAnimationSpeed",
	    "GAAzmeriRobotArgusSlam",
	    "EGAzmeriRobotArgusOverchargeBuff",
	    "EGAzmeriRobotArgusActivationWave",
	    "SSMAzmeriRobotArgusItsRainingMannequins"
	   ]
	  },
	  "Metadata/Monster/CageSpider/CageSpider2": {
	   "accuracy": 1,
	   "armour": 0.75,
	   "attackRange": 12,
	   "attackTime": 1.755,
	   "chaosResist": 0,
	   "coldResist": 0,
	   "damage": 1.28,
	   "damageSpread": 0.2,
	   "fireResist": 0,
	   "id": "Metadata/Monster/CageSpider/CageSpider2",
	   "life": 4.4,
	   "lightningResist": 40,
	   "limit": "ActiveSpectreLimit",
	   "loc": 514,
	   "modList": [],
	   "monsterTags": [
	    "caster",
	    "flesh_armour",
	    "is_unarmed",
	    "large_height",
	    "melee",
	    "not_dex",
	    "not_int",
	    "physical_affinity",
	    "ranged",
	    "skeleton",
	    "slashing_weapon",
	    "slow_movement",
	    "undead"
	   ],
	   "name": "Sandworn Slaves",
	   "skillList": [
	    "Melee",
	    "CageSpiderCycloneTriggerSandstorms",
	    "CageSpiderSandSpark"
	   ]
	  },
	  "Metadata/Monster/KitavaDemon/KitavaDemon": {
	   "accuracy": 1,
	   "armour": 0.5,
	   "attackRange": 10,
	   "attackTime": 1.455,
	   "chaosResist": 0,
	   "coldResist": 37,
	   "damage": 1.4,
	   "damageFixup": 0.33,
	   "damageSpread": 0.2,
	   "evasion": 0.4,
	   "fireResist": 37,
	   "id": "Metadata/Monster/KitavaDemon/KitavaDemon",
	   "life": 3.6,
	   "lightningResist": 37,
	   "limit": "ActiveSpectreLimit",
	   "loc": 1530,
	   "modList": [],
	   "monsterTags": [
	    "cannot_be_monolith",
	    "caster",
	    "cleaving_weapon",
	    "cultist",
	    "demon",
	    "fire_affinity",
	    "flesh_armour",
	    "has_one_hand_sword",
	    "has_one_handed_melee",
	    "humanoid",
	    "medium_height",
	    "melee",
	    "not_int",
	    "physical_affinity",
	    "red_blood",
	    "slow_movement"
	   ],
	   "name": "Kitava's Herald",
	   "skillList": [
	    "KitavaDemonLeapSlam",
	    "KitavaDemonCleave",
	    "KitavaDemonWhirlingBlades",
	    "KitavaDemonXMortar",
	    "Melee"
	   ],
	   "weaponType1": "One Handed Sword"
	  }
	 }
	}`
}

func mdSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, mdPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{mdPy: string(py), "dump.json": mdDump(), "adir/keep": "a dir\n"}
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

func mdPut(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func mdSpec(bin string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool: "minion-data-emit", PythonScript: mdPy, PythonArgs: args,
		GoBin: bin, GoArgs: append([]string{"minion-data-emit"}, args...),
		Stderr:       difftest.StderrExact,
		Files:        []string{"out.go"},
		RefusalSites: 1,
	}
}

func TestMinionDataEmit(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := mdSandbox(t, repo)

	spec := mdSpec(bin, "dump.json", "out.go", "minions")
	spec.Fixtures = []difftest.Fixture{
		{Name: "happy", Passes: true},

		{Name: "no-args", Refusal: "usage: minion_data_emit.py <dump.json> <out.go> <minions|spectres>",
			Break: func(string) error { return nil }},
		{Name: "four-args", Refusal: "usage: minion_data_emit.py <dump.json> <out.go> <minions|spectres>",
			Break: func(string) error { return nil }},

		{Name: "unknown-table", Refusal: "REFUSING: unknown table 'nope'; have minions, spectres",
			Break: func(string) error { return nil }},

		{Name: "dump-missing",
			Refusal: "REFUSING: the dump (dump.json) is not readable: No such file or directory",
			Break:   func(sb string) error { return os.Remove(filepath.Join(sb, "dump.json")) }},
		{Name: "dump-not-json", Refusal: "REFUSING: the dump (dump.json) is not valid JSON",
			Break: func(sb string) error { return mdPut(sb, "dump.json", "{ nope") }},
		{Name: "dump-not-an-object", Refusal: "REFUSING: the dump (dump.json) is not a JSON object",
			Break: func(sb string) error { return mdPut(sb, "dump.json", "[]") }},
		{Name: "table-is-empty",
			Refusal: "REFUSING: the dump has no 'minions' entries. Re-run `go run ./scripts/gamedatadump minion-data`.",
			Break:   func(sb string) error { return mdPut(sb, "dump.json", `{"minions": {}}`) }},

		{Name: "a-row-has-no-loc",
			Refusal: "REFUSING: Alpha: the dumper found no declaring line to cite",
			Break: func(sb string) error {
				return mdPut(sb, "dump.json", `{"minions": {"A": {"id": "Alpha", "name": "A"}}}`)
			}},
		{Name: "a-row-has-a-zero-loc",
			Refusal: "REFUSING: Alpha: the dumper found no declaring line to cite",
			Break: func(sb string) error {
				return mdPut(sb, "dump.json", `{"minions": {"A": {"id": "Alpha", "name": "A", "loc": 0}}}`)
			}},
		{Name: "a-row-has-a-non-numeric-loc",
			Refusal: "REFUSING: Alpha: the declaring line 'abc' is not a number",
			Break: func(sb string) error {
				return mdPut(sb, "dump.json", `{"minions": {"A": {"id": "Alpha", "name": "A", "loc": "abc"}}}`)
			}},

		{Name: "out-path-is-a-directory",
			Refusal: "REFUSING: adir is not writable: Is a directory",
			Break:   func(string) error { return nil }},
	}

	pySrc := difftest.ReadOriginal(t, repo, mdPy)
	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			switch f.Name {
			case "no-args":
				s.PythonArgs = nil
			case "four-args":
				s.PythonArgs = []string{"dump.json", "out.go", "minions", "extra"}
			case "unknown-table":
				s.PythonArgs = []string{"dump.json", "out.go", "nope"}
			case "out-path-is-a-directory":
				s.PythonArgs = []string{"dump.json", "adir", "minions"}
			}
			s.GoArgs = append([]string{"minion-data-emit"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestMinionDataEmitSpectres: THE SPECTRES SIDE IS A SEPARATE RUN, NOT A FIXTURE.
func TestMinionDataEmitSpectres(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	difftest.RunFixture(t, t.Context(), mdSpec(bin, "dump.json", "out.go", "spectres"),
		difftest.Fixture{Name: "spectres", Passes: true}, mdSandbox(t, repo))
}

// TestMinionDataEmitKeyDeletion: THE REVERSE DIRECTION: delete each key of the input in turn and require the two implementations to still agree.
func TestMinionDataEmitKeyDeletion(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	difftest.AssertKeyDeletionAgrees(t, t.Context(), mdSpec(bin, "dump.json", "out.go", "spectres"),
		[]string{"dump.json"}, mdSandbox(t, repo))
}
