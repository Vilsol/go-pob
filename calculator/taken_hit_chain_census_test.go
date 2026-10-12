package calculator

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

var takenHitExonerated = []string{
	"ArmourDefense", "DamageReductionMax", "PhysicalResist", "PhysicalEnemyPen",
	"PhysicalEnemyOverwhelm",
}

const takenHitArisesHereCeiling = 150

// TestTakenHitChainCensus answers "where in the incoming-damage chain does <type>TakenHit go wrong".
func TestTakenHitChainCensus(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	rungs := []string{
		"PhysicalTakenDamage", "PhysicalEnemyDamage", "totalEnemyDamage", "PhysicalDamageReduction",
		"Armour", "Evasion", "ConfiguredEvadeChance", "EnemyCritChance", "EnemyCritEffect",
		"PhysicalEnemyDamageMult", "PhysicalTakenHitMult",
	}
	seenRung := map[string]bool{}
	for _, k := range rungs {
		seenRung[k] = true
	}
	for _, k := range takenHitExonerated {
		if !seenRung[k] {
			seenRung[k] = true
			rungs = append(rungs, k)
		}
	}

	wrongAt := func(env *Environment, rec *oracleRecord, key string) (bool, bool) {
		want, has := rec.Output[key]
		if !has {
			return false, false
		}
		return !oracleClose(env.Player.Output[key], want, oracleDumpTolerance), true
	}

	counts := map[string]int{}
	var seen, wrong, arisesHere int
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		bad, present := wrongAt(env, &rec, "PhysicalTakenHit")
		if !present {
			continue
		}
		seen++
		if !bad {
			continue
		}
		wrong++
		for _, k := range rungs {
			if w, p := wrongAt(env, &rec, k); p && w {
				counts[k]++
			}
		}
		dmgBad, _ := wrongAt(env, &rec, "PhysicalTakenDamage")
		drBad, _ := wrongAt(env, &rec, "PhysicalDamageReduction")
		if !dmgBad && !drBad {
			arisesHere++
		}
	}

	if seen == 0 || wrong == 0 {
		t.Fatalf("REFUSING: PhysicalTakenHit graded on %d builds, %d wrong — a census over an empty "+
			"set describes nothing", seen, wrong)
	}

	fmt.Printf("\n=== TakenHit CHAIN CENSUS === PhysicalTakenHit %d graded, %d wrong\n", seen, wrong)
	fmt.Printf("  %-28s %5d  %5.1f%%   <- arises in CalcDefence.lua:2041 itself\n",
		"INPUTS AGREE", arisesHere, 100*float64(arisesHere)/float64(wrong))
	keys := append([]string(nil), rungs...)
	sort.Slice(keys, func(a, b int) bool { return counts[keys[a]] > counts[keys[b]] })
	for _, k := range keys {
		fmt.Printf("  %-28s %5d  %5.1f%%\n", k, counts[k], 100*float64(counts[k])/float64(wrong))
	}

	if arisesHere > takenHitArisesHereCeiling {
		t.Errorf("%d of %d PhysicalTakenHit failures have both inputs AGREEING, ceiling %d — the "+
			"defect has moved INTO this step and the chain ranking above no longer describes it",
			arisesHere, wrong, takenHitArisesHereCeiling)
	}
	for _, k := range takenHitExonerated {
		if counts[k] != 0 {
			t.Errorf("%s overlaps %d PhysicalTakenHit failures, measured 0 — this rung was "+
				"exonerated and is now implicated; re-rank before trusting the census", k, counts[k])
		}
	}
}
