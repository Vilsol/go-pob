package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const ctsSh = "scripts/classify_tree_sources.sh"

func ctsID(id string) string { return fmt.Sprintf("\t\t[\"id\"] = %q,\n", id) }

func ctsLua(ids ...string) string {
	var b strings.Builder
	b.WriteString("return {\n")
	for _, id := range ids {
		b.WriteString(ctsID(id))
	}
	b.WriteString("}\n")
	return b.String()
}

var ctsDumps = map[string][]string{
	"b1": {"Tree:t_alpha", "Tree:l_notable_y"},
	"b2": {"Tree:t_alpha", "Tree:l_keystone_x"},
	"b3": {"Tree:t_bravo", "Tree:u_mystery"},
	"b4": {"Tree:t_charlie", "Tree:templar_devotion_node"},
	"b5": {"Tree:t_alpha", "Tree:t_alpha"},
	"b6": {"Tree:34171"},
	"b7": {"Tree:l_keystoneish_notable"},
}

func ctsSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	sh := difftest.ReadOriginal(t, repo, ctsSh)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			ctsSh:                             string(sh),
			"pob/src/Data/TattooPassives.lua": ctsLua("t_alpha", "t_bravo", "t_charlie"),
			"pob/src/Data/TimelessJewelData/LegionPassives.lua": ctsLua(
				"l_keystone_x", "templar_devotion_node", "eternal_small_blank", "l_notable_y",
				"l_notable_z",
				"l_keystoneish_notable"),
		}
		for b, sources := range ctsDumps {
			var sb2 strings.Builder
			sb2.WriteString("{\n")
			for _, s := range sources {
				fmt.Fprintf(&sb2, "  %q: 1,\n", s)
			}
			sb2.WriteString("}\n")
			files[filepath.Join("dumps", b+".json")] = sb2.String()
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

func TestClassifyTreeSourcesPortIsIndistinguishable(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ctsSandbox(t, repo)

	shSrc := difftest.ReadOriginal(t, repo, ctsSh)

	spec := difftest.Spec{
		Tool:         "classify-tree-sources",
		PythonScript: ctsSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   []string{"dumps"},
		GoBin:        bin,
		GoArgs:       []string{"classify-tree-sources", "dumps"},
		Env:          []string{"POB_DIR=pob", "GOPOB_NICED=1"},
		Stderr:       difftest.StderrExact,
		RefusalSites: 4,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "tattoo-unreadable", Refusal: "REFUSING: cannot read pob/src/Data/TattooPassives.lua",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "pob/src/Data/TattooPassives.lua"), 0o000)
				}},

			{Name: "no-dump-set", Refusal: "REFUSING: no dump set at dumps",
				Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "dumps")) }},

			{Name: "id-sets-overlap", Refusal: "REFUSING: id sets overlap; classification is not well defined",
				Break: func(sb string) error {
					return put(sb, "pob/src/Data/TimelessJewelData/LegionPassives.lua",
						ctsLua("l_keystone_x", "t_alpha"))
				}},

			{Name: "no-named-sources", Refusal: "REFUSING: not one named Tree: source in 1 dumps",
				Break: func(sb string) error {
					if err := os.RemoveAll(filepath.Join(sb, "dumps")); err != nil {
						return fmt.Errorf("fixture: %w", err)
					}
					if err := os.MkdirAll(filepath.Join(sb, "dumps"), 0o750); err != nil {
						return fmt.Errorf("fixture: %w", err)
					}
					return put(sb, "dumps/only.json", "{ \"Tree:34171\": 1 }\n")
				}},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, shSrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "tattoo-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			difftest.RunFixture(t, t.Context(), spec, f, mk)
		})
	}
}

func TestClassifyTreeSourcesIsStableAcrossLocales(t *testing.T) {
	repo := repoRoot(t)
	mk := ctsSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "classify-tree-sources",
		PythonScript: ctsSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   []string{"dumps"},
		Env:          []string{"POB_DIR=pob", "GOPOB_NICED=1"},
	}, mk, []string{"C", "en_US.utf8", "C.utf8"})
}
