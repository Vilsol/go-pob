package calculator

import "testing"

// TestDecayDPSUsesDecayCfg pins CalcOffence.lua:5599-5602: Decay's MORE is unrounded and its dot multiplier reads the decay cfg.
func TestDecayDPSUsesDecayCfg(t *testing.T) {
	assertBuildStats(t, "12810-ladder-Zeblain_5106.Oli_Where_CA_Buffs.xml", []string{"DecayDPS"}, nil)
}
