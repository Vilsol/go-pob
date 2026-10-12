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

// TestProducerCensus answers one question for every stat the authoritative gate grades: DOES ANYTHING IN THIS TREE EVER WRITE IT?
func TestProducerCensus(t *testing.T) {
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
		seen       int
		present    int
		nonzeroPoB int
		recover    int
		ourNonzero int
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
			for k, want := range pair.exp {
				got, present := pair.got[k]
				a := local[pair.prefix+k]
				a.seen++
				if present {
					a.present++
				}
				if got != 0 {
					a.ourNonzero++
				}
				if want != 0 {
					a.nonzeroPoB++
					if !oracleClose(got, want, oracleDumpTolerance) {
						a.recover++
					}
				}
				local[pair.prefix+k] = a
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
			e.seen += v.seen
			e.present += v.present
			e.nonzeroPoB += v.nonzeroPoB
			e.recover += v.recover
			e.ourNonzero += v.ourNonzero
		}
		mu.Unlock()
	})

	if builtCount < len(names)/2 {
		t.Fatalf("REFUSING: only %d of %d builds produced an environment; the census would be "+
			"measuring a build failure, not a producer gap", builtCount, len(names))
	}

	for _, known := range []string{"Life", "Mana", "Armour", "EnergyShield", "ColdMaximumHitTaken",
		"EnergyShieldOnBody Armour", "Minion.Life", "MainHand.Accuracy"} {
		a, ok := stats[known]
		if !ok {
			t.Fatalf("REFUSING: control stat %q is not graded at all; the census keyspace is wrong", known)
		}
		if a.present == 0 {
			t.Fatalf("REFUSING: control stat %q reports ABSENT. It is produced -- "+
				"%q covers the prefix-concatenation case and %q the suffix case. The probe is broken.",
				known, "ColdMaximumHitTaken", "EnergyShieldOnBody Armour")
		}
	}
	{
		probe := corpusEnvForLoadedData(t, names[0])
		const fabricated = "ZzFabricatedStatThatNothingWrites"
		if _, present := probe.Player.Output[fabricated]; present {
			t.Fatalf("REFUSING: fabricated key %q reports PRESENT on %s; the presence test cannot "+
				"distinguish produced from unproduced", fabricated, names[0])
		}
		if _, present := probe.Player.Output["Life"]; !present {
			t.Fatalf("REFUSING: %q reports ABSENT on %s; the presence test cannot answer YES either",
				"Life", names[0])
		}
	}

	type row struct {
		stat    string
		class   string
		seen    int
		present int
		nzPoB   int
		recover int
	}
	rows := make([]row, 0, len(stats))
	for k, a := range stats {
		class := "LIVE"
		switch {
		case a.present == 0:
			class = "ABSENT"
		case a.ourNonzero == 0:
			class = "ZERO"
		}
		rows = append(rows, row{k, class, a.seen, a.present, a.nonzeroPoB, a.recover})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].recover != rows[j].recover {
			return rows[i].recover > rows[j].recover
		}
		return rows[i].stat < rows[j].stat
	})

	var absentPairs, zeroPairs, livePairs int
	for _, r := range rows {
		switch r.class {
		case "ABSENT":
			absentPairs += r.recover
		case "ZERO":
			zeroPairs += r.recover
		default:
			livePairs += r.recover
		}
	}
	total := absentPairs + zeroPairs + livePairs

	t.Logf("PRODUCER CENSUS over %d builds, %d graded stats", builtCount, len(rows))
	t.Logf("  recoverable pairs (PoB nonzero AND we disagree): %d", total)
	t.Logf("  ABSENT (we never write the key) : %6d pairs  %5.2f%% of recoverable",
		absentPairs, 100*float64(absentPairs)/float64(max(total, 1)))
	t.Logf("  ZERO   (written, always 0)      : %6d pairs  %5.2f%%",
		zeroPairs, 100*float64(zeroPairs)/float64(max(total, 1)))
	t.Logf("  LIVE   (written, sometimes right): %6d pairs  %5.2f%%",
		livePairs, 100*float64(livePairs)/float64(max(total, 1)))

	t.Log("TOP 40 ABSENT PRODUCERS BY RECOVERABLE PAIRS")
	t.Logf("  %-40s %8s %8s %8s", "stat", "seen", "nzPoB", "recover")
	shown := 0
	for _, r := range rows {
		if r.class != "ABSENT" || shown >= 40 {
			continue
		}
		t.Logf("  %-40s %8d %8d %8d", r.stat, r.seen, r.nzPoB, r.recover)
		shown++
	}

	if dir := os.Getenv("CENSUS_OUT"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("CENSUS_OUT: %v", err)
		}
		var b strings.Builder
		b.WriteString("stat\tclass\tseen\tpresent\tnzPoB\trecover\n")
		for _, r := range rows {
			fmt.Fprintf(&b, "%s\t%s\t%d\t%d\t%d\t%d\n", r.stat, r.class, r.seen, r.present, r.nzPoB, r.recover)
		}
		if err := os.WriteFile(filepath.Join(dir, "producer_census.tsv"), []byte(b.String()), 0o600); err != nil {
			t.Fatalf("CENSUS_OUT: %v", err)
		}
		t.Logf("wrote %s", filepath.Join(dir, "producer_census.tsv"))
	}
}
