package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const tcPy = "scripts/tag_census_check.py"

func tcDump(treeSources int, mods map[string][]map[string]any) string {
	modDB := map[string][]map[string]any{}
	srcs := make([]map[string]any, 0, treeSources+1)
	carrier := func(src string) map[string]any {
		return map[string]any{
			"source": src, "type": "BASE", "flags": "-", "kw": "-",
			"fmt": "base = TreeCarrier|BASE|-|-|-", "tags": []string{},
		}
	}
	for i := range treeSources {
		srcs = append(srcs, carrier(fmt.Sprintf("Tree:%d", 1000+i)))
	}
	srcs = append(srcs, carrier("Tree:70000"))
	modDB["TreeCarrier"] = srcs
	for k, v := range mods {
		modDB[k] = v
	}
	b, _ := json.Marshal(map[string]any{"modDB": modDB})
	return string(b)
}

func tcFiles() map[string]string {
	out := map[string]string{}
	for i := range 184 {
		out[fmt.Sprintf("dumps/x%04d.json", i)] = tcDump(4, nil)
	}
	mk := func(name, typ, flags, kw, tail string, tags []string) map[string]any {
		m := map[string]any{
			"source": "Item", "type": typ, "flags": flags, "kw": kw,
			"fmt": fmt.Sprintf("blah = %s|%s|%s|%s|%s", name, typ, flags, kw, tail),
		}
		if tags != nil {
			m["tags"] = tags
		}
		return m
	}
	out["dumps/a0001.json"] = tcDump(5, map[string][]map[string]any{
		"Life":     {mk("Life", "BASE", "-", "-", "type=Condition/var=x", []string{"Condition"})},
		"Mana":     {mk("Mana", "INC", "-", "-", "type=Multiplier", []string{"Multiplier"})},
		"NoMarker": {{"source": "Item", "type": "BASE", "fmt": "nothing matches here"}},
		"Untyped":  {mk("Untyped", "BASE", "-", "-", "var=only", []string{"Condition"})},
	})
	out["dumps/a0002.json"] = tcDump(5, map[string][]map[string]any{
		"Disagree": {mk("Disagree", "BASE", "-", "-", "type=SkillType", []string{"Condition"})},
		"TieOne":   {mk("TieOne", "BASE", "-", "-", "type=AlphaTag", []string{"AlphaTag"})},
		"TieTwo":   {mk("TieTwo", "BASE", "-", "-", "type=BravoTag", []string{"BravoTag"})},
		"Repeated": {{
			"source": "Item", "type": "BASE", "flags": "-", "kw": "-",
			"fmt":  "a = Repeated|BASE|-|-|type=First blah = Repeated|BASE|-|-|type=Second",
			"tags": []string{"Second"},
		}},
	})
	tied := map[string][]map[string]any{}
	for i := range 14 {
		n := fmt.Sprintf("Tied%02d", i)
		tied[n] = []map[string]any{mk(n, "BASE", "-", "-", "type=TT"+n, []string{"TT" + n})}
	}
	heavy := make([]map[string]any, 0, 20)
	for range 20 {
		heavy = append(heavy, mk("Heavy", "BASE", "-", "-", "type=LateHeavy", []string{"LateHeavy"}))
	}
	tied["Heavy"] = heavy
	out["dumps/a0003.json"] = tcDump(5, tied)
	return out
}

func tcCleanFiles() map[string]string {
	out := map[string]string{}
	for i := range 184 {
		out[fmt.Sprintf("dumps/x%04d.json", i)] = tcDump(4, nil)
	}
	mk := func(name, tail string, tags []string) map[string]any {
		return map[string]any{
			"source": "Item", "type": "BASE", "flags": "-", "kw": "-",
			"fmt":  fmt.Sprintf("blah = %s|BASE|-|-|%s", name, tail),
			"tags": tags,
		}
	}
	out["dumps/a0001.json"] = tcDump(5, map[string][]map[string]any{
		"Life": {mk("Life", "type=Condition/var=x", []string{"Condition"})},
		"Mana": {mk("Mana", "type=Multiplier", []string{"Multiplier"})},
	})
	return out
}

func tcSandboxFrom(t *testing.T, repo string, files map[string]string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, tcPy)
	files[tcPy] = string(py)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
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

func tcSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, tcPy)
	_ = py
	return tcSandboxFrom(t, repo, tcFiles())
}

// TestTagCensusCheckCleanExtractionMatches: THE SUCCESS BRANCH, WHICH THE MAIN FIXTURE STRUCTURALLY CANNOT REACH.
func TestTagCensusCheckCleanExtractionMatches(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := tcSandboxFrom(t, repo, tcCleanFiles())
	spec := difftest.Spec{
		Tool:         "tag-census-check",
		PythonScript: tcPy,
		PythonArgs:   []string{"dumps"},
		GoBin:        bin,
		GoArgs:       []string{"tag-census-check", "dumps"},
		Stderr:       difftest.StderrExact,
	}
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "clean", Passes: true, Break: func(string) error { return nil }}, mk)

	difftest.AssertKeyDeletionAgrees(t, t.Context(), spec, []string{"dumps/a0001.json"}, mk)
}

func TestTagCensusCheckPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := tcSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, tcPy)

	spec := difftest.Spec{
		Tool:         "tag-census-check",
		PythonScript: tcPy,
		PythonArgs:   []string{"dumps"},
		GoBin:        bin,
		GoArgs:       []string{"tag-census-check", "dumps"},
		Stderr:       difftest.StderrExact,
		RefusalSites: 5,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-dumps", Refusal: "no dumps in dumps",
				Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "dumps")) }},
			{Name: "dump-unreadable", Refusal: "is not readable:",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "dumps/a0001.json"), 0o000)
				}},
			{Name: "dump-invalid-json", Refusal: "is not valid JSON",
				Break: func(sb string) error { return put(sb, "dumps/a0001.json", "{ not json") }},
			{Name: "exclusion-count-moved", Refusal: "exclusion count moved: the dump set changed",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "dumps/x0000.json")) }},
			{Name: "extraction-not-clean", Refusal: "extraction is not clean",
				Break: func(string) error { return nil }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "dump-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			difftest.RunFixture(t, t.Context(), spec, f, mk)
		})
	}
}

func TestTagCensusCheckPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := tcSandboxFrom(t, repo, tcCleanFiles())
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "tag-census-check",
		PythonScript: tcPy,
		PythonArgs:   []string{"dumps"},
	}, mk, []string{"0", "12345", "999983"})
}
