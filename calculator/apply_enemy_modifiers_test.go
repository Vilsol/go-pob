package calculator

import (
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestEnemyModifiersApplyBeforeCurses pins CalcPerform.lua:1314-1318.
func TestEnemyModifiersApplyBeforeCurses(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0929-luminary-animate-guardian-8.xml")
	var got []float64
	for _, r := range env.EnemyModDB.Tabulate(mod.TypeBase, nil, "SelfCritMultiplier") {
		if r.Mod.GetSource() == "Skill:AssassinsMark" {
			got = append(got, r.Value)
		}
	}
	if len(got) != 1 || got[0] != 57 {
		t.Errorf("Assassin's Mark SelfCritMultiplier = %v, want one mod at 57 (PoB dump)", got)
	}
	found := false
	for _, r := range env.EnemyModDB.Tabulate(mod.TypeIncrease, nil, "CurseEffectOnSelf") {
		if strings.Contains(string(r.Mod.GetSource()), "Dying Breath, Iron Staff") {
			found = true
		}
	}
	if !found {
		t.Errorf("enemy CurseEffectOnSelf carries no Dying Breath source; applyEnemyModifiers' setSource is missing")
	}

	assertBuildStats(t, "10114-ladder-GhostOfItachi_1415.SimpingForMercss.xml",
		[]string{"ColdEnemyDamage", "FireEnemyDamage"}, nil)
}
