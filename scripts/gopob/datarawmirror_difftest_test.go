package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const (
	dmPy    = "scripts/dataraw_mirror.py"
	dmLuaPy = "scripts/lua_data.py"
)

func dmDumpTest() string {
	return `package raw

import (
	"os"
	"testing"
)

func TestDumpMirrorTables(t *testing.T) {
	path := os.Getenv("MIRROR_DUMP")
	if path == "" {
		t.Skip("no MIRROR_DUMP")
	}
	if err := os.WriteFile(path, []byte(dumpBlob), 0o600); err != nil {
		t.Fatal(err)
	}
}
`
}

func dmDumpBlob() string {
	return `package raw

const dumpBlob = ` + "`" + `{
  "Minions": {
    "Zombie": {"name": "Zombie", "life": 2, "energyShield": 0, "armour": 0,
      "fireResist": 0, "coldResist": 0, "lightningResist": 0, "chaosResist": 0,
      "damage": 1.5, "damageSpread": 0, "damageFixup": 0, "attackTime": 0,
      "attackRange": 0, "accuracy": 1, "weaponType1": "", "weaponType2": "",
      "limit": "", "skillList": ["MeleeAtAnimationSpeed"], "modList": ["Damage"]}
  },
  "Spectres": {
    "Ghost": {"name": "Ghost", "life": 99, "energyShield": 0, "armour": 0,
      "fireResist": 0, "coldResist": 0, "lightningResist": 0, "chaosResist": 0,
      "damage": 0, "damageSpread": 0, "damageFixup": 0, "attackTime": 0,
      "attackRange": 0, "accuracy": 1, "weaponType1": "", "weaponType2": "",
      "limit": "ActiveSpectreLimit", "skillList": [], "modList": []}
  },
  "SkillBaseFlags": {
    "Fireball": ["spell"],
    "OnlyOurs": ["attack"]
  },
  "SkillAddFlags": {},
  "PoBSkills": {
    "Fireball": {"name": "Fireball", "hidden": false, "fromItem": false,
      "fromTree": false, "support": false},
    "Aliased": {"name": "Aliased", "hidden": false, "fromItem": false,
      "fromTree": false, "support": false}
  },
  "SkillMinionLists": {},
  "SupportAddMinionLists": {},
  "SkillAddSkillTypes": {},
  "SkillMinionHasItemSet": {},
  "Gems": {"Metadata/Fireball": {"name": "Fireball", "tags": ["fire", "spell"]}},
  "SkillParts": {"Fireball": []},
  "SkillHiddenLevels": {},
  "SkillBaseMods": {},
  "SkillStatMap": {},
  "SkillStatMaps": {},
  "Unswept": {"x": 1}
}` + "`" + `
`
}

func dmFixtureLua(rel string) string {
	switch rel {
	case "src/Data/Minions.lua":
		return "\nminions[\"Zombie\"] = {\n" +
			"\tname = \"Zombie\",\n\tlife = 2,\n\tdamage = 1.5,\n" +
			"\tskillList = { \"MeleeAtAnimationSpeed\" },\n" +
			"\tmodList = { mod(\"Damage\", \"BASE\", 1) },\n}\n"
	case "src/Data/Spectres.lua":
		return "\nminions[\"Ghost\"] = {\n\tname = \"Ghøst\",\n\tlife = 3,\n}\n"
	case "src/Data/Skills/fake.lua":
		return "\nskills[\"Fireball\"] = {\n" +
			"\tname = \"Fireball\",\n" +
			"\tbaseFlags = { spell = true, notThisOne = false },\n" +
			"\tparts = { { name = \"P\", spell = true } },\n}\n" +
			"skills[\"Aliased\"] = {\n\tname = \"Aliased\",\n" +
			"\tparts = skills.Fireball.parts,\n}\n"
	case "src/Data/Gems.lua":
		return "return {\n\t[\"Metadata/Fireball\"] = { name = \"Fireball\", " +
			"tags = { fire = true, spell = true } },\n}\n"
	case "src/Data/SkillStatMap.lua":
		return "return {\n}\n"
	}
	return ""
}

var dmFixtureLuaFiles = []string{
	"src/Data/Minions.lua", "src/Data/Spectres.lua", "src/Data/Skills/fake.lua",
	"src/Data/Gems.lua", "src/Data/SkillStatMap.lua",
}

func dmParserGo() string {
	return "package builds\n\nvar pobGemIDtoGameGemIDs = map[string]string{\n" +
		"\t\"Metadata/OldName\": \"Metadata/NewName\",\n}\n"
}

func dmSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, dmPy)
	luaPy := difftest.ReadOriginal(t, repo, dmLuaPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			dmPy:                           string(py),
			dmLuaPy:                        string(luaPy),
			"go.mod":                       "module example.com/sandbox\n\ngo 1.24\n",
			"builds/parser.go":             dmParserGo(),
			"data/raw/mirror_dump_test.go": dmDumpTest(),
			"data/raw/mirror_dump_blob.go": dmDumpBlob(),
		}
		for _, rel := range dmFixtureLuaFiles {
			files["pob/"+rel] = dmFixtureLua(rel)
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
		for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}} {
			cmd := exec.CommandContext(t.Context(), "git", args...)
			cmd.Dir = sb
			if out, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("sandbox: git %s: %w: %s", args[0], err, out)
			}
		}
		return sb, nil
	}
}

func dmSpecFor(bin string, env []string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool:         "dataraw-mirror",
		PythonScript: dmPy,
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"dataraw-mirror"}, args...),
		Env:          env,
		Stderr:       difftest.StderrExact,
		Library:      []string{dmLuaPy},
	}
}

