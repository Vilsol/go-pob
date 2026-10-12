package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestReverseConsumerSweep is the MIRROR of TestUnreadProducerSweep, and it exists because that sweep is structurally blind to this direction.
func TestReverseConsumerSweep(t *testing.T) {
	pobSrc := pinnedPoBSrcOrSkip(t)
	root := ".."

	oracleGraded, oracleNonzero := reverseSweepOracleKeys(t)
	readerBodies := reverseSweepReaderBodies(t, root)

	fmt.Printf("\n=== REVERSE CONSUMER SWEEP === %d list pairs, %d graded oracle keys, %d source files\n",
		len(reverseSweepPairs), len(oracleGraded), len(readerBodies))
	fmt.Printf("OURS-ABSENT = PoB's list carries it and ours does not.  " +
		"OURS-EXTRA = we carry it and PoB does not (over-application).\n")

	for _, pair := range reverseSweepPairs {
		luaNames := extractNames(t, filepath.Join(pobSrc, pair.luaFile), pair.luaStart, pair.luaEnd, pair.luaRe, "PoB "+pair.name)
		goNames := extractNames(t, filepath.Join(root, pair.goFile), pair.goStart, pair.goEnd, pair.goRe, "ours "+pair.name)

		if !luaNames[pair.control] || !goNames[pair.control] {
			t.Fatalf("REFUSING: %s — control name %q is on lua=%v go=%v, so the extraction has "+
				"moved and this pair would report every name as a divergence",
				pair.name, pair.control, luaNames[pair.control], goNames[pair.control])
		}

		oursAbsent := setDifference(luaNames, goNames)
		oursExtra := setDifference(goNames, luaNames)

		fmt.Printf("\n---- %s  [%s]  %s  vs  %s\n", pair.name, pair.kind, pair.luaFile, pair.goFile)
		fmt.Printf("     PoB carries %d names, ours %d; %d ours-absent, %d ours-extra\n",
			len(luaNames), len(goNames), len(oursAbsent), len(oursExtra))

		if len(oursAbsent) > 0 {
			fmt.Printf("  OURS-ABSENT                            readers  oracle\n")
			for _, name := range oursAbsent {
				fmt.Printf("    %-38s %5d  %s\n", name,
					countReaders(readerBodies, name), oracleCell(oracleGraded, oracleNonzero, name))
			}
		}
		if len(oursExtra) > 0 {
			fmt.Printf("  OURS-EXTRA (no PoB counterpart in this region — check for a RENAME " +
				"against the ours-absent list above before calling it invented):\n")
			for _, name := range oursExtra {
				fmt.Printf("    %-38s        %s\n", name, oracleCell(oracleGraded, oracleNonzero, name))
			}
		}
	}
}

type reverseSweepPair struct {
	name                      string
	kind                      string
	luaFile, luaStart, luaEnd string
	luaRe                     *regexp.Regexp
	goFile, goStart, goEnd    string
	goRe                      *regexp.Regexp
	control                   string
}

var reverseSweepPairs = []reverseSweepPair{
	{
		name:     "item calcLocal drains",
		kind:     "DRAIN",
		luaFile:  "Classes/Item.lua",
		luaStart: "function ItemClass:BuildModListForSlotNum",
		luaEnd:   "function ItemClass:CanHaveMod",
		luaRe:    regexp.MustCompile(`calcLocal\((?:modList|baseList), "([A-Za-z]+)"`),
		goFile:   "calculator/item.go",
		goRe:     regexp.MustCompile(`calcLocal(?:Flag)?\(&?(?:modList|baseList), "([A-Za-z]+)"`),
		control:  "AbyssalSocketCount",
	},
	{
		name:     "initModDB base mods",
		kind:     "DISPATCH",
		luaFile:  "Modules/CalcSetup.lua",
		luaStart: "function calcs.initModDB",
		luaEnd:   "function calcs.buildModListForNode",
		luaRe:    regexp.MustCompile(`NewMod\("([A-Za-z:]+)"`),
		goFile:   "calculator/env.go",
		goStart:  "func initModDBMods(",
		goEnd:    "\nfunc ",
		goRe:     regexp.MustCompile(`mod\.New(?:Float|Flag)\("([A-Za-z:]+)"`),
		control:  "FireResistMax",
	},
	{
		name:     "weaponData fields",
		kind:     "FIELD SET",
		luaFile:  "Classes/Item.lua",
		luaStart: "function ItemClass:BuildModListForSlotNum",
		luaEnd:   "function ItemClass:CanHaveMod",
		luaRe:    regexp.MustCompile(`\bweaponData\.([A-Za-z]+)`),
		goFile:   "calculator/item.go",
		goRe:     regexp.MustCompile(`\bweaponData\["([A-Za-z]+)"\]`),
		control:  "AttackRate",
	},
	{
		name:     "armourData fields",
		kind:     "FIELD SET",
		luaFile:  "Classes/Item.lua",
		luaStart: "function ItemClass:BuildModListForSlotNum",
		luaEnd:   "function ItemClass:CanHaveMod",
		luaRe:    regexp.MustCompile(`\barmourData\.([A-Za-z]+)`),
		goFile:   "calculator/item.go",
		goRe:     regexp.MustCompile(`\barmourData\["([A-Za-z]+)"\]`),
		control:  "Armour",
	},
	{
		name:     "flaskData fields",
		kind:     "FIELD SET",
		luaFile:  "Classes/Item.lua",
		luaStart: "function ItemClass:BuildModListForSlotNum",
		luaEnd:   "function ItemClass:CanHaveMod",
		luaRe:    regexp.MustCompile(`\bflaskData\.([A-Za-z]+)`),
		goFile:   "calculator/item.go",
		goRe:     regexp.MustCompile(`\bflaskData\["([A-Za-z]+)"\]`),
		control:  "duration",
	},
	{
		name:     "item slot base names",
		kind:     "DATA LIST",
		luaFile:  "Classes/ItemsTab.lua",
		luaStart: "local baseSlots = {",
		luaEnd:   "\n",
		luaRe:    regexp.MustCompile(`"([A-Za-z0-9 ]+)"`),
		goFile:   "calculator/item_setup.go",
		goStart:  "var itemSlotBaseNames = []string{",
		goEnd:    "\n}",
		goRe:     regexp.MustCompile(`"([A-Za-z0-9 ]+)"`),
		control:  "Ring 3",
	},
	{
		name:    "config var list",
		kind:    "DISPATCH",
		luaFile: "Modules/ConfigOptions.lua",
		luaRe:   regexp.MustCompile(`\{\s*var = "([A-Za-z0-9_]+)"`),
		goFile:  "calculator/configs.go",
		goStart: "var configOrder = []string{",
		goEnd:   "\n}",
		goRe:    regexp.MustCompile(`"([A-Za-z0-9_]+)"`),
		control: "enemyIsBoss",
	},
}

