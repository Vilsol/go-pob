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

// TestItemGrantedSupportNamesMatchPinnedPoB asserts the property `itemGrantedSupportNames`' own comment already CLAIMED.
func TestItemGrantedSupportNamesMatchPinnedPoB(t *testing.T) {
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
			"map matches upstream — it has told you it could not look.", skillDir)
	}

	pin, err := pinItemGrantedSupports(skillDir)
	if err != nil {
		t.Fatalf("REFUSING: cannot read the pinned skill data: %v", err)
	}
	if len(pin) < 15 || len(pin) > 60 {
		t.Fatalf("REFUSING: the scan found %d item-granted supports; that is outside any "+
			"plausible range and the Lua scan is broken", len(pin))
	}
	if pin["SupportTriggerSpellOnAttack"] != "Poet's Pen" {
		t.Fatalf("REFUSING: control failed — the pin declares SupportTriggerSpellOnAttack as "+
			"a fromItem support named \"Poet's Pen\"; the scan found %q",
			pin["SupportTriggerSpellOnAttack"])
	}
	if _, ok := pin["SupportElementalPenetration"]; ok {
		t.Fatal("REFUSING: control failed — SupportElementalPenetration is `support = true` and " +
			"NOT `fromItem` in the pin, so a scan that finds it is matching lines it should not")
	}

	ours, err := goStringMapLiteral(filepath.Join("..", "calculator", "calc_perform.go"), "itemGrantedSupportNames")
	if err != nil {
		t.Fatalf("REFUSING: cannot parse our map: %v", err)
	}
	if len(ours) == 0 {
		t.Fatal("REFUSING: parsed zero entries out of itemGrantedSupportNames; the AST reader is broken")
	}

	for _, id := range sortedStringMapKeys(ours) {
		want, ok := pin[id]
		if !ok {
			t.Errorf("OVER-APPLICATION: itemGrantedSupportNames carries %q and the pinned PoB "+
				"marks no such skill `fromItem`. getUniqueItemTriggerName returns the FIRST match "+
				"in supportList, so this entry can HIJACK the name on any build carrying it and "+
				"send a real item trigger to a case that does not exist. Delete it, or — if it is "+
				"a deliberate divergence — say so AT THE SITE with the pin range showing upstream's "+
				"absence.", id)
			continue
		}
		if ours[id] != want {
			t.Errorf("WRONG NAME: itemGrantedSupportNames[%q] = %q, want %q — the map's contract "+
				"is the string PoB's supportList scan yields, which is grantedEffect.name",
				id, ours[id], want)
		}
	}
	for _, id := range sortedStringMapKeys(pin) {
		if _, ok := ours[id]; !ok {
			t.Errorf("MISSING: the pinned PoB marks %q as a `fromItem` support (name %q) and the "+
				"map does not carry it. getUniqueItemTriggerName then skips it and returns a LATER "+
				"support's name, which is a different branch — not merely a missing one.",
				id, pin[id])
		}
	}
}

func pinItemGrantedSupports(dir string) (map[string]string, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.lua"))
	if err != nil {
		return nil, fmt.Errorf("globbing %s: %w", dir, err)
	}
	var (
		skillRe = regexp.MustCompile(`^skills\["([^"]+)"\]\s*=\s*\{`)
		nameRe  = regexp.MustCompile(`^\tname = "(.*)",\s*$`)
	)
	out := map[string]string{}
	for _, path := range entries {
		raw, err := os.ReadFile(path) //nolint:gosec // POB_DIR-relative, as elsewhere in this package
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
		cur, name, support, fromItem := "", "", false, false
		flush := func() {
			if cur != "" && support && fromItem {
				out[cur] = name
			}
			cur, name, support, fromItem = "", "", false, false
		}
		for _, line := range lines {
			if m := skillRe.FindStringSubmatch(line); m != nil {
				flush()
				cur = m[1]
				continue
			}
			if cur == "" {
				continue
			}
			switch {
			case line == "\tsupport = true,":
				support = true
			case line == "\tfromItem = true,":
				fromItem = true
			default:
				if m := nameRe.FindStringSubmatch(line); m != nil && name == "" {
					name = m[1]
				}
			}
		}
		flush()
	}
	return out, nil
}

func goStringMapLiteral(path, varName string) (map[string]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	out := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || vs.Names[0].Name != varName || len(vs.Values) != 1 {
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
			k, kok := litString(kv.Key)
			v, vok := litString(kv.Value)
			if kok && vok {
				out[k] = v
			}
		}
		return false
	})
	return out, nil
}

func sortedStringMapKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
