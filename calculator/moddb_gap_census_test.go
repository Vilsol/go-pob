package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestModDBGapCensus ranks, across the WHOLE corpus.
func TestModDBGapCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	dir := diagDumpsDir(t)
	names := diagCorpusNames(t)
	corpus := corpusDirForLoadedData(t)

	limit := len(names)
	if v := os.Getenv("LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			t.Fatalf("REFUSING: LIMIT=%q is not a positive integer (%v)", v, err)
		}
		limit = min(n, len(names))
	}

	type gap struct {
		builds  int
		occurs  int
		example string
	}
	onlyPoB := map[string]*gap{}
	onlyOurs := map[string]*gap{}
	scanned, skipped := 0, 0

	for _, name := range names {
		if scanned >= limit {
			break
		}
		d, healthy := diagLoadDump(dir, name)
		if d == nil || !healthy {
			skipped++
			continue
		}
		file, err := os.ReadFile(filepath.Join(corpus, name+".xml"))
		if err != nil {
			skipped++
			continue
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			skipped++
			continue
		}
		env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
		if env == nil || env.Player == nil || env.Player.ModDB == nil {
			skipped++
			continue
		}
		key := func(modName, typ string, val float64) string {
			return fmt.Sprintf("%-34s %-8s %12.4f", modName, typ, val)
		}
		which := os.Getenv("ACTOR")
		pobMods, ourMods := d.ModDB, env.Player.ModDB.Mods
		switch which {
		case "enemy":
			pobMods = d.EnemyModDB
			if env.Enemy == nil || env.Enemy.ModDB == nil {
				skipped++
				continue
			}
			ourMods = env.Enemy.ModDB.Mods
		case "minion":
			pobMods = d.MinionModDB
			if env.Minion == nil || env.Minion.ModDB == nil || len(pobMods) == 0 {
				skipped++
				continue
			}
			ourMods = env.Minion.ModDB.Mods
		}
		scanned++
		pobCount := map[string]int{}
		for modName, mods := range pobMods {
			for _, m := range mods {
				pobCount[key(modName, m.Type, pobModValue(m.Type, m.Value))]++
			}
		}
		ourCount := map[string]int{}
		for modName, mods := range ourMods {
			for _, m := range mods {
				ourCount[key(modName, string(m.Type()), ourModValue(m))]++
			}
		}

		seen := map[string]bool{}
		for k := range pobCount {
			seen[k] = true
		}
		for k := range ourCount {
			seen[k] = true
		}
		for k := range seen {
			switch d := pobCount[k] - ourCount[k]; {
			case d > 0:
				g := onlyPoB[k]
				if g == nil {
					g = &gap{example: name}
					onlyPoB[k] = g
				}
				g.builds++
				g.occurs += d
			case d < 0:
				g := onlyOurs[k]
				if g == nil {
					g = &gap{example: name}
					onlyOurs[k] = g
				}
				g.builds++
				g.occurs += -d
			}
		}
	}

	report := func(title string, m map[string]*gap) {
		keys := make([]string, 0, len(m))
		universal := 0
		for k := range m {
			if m[k].builds >= scanned {
				universal++
				continue
			}
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if m[keys[i]].builds != m[keys[j]].builds {
				return m[keys[i]].builds > m[keys[j]].builds
			}
			return keys[i] < keys[j]
		})
		fmt.Printf("\n=== %s (%d build-specific rows; %d universal rows excluded as base scaffolding) ===\n", title, len(keys), universal)
		fmt.Printf("%7s %8s  %-34s %-8s %12s  %s\n", "BUILDS", "OCCURS", "MOD", "TYPE", "VALUE", "EXAMPLE BUILD")
		for i, k := range keys {
			if i >= censusRowCap() {
				fmt.Printf("... %d further rows not shown (ROWS=n to change)\n", len(keys)-censusRowCap())
				break
			}
			fmt.Printf("%7d %8d  %s  %s\n", m[k].builds, m[k].occurs, k, m[k].example)
		}
	}

	fmt.Printf("=== MODDB GAP CENSUS === scanned %d builds, skipped %d (no dump / unhealthy / unparsable)\n", scanned, skipped)
	if scanned == 0 {
		t.Fatalf("REFUSING: scanned 0 builds. %s holds no usable dumps — empty must mean "+
			"'I looked and found nothing', never 'I did not look'", dir)
	}
	report("ONLY POB — a mod row PoB has and we do not", onlyPoB)
	report("ONLY OURS — a mod row we invent", onlyOurs)
	_ = strings.TrimSpace
}

func censusRowCap() int {
	if v := os.Getenv("ROWS"); v != "" {
		n := 0
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return 45
}
