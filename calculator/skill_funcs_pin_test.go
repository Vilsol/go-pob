package calculator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestSkillFuncsMatchPinnedPoB is a two-sided name-set diff between `skillFuncs` and the callback fields the pinned PoB declares inside its own `skills[...]` tables.
func TestSkillFuncsMatchPinnedPoB(t *testing.T) {
	pobRoot := os.Getenv("POB_DIR")
	if pobRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("NOT-RUN: cannot resolve $HOME to find the pinned PoB: %v", err)
		}
		pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
	}
	skillDir := filepath.Join(pobRoot, "src", "Data", "Skills")
	if st, err := os.Stat(skillDir); err != nil || !st.IsDir() {
		t.Skipf("NOT-RUN: no pinned PoB skill data at %s (set POB_DIR). This has NOT told you the "+
			"registry matches upstream — it has told you it could not look.", skillDir)
	}

	pin, err := pinSkillCallbacks(skillDir)
	if err != nil {
		t.Fatalf("REFUSING: cannot read the pinned skill data: %v", err)
	}
	ours, err := goSkillFuncKeys(filepath.Join("..", "calculator", "skill_funcs.go"))
	if err != nil {
		t.Fatalf("REFUSING: cannot parse our registry: %v", err)
	}

	if len(pin) < 100 {
		t.Fatalf("REFUSING: only %d pinned skills carry a callback; the Lua scan is broken and "+
			"every registry entry would read as over-application", len(pin))
	}
	if !pin["Vortex"]["preDamageFunc"] {
		t.Fatal("REFUSING: control failed — the pin declares `Vortex.preDamageFunc` and the scan " +
			"did not find it")
	}
	if pin["Cyclone"]["preDamageFunc"] {
		t.Fatal("REFUSING: control failed — the pin declares no `Cyclone.preDamageFunc`, so the " +
			"scan is matching lines it should not")
	}
	if len(ours) < 50 || !ours["Barrage"]["preDamageFunc"] {
		t.Fatalf("REFUSING: control failed — parsed %d registry entries and did not find "+
			"`Barrage.preDamageFunc`", len(ours))
	}

	var extra []string
	for id, fns := range ours {
		for fn := range fns {
			if !pin[id][fn] {
				extra = append(extra, id+"."+fn)
			}
		}
	}
	sort.Strings(extra)
	for _, e := range extra {
		t.Errorf("OVER-APPLICATION: skillFuncs registers %s and the pinned PoB declares no such "+
			"callback on that skill. We run a function upstream never runs. Delete it, or — if it "+
			"is a deliberate divergence — say so AT THE SITE, with the measurement that "+
			"justifies it and the pin range that shows upstream's absence.", e)
	}

	var missing []string
	for id, fns := range pin {
		for fn := range fns {
			if !ours[id][fn] {
				missing = append(missing, id+"."+fn)
			}
		}
	}
	sort.Strings(missing)
	for _, m := range missing {
		if _, outside := skillFuncsPortedOutsideRegistry[m]; outside {
			continue
		}
		if !slices.Contains(skillFuncsNotPorted, m) {
			t.Errorf("UNPORTED AND UNDECLARED: the pinned PoB declares %s and we do not. Port it, "+
				"or add it to skillFuncsNotPorted — a callback nobody listed is indistinguishable "+
				"from one nobody looked at.", m)
		}
	}
	for _, d := range skillFuncsNotPorted {
		if _, outside := skillFuncsPortedOutsideRegistry[d]; outside {
			t.Errorf("DOUBLE DECLARATION: %s is named by BOTH skillFuncsNotPorted and "+
				"skillFuncsPortedOutsideRegistry. One of them is wrong and the worklist cannot "+
				"be read while both stand.", d)
			continue
		}
		if !slices.Contains(missing, d) {
			t.Errorf("STALE DECLARATION: skillFuncsNotPorted names %s, which is either ported "+
				"already or no longer declared by the pin. Remove it.", d)
		}
	}

	decls, calls, err := goFuncDeclsAndCalls(filepath.Join("..", "calculator"))
	if err != nil {
		t.Fatalf("REFUSING: cannot scan the calculator package for out-of-registry callbacks: %v", err)
	}
	if !decls["skillFuncsFor"] || !calls["skillFuncsFor"] {
		t.Fatalf("REFUSING: control failed — skillFuncsFor is declared and called in this "+
			"package and the scan found decl=%v call=%v", decls["skillFuncsFor"], calls["skillFuncsFor"])
	}
	for m, goFunc := range skillFuncsPortedOutsideRegistry {
		if !slices.Contains(missing, m) {
			t.Errorf("STALE DECLARATION: skillFuncsPortedOutsideRegistry names %s, which the pin "+
				"no longer declares (or which is now in skillFuncs). Remove it.", m)
		}
		if !decls[goFunc] {
			t.Errorf("UNBACKED DECLARATION: skillFuncsPortedOutsideRegistry says %s is ported as "+
				"%s, and no such function is declared in this package.", m, goFunc)
			continue
		}
		if !calls[goFunc] {
			t.Errorf("DEAD PORT: %s is declared as %s and NOTHING CALLS IT. Upstream dispatches "+
				"this callback from a named site; port that site too, or the closure is correct "+
				"and never runs.", m, goFunc)
		}
	}

	t.Logf("SKILL CALLBACKS: %d registered here, %d pinned skills carry one, %d unported (declared)",
		len(ours), len(pin), len(skillFuncsNotPorted))
}

