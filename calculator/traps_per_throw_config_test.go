package calculator

import (
	"math"
	"os"
	"testing"
)

// TestTrapsPerThrowConfig pins ConfigOptions.lua:1587-1589.
func TestTrapsPerThrowConfig(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1634-pathfinder-seismic-trap-of-swells.xml")
	for key, want := range map[string]float64{"TrapThrowCount": 3, "Speed": 5.45} {
		if got := env.Player.Output[key]; math.Abs(got-want) > want*1e-6 {
			t.Errorf("%s = %v, PoB has %v", key, got, want)
		}
	}
}
