package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestProjectileKeystoneRampsMatchThePin pins the DistanceRamp tables of the two projectile keystones against the PINNED PoB, digit for digit.
func TestProjectileKeystoneRampsMatchThePin(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  [][]float64
		want [][]float64
		lua  string
	}{
		{"PointBlank", pointBlankDistanceRamp, [][]float64{{10, 1}, {35, 0}, {120, -1}}, "CalcOffence.lua:1049"},
		{"FarShot", farShotDistanceRamp, [][]float64{{10, -0.2}, {25, 0}, {70, 0.6}}, "CalcOffence.lua:1052"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.got) != len(tc.want) {
				t.Fatalf("%s ramp has %d steps, pin (%s) has %d", tc.name, len(tc.got), tc.lua, len(tc.want))
			}
			for i := range tc.want {
				for j := range tc.want[i] {
					if tc.got[i][j] != tc.want[i][j] {
						t.Errorf("%s ramp[%d][%d] = %v, pin (%s) has %v",
							tc.name, i, j, tc.got[i][j], tc.lua, tc.want[i][j])
					}
				}
			}
		})
	}
}

// TestProjectileKeystoneRampsEvaluate is the behavioural half.
func TestProjectileKeystoneRampsEvaluate(t *testing.T) {
	for _, tc := range []struct {
		name string
		ramp [][]float64
		base float64
		dist float64
		want float64
	}{
		{"PointBlank at its floor", pointBlankDistanceRamp, 30, 120, -30},
		{"FarShot at its zero crossing", farShotDistanceRamp, 100, 25, 0},
		{"CONTROL: PointBlank inside its flat head", pointBlankDistanceRamp, 30, 10, 30},
		{"CONTROL: FarShot at its far end", farShotDistanceRamp, 100, 70, 60},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := moddb.NewModDB()
			db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, tc.base).Tag(mod.DistanceRamp(tc.ramp)))
			got := db.Sum(mod.TypeIncrease, &moddb.ListCfg{SkillDist: tc.dist, SkillDistSet: true}, "Damage")
			if got != tc.want {
				t.Errorf("at distance %v: got %v, want %v", tc.dist, got, tc.want)
			}
		})
	}
}
