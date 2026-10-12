package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator/calclib"
	"github.com/Vilsol/go-pob/data"
)

// TestUberEnemyEvasionUsesEvasionMean pins the Uber arm of the enemyIsBoss placeholder writer (ConfigOptions.lua:2193-2194) against the PINNED 3.29 PoB.
func TestUberEnemyEvasionUsesEvasionMean(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	if data.BossStats.UberArmourMean == data.BossStats.UberEvasionMean {
		t.Fatalf("UberArmourMean == UberEvasionMean == %v: this test can no longer tell the arms apart",
			data.BossStats.UberArmourMean)
	}

	const build = "0369-necromancer-automation"
	want := math.Floor(data.MonsterEvasionTable[85]*(data.BossStats.UberEvasionMean/100) + 0.5)

	file, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	pob, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatal(err)
	}
	env := (&Calculator{PoB: pob}).BuildOutput(OutputModeMain)

	if got := calclib.Val(env.EnemyModDB, "Evasion"); got != want {
		t.Errorf("enemyIsBoss=Uber enemy Evasion = %v, PoB %v (= round(%v * UberEvasionMean/100))",
			got, want, data.MonsterEvasionTable[85])
	}
}
