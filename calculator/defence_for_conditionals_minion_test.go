package calculator

import (
	"os"
	"testing"
)

// TestDefenceForConditionalsRunsForTheMinion pins CalcPerform.lua:3419-3421.
func TestDefenceForConditionalsRunsForTheMinion(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, build := range []string{
		"0342-luminary-animate-guardian.xml",
		"0582-luminary-animate-guardian-2.xml",
		"0662-luminary-animate-guardian-of-smiting.xml",
	} {
		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			if env.Minion == nil {
				t.Fatalf("REFUSING: %s has no minion", build)
			}

			want := map[string]float64{}
			for _, slot := range []string{"Helmet", "Gloves", "Boots", "Body Armour", "Weapon 2", "Weapon 3"} {
				item := env.Minion.Actor.ItemList[slot]
				if item == nil || item.ArmourData == nil {
					continue
				}
				if item.ArmourData.Armour > 0 {
					want["ArmourOn"+slot] = item.ArmourData.Armour
				}
				if item.ArmourData.EnergyShield > 0 {
					want["EnergyShieldOn"+slot] = item.ArmourData.EnergyShield
				}
				if item.ArmourData.Evasion > 0 {
					want["EvasionOn"+slot] = item.ArmourData.Evasion
				}
			}
			if len(want) == 0 {
				t.Fatalf("REFUSING: %s's minion carries no armour in any slot — the fixture "+
					"cannot exercise the minion call site", build)
			}

			for key, value := range want {
				got, present := env.Minion.Actor.Output[key]
				if !present {
					t.Errorf("%s: the minion's %s was never published; its item carries %v",
						build, key, value)
					continue
				}
				if got != value {
					t.Errorf("%s: the minion's %s = %v, its item carries %v", build, key, got, value)
				}
			}
		})
	}
}
