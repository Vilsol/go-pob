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
	requireGameData(t, "3.20")

	cases := []struct {
		build string
		note  string
		want  float64
	}{
		{"0001-juggernaut-molten-strike", "no <Input>", 1},
		{"0287-occultist-frostbite-2", "<Input> 0", 1},
		{"0973-raider-cyclone", "<Input> -7 + INC", 0.99118},
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
	requireGameData(t, "3.20")

	dir := corpusDirForLoadedData(t)
	for _, tc := range []struct{ build, isBoss string }{
		{"0001-juggernaut-molten-strike", "Boss"},
		{"0002-slayer-flicker-strike", "Pinnacle"},
		{"0020-deadeye-fireball", "Uber"},
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
	requireGameData(t, "3.20")

	cases := []struct {
		build   string
		isBoss  string
		armour  float64
		evasion float64
	}{
		{"0007-ascendant-hexblast", "None", 17268, 8388},
		{"0001-juggernaut-molten-strike", "Boss", 17268, 8388},
		{"0002-slayer-flicker-strike", "Pinnacle", 24752, 10143},
		{"0020-deadeye-fireball", "Uber", 23830, 11575},
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
