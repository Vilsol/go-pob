package calculator

import (
	"math"
	"testing"
)

// TestConfigSkillNameIncludesTransfigured: perforateSpikeOverlap reaches Perforate of Duality (ConfigOptions.lua:555-557).
func TestConfigSkillNameIncludesTransfigured(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0235-champion-sunder.xml")
	want := 21478916.214712
	if got := env.Player.Output["TotalDPS"]; math.Abs(got-want) > want*1e-6 {
		t.Errorf("TotalDPS = %v, want %v (PoB)", got, want)
	}
}
