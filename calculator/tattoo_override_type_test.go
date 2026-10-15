package calculator

import "testing"

// TestTattooTypeMultipliers pins CalcSetup.lua:691-695: one multiplier per allocated tattoo overrideType.
func TestTattooTypeMultipliers(t *testing.T) {
	env := (&Calculator{PoB: loadCorpusBuild(t, "15773-pathfinder-vaal-spark.xml")}).BuildOutput(OutputModeMain)
	for name, want := range map[string]float64{"RamakoTattoo": 5, "UnknownTattoo": 24} {
		if got := env.ModDB.Multipliers[name]; got != want {
			t.Errorf("Multiplier %s = %v, want %v", name, got, want)
		}
	}
	if got, want := env.Player.Output["SpellSuppressionEffect"], 63.0; got != want {
		t.Errorf("SpellSuppressionEffect = %v, want %v", got, want)
	}
}
