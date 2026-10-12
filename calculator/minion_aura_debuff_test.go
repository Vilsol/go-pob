package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMinionAuraDebuffReachesEnemy pins CalcPerform.lua:2889-2935.
func TestMinionAuraDebuffReachesEnemy(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "1342-necromancer-flame-dash.xml")
	rows := env.EnemyModDB.Tabulate(mod.TypeMore, nil, "PhysicalDamageTaken")
	got := make([]float64, 0, len(rows))
	found := false
	for _, r := range rows {
		got = append(got, r.Value)
		if r.Value == 34 {
			found = true
		}
	}
	if !found {
		t.Errorf("enemy PhysicalDamageTaken MORE rows = %v, want one at 34 (PoB dump)", got)
	}
}
