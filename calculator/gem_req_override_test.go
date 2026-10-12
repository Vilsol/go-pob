package calculator

import "testing"

// TestIgnoreSocketedGemRequirements pins ModParser.lua:2365 and CalcSetup.lua:322.
func TestIgnoreSocketedGemRequirements(t *testing.T) {
	assertBuildStats(t, "3577-ladder-ketz3r_4992.NiceToFreezeYou.xml", []string{"ReqStr", "ReqDex", "ReqInt"}, nil)
}
