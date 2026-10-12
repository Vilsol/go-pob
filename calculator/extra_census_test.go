package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestExtraCensus answers the one question NOTHING on this project could answer: WHICH STATS DO WE WRITE THAT PoB DOES NOT?
func TestExtraCensus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	corpusDir, names := corpusNamesForLoadedData(t)

	type acc struct {
		oursNonzero int
		pobHasKey   int
		pobNonzero  int
		phantom     int
		orphanPairs int
		nonUnit     int
		oursHasKey  int
		orphanZero  int
	}

	var mu sync.Mutex
	stats := map[string]*acc{}
	builtCount := 0

	runConcurrently(len(names), 12, func(i int, cache *EnvironmentCache) {
		name := names[i]
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			return
		}
		file, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			return
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			return
		}
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		if env == nil {
			return
		}

		local := map[string]acc{}
		for _, pair := range oracleTablePairs(rec, env) {
			if pair.exp == nil {
				continue
			}
			for k, got := range pair.got {
				key := pair.prefix + k
				a := local[key]
				want, present := pair.exp[k]
				if present {
					a.pobHasKey++
					if want != 0 {
						a.pobNonzero++
					}
				}
				a.oursHasKey++
				if got != 0 {
					a.oursNonzero++
					if got != 1 {
						a.nonUnit++
					}
					switch {
					case !present:
						a.orphanPairs++
					case want == 0:
						a.phantom++
					}
				} else if !present {
					a.orphanZero++
				}
				local[key] = a
			}
			for k, want := range pair.exp {
				if _, ours := pair.got[k]; ours {
					continue
				}
				key := pair.prefix + k
				a := local[key]
				a.pobHasKey++
				if want != 0 {
					a.pobNonzero++
				}
				local[key] = a
			}
		}

		mu.Lock()
		builtCount++
		for k, v := range local {
			e, ok := stats[k]
			if !ok {
				e = &acc{}
				stats[k] = e
			}
			e.oursNonzero += v.oursNonzero
			e.pobHasKey += v.pobHasKey
			e.pobNonzero += v.pobNonzero
			e.phantom += v.phantom
			e.orphanPairs += v.orphanPairs
			e.nonUnit += v.nonUnit
			e.oursHasKey += v.oursHasKey
			e.orphanZero += v.orphanZero
		}
		mu.Unlock()
	})

	if builtCount < len(names)/2 {
		t.Fatalf("REFUSING: only %d of %d builds produced an environment; this census would be "+
			"measuring a build failure, not an extra-key defect", builtCount, len(names))
	}

	const knownOrphan = "LifeEfficiency"
	if a, ok := stats[knownOrphan]; !ok || a.pobHasKey != 0 || a.oursNonzero == 0 {
		pobHas, oursNZ := -1, -1
		if ok {
			pobHas, oursNZ = a.pobHasKey, a.oursNonzero
		}
		t.Fatalf("REFUSING: control key %q did not come back as ours-only (in keyspace=%v, "+
			"PoB emits it on %d builds, we write it nonzero on %d; -1 means absent from the "+
			"keyspace). Either this census cannot see an extra key at all, or that divergence was "+
			"closed and this control needs replacing", knownOrphan, ok, pobHas, oursNZ)
	}

	const knownZeroOrphan = "AnyGuard"
	if a, ok := stats[knownZeroOrphan]; !ok || a.pobHasKey != 0 || a.orphanZero == 0 || a.oursNonzero != 0 {
		pobHas, oz, nz := -1, -1, -1
		if ok {
			pobHas, oz, nz = a.pobHasKey, a.orphanZero, a.oursNonzero
		}
		t.Fatalf("REFUSING: control key %q did not come back as an always-zero ours-only key "+
			"(in keyspace=%v, PoB emits it on %d builds, orphanZero=%d, oursNonzero=%d; -1 means "+
			"absent). Either the `got == 0` arm is not tallying, or this key changed and the "+
			"control needs replacing", knownZeroOrphan, ok, pobHas, oz, nz)
	}

	for _, shared := range []string{"Life", "Mana", "Armour", "EnergyShield"} {
		a, ok := stats[shared]
		if !ok {
			t.Fatalf("REFUSING: control key %q is not in the census keyspace at all", shared)
		}
		if a.pobHasKey == 0 {
			t.Fatalf("REFUSING: control key %q reports PoB-absent on every build. The census "+
				"cannot tell an extra key from a shared one", shared)
		}
	}

	type row struct {
		stat        string
		class       string
		oursNonzero int
		pobHasKey   int
		pobNonzero  int
		nonUnit     int
		pairs       int
		orphanZero  int
	}
	rows := make([]row, 0, len(stats))
	for k, a := range stats {
		pairs := a.phantom + a.orphanPairs
		if pairs == 0 && a.orphanZero == 0 {
			continue
		}
		class := "PHANTOM"
		switch {
		case a.pobHasKey != 0:
		case a.orphanPairs == 0:
			class = "ORPHAN-0"
		default:
			class = "ORPHAN"
			if a.nonUnit == 0 {
				class = "ORPHAN?B"
			}
		}
		rows = append(rows, row{k, class, a.oursNonzero, a.pobHasKey, a.pobNonzero, a.nonUnit, pairs, a.orphanZero})
	}
	sort.Slice(rows, func(i, j int) bool {
		li, lj := rows[i].pairs+rows[i].orphanZero, rows[j].pairs+rows[j].orphanZero
		if li != lj {
			return li > lj
		}
		return rows[i].stat < rows[j].stat
	})

	var orphanPairs, boolPairs, phantomPairs, zeroPairs int
	var orphanRows, boolRows, phantomRows, zeroRows int
	for _, r := range rows {
		switch r.class {
		case "ORPHAN":
			orphanPairs += r.pairs
			orphanRows++
		case "ORPHAN?B":
			boolPairs += r.pairs
			boolRows++
		case "ORPHAN-0":
			zeroPairs += r.orphanZero
			zeroRows++
		default:
			phantomPairs += r.pairs
			phantomRows++
		}
	}

	t.Logf("EXTRA CENSUS over %d builds — keys WE write that PoB does not", builtCount)
	t.Logf("  ORPHAN   (PoB never emits it, we write real numbers) : %6d pairs in %d stat(s)  <- NO GATE SEES THIS",
		orphanPairs, orphanRows)
	t.Logf("  ORPHAN?B (same, but we only ever write 0 or 1)      : %6d pairs in %d stat(s)  <- MOSTLY LUA BOOLEANS, NOT DEFECTS",
		boolPairs, boolRows)
	t.Logf("  PHANTOM  (PoB emits it and reports 0)               : %6d pairs in %d stat(s)  <- graded, unrankable",
		phantomPairs, phantomRows)
	t.Logf("  ORPHAN-0 (PoB never emits it, we ALWAYS write 0)    : %6d pairs in %d stat(s)  <- WAS INVISIBLE; MOSTLY BOOLEANS, GREP EACH",
		zeroPairs, zeroRows)
	if len(rows) == 0 {
		t.Log("  NO EXTRA KEYS AT ALL. The controls above passed, so this is a measurement.")
	}

	t.Log("TOP 30 BY PAIRS")
	t.Logf("  %-44s %-9s %8s %8s %8s %8s %8s", "stat", "class", "oursNZ", "nonUnit", "pobHas", "pairs", "orphan0")
	for i, r := range rows {
		if i >= 30 {
			break
		}
		t.Logf("  %-44s %-9s %8d %8d %8d %8d %8d", r.stat, r.class, r.oursNonzero, r.nonUnit, r.pobHasKey, r.pairs, r.orphanZero)
	}

	if dir := os.Getenv("CENSUS_OUT"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("CENSUS_OUT: %v", err)
		}
		var b strings.Builder
		b.WriteString("stat\tclass\toursNonzero\tnonUnit\tpobHasKey\tpobNonzero\tpairs\torphanZero\n")
		for _, r := range rows {
			fmt.Fprintf(&b, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
				r.stat, r.class, r.oursNonzero, r.nonUnit, r.pobHasKey, r.pobNonzero, r.pairs, r.orphanZero)
		}
		out := filepath.Join(dir, "extra_census.tsv")
		if err := os.WriteFile(out, []byte(b.String()), 0o600); err != nil {
			t.Fatalf("CENSUS_OUT: %v", err)
		}
		t.Logf("wrote %s", out)
	}
}
