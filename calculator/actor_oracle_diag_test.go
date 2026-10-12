package calculator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type minionDiagDump struct {
	MinionModDB map[string][]diagDumpMod `json:"minionModDB"`
}

// TestDiagMinionMods is the minion counterpart of TestDiagBuildMods, and it walks EVERY mod name at once rather than one.
func TestDiagMinionMods(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	dir := diagDumpsDir(t)
	buildName := strings.TrimSuffix(os.Getenv("BUILD"), ".xml")
	if buildName == "" {
		t.Skip("BUILD unset")
	}

	raw, err := os.ReadFile(filepath.Join(dir, buildName+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var d minionDiagDump
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if d.MinionModDB == nil {
		t.Fatalf("%s: the dump carries no minionModDB (PoB built no minion, or DUMPS is not a v3 set)", buildName)
	}

	env := corpusEnvForLoadedData(t, buildName)
	if env.Minion == nil {
		t.Fatalf("%s: go-pob built no minion actor", buildName)
	}

	only := map[string]bool{}
	for _, n := range strings.Split(os.Getenv("NAME"), ",") {
		if n != "" {
			only[n] = true
		}
	}

	bySource := os.Getenv("BYSOURCE") == "1"
	key := func(src, typ string, val float64) string {
		if !bySource {
			src = ""
		}
		return fmt.Sprintf("%-46s %-6s %10.4f", src, typ, val)
	}
	pob := map[string]map[string]int{}
	for name, mods := range d.MinionModDB {
		if len(only) > 0 && !only[name] {
			continue
		}
		m := map[string]int{}
		for _, e := range mods {
			m[key(normalizeSource(e.Source), e.Type, pobModValue(e.Type, e.Value))]++
		}
		pob[name] = m
	}
	ours := map[string]map[string]int{}
	for name, mods := range env.Minion.ModDB.Mods {
		if len(only) > 0 && !only[name] {
			continue
		}
		m := map[string]int{}
		for _, mm := range mods {
			m[key(normalizeSource(string(mm.GetSource())), string(mm.Type()), ourModValue(mm))]++
		}
		ours[name] = m
	}

	names := map[string]bool{}
	for n := range pob {
		names[n] = true
	}
	for n := range ours {
		names[n] = true
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)

	fmt.Printf("\n=== MINION MODDB %s  (pob %d names, ours %d names) ===\n", buildName, len(pob), len(ours))
	for _, n := range sorted {
		p, o := pob[n], ours[n]
		rows := map[string]bool{}
		for k := range p {
			rows[k] = true
		}
		for k := range o {
			rows[k] = true
		}
		ks := make([]string, 0, len(rows))
		for k := range rows {
			if p[k] != o[k] {
				ks = append(ks, k)
			}
		}
		if len(ks) == 0 {
			continue
		}
		sort.Strings(ks)
		fmt.Printf("--- %s\n", n)
		for _, k := range ks {
			fmt.Printf("%4d %4d ** %s\n", p[k], o[k], k)
		}
	}
}

type oracleDiagRow struct {
	key      string
	exp, got float64
	rel      float64
}

func TestDiagActorOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	build := os.Getenv("BUILD")
	if build == "" {
		t.Skip("set BUILD=<corpus build name>")
	}
	if !strings.HasSuffix(build, ".xml") {
		build += ".xml"
	}
	table, tableSet := os.LookupEnv("TABLE")
	switch strings.ToLower(table) {
	case "player", "main":
		table = ""
	}
	if table != "" && !strings.HasSuffix(table, ".") {
		table += "."
	}

	oracles, err := oracleRecordsForBuilds(build)
	if err != nil {
		t.Fatalf("cannot load %s from the oracle blob: %v", build, err)
	}
	rec := oracles[build]

	if rec.TreeSources < oracleMinTreeSources {
		t.Fatalf("%s is an EXCLUDED build (treeSources %d < %d), so the oracle never grades it and\n"+
			"every row below would be meaningless. Headless PoB failed to apply this build's passive\n"+
			"tree — the excluded set is dominated by Glorious Vanity sockets — so PoB's own numbers\n"+
			"are garbage here, not ours. Pick a graded build; there is a wide gap between the two\n"+
			"populations, so any build the gate grades is far above the threshold.",
			build, rec.TreeSources, oracleMinTreeSources)
	}

	env := corpusEnvPinnedForLoadedData(t, build)
	pairs := oracleTablePairs(rec, env)

	avail := make([]string, 0, len(pairs))
	for _, p := range pairs {
		if p.exp == nil {
			continue
		}
		name := p.prefix
		if name == "" {
			name = "Player"
		}
		avail = append(avail, fmt.Sprintf("%s(%d keys)", name, len(p.exp)))
	}

	var selected []oracleTablePair
	for _, p := range pairs {
		if p.exp == nil {
			continue
		}
		if !tableSet || p.prefix == table {
			selected = append(selected, p)
		}
	}
	if len(selected) == 0 {
		if tableSet {
			t.Fatalf("no table with prefix %q on %s; this build has %v\n"+
				"(TABLE=Player is the player table; UNSET prints every table at once, which is the\n"+
				"default and is usually what you want)", table, build, avail)
		}
		t.Fatalf("PoB emitted no gradeable table at all for %s", build)
	}

	for _, pair := range selected {
		rows := make([]oracleDiagRow, 0, len(pair.exp))
		for _, k := range sortedKeys(pair.exp) {
			exp := pair.exp[k]
			got := pair.got[k]
			var rel float64
			if exp != 0 {
				rel = math.Abs((got - exp) / exp)
			} else if got != 0 {
				rel = math.Inf(1)
			} else {
				rel = 0
			}
			rows = append(rows, oracleDiagRow{k, exp, got, rel})
		}
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].rel > rows[j].rel })

		label := pair.prefix
		if label == "" {
			label = "Player"
		}
		var shown int
		for _, r := range rows {
			if !oracleClose(r.got, r.exp, oracleDumpTolerance) {
				shown++
			}
		}
		fmt.Printf("\n=== %s  table %q  (%d keys, ours has %d) — %d MISMATCHING ===\n",
			build, label, len(pair.exp), len(pair.got), shown)
		if shown == 0 {
			fmt.Printf("(this table agrees on every key — a build is NOT clean until every table above is)\n")
			continue
		}
		fmt.Printf("%-44s %18s %18s %10s\n", "KEY", "PoB", "ours", "got/PoB")
		printOracleDiagRows(rows)
	}
}

func printOracleDiagRows(rows []oracleDiagRow) {
	for _, r := range rows {
		if oracleClose(r.got, r.exp, oracleDumpTolerance) {
			continue
		}
		ratio := math.NaN()
		if r.exp != 0 {
			ratio = r.got / r.exp
		}
		fmt.Printf("%-44s %18.6g %18.6g %10.5f\n", r.key, r.exp, r.got, ratio)
	}
}
