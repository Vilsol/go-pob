package calculator

import "testing"

// TestMainModeBuildsBreakdowns pins CalcSetup.lua:500 and CalcPerform.lua:1514-1522.
func TestMainModeBuildsBreakdowns(t *testing.T) {
	assertPlayerOutputs(t, "0002-pathfinder-unknown.xml",
		map[string]float64{"TinctureLimit": 1, "FlaskEffect": 15, "LifeFlaskChargeGen": 1})
	assertPlayerOutputs(t, "0020-assassin-storm-burst.xml",
		map[string]float64{"IgniteDotMulti": 1.4990930959390001})
}
