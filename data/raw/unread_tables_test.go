//go:build !js

package raw

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/fields"
)

const moduleRoot = "../.."

var initHelperRe = regexp.MustCompile(`loader\.InitHelper\(ctx, version, "([A-Za-z]+)", &([A-Za-z]+),`)

func poeGlobals(t *testing.T) map[string]string {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), "go", "list", "-m", "-f", "{{.Dir}}", DataModulePath).Output()
	if err != nil {
		t.Fatalf("go list %s: %v", DataModulePath, err)
	}
	dir := filepath.Join(strings.TrimSpace(string(out)), "poe")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	globals := map[string]string{}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_test.go") || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range initHelperRe.FindAllStringSubmatch(string(b), -1) {
			globals[m[1]] = m[2]
		}
	}
	if len(globals) < 30 {
		t.Fatalf("found %d poe loaders in %s; a scan that sees nothing makes every check pass", len(globals), dir)
	}
	return globals
}

var poeOwnReads = map[string][]string{
	"AlternateTreeArt":              {"AscendancyFrameArt", "ConnectionArt", "KeystoneFrameArt", "NotableFrameArt"},
	"BuffTemplates":                 {"ID", "StatsKey"},
	"ClientStrings":                 {"ID"},
	"CurrencyExchange":              {"Item"},
	"FlavourText":                   {"ID"},
	"GameConstants":                 {"ID"},
	"GemEffects":                    {"GemTags", "GrantedEffect", "GrantedEffect2", "ID"},
	"GemTags":                       {"ID"},
	"ItemExperiencePerLevel":        {"ItemExperienceType"},
	"ItemVisualIdentity":            {"ID"},
	"ModFamily":                     {"ID"},
	"ModType":                       {"Stats"},
	"DefaultMonsterStats":           {"AltDamage1", "AltDamage2", "AltLife1", "AltLife2", "Armour", "EvasiveEvasion", "MinionDamage", "MinionLife", "MonsterLife"},
	"MonsterMapBossDifficulty":      {"BossAilmentPercentDecrease", "BossDamagePercentIncrease", "BossIncItemQuantity", "BossIncItemRarity", "BossLifePercentIncrease", "Stat1", "Stat2", "Stat3", "Stat4", "Stat5"},
	"MonsterMapDifficulty":          {"DamagePercentIncrease", "LifePercentIncrease", "Stat1", "Stat2", "Stat3", "Stat4"},
	"MonsterTypes":                  {"ID"},
	"PassiveJewelSlots":             {"ClusterJewelSize"},
	"PassiveOverrideLimits":         {"ID"},
	"PassiveSkillOverrideTypes":     {"CounterStat", "ID"},
	"PassiveSkillOverrides":         {"Effect", "ID", "Limit", "Stats", "Type"},
	"PassiveSkillTattooTargetSets":  {"ID"},
	"PassiveSkillTattoos":           {"Override", "OverrideType", "Set", "Tattoo"},
	"PassiveSkillTreeConnectionArt": {"ID"},
	"PassiveSkillTreeNodeFrameArt":  {"ID"},
	"UniqueStashLayout":             {"ItemVisualIdentityKey", "UniqueStashTypesKey", "WordsKey"},
	"UniqueStashTypes":              {"ID"},
	"VirtualStatContextFlags":       {"ID"},
	"WeaponPassiveSkillTypes":       {"ID"},
	"WeaponPassiveSkills":           {"ID", "Mod", "Type"},
}

var aheadOfPort = map[string][]string{
	"BuffTemplates": {"AuraRadius", "BuffDefinitionsKey", "BuffStatValues", "BuffVisualsKey", "ID", "Key", "StatsKey"},
	"FlavourText":   {"ID", "Text"},
	"ModType":       {"Name", "Stats"},
}

