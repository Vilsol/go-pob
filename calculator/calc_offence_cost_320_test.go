package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

const offenceCostSample = 20

var offenceCostRareBuilds = []string{
	"0801-hierophant-consecrated-path.xml",
	"1713-raider-lightning-trap.xml",
	"1714-raider-lightning-trap-2.xml",
	"3813-berserker-vaal-ground-slam-2.xml",
}

var offenceCostKeyFloor = map[string]int{
	"LifeOnKill":         104,
	"EnergyShieldOnKill": 34,
	"ManaOnKill":         17,

	"ManaCost": 196,
	"LifeCost": 32,

	"SoulCost": 9,

	"ManaPerSecondCost": 90,
	"ESPerSecondCost":   54,
	"LifePerSecondCost": 28,
	"SoulPerSecondCost": 6,
	"RagePerSecondCost": 1,
}

// TestOffenceCost3_20 pins the resource-cost and on-kill outputs this port gained for PoB 3.20.
func TestOffenceCost3_20(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.20")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v\nBuild it with `task oracle-corpus` after a dump.", err)
	}

	corpusDir, names := corpusNamesForLoadedData(t)

	seen := map[string]bool{}
	var sample []string
	for i := 0; i < len(names); i += offenceCostSample {
		sample = append(sample, names[i])
		seen[names[i]] = true
	}
	for _, name := range offenceCostRareBuilds {
		if !seen[name] {
			sample = append(sample, name)
		}
	}
	if len(sample) < 250 {
		t.Fatalf("REFUSING: %d sampled builds of %d in the corpus, expected ~306. "+
			"The floors below were measured against the full corpus and mean nothing on a short one.",
			len(sample), len(names))
	}

	type counts struct{ ok, graded int }
	results := make([]map[string]counts, len(sample))

	runConcurrently(len(sample), evalWorkers(), func(i int, cache *EnvironmentCache) {
		rec, ok := oracle[sample[i]]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			return
		}
		file, err := os.ReadFile(filepath.Join(corpusDir, sample[i]))
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

		out := map[string]counts{}
		for _, pair := range oracleTablePairs(rec, env) {
			if pair.prefix != "" {
				continue
			}
			for key := range offenceCostKeyFloor {
				want, present := pair.exp[key]
				if !present || want == 0 {
					continue
				}
				got := pair.got[key]
				c := out[key]
				c.graded++
				if !math.IsNaN(got) && !math.IsInf(got, 0) && oracleClose(got, want, oracleDumpTolerance) {
					c.ok++
				}
				out[key] = c
			}
		}
		results[i] = out
	})

	total := map[string]counts{}
	for _, r := range results {
		for k, c := range r {
			t := total[k]
			t.ok += c.ok
			t.graded += c.graded
			total[k] = t
		}
	}

	keys := make([]string, 0, len(offenceCostKeyFloor))
	for k := range offenceCostKeyFloor {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Printf("\n=== CalcOffence 3.20 cost / on-kill pins (%d sampled builds, NONZERO only) ===\n", len(sample))
	for _, k := range keys {
		c := total[k]
		fmt.Printf("  %-24s ok %4d / graded %4d   floor %4d\n", k, c.ok, c.graded, offenceCostKeyFloor[k])
	}

	for _, k := range keys {
		c := total[k]
		if c.graded == 0 {
			t.Errorf("%s: PoB emits it nonzero on 0 of %d sampled builds. Either the blob lacks the "+
				"key or the sample lost its population; a zero denominator is not evidence.",
				k, len(sample))
			continue
		}
		if c.ok < offenceCostKeyFloor[k] {
			t.Errorf("%s: %d of %d sampled builds match PoB, floor is %d",
				k, c.ok, c.graded, offenceCostKeyFloor[k])
		}
	}
}
