package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const tdPy = "oracle/tree_divergence.py"

func tdOursJSON() string {
	return `{
  "version": "3_29",
  "nodes": {
    "100": {"name": "Ours Only Wording", "stats": ["Cannot be Blinded"]},
    "300": {"name": "Reordered", "stats": ["10% increased Damage", "5% increased Life"]},
    "5087": {"name": "Born in the Shadows", "stats": ["Cannot be Blinded", "  ", ""]},
    "49639": {"name": "Supreme Ego", "stats": ["Split A\nSplit B"]},
    "900": {"name": "Agrees", "stats": ["Identical Line"]}
  }
}`
}

func tdPobJSON() string {
	return `{
  "version": "2.67.2-fixture",
  "nodes": {
    "5087": {"name": "Born in the Shadows", "stats": ["100% chance to Avoid Blind\n"]},
    "49639": {"name": "Supreme Ego", "stats": ["Split A", "Split B"]},
    "100": {"name": "Ours Only Wording", "stats": ["Cannot be Blinded", "and one more"]},
    "300": {"name": "Reordered", "stats": ["5% increased Life", "10% increased Damage"]},
    "900": {"name": "Agrees", "stats": ["Identical Line"]},
    "777": {"name": "PoB Only Node", "stats": ["Some Stat"]},
    "20": {"name": "PoB Only Node Sorting Second", "stats": ["Other Stat"]}
  }
}`
}

func tdSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	src := difftest.ReadOriginal(t, repo, tdPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			tdPy:        string(src),
			"ours.json": tdOursJSON(),
			"pob.json":  tdPobJSON(),
			"notjson":   "{ nope",
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

func tdPut(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func TestTreeDivergence(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := tdSandbox(t, repo)
	pySrc := difftest.ReadOriginal(t, repo, tdPy)

	base := []string{"ours.json", "pob.json"}
	spec := difftest.Spec{
		Tool:         "tree-divergence",
		PythonScript: tdPy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"tree-divergence"}, base...),
		Stderr:       difftest.StderrExact,
		RefusalSites: 1,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "usage: tree_divergence.py <ours.json> <pob.json>",
				Break: func(string) error { return nil }},
			{Name: "ours-missing", Refusal: "REFUSING: ours.json is not readable: No such file or directory",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "ours.json")) }},
			{Name: "pob-missing", Refusal: "REFUSING: pob.json is not readable: No such file or directory",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "pob.json")) }},
			{Name: "ours-not-json", Refusal: "REFUSING: ours.json is not valid JSON",
				Break: func(sb string) error { return tdPut(sb, "ours.json", "{ nope") }},
			{Name: "ours-not-an-object", Refusal: "REFUSING: ours.json is not a JSON object",
				Break: func(sb string) error { return tdPut(sb, "ours.json", `[]`) }},
			{Name: "ours-has-no-nodes-key", Refusal: "REFUSING: ours.json has no 'nodes' object",
				Break: func(sb string) error { return tdPut(sb, "ours.json", `{"version": "3_29"}`) }},
			{Name: "ours-has-empty-nodes", Refusal: "REFUSING: one of the dumps has no nodes",
				Break: func(sb string) error {
					return tdPut(sb, "ours.json", `{"version": "3_29", "nodes": {}}`)
				}},
			{Name: "a-node-has-no-name", Refusal: "REFUSING: node 777 has no 'name' string",
				Break: func(sb string) error {
					return tdPut(sb, "pob.json", `{"version": "v", "nodes": {
						"777": {"stats": ["Some Stat"]},
						"900": {"name": "Agrees", "stats": ["Identical Line"]}}}`)
				}},
			{Name: "a-node-has-no-stats", Refusal: "REFUSING: node 900 has no 'stats' list",
				Break: func(sb string) error {
					return tdPut(sb, "pob.json", `{"version": "v", "nodes": {
						"900": {"name": "Agrees"}}}`)
				}},
			{Name: "a-stat-is-not-a-string", Refusal: "REFUSING: a node's stats hold a int, not a string",
				Break: func(sb string) error {
					return tdPut(sb, "pob.json", `{"version": "v", "nodes": {
						"900": {"name": "Agrees", "stats": [7]}}}`)
				}},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			if f.Name == "no-args" {
				s.PythonArgs = []string{"ours.json"}
			}
			s.GoArgs = append([]string{"tree-divergence"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}

	difftest.AssertKeyDeletionAgrees(t, t.Context(), spec, []string{"ours.json", "pob.json"}, mk)
}

// TestTreeDivergenceOrderIsStableAcrossRuns: THE ORDERING CONTROL, AND IT IS THE ONE THAT MUST REPEAT.
func TestTreeDivergenceOrderIsStableAcrossRuns(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := tdSandbox(t, repo)
	base := []string{"ours.json", "pob.json"}
	spec := difftest.Spec{
		Tool:         "tree-divergence",
		PythonScript: tdPy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"tree-divergence"}, base...),
		Stderr:       difftest.StderrExact,
		RefusalSites: 1,
	}
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprintf("run%d", i), func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), spec, difftest.Fixture{Name: "happy"}, mk)
		})
	}
}

func TestTreeDivergencePythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := tdSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "tree-divergence",
		PythonScript: tdPy,
		PythonArgs:   []string{"ours.json", "pob.json"},
	}, mk, []string{"0", "12345", "999983"})
}
