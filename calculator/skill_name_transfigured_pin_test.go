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

// TestSkillNameTagsCarryIncludeTransfigured grades every `mod.SkillName(...)` tag in calculator/mod_parser.go against the SAME tag in the pinned PoB's ModParser.lua.
func TestSkillNameTagsCarryIncludeTransfigured(t *testing.T) {
	pobRoot := os.Getenv("POB_DIR")
	if pobRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("NOT-RUN: cannot resolve $HOME to find the pinned PoB: %v", err)
		}
		pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
	}
	luaPath := filepath.Join(pobRoot, "src", "Modules", "ModParser.lua")
	luaSrc, err := os.ReadFile(luaPath)
	if err != nil {
		t.Skipf("NOT-RUN: no pinned ModParser.lua at %s (set POB_DIR). This has NOT told you the "+
			"tags agree — it has told you it could not look.", luaPath)
	}

	withField, withoutField := pinnedSkillNameTagUsage(string(luaSrc))
	if len(withField) < 20 {
		t.Fatalf("REFUSING: only %d skill names carry includeTransfigured in the pinned "+
			"ModParser.lua; the tag spelling has moved and this test is measuring nothing", len(withField))
	}

	ours := goSkillNameTags(t, "mod_parser.go")
	if len(ours) < 100 {
		t.Fatalf("REFUSING: extracted only %d mod.SkillName tags from mod_parser.go; the call "+
			"shape this test recognises has moved", len(ours))
	}

	var bad []string
	for _, tag := range ours {
		if tag.transfigured || len(tag.names) == 0 {
			continue
		}
		uniform := true
		for _, n := range tag.names {
			if !withField[n] || withoutField[n] {
				uniform = false
				break
			}
		}
		if uniform {
			bad = append(bad, tag.pos+"  mod.SkillName("+strings.Join(tag.names, ", ")+")")
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 {
		t.Fatalf("%d SkillName tag(s) omit .Transfigured(true) where the pinned ModParser.lua "+
			"sets includeTransfigured on every tag naming those skills — each one is dead on a "+
			"transfigured gem:\n%s", len(bad), strings.Join(bad, "\n"))
	}
}

func pinnedSkillNameTagUsage(src string) (map[string]bool, map[string]bool) {
	withField := map[string]bool{}
	withoutField := map[string]bool{}
	const marker = `{ type = "SkillName"`
	for idx := 0; ; {
		i := strings.Index(src[idx:], marker)
		if i < 0 {
			break
		}
		start := idx + i
		depth := 0
		end := start
		for ; end < len(src); end++ {
			switch src[end] {
			case '{':
				depth++
			case '}':
				depth--
			}
			if depth == 0 {
				break
			}
		}
		if end >= len(src) {
			break
		}
		tag := src[start : end+1]
		idx = end + 1

		transf := strings.Contains(tag, "includeTransfigured = true")
		for _, name := range quotedStrings(strings.Replace(tag, `"SkillName"`, "", 1)) {
			if transf {
				withField[name] = true
			} else {
				withoutField[name] = true
			}
		}
	}
	return withField, withoutField
}

func quotedStrings(s string) []string {
	var out []string
	for {
		i := strings.IndexByte(s, '"')
		if i < 0 {
			return out
		}
		j := strings.IndexByte(s[i+1:], '"')
		if j < 0 {
			return out
		}
		out = append(out, s[i+1:i+1+j])
		s = s[i+2+j:]
	}
}

type goSkillNameTag struct {
	names        []string
	transfigured bool
	pos          string
}

func goSkillNameTags(t *testing.T, file string) []goSkillNameTag {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("REFUSING: cannot parse %s: %v", file, err)
	}

	isSkillName := func(call *ast.CallExpr) bool {
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "SkillName" {
			return false
		}
		pkg, ok := sel.X.(*ast.Ident)
		return ok && pkg.Name == "mod"
	}

	transfigured := map[*ast.CallExpr]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Transfigured" {
			return true
		}
		if inner, ok := sel.X.(*ast.CallExpr); ok && isSkillName(inner) {
			transfigured[inner] = true
		}
		return true
	})

	var out []goSkillNameTag
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isSkillName(call) {
			return true
		}
		tag := goSkillNameTag{transfigured: transfigured[call], pos: fset.Position(call.Pos()).String()}
		for _, arg := range call.Args {
			lit, ok := arg.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			tag.names = append(tag.names, s)
		}
		out = append(out, tag)
		return true
	})
	return out
}
