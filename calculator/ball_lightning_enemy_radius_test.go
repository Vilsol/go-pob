package calculator

import "testing"

// TestBallLightningStrikesInRange pins act_int.lua:1086-1167 (upstream #10274).
func TestBallLightningStrikesInRange(t *testing.T) {
	assertPlayerOutputs(t, "1140-hierophant-ball-lightning.xml", map[string]float64{"NormalHitsPerCast": 13})
	assertPlayerOutputs(t, "0482-luminary-intuitive-link.xml", map[string]float64{"NormalHitsPerCast": 7})
}
