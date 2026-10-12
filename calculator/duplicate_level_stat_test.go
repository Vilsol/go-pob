package calculator

import "testing"

// TestDuplicateLevelStatsKeepTheirColumns.
func TestDuplicateLevelStatsKeepTheirColumns(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1192-elementalist-reap-8.xml")
	if got := env.Player.Output["ChainMax"]; got != 11 {
		t.Errorf("ChainMax = %v, want 11 (PoB)", got)
	}
}
