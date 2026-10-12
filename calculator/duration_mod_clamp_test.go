package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/calculator/calclib"
)

// TestDurationModClampsAtZero pins CalcOffence.lua:1463: a net -101% duration clamps DurationMod at 0.
func TestDurationModClampsAtZero(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const name = "5244-ladder-LATERi_2248.OnemanGroupPlay.xml"
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	want, ok := oracle[name].Output["DurationMod"]
	if !ok {
		t.Fatalf("REFUSING: PoB emits no DurationMod for %s", name)
	}
	env := corpusEnvForLoadedData(t, name)
	s := env.Player.MainSkill
	if raw := calclib.Mod(s.SkillModList, s.SkillCfg, "Duration", "PrimaryDuration"); raw >= 0 {
		t.Fatalf("REFUSING: the unclamped duration mod is %v, so the clamp is not exercised", raw)
	}
	if got := env.Player.Output["DurationMod"]; got != want {
		t.Errorf("DurationMod = %v, PoB %v", got, want)
	}
}
