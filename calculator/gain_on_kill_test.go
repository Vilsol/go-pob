package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestGainOnHitAndKillGates pins CalcOffence.lua:3669-3692: CannotRecoverLifeOutsideLeech zeroes life on hit and on kill.
func TestGainOnHitAndKillGates(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	for _, name := range []string{"1066-slayer-wild-strike.xml", "3845-ladder-김도_8030.PELEL.xml"} {
		t.Run(name, func(t *testing.T) {
			rec, ok := oracle[name]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", name)
			}
			want, ok := rec.Output["LifeOnKill"]
			if !ok {
				t.Fatalf("REFUSING: PoB emits no LifeOnKill for %s", name)
			}
			env := corpusEnvForLoadedData(t, name)
			if env.Player.MainSkill.SkillModList.Sum(mod.TypeBase, nil, "LifeOnKill") == 0 {
				t.Fatalf("REFUSING: %s carries no LifeOnKill to gate", name)
			}
			if got := env.Player.Output["LifeOnKill"]; !oracleClose(got, want, oracleDumpTolerance) {
				t.Errorf("LifeOnKill = %g, PoB %g", got, want)
			}
		})
	}
}
