package calculator

import (
	"os"
	"testing"
)

// TestLifeOverrideWins pins the term dropped from CalcPerform.lua:88/90.
//
//	local override = modDB:Override(nil, "Life")
//	output.Life = override or m_max(round(base * (1 + inc/100) * more * (1 - conv/100)), 1)
func TestLifeOverrideWins(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0019-champion-herald-of-agony.xml")
	if env.Minion == nil || env.Minion.Actor == nil {
		t.Fatal("fixture is useless: this build no longer grants a minion actor")
	}
	if ov := env.Minion.ModDB.Override(nil, "Life"); ov == nil {
		t.Fatal("fixture is useless: the minion no longer carries a Life OVERRIDE, so the arm " +
			"under test is never reached and this test would pass vacuously")
	}

	if got := env.Minion.Output["Life"]; got != 1 {
		t.Errorf("minion Life = %v, want 1 (PoB: minionOutput.Life) — CalcPerform.lua:88 reads "+
			"Override(nil, \"Life\") and :90 lets it win outright", got)
	}

	if got := env.Player.Output["Life"]; got != 2095 {
		t.Errorf("CONTROL: player Life = %v, want 2095 (PoB) — the non-override path has been "+
			"disturbed", got)
	}
}
