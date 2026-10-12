package calculator

import (
	"math"
	"os"
	"testing"
)

// TestUniqueItemTriggerRung pins CalcTriggers.lua:1596/:1606.
func TestUniqueItemTriggerRung(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	for build, want := range map[string]float64{
		"0772-slayer-starfall.xml":  4.4689637735792,
		"0786-saboteur-unknown.xml": 7.9456,
	} {
		env := corpusEnvForLoadedData(t, build)
		if got := env.Player.Output["Speed"]; math.Abs(got-want) > want*1e-6 {
			t.Errorf("%s Speed = %v, PoB has %v", build, got, want)
		}
	}
}
