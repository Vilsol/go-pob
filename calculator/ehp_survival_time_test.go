package calculator

import (
	"os"
	"testing"
)

// TestEHPSurvivalTimeKeyCasing pins CalcDefence.lua:2922.
func TestEHPSurvivalTimeKeyCasing(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0001-necromancer-unknown.xml")
	out := env.Player.Output

	got, ok := out["EHPSurvivalTime"]
	if !ok {
		t.Fatal("EHPSurvivalTime absent; PoB publishes it on 1620 corpus builds")
	}
	if _, stale := out["EHPsurvivalTime"]; stale {
		t.Error("EHPsurvivalTime (lowercase s) is still written — PoB's key is EHPSurvivalTime, " +
			"and publishing both would let a consumer keep reading the wrong one")
	}
	if want := out["TotalNumberOfHits"] * out["enemySkillTime"]; got != want {
		t.Errorf("EHPSurvivalTime = %v, want TotalNumberOfHits * enemySkillTime = %v", got, want)
	}
}
