package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator/calclib"
)

// TestEnemyCritEffectMatchesPoB: TestEnemyArmourEvasionMatchesPoB pins the enemy's base Armour and Evasion against the values the pinned PoB actually built.
func TestEnemyCritEffectMatchesPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	cases := []struct {
		build string
		note  string
		want  float64
	}{
		{"4243-ladder-BoBricek_5410.BoBricekScion", "no <Input>", 1.0105},
		{"0549-guardian-tempest-shield", "<Input> 0", 1},
		{"1496-elementalist-winter-orb-34", "<Input> 6", 1.018},
	}

	dir := corpusDirForLoadedData(t)
	for _, tc := range cases {
		t.Run(tc.build, func(t *testing.T) {
			file, err := os.ReadFile(filepath.Join(dir, tc.build+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			build, err := builds.ParseBuild(file)
			if err != nil {
				t.Fatal(err)
			}
			env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
			got := env.Player.Output["EnemyCritEffect"]
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("%s: EnemyCritEffect = %v, PoB dumps %v", tc.note, got, tc.want)
			}
		})
	}
}

// TestBossAddsNoCurseEffectOnSelf pins the ABSENCE of a Boss-sourced CurseEffectOnSelf on the enemy.
func TestBossAddsNoCurseEffectOnSelf(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	dir := corpusDirForLoadedData(t)
	for _, tc := range []struct{ build, isBoss string }{
		{"15363-pathfinder-caustic-arrow", "Boss"},
		{"0005-occultist-vaal-double-strike", "Pinnacle"},
		{"1550-chieftain-ethereal-knives-4", "Uber"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			file, err := os.ReadFile(filepath.Join(dir, tc.build+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			build, err := builds.ParseBuild(file)
			if err != nil {
				t.Fatal(err)
			}
			env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
			if got := env.EnemyModDB.More(nil, "CurseEffectOnSelf"); got != 1 {
				t.Errorf("enemyIsBoss=%s enemy CurseEffectOnSelf MORE = %v, PoB applies none (1)",
					tc.isBoss, got)
			}
		})
	}
}

func TestEnemyArmourEvasionMatchesPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	cases := []struct {
		build   string
		isBoss  string
		armour  float64
		evasion float64
	}{
		{"0061-trickster-spark-of-the-nova-3", "None", 44591, 8388},
		{"15363-pathfinder-caustic-arrow", "Boss", 44591, 8388},
		{"0005-occultist-vaal-double-strike", "Pinnacle", 61989, 10143},
		{"1550-chieftain-ethereal-knives-4", "Uber", 55739, 9778},
	}

	dir := corpusDirForLoadedData(t)
	for _, tc := range cases {
		t.Run(tc.build, func(t *testing.T) {
			file, err := os.ReadFile(filepath.Join(dir, tc.build+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			build, err := builds.ParseBuild(file)
			if err != nil {
				t.Fatal(err)
			}
			env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)

			for _, stat := range []struct {
				name string
				want float64
			}{{"Armour", tc.armour}, {"Evasion", tc.evasion}} {
				if got := calclib.Val(env.EnemyModDB, stat.name); got != stat.want {
					t.Errorf("enemyIsBoss=%s enemy %s = %v, PoB dumps %v",
						tc.isBoss, stat.name, got, stat.want)
				}
			}
		})
	}
}
