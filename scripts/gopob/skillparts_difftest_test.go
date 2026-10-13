package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const spPy = "oracle/skill_parts.py"

func spPartLua(name string, flags ...string) string {
	b := "\t\t{\n\t\t\tname = \"" + name + "\",\n"
	for _, f := range flags {
		b += "\t\t\t" + f + "\n"
	}
	return b + "\t\t},\n"
}

func spSkillLua(id string, parts ...string) string {
	b := "skills[\"" + id + "\"] = {\n\tname = \"" + id + " Display\",\n\tparts = {\n"
	for _, p := range parts {
		b += p
	}
	return b + "\t},\n}\n"
}

func spTree() map[string]string {
	files := map[string]string{
		"act_dex.lua": spSkillLua("ZetaStrike", spPartLua("Slash"), spPartLua("Thrust",
			"trap = true,", "mine = false,")) +
			spSkillLua("YankeeShot", spPartLua("Single"), spPartLua("Volley", "totem = true,")) +
			spSkillLua("TiedId", spPartLua("From act_dex.lua")),
		"act_int.lua": "skills[\"Autopulse\"] = {\n\tname = \"Autopulse Display\",\n\tparts = {\n" +
			"\t\t{\n\t\t\t\tname = \"Deep Indent\",\n\n\t\t\t\tspell = true,\n\t\t},\n\t},\n}\n" +
			spSkillLua("XrayBolt", spPartLua("Only")),
		"act_str.lua": spSkillLua("WarcrySlam", spPartLua("Melee", "attack = true,")) +
			spSkillLua("VaultCharge", spPartLua("Charge"), spPartLua("Impact")),
		"glove.lua": "skills[\"UmbralGrasp\"] = {\n\tname = \"Umbral Display\",\n" +
			"\tmeta = {\n\t\tparts = {\n\t\t\tignored = true,\n\t\t},\n\t},\n" +
			"\tparts = {\n" + spPartLua("Backslash \\\\b Kept") + "\t},\n}\n",
		"minion.lua": spSkillLua("TangleSummon", spPartLua("Alpha"), spPartLua("Beta")) +
			spSkillLua("TiedId", spPartLua("From minion.lua")),
		"other.lua": spSkillLua("SableAura", spPartLua("Aura", "aura = true,", "buff = false,")) +
			spSkillLua("ControlChar", spPartLua("Bell\x01Inside")),
		"spectre.lua": spSkillLua("RavenGhost", spPartLua("Haunt")),
		"sup_dex.lua": spSkillLua("QuiverSupport", spPartLua("Support Part")) +
			"skills[\"NbspSkill\"] = {\n\tparts = {\n\t\t{\n\t\t\tname = \"Nbsp Name\",\n\t\t},\n\t},\n}\n",
		"sup_int.lua": spSkillLua("PrismSupport", spPartLua("First"), spPartLua("Second")) +
			spSkillLua("OrbSupport", spPartLua("Lone")) +
			spSkillLua("TiedId", spPartLua("From sup_int.lua")),
		"sup_str.lua": spSkillLua("NimbusSupport", spPartLua("One")) +
			spSkillLua("MarrowSupport", spPartLua("Two")) +
			spSkillLua("AlphaSupport", spPartLua("Three")),
	}
	files["spectre.lua"] = strings.ReplaceAll(files["spectre.lua"], "\n", "\r\n")
	out := map[string]string{}
	for name, body := range files {
		out["pob/src/Data/Skills/"+name] = body
	}
	return out
}

func spSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	src := difftest.ReadOriginal(t, repo, spPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{spPy: string(src)}
		for k, v := range spTree() {
			files[k] = v
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

func spPutLua(sb, name, body string) error {
	dst := filepath.Join(sb, "pob/src/Data/Skills", name)
	if err := os.WriteFile(dst, []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func TestSkillParts(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := spSandbox(t, repo)
	pySrc := difftest.ReadOriginal(t, repo, spPy)

	spec := difftest.Spec{
		Tool:         "skill-parts",
		PythonScript: spPy,
		PythonArgs:   []string{"out.go"},
		GoBin:        bin,
		GoArgs:       []string{"skill-parts", "out.go"},
		Env:          []string{"POB_DIR=pob"},
		Stderr:       difftest.StderrExact,
		Files:        []string{"out.go"},
		RefusalSites: 1,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "usage: skill_parts.py <out.go>",
				Break: func(string) error { return nil }},
			{Name: "two-args", Refusal: "usage: skill_parts.py <out.go>",
				Break: func(string) error { return nil }},
			{Name: "a-skill-file-is-missing",
				Refusal: "REFUSING: src/Data/Skills/act_dex.lua is not readable: No such file or directory",
				Break: func(sb string) error {
					return os.Remove(filepath.Join(sb, "pob/src/Data/Skills/act_dex.lua"))
				}},
			{Name: "unrecognised-line-in-a-part",
				Refusal: "REFUSING: src/Data/Skills/act_dex.lua:5: unrecognised line in a part: '\\t\\t\\tlevel = 3,'",
				Break: func(sb string) error {
					return spPutLua(sb, "act_dex.lua", "skills[\"Z\"] = {\n\tparts = {\n"+
						"\t\t{\n\t\t\tname = \"P\",\n\t\t\tlevel = 3,\n\t\t},\n\t},\n}\n")
				}},
			{Name: "not-a-part-constructor",
				Refusal: "REFUSING: src/Data/Skills/act_dex.lua:3: expected a part constructor, got '\\t\\tnope = 1,'",
				Break: func(sb string) error {
					return spPutLua(sb, "act_dex.lua", "skills[\"Z\"] = {\n\tparts = {\n"+
						"\t\tnope = 1,\n\t},\n}\n")
				}},
			{Name: "block-runs-past-end-of-file",
				Refusal: "REFUSING: src/Data/Skills/act_dex.lua: a parts block runs past the end of the file",
				Break: func(sb string) error {
					return spPutLua(sb, "act_dex.lua", "skills[\"Z\"] = {\n\tparts = {\n\t\t{\n")
				}},
			{Name: "a-part-declares-no-name",
				Refusal: "REFUSING: src/Data/Skills/act_dex.lua:5: a part declares no name",
				Break: func(sb string) error {
					return spPutLua(sb, "act_dex.lua", "skills[\"Z\"] = {\n\tparts = {\n"+
						"\t\t{\n\t\t\ttrap = true,\n\t\t},\n\t},\n}\n")
				}},
			{Name: "output-is-a-directory",
				Refusal: "REFUSING: out.go is not writable: Is a directory",
				Break:   func(sb string) error { return os.Mkdir(filepath.Join(sb, "out.go"), 0o750) }},
			{Name: "crlf-file-reaches-a-refusal",
				Refusal: `REFUSING: src/Data/Skills/act_dex.lua:5: unrecognised line in a part: '\t\t\tlevel = 3,'`,
				Break: func(sb string) error {
					return spPutLua(sb, "act_dex.lua", strings.ReplaceAll(
						"skills[\"Z\"] = {\n\tparts = {\n"+
							"\t\t{\n\t\t\tname = \"P\",\n\t\t\tlevel = 3,\n\t\t},\n\t},\n}\n",
						"\n", "\r\n"))
				}},
			{Name: "an-escaped-quote-is-not-accepted",
				Refusal: `REFUSING: src/Data/Skills/act_dex.lua:4: unrecognised line in a part: '\t\t\tname = "Quote \\" Here",'`,
				Break: func(sb string) error {
					return spPutLua(sb, "act_dex.lua", "skills[\"Z\"] = {\n\tparts = {\n"+
						"\t\t{\n\t\t\tname = \"Quote \\\" Here\",\n\t\t},\n\t},\n}\n")
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
			case "two-args":
				s.PythonArgs = []string{"out.go", "extra"}
			}
			if f.Name != "happy" {
				s.Files = nil
			}
			s.GoArgs = append([]string{"skill-parts"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

func TestSkillPartsPythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := spSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "skill-parts",
		PythonScript: spPy,
		PythonArgs:   []string{"out.go"},
		Env:          []string{"POB_DIR=pob"},
		Files:        []string{"out.go"},
	}, mk, []string{"0", "12345", "999983"})
}
