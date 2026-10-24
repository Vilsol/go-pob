package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

const offence320Sample = 20

var offence320KeyFloor = map[string]int{
	"AccuracyHitChance":           791,
	"MainHand.AccuracyHitChance":  186,
	"AverageBurstHits":            791,
	"DoubleDamageEffect":          592,
	"TripleDamageEffect":          592,
	"MainHand.DoubleDamageEffect": 186,
	"MainHand.TripleDamageEffect": 186,
	"StoredUses":                  314,
	"IgniteChancePerHit":          262,
}

// TestOffence3_20NewOutputs pins the CalcOffence.lua outputs this port gained for PoB 3.20.
func TestOffence3_20NewOutputs(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v\nBuild it with `task oracle-corpus` after a dump.", err)
	}

	corpusDir, names := corpusNamesForLoadedData(t)

	var sample []string
	for i := 0; i < len(names); i += offence320Sample {
		sample = append(sample, names[i])
	}
	if len(sample) < 700 {
		t.Fatalf("REFUSING: %d sampled builds of %d in the corpus, expected ~790. "+
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
			for key := range offence320KeyFloor {
				name, found := strings.CutPrefix(key, pair.prefix)
				if !found || strings.Contains(name, ".") {
					continue
				}
				want, present := pair.exp[name]
				if !present {
					continue
				}
				got := pair.got[name]
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

	keys := make([]string, 0, len(offence320KeyFloor))
	for k := range offence320KeyFloor {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Printf("\n=== CalcOffence output pins (%d sampled builds) ===\n", len(sample))
	for _, k := range keys {
		c := total[k]
		fmt.Printf("  %-32s ok %4d / graded %4d   floor %4d\n", k, c.ok, c.graded, offence320KeyFloor[k])
	}

	for _, k := range keys {
		c := total[k]
		if c.graded == 0 {
			t.Errorf("%s: PoB emits it on 0 of %d sampled builds. Either the blob lacks the key or "+
				"the table prefix is wrong; a zero denominator is not evidence of anything.",
				k, len(sample))
			continue
		}
		if c.ok < offence320KeyFloor[k] {
			t.Errorf("%s: %d of %d sampled builds match PoB, floor is %d",
				k, c.ok, c.graded, offence320KeyFloor[k])
		}
	}
}
