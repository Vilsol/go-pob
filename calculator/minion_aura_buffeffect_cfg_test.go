package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestMinionAuraBuffEffectHonoursSkillCfg pins CalcPerform.lua:2428-2429.
func TestMinionAuraBuffEffectHonoursSkillCfg(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	dir := corpusDirForLoadedData(t)

	stats := []string{"ColdResist", "ColdResistTotal", "FireResist", "LightningResist", "AverageDamage"}

	for _, name := range []string{
		"6647-ladder-Yulana_6514.Lauritas_CotA",
		"3575-ladder-keklik223_2656.ALLF_ABOBA",
		"3656-ladder-nguyenmanh16111988_9826.AllFlame_Duelist_Chaos",
		"6754-ladder-abuseritualpoe2_3043.aurasixsevenyoloyvl",
	} {
		t.Run(name, func(t *testing.T) {
			rec, ok := oracle[name+".xml"]
			if !ok {
				t.Fatalf("build not in the oracle blob")
			}
			file, err := os.ReadFile(filepath.Join(dir, name+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			pob, err := builds.ParseBuild(file)
			if err != nil {
				t.Fatal(err)
			}
			env := (&Calculator{PoB: pob}).BuildOutput(OutputModeMain)
			if env.Minion == nil {
				t.Fatalf("no minion: the fixture no longer exercises the ally aura arm")
			}

			checked := 0
			for _, stat := range stats {
				want, ok := rec.Minion[stat]
				if !ok || want == 0 {
					continue
				}
				checked++
				got := env.Minion.Actor.Output[stat]
				if math.Abs(got-want)/math.Abs(want) > oracleDumpTolerance {
					t.Errorf("Minion.%s = %v, PoB %v", stat, got, want)
				}
			}
			if checked == 0 {
				t.Fatalf("none of %v is present and nonzero in the oracle record", stats)
			}
		})
	}
}
