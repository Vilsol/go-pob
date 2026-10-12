package calculator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestSharedSkillFuncsHaveIdenticalPinBodies asserts that whenever skillFuncs registers ONE named function under SEVERAL granted-effect ids.
func TestSharedSkillFuncsHaveIdenticalPinBodies(t *testing.T) {
	skillDir := pinSkillDirOrSkip(t)

	shared, err := goSharedSkillFuncIdents(filepath.Join("..", "calculator", "skill_funcs.go"))
	if err != nil {
		t.Fatalf("REFUSING: cannot parse our registry: %v", err)
	}

	if len(shared) < 5 {
		t.Fatalf("REFUSING: only %d shared-function groups resolved out of skillFuncs; the AST "+
			"reader is broken and every group would pass by not being looked at", len(shared))
	}
	if got := len(shared[sharedKey{"preDamageFunc", "brandPreDamageFunc"}]); got < 6 {
		t.Fatalf("REFUSING: control failed — brandPreDamageFunc resolved %d ids, expected the six "+
			"brand skills", got)
	}

	base, err := pinCallbackBody(skillDir, "MoltenStrike", "preDamageFunc")
	if err != nil {
		t.Fatalf("REFUSING: control failed — cannot read MoltenStrike.preDamageFunc: %v", err)
	}
	zenith, err := pinCallbackBody(skillDir, "MoltenStrikeAltX", "preDamageFunc")
	if err != nil {
		t.Fatalf("REFUSING: control failed — cannot read MoltenStrikeAltX.preDamageFunc: %v", err)
	}
	if len(base) < 200 || normalisePinBody(base) == normalisePinBody(zenith) {
		t.Fatalf("REFUSING: control failed — MoltenStrike (%d chars) and MoltenStrikeAltX (%d) "+
			"normalise to the same body, so this comparator cannot see a real divergence",
			len(base), len(zenith))
	}

	keys := make([]sharedKey, 0, len(shared))
	for k := range shared {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].fn != keys[j].fn {
			return keys[i].fn < keys[j].fn
		}
		return keys[i].callback < keys[j].callback
	})

	compared := 0
	for _, k := range keys {
		ids := shared[k]
		sort.Strings(ids)
		groups := map[string][]string{}
		for _, id := range ids {
			body, err := pinCallbackBody(skillDir, id, k.callback)
			if err != nil {
				t.Errorf("UNREADABLE: %s.%s is registered under %s and its pinned body could not "+
					"be extracted: %v", id, k.callback, k.fn, err)
				continue
			}
			n := normalisePinBody(body)
			groups[n] = append(groups[n], id)
		}
		compared++
		if len(groups) <= 1 {
			continue
		}
		summary := make([]string, 0, len(groups))
		for _, v := range groups {
			sort.Strings(v)
			summary = append(summary, "{"+strings.Join(v, ", ")+"}")
		}
		sort.Strings(summary)
		t.Errorf("SHARED CLOSURE OVER DIVERGENT BODIES: skillFuncs registers %s as the %q for %v, "+
			"and the pinned PoB gives them %d DIFFERENT bodies: %s. One of those skills is running "+
			"another skill's function. Split them, or — if the only difference is a per-skill "+
			"constant — parameterise the function the way lancingSteelPreDamageFunc does.",
			k.fn, k.callback, ids, len(groups), strings.Join(summary, " vs "))
	}
	t.Logf("SHARED CLOSURES: %d named functions cover more than one granted-effect id", compared)
}

type sharedKey struct {
	callback string
	fn       string
}

func pinSkillDirOrSkip(t *testing.T) string {
	t.Helper()
	pobRoot := os.Getenv("POB_DIR")
	if pobRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("NOT-RUN: cannot resolve $HOME to find the pinned PoB: %v", err)
		}
		pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
	}
	dir := filepath.Join(pobRoot, "src", "Data", "Skills")
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		t.Skipf("NOT-RUN: no pinned PoB skill data at %s (set POB_DIR). This has NOT told you the "+
			"shared closures agree — it has told you it could not look.", dir)
	}
	return dir
}

func goSharedSkillFuncIdents(path string) (map[sharedKey][]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	all := map[sharedKey][]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "skillFuncs" || len(vs.Values) != 1 {
			return true
		}
		lit, ok := vs.Values[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, el := range lit.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			id, ok := litString(kv.Key)
			if !ok {
				continue
			}
			inner, ok := kv.Value.(*ast.CompositeLit)
			if !ok {
				continue
			}
			for _, iel := range inner.Elts {
				ikv, ok := iel.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				cb, ok := litString(ikv.Key)
				if !ok {
					continue
				}
				fn, ok := ikv.Value.(*ast.Ident)
				if !ok {
					continue
				}
				all[sharedKey{cb, fn.Name}] = append(all[sharedKey{cb, fn.Name}], id)
			}
		}
		return false
	})
	out := map[sharedKey][]string{}
	for k, v := range all {
		if len(v) > 1 {
			out[k] = v
		}
	}
	return out, nil
}

var pinSkillHeadRe = regexp.MustCompile(`^skills\["([^"]+)"\]\s*=\s*\{`)

func pinCallbackBody(dir, id, callback string) (string, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.lua"))
	if err != nil {
		return "", fmt.Errorf("globbing %s: %w", dir, err)
	}
	head := regexp.MustCompile(`^(\s+)` + regexp.QuoteMeta(callback) + `\s*=\s*function`)
	for _, path := range entries {
		raw, err := os.ReadFile(path) //nolint:gosec // POB_DIR-relative, as elsewhere in this package
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", path, err)
		}
		lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
		inSkill := false
		for i, line := range lines {
			if m := pinSkillHeadRe.FindStringSubmatch(line); m != nil {
				inSkill = m[1] == id
				continue
			}
			if strings.HasPrefix(line, "}") {
				inSkill = false
				continue
			}
			if !inSkill {
				continue
			}
			m := head.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			indent := m[1]
			for j := i + 1; j < len(lines); j++ {
				trimmed := strings.TrimRight(lines[j], " \t")
				if trimmed == indent+"end," || trimmed == indent+"end" {
					return strings.Join(lines[i+1:j], "\n"), nil
				}
			}
			return "", fmt.Errorf("%s.%s in %s: no `end` at the callback's own indent",
				id, callback, filepath.Base(path))
		}
	}
	return "", fmt.Errorf("%s.%s: not declared in the pinned skill data", id, callback)
}

var (
	luaLineCommentRe = regexp.MustCompile(`--.*`)
	whitespaceRunRe  = regexp.MustCompile(`\s+`)
)

func normalisePinBody(body string) string {
	return strings.TrimSpace(whitespaceRunRe.ReplaceAllString(luaLineCommentRe.ReplaceAllString(body, ""), " "))
}
