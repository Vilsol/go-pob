package calculator

import (
	"testing"
)

// TestMinionSkillBuffMergesModListTwice: a minion skill's ally buff is merged scaled and then unscaled.
func TestMinionSkillBuffMergesModListTwice(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1292-luminary-summon-reaper-of-revenants.xml")
	for k, want := range map[string]float64{"Speed": 2.9461077844311, "TotalDPS": 11020041.255833} {
		if got := env.Minion.Output[k]; !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("Minion %s = %v, want %v (PoB)", k, got, want)
		}
	}
}
