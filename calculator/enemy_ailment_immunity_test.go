package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestEnemyAilmentImmunityGatesTheAilmentLoop pins the two negated conjuncts of CalcPerform.lua:3500-3502 that this port had dropped.
func TestEnemyAilmentImmunityGatesTheAilmentLoop(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build         string
		wantChill     float64
		wantSkillTime float64
		role          string
	}{
		{"0161-assassin-viper-strike-of-the-mamba", 0, 0.7, "immune: no chill, so the bare 700ms interval"},
		{"0013-inquisitor-divine-ire", -10, 0.7 / 0.9, "chillable: the ailment must still land"},
		{"0004-trickster-lightning-tendrils-of-eccentricity", -13, 0.7 / 0.87, "chillable at 13, not the 10 default"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), tc.build+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			build, err := builds.ParseBuild(xml)
			if err != nil {
				t.Fatal(err)
			}
			env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
			if env == nil || env.Enemy == nil {
				t.Fatal("no env")
			}

			got := float64(0)
			for _, m := range env.Enemy.ModDB.Mods["ActionSpeed"] {
				if m.GetSource() == mod.Source("Chill") {
					got += m.Value().Float()
				}
			}
			if got != tc.wantChill {
				t.Errorf("enemy ActionSpeed from Chill = %v, want %v (%s)", got, tc.wantChill, tc.role)
			}

			if st := env.Player.Output["enemySkillTime"]; math.Abs(st-tc.wantSkillTime) > tc.wantSkillTime*1e-6 {
				t.Errorf("enemySkillTime = %v, want %v (%s)", st, tc.wantSkillTime, tc.role)
			}
		})
	}
}
