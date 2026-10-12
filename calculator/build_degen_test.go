package calculator

import (
	"os"
	"testing"
)

// TestBuildDegenOutputs covers CalcDefence.lua:3339-3394 and the shift table at :1802-1823.
func TestBuildDegenOutputs(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	keys := []string{
		"TotalBuildDegen", "PhysicalBuildDegen", "FireBuildDegen", "ChaosBuildDegen",
		"LightningBuildDegen", "ColdBuildDegen", "NetLifeRegen", "NetManaRegen",
		"NetEnergyShieldRegen",
	}
	seen := map[string]int{}
	agreed := map[string]int{}
	var builds int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		if _, has := rec.Output["TotalBuildDegen"]; !has {
			continue
		}
		builds++
		env := corpusEnvForLoadedData(t, name)
		for _, key := range keys {
			want, has := rec.Output[key]
			if !has || want == 0 {
				continue
			}
			seen[key]++
			if oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
				agreed[key]++
			}
		}
		if builds >= 120 {
			break
		}
	}

	if builds == 0 {
		t.Fatal("REFUSING: no graded build publishes TotalBuildDegen; this test no longer " +
			"measures the mechanism it was written for")
	}
	for _, key := range keys {
		t.Logf("%-22s %d of %d", key, agreed[key], seen[key])
	}

	floors := map[string]int{
		"TotalBuildDegen":      115,
		"PhysicalBuildDegen":   76,
		"FireBuildDegen":       43,
		"ChaosBuildDegen":      4,
		"NetLifeRegen":         99,
		"NetManaRegen":         111,
		"NetEnergyShieldRegen": 14,
	}
	for _, key := range keys {
		if agreed[key] < floors[key] {
			t.Errorf("%s agrees on %d of %d graded builds, below the floor of %d "+
				"(CalcDefence.lua:3339-3394)", key, agreed[key], seen[key], floors[key])
		}
	}
}
