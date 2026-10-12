package calculator

import (
	"testing"
)

// TestCaneOfKulemakScalesUnveiledLines: Cane of Kulemak's explicits count as unveiled (Item.lua:1369-1376), so its magnitude line scales them (:1399-1419).
func TestCaneOfKulemakScalesUnveiledLines(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0012-occultist-unknown.xml")
	for k, want := range map[string]float64{"TotalDotDPS": 2184602.4626516, "DurationMod": 0.9804} {
		if got := env.Player.Output[k]; !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}