func extractNames(t *testing.T, path, start, end string, re *regexp.Regexp, label string) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("REFUSING: %s — cannot read %s: %v", label, path, err)
	}
	body := string(raw)
	if start != "" {
		i := strings.Index(body, start)
		if i < 0 {
			t.Fatalf("REFUSING: %s — start marker %q is not in %s; the region moved and a "+
				"whole-file comparison would be a different measurement wearing this one's name",
				label, start, path)
		}
		body = body[i:]
	}
	if end != "" {
		if j := strings.Index(body, end); j > 0 {
			body = body[:j]
		} else {
			t.Fatalf("REFUSING: %s — end marker %q is not in %s after the start marker",
				label, end, path)
		}
	}
	out := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		out[m[1]] = true
	}
	if len(out) == 0 {
		t.Fatalf("REFUSING: %s — extracted 0 names from %s. Empty here would print as "+
			"'no divergences', which is the most reassuring possible way to be broken", label, path)
	}
	return out
}

func setDifference(a, b map[string]bool) []string {
	var out []string
	for name := range a {
		if !b[name] {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func reverseSweepOracleKeys(t *testing.T) (map[string]int, map[string]int) {
	t.Helper()
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot read the oracle blob, so the ORACLE column would read '-' "+
			"for every name and rank real work as ungradeable: %v", err)
	}
	graded, nonzero := map[string]int{}, map[string]int{}
	count := func(table map[string]float64) {
		for k, v := range table {
			graded[k]++
			if v != 0 {
				nonzero[k]++
			}
		}
	}
	for _, rec := range oracle {
		count(rec.Output)
		count(rec.EnemyOutput)
	}
	keys := graded
	if keys["TotalDPS"] == 0 {
		t.Fatalf("REFUSING: control failed — TotalDPS is not in the oracle key set (%d keys), "+
			"so the ORACLE column is measuring nothing", len(keys))
	}
	if nonzero["TotalDPS"] == 0 {
		t.Fatalf("REFUSING: control failed — TotalDPS is graded on %d builds and NONZERO on none, "+
			"so the nonzero half of the ORACLE column is broken", graded["TotalDPS"])
	}
	return graded, nonzero
}

func oracleCell(graded, nonzero map[string]int, name string) string {
	n, ok := graded[name]
	if !ok {
		return "-"
	}
	return fmt.Sprintf("%d/%d", nonzero[name], n)
}

func reverseSweepReaderBodies(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	for _, dir := range []string{"calculator", "moddb", "mod", "data", "pob", "builds", "wasm", "utils"} {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil //nolint:nilerr
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			if strings.Contains(p, string(filepath.Separator)+"raw"+string(filepath.Separator)) {
				return nil
			}
			paths = append(paths, p)
			return nil
		})
	}
	if len(paths) == 0 {
		t.Fatalf("REFUSING: found 0 source files under %s to search for readers", root)
	}
	bodies := make([]string, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p) //nolint:gosec // path came from our own WalkDir over the repo
		if err != nil {
			t.Fatalf("REFUSING: cannot read %s: %v", p, err)
		}
		bodies = append(bodies, string(b))
	}
	return bodies
}

func countReaders(bodies []string, name string) int {
	needle := `"` + name + `"`
	n := 0
	for _, b := range bodies {
		n += strings.Count(b, needle)
	}
	return n
}

func pinnedPoBSrcOrSkip(t *testing.T) string {
	t.Helper()
	pobRoot := os.Getenv("POB_DIR")
	if pobRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("NOT-RUN: cannot resolve $HOME to find the pinned PoB: %v", err)
		}
		pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
	}
	src := filepath.Join(pobRoot, "src")
	if st, err := os.Stat(src); err != nil || !st.IsDir() {
		t.Skipf("NOT-RUN: no pinned PoB source at %s (set POB_DIR). This sweep has NOT told you "+
			"the lists agree — it has told you it could not look.", src)
	}
	return src
}
