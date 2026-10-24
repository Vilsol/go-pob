package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const ttePy = "oracle/tree_tables_emit.py"

func tteOursJSON() string {
	return `{
  "version": "3_29",
  "nodes": {
    "1000": {"name": "Identical", "stats": ["Same Line"]},
    "2000": {"name": "Wording", "stats": ["Cannot be Blinded"]},
    "3000": {"name": "Purposeful Harbinger", "stats": ["Auras from your Skills have 8% increased Effect on you for\neach Herald affecting you, up to 40%"]},
    "4000": {"name": "Candidate Join", "stats": ["Projectiles gain Damage — as they travel farther, dealing up\nto 50% more Damage"]},
    "5000": {"name": "Complete Lines", "stats": ["First Complete Line\nSecond Complete Line"]},
    "6000": {"name": "Real Wording", "stats": ["Our Wording Here"]},
    "root": {"name": "Root", "stats": ["Not A Number Id"]}
  }
}`
}

func ttePobJSON() string {
	return `{
  "version": "2.67.2-fixture",
  "nodes": {
    "7000": {"name": "PoB Only", "stats": ["Missing Node Stat"], "group": 123, "ascendancy": "Slayer"},
    "42469": {"name": "Fatal Flourish", "stats": ["Aliased Stat"], "group": 456},
    "6000": {"name": "Real Wording", "stats": ["PoB Wording Here"]},
    "4000": {"name": "Candidate Join", "stats": ["Projectiles gain Damage — as they travel farther, dealing up", "to 50% more Damage"]},
    "2000": {"name": "Wording", "stats": ["100% chance to Avoid Blind"]},
    "5000": {"name": "Complete Lines", "stats": ["First Complete Line", "Second Complete Line"]},
    "1000": {"name": "Identical", "stats": ["Same Line"]},
    "3000": {"name": "Purposeful Harbinger", "stats": ["Auras from your Skills have 8% increased Effect on you for", "each Herald affecting you, up to 40%"]}
  }
}`
}

func tteSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	src := difftest.ReadOriginal(t, repo, ttePy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			ttePy:       string(src),
			"ours.json": tteOursJSON(),
			"pob.json":  ttePobJSON(),
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

func ttePut(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func tteOneNode(id, node string) string {
	return `{"version": "v", "nodes": {"` + id + `": ` + node + `}}`
}

func TestTreeTablesEmit(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := tteSandbox(t, repo)
	pySrc := difftest.ReadOriginal(t, repo, ttePy)

	base := []string{"ours.json", "pob.json", "missing.go", "fixups.go"}
	spec := difftest.Spec{
		Tool:         "tree-tables-emit",
		PythonScript: ttePy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"tree-tables-emit"}, base...),
		Stderr:       difftest.StderrExact,
		Files:        []string{"missing.go", "fixups.go"},
		RefusalSites: 1,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "too-few-args", Refusal: "python3 oracle/tree_tables_emit.py <ours.json>",
				Break: func(string) error { return nil }},

			{Name: "ours-missing", Refusal: "REFUSING: ours.json is not readable: No such file or directory",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "ours.json")) }},
			{Name: "pob-missing", Refusal: "REFUSING: pob.json is not readable: No such file or directory",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "pob.json")) }},
			{Name: "ours-not-json", Refusal: "REFUSING: ours.json is not valid JSON",
				Break: func(sb string) error { return ttePut(sb, "ours.json", "{ nope") }},
			{Name: "ours-not-an-object", Refusal: "REFUSING: ours.json is not a JSON object",
				Break: func(sb string) error { return ttePut(sb, "ours.json", `[]`) }},
			{Name: "ours-has-no-nodes-key", Refusal: "REFUSING: ours.json has no 'nodes' object",
				Break: func(sb string) error { return ttePut(sb, "ours.json", `{"version": "v"}`) }},
			{Name: "ours-has-empty-nodes", Refusal: "REFUSING: a dump has no nodes",
				Break: func(sb string) error { return ttePut(sb, "ours.json", `{"nodes": {}}`) }},
			{Name: "a-node-is-not-an-object", Refusal: "REFUSING: pob.json: node 7000 is not a JSON object",
				Break: func(sb string) error { return ttePut(sb, "pob.json", tteOneNode("7000", `"scalar"`)) }},

			{Name: "a-pob-node-id-is-not-a-number",
				Refusal: `REFUSING: node id 'rootish' in the PoB tree is not a number`,
				Break: func(sb string) error {
					return ttePut(sb, "pob.json", tteOneNode("rootish",
						`{"name": "N", "stats": ["S"], "group": 1}`))
				}},
			{Name: "a-missing-node-has-no-group",
				Refusal: "REFUSING: node 7000 (PoB Only) has no group",
				Break: func(sb string) error {
					return ttePut(sb, "pob.json", tteOneNode("7000", `{"name": "PoB Only", "stats": ["S"]}`))
				}},
			{Name: "missing-output-is-a-directory",
				Refusal: "REFUSING: missing.go is not writable: Is a directory",
				Break:   func(sb string) error { return os.Mkdir(filepath.Join(sb, "missing.go"), 0o750) }},
			{Name: "fixups-output-is-a-directory",
				Refusal: "REFUSING: fixups.go is not writable: Is a directory",
				Break:   func(sb string) error { return os.Mkdir(filepath.Join(sb, "fixups.go"), 0o750) }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			if f.Name == "too-few-args" {
				s.PythonArgs = []string{"ours.json", "pob.json", "missing.go"}
			}
			if f.Name != "happy" {
				s.Files = nil
			}
			s.GoArgs = append([]string{"tree-tables-emit"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}

	difftest.AssertKeyDeletionAgrees(t, t.Context(), spec, []string{"ours.json", "pob.json"}, mk)
}

// TestTreeTablesEmitOrderIsStableAcrossRuns: REPEATED, BECAUSE Go MAP ORDER VARIES PER PROCESS AND THE DIVERGENCE IS IN THE DATA.
func TestTreeTablesEmitOrderIsStableAcrossRuns(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := tteSandbox(t, repo)
	base := []string{"ours.json", "pob.json", "missing.go", "fixups.go"}
	spec := difftest.Spec{
		Tool:         "tree-tables-emit",
		PythonScript: ttePy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"tree-tables-emit"}, base...),
		Stderr:       difftest.StderrExact,
		Files:        []string{"missing.go", "fixups.go"},
		RefusalSites: 1,
	}
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprintf("run%d", i), func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), spec, difftest.Fixture{Name: "happy"}, mk)
		})
	}
}