// TestDataRawMirrorAgainstPython is the sweep plus every refusal a sandbox state can reach.
func TestDataRawMirrorAgainstPython(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := dmSandbox(t, repo)
	s := dmSpecFor(bin, []string{"POB_DIR=pob"})
	s.RefusalSites = 2
	s.Fixtures = []difftest.Fixture{
		{Name: "happy"},
		{
			Name:    "not-a-git-repository",
			Refusal: "git rev-parse --show-toplevel failed: exit status 128",
			Break: func(sb string) error {
				return os.RemoveAll(filepath.Join(sb, ".git"))
			},
		},
		{
			Name:    "pob-dir-has-no-src",
			Refusal: "has no src/ -- set POB_DIR",
			Break: func(sb string) error {
				return os.RemoveAll(filepath.Join(sb, "pob", "src"))
			},
		},
		{
			Name:    "a-source-is-not-utf8",
			Refusal: "is not valid UTF-8",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "pob/src/Data/Minions.lua"),
					[]byte("\nminions[\"Z\"] = { name = \"\xff\xfe\" }\n"), 0o600)
			},
		},
		{
			Name:    "the-go-dump-fails",
			Refusal: "the Go dump failed; nothing was measured",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "data/raw/mirror_dump_blob.go"),
					[]byte("package raw\n\nthis is not go\n"), 0o600)
			},
		},
		{
			Name:    "our-own-parser-go-is-gone",
			Refusal: "builds/parser.go is not readable",
			Break: func(sb string) error {
				return os.Remove(filepath.Join(sb, "builds/parser.go"))
			},
		},
	}
	difftest.AssertRefusalSiteCount(t, s, dmReadPy(t, repo))
	for _, f := range s.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.NoFixture {
				t.Skip("no fixture reaches this site: " + f.Note)
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

func dmReadPy(t *testing.T, repo string) []byte {
	t.Helper()
	b := difftest.ReadOriginal(t, repo, dmPy)
	return b
}

// TestDataRawMirrorModes sweeps --rows on BOTH sides of the truncation boundary.
func TestDataRawMirrorModes(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := dmSandbox(t, repo)
	for _, tc := range []struct {
		name string
		env  []string
		args []string
	}{
		{"default-rows", []string{"POB_DIR=pob"}, nil},
		{"rows-1-truncates", []string{"POB_DIR=pob"}, []string{"--rows", "1"}},
		{"rows-0", []string{"POB_DIR=pob"}, []string{"--rows", "0"}},
		{"rows-40", []string{"POB_DIR=pob"}, []string{"--rows", "40"}},
		{"rows-equals-form", []string{"POB_DIR=pob"}, []string{"--rows=2"}},
		{"report-only", []string{"POB_DIR=pob", "REPORT=1"}, nil},
		{"report-0-is-not-report", []string{"POB_DIR=pob", "REPORT=0"}, nil},
		{"selftest", []string{"POB_DIR=pob"}, []string{"--selftest"}},
		{"help", []string{"POB_DIR=pob"}, []string{"--help"}},
		{"help-short", []string{"POB_DIR=pob"}, []string{"-h"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), dmSpecFor(bin, tc.env, tc.args...),
				difftest.Fixture{Name: tc.name}, mk)
		})
	}
}

func TestDataRawMirrorArgumentGrammar(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := dmSandbox(t, repo)
	for _, tc := range []struct {
		name    string
		args    []string
		refusal string
	}{
		{"unknown-flag", []string{"--nope"}, "unknown argument '--nope'"},
		{"positional", []string{"wat"}, "unknown argument 'wat'"},
		{"rows-needs-a-value", []string{"--rows"}, "--rows needs a value"},
		{"rows-not-an-integer", []string{"--rows", "abc"}, "--rows wants an integer, not 'abc'"},
		{"rows-empty", []string{"--rows", ""}, "--rows wants an integer, not ''"},
		{"selftest-takes-no-value", []string{"--selftest=1"}, "--selftest takes no value"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), dmSpecFor(bin, []string{"POB_DIR=pob"}, tc.args...),
				difftest.Fixture{Name: tc.name, Refusal: tc.refusal}, mk)
		})
	}
}

// TestDataRawMirrorPythonIsStableAcrossHashSeeds guards against a source of nondeterminism a differential run cannot see.
func TestDataRawMirrorPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := dmSandbox(t, repo)
	s := difftest.Spec{
		Tool:         "dataraw-mirror",
		PythonScript: dmPy,
		PythonArgs:   []string{"--rows", "40"},
		Env:          []string{"POB_DIR=pob", "REPORT=1"},
	}
	difftest.SelfIdentity(t, t.Context(), s, mk, []string{"0", "12345", "999983"})
}

// TestDataRawMirrorGoDumpIsActuallyRun is the control on the sandbox, not on the tool.
func TestDataRawMirrorGoDumpIsActuallyRun(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := dmSandbox(t, repo)
	sb, err := mk()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), bin, "dataraw-mirror")
	cmd.Dir = sb
	cmd.Env = append(os.Environ(), "POB_DIR=pob")
	out, _ := cmd.Output()
	body := string(out)
	for _, want := range []string{
		"Minions                 IDENTICAL",
		"Spectres                DIFFERS",
		"SkillParts              UNMAPPED",
		"⚠️  IN THE Go DUMP BUT NOT IN THIS SPEC -- unswept: Unswept",
		"⚠️  SPECED BUT ABSENT FROM THE Go DUMP: SkillMinionUses",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("the fixture never reached the code it claims to cover: %q missing\n%s", want, body)
		}
	}
}