// TestUnreadTablesStayUnread: InitializeAll leaves unreadTables' poe globals nil.
func TestUnreadTablesStayUnread(t *testing.T) {
	globals := poeGlobals(t)

	loaded := make([]string, 0, len(tables)+len(deferredTables))
	for _, tb := range slices.Concat(tables, deferredTables) {
		loaded = append(loaded, tb.Name)
	}
	for name := range globals {
		if slices.Contains(loaded, name) == slices.Contains(unreadTables, name) {
			t.Errorf("poe loads %s: it must be in exactly one of tables and unreadTables", name)
		}
	}
	for _, name := range append(slices.Clone(loaded), unreadTables...) {
		if _, ok := globals[name]; !ok {
			t.Errorf("%s is not a table poe loads", name)
		}
	}

	list, ok := fields.For(DefaultVersion)
	if !ok {
		t.Fatalf("go-pob-data publishes no field list for %s", DefaultVersion)
	}
	for _, name := range unreadTables {
		a, ok := list.Assets["raw/"+name]
		if !ok {
			continue
		}
		if slices.Contains(timelessTables, name) {
			continue
		}
		var ours []string
		for _, k := range a.Keep {
			if !slices.Contains(poeOwnReads[name], k) && !slices.Contains(aheadOfPort[name], k) {
				ours = append(ours, k)
			}
		}
		if len(ours) > 0 {
			t.Errorf("go-pob reads fields of raw/%s (%v): move it from unreadTables to tables", name, ours)
		}
		for _, k := range poeOwnReads[name] {
			if !slices.Contains(a.Keep, k) {
				t.Errorf("poeOwnReads[%s] lists %s, which the data-fields list does not keep", name, k)
			}
		}
	}
	for name := range poeOwnReads {
		if !slices.Contains(unreadTables, name) {
			t.Errorf("poeOwnReads lists %s, which is not in unreadTables", name)
		}
	}
	for name := range aheadOfPort {
		if !slices.Contains(unreadTables, name) {
			t.Errorf("aheadOfPort lists %s, which is not in unreadTables: the port landed, delete the entry", name)
		}
	}

	unreadGlobals := map[string]string{}
	for _, name := range unreadTables {
		unreadGlobals[globals[name]] = name
	}
	deferredGlobals := map[string]string{}
	for _, tb := range deferredTables {
		deferredGlobals[globals[tb.Name]] = tb.Name
	}
	if len(deferredGlobals) != len(deferredTables) || deferredGlobals[""] != "" {
		t.Fatalf("a deferred table has no poe global: %v", deferredGlobals)
	}
	scanned := 0
	fset := token.NewFileSet()
	err := filepath.WalkDir(moduleRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); path != moduleRoot && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "testdata") {
				return filepath.SkipDir
			}
			if path == filepath.Join(moduleRoot, "scripts", "gamedatadump") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		scanned++
		poeName := ""
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == DataModulePath+"/poe" {
				poeName = "poe"
				if imp.Name != nil {
					poeName = imp.Name.Name
				}
			}
		}
		if poeName == "" {
			return nil
		}
		inRaw := filepath.Dir(path) == filepath.Join(moduleRoot, "data", "raw")
		loadsEverything := false
		var hits, deferredHits []*ast.SelectorExpr
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == poeName {
				if sel.Sel.Name == "InitializeAll" {
					loadsEverything = true
				}
				if _, unread := unreadGlobals[sel.Sel.Name]; unread {
					hits = append(hits, sel)
				}
				if _, deferred := deferredGlobals[sel.Sel.Name]; deferred && !inRaw {
					deferredHits = append(deferredHits, sel)
				}
			}
			return true
		})
		if loadsEverything {
			return nil
		}
		for _, sel := range hits {
			t.Errorf("%s reads poe.%s, which raw.InitializeAll does not load (move %s to tables)",
				fset.Position(sel.Pos()), sel.Sel.Name, unreadGlobals[sel.Sel.Name])
		}
		for _, sel := range deferredHits {
			t.Errorf("%s reads poe.%s directly; it is deferred, read it through raw.%s()",
				fset.Position(sel.Pos()), sel.Sel.Name, deferredGlobals[sel.Sel.Name])
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 200 {
		t.Fatalf("scanned %d Go files; the walk is not seeing the module", scanned)
	}
}
