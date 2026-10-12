package calculator

import "testing"

// TestMaxStagesSetBeforeBuffs pins CalcPerform.lua:2226-2254.
func TestMaxStagesSetBeforeBuffs(t *testing.T) {
	assertBuildStats(t, "15418-variant-10008.scorching-ray-part2.xml", []string{"FireDotEffMult", "FireDot"}, nil)
}