func pinSkillCallbacks(dir string) (map[string]map[string]bool, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.lua"))
	if err != nil {
		return nil, fmt.Errorf("globbing %s: %w", dir, err)
	}
	var (
		skillRe = regexp.MustCompile(`^skills\["([^"]+)"\]\s*=\s*\{`)
		funcRe  = regexp.MustCompile(`^\s+(\w+[Ff]unc)\s*=\s*`)
	)
	out := map[string]map[string]bool{}
	for _, path := range entries {
		raw, err := os.ReadFile(path) //nolint:gosec // POB_DIR-relative, as elsewhere in this package
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		cur := ""
		for _, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
			if m := skillRe.FindStringSubmatch(line); m != nil {
				cur = m[1]
				continue
			}
			if strings.HasPrefix(line, "}") {
				cur = ""
				continue
			}
			if cur == "" {
				continue
			}
			if m := funcRe.FindStringSubmatch(line); m != nil {
				if out[cur] == nil {
					out[cur] = map[string]bool{}
				}
				out[cur][m[1]] = true
			}
		}
	}
	return out, nil
}

func goSkillFuncKeys(path string) (map[string]map[string]bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	out := map[string]map[string]bool{}
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
			out[id] = map[string]bool{}
			for _, iel := range inner.Elts {
				ikv, ok := iel.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if fn, ok := litString(ikv.Key); ok {
					out[id][fn] = true
				}
			}
		}
		return false
	})
	return out, nil
}

func litString(e ast.Expr) (string, bool) {
	bl, ok := e.(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(bl.Value)
	return v, err == nil
}

var skillFuncsPortedOutsideRegistry = map[string]string{
	"ExplosiveArrow.explosiveArrowFunc": "explosiveArrowFunc",
}

var skillFuncsNotPorted = []string{
	"AzmeriHydraBarrage.preDamageFunc",
	"BloodSacramentUnique.initialFunc",
	"ChargedDashAltX.preDamageFunc",
	"ChargedDash.preDamageFunc",
	"DeathWish.preDamageFunc",
	"ExplosiveTrapAltX.preDamageFunc",
	"EyeOfWinterAltX.preDamageFunc",
	"EyeOfWinterAltY.preDamageFunc",
	"FlamethrowerTrapAltX.preDamageFunc",
	"FlamethrowerTrap.preDamageFunc",
	"FrozenSweepAltX.preDamageFunc",
	"FrozenSweep.preDamageFunc",
	"HeraldOfAsh.preDamageFunc",
	"MoltenShell.preDamageFunc",
	"ShrapnelBallistaAltX.preDamageFunc",
	"ShrapnelBallista.preDamageFunc",
	"VaalMoltenShell.preDamageFunc",
	"WaveOfConvictionAltY.preDamageFunc",
	"WaveOfConviction.preDamageFunc",
}

func goFuncDeclsAndCalls(dir string) (map[string]bool, map[string]bool, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, nil, fmt.Errorf("globbing %s: %w", dir, err)
	}
	decls, calls := map[string]bool{}, map[string]bool{}
	fset := token.NewFileSet()
	for _, path := range entries {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.FuncDecl:
				if v.Recv == nil {
					decls[v.Name.Name] = true
				}
			case *ast.CallExpr:
				if id, ok := v.Fun.(*ast.Ident); ok {
					calls[id.Name] = true
				}
			}
			return true
		})
	}
	return decls, calls, nil
}
