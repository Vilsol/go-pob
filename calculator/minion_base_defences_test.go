package calculator

import (
	"math"
	"os"
	"testing"
)

// TestMinionBaseArmourEvasionFromMonsterTables pins CalcPerform.lua:1139-1140.
//
//	minion.modDB:NewMod("Armour",  "BASE", round(env.data.monsterArmourTable[minion.level]  * (minion.minionData.armour  or 1)), "Base")
//	minion.modDB:NewMod("Evasion", "BASE", round(env.data.monsterEvasionTable[minion.level] * (minion.minionData.evasion or 1)), "Base")
func TestMinionBaseArmourEvasionFromMonsterTables(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	cases := []struct {
		build                string
		level                int
		wantArmour, wantEvas float64
		note                 string
	}{
		{"0081-elementalist-summon-ice-golem.xml", 68, 12009, 4739, "Ice Golem, no multipliers"},
		{"0645-elementalist-summon-stone-golem.xml", 69, 12989, 1963, "Stone Golem, evasion multiplier 0.4"},
		{"0667-guardian-summon-raging-spirit.xml", 4, 23, 124, "Raging Spirit at level 4"},
	}

	for _, tc := range cases {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			if env.Minion == nil {
				t.Fatal("no minion actor built; PoB builds one for this fixture")
			}
			if got := env.Minion.Level; got != tc.level {
				t.Fatalf("premise broken: minion level = %d, PoB %d — this fixture was picked "+
					"because the levels already agree; re-pick it rather than tuning the table",
					got, tc.level)
			}
			for _, k := range []struct {
				name string
				want float64
			}{
				{"Armour", tc.wantArmour},
				{"Evasion", tc.wantEvas},
			} {
				got, ok := env.Minion.Output[k.name]
				if !ok {
					t.Errorf("Minion.%s absent", k.name)
					continue
				}
				if math.Abs(got-k.want) > 0.5 {
					t.Errorf("Minion.%s = %v, PoB %v (%s)", k.name, got, k.want, tc.note)
				}
			}
		})
	}
}
