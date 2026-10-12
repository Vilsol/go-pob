package calculator

import (
	"os"
	"testing"
)

// TestEnemyRegenOutputs covers CalcOffence.lua:3694-3697.
func TestEnemyRegenOutputs(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	keys := []string{"EnemyLifeRegen", "EnemyManaRegen", "EnemyEnergyShieldRegen"}
	seen := map[string]int{}
	agreed := map[string]int{}
	var builds, nonZero int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		if _, has := rec.Output["EnemyLifeRegen"]; !has {
			continue
		}
		builds++
		env := corpusEnvForLoadedData(t, name)
		for _, key := range keys {
			want, has := rec.Output[key]
			if !has {
				continue
			}
			seen[key]++
			if key == "EnemyLifeRegen" && want != 0 {
				nonZero++
			}
			if oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
				agreed[key]++
			}
		}
		if builds >= 150 {
			break
		}
	}

	if builds == 0 {
		t.Fatal("REFUSING: no graded build publishes EnemyLifeRegen; this test no longer measures " +
			"the mechanism it was written for")
	}
	for _, key := range keys {
		t.Logf("%-26s %d of %d", key, agreed[key], seen[key])
	}
	t.Logf("EnemyLifeRegen is nonzero on %d of the %d sampled builds", nonZero, builds)
	if nonZero == 0 {
		t.Error("REFUSING: PoB reports EnemyLifeRegen 0 on every sampled build, so this test " +
			"cannot distinguish the port from writing a constant zero")
	}

	floors := map[string]int{
		"EnemyLifeRegen":         149,
		"EnemyManaRegen":         150,
		"EnemyEnergyShieldRegen": 150,
	}
	for _, key := range keys {
		if agreed[key] < floors[key] {
			t.Errorf("%s agrees on %d of %d graded builds, below the floor of %d "+
				"(CalcOffence.lua:3694-3697)", key, agreed[key], seen[key], floors[key])
		}
	}
}
