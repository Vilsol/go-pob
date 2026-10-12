package calculator

import (
	"math"
	"testing"
)

// TestWarcryUptimeUsesCombinedSpeed: exert uptime divides by globalOutput.Speed, the hands' combined speed (CalcOffence.lua:2444, :2598, :2644).
func TestWarcryUptimeUsesCombinedSpeed(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1852-ladder-Nyiou_7641.Nyu_Clutch.xml")
	for k, want := range map[string]float64{"RallyingUpTimeRatio": 51.376621906495, "IntimidatingUpTimeRatio": 24.494712666531} {
		if got := env.Player.Output[k]; math.Abs(got-want) > 1e-6 {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}
