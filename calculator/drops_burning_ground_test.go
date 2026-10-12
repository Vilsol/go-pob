package calculator

import "testing"

// TestDropsBurningGround pins ModParser.lua:3796 and CalcOffence.lua:5619-5645 (Sol Spur's burning ground).
func TestDropsBurningGround(t *testing.T) {
	assertBuildStats(t, "2617-ladder-Xen0s_1227.StopNerfingVaalTempleQQ.xml", []string{"BurningGroundDPS", "TotalDotDPS"}, nil)
}
