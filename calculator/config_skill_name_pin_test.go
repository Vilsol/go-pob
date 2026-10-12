package calculator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestConfigSkillNameTagsMatchPinned grades the `mod.SkillName(...)` strings inside each config handler in configs.go against the `skillName = "..."` strings inside the SAME `var` entry of the pinned ConfigOptions.lua.
func TestConfigSkillNameTagsMatchPinned(t *testing.T) {
	pobRoot := os.Getenv("POB_DIR")
	if pobRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("NOT-RUN: cannot resolve $HOME to find the pinned PoB: %v", err)
		}
		pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
	}
	luaPath := filepath.Join(pobRoot, "src", "Modules", "ConfigOptions.lua")
	luaSrc, err := os.ReadFile(luaPath)
	if err != nil {
		t.Skipf("NOT-RUN: no pinned ConfigOptions.lua at %s (set POB_DIR). This has NOT told you "+
			"the tags agree — it has told you it could not look.", luaPath)
	}

	upstream := pinnedConfigSkillNames(string(luaSrc))
	if len(upstream) < 10 {
		t.Fatalf("REFUSING: found skillName tags in only %d config vars of the pinned "+
			"ConfigOptions.lua; the entry shape has moved and this test measures nothing", len(upstream))
	}

	ours := goConfigSkillNames(t)
	if len(ours) < 5 {
		t.Fatalf("REFUSING: found mod.SkillName tags in only %d handlers of configs.go; the "+
			"map shape this test recognises has moved", len(ours))
	}

	var bad []string
	for v, got := range ours {
		want, ok := upstream[v]
		if !ok {
			continue
		}
		if !sameStringSet(got, want) {
			bad = append(bad, "  "+v+": ours "+strings.Join(sortedCopy(got), ", ")+
				"  |  pinned ConfigOptions.lua "+strings.Join(sortedCopy(want), ", "))
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 {
		t.Fatalf("%d config handler(s) tag a different skill than the pinned ConfigOptions.lua "+
			"entry of the same name — the tag matches nothing on the builds that can set the "+
			"option:\n%s", len(bad), strings.Join(bad, "\n"))
	}
}

func pinnedConfigSkillNames(src string) map[string][]string {
	out := map[string][]string{}
	const head = `{ var = "`
	idxs := []int{}
	for i := 0; ; {
		j := strings.Index(src[i:], head)
		if j < 0 {
			break
		}
		idxs = append(idxs, i+j)
		i += j + len(head)
	}
	for n, start := range idxs {
		end := len(src)
		if n+1 < len(idxs) {
			end = idxs[n+1]
		}
		chunk := src[start:end]
		nameEnd := strings.IndexByte(chunk[len(head):], '"')
		if nameEnd < 0 {
			continue
		}
		v := chunk[len(head) : len(head)+nameEnd]
		var names []string
		for rest := chunk; ; {
			k := strings.Index(rest, `skillName = "`)
			if k < 0 {
				break
			}
			rest = rest[k+len(`skillName = "`):]
			q := strings.IndexByte(rest, '"')
			if q < 0 {
				break
			}
			names = append(names, rest[:q])
			rest = rest[q+1:]
		}
		if len(names) > 0 {
			out[v] = names
		}
	}
	return out
}

func goConfigSkillNames(t *testing.T) map[string][]string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "configs.go", nil, 0)
	if err != nil {
		t.Fatalf("REFUSING: cannot parse configs.go: %v", err)
	}

	out := map[string][]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		key, ok := kv.Key.(*ast.BasicLit)
		if !ok || key.Kind != token.STRING {
			return true
		}
		fn, ok := kv.Value.(*ast.FuncLit)
		if !ok {
			return true
		}
		v, err := strconv.Unquote(key.Value)
		if err != nil {
			return true
		}
		var names []string
		ast.Inspect(fn, func(m ast.Node) bool {
			call, ok := m.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "SkillName" {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "mod" {
				return true
			}
			for _, arg := range call.Args {
				lit, ok := arg.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				if s, err := strconv.Unquote(lit.Value); err == nil {
					names = append(names, s)
				}
			}
			return true
		})
		if len(names) > 0 {
			out[v] = names
		}
		return true
	})
	return out
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func sameStringSet(a, b []string) bool {
	sa, sb := map[string]bool{}, map[string]bool{}
	for _, s := range a {
		sa[s] = true
	}
	for _, s := range b {
		sb[s] = true
	}
	if len(sa) != len(sb) {
		return false
	}
	for s := range sa {
		if !sb[s] {
			return false
		}
	}
	return true
}
