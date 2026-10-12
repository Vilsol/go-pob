package calculator

import "testing"

// TestVaalTransfiguredSecondaryKeepsVaalTags pins Data.lua:1098-1117: a transfigured Vaal gem is one gem row whose tags are the Vaal gem's.
func TestVaalTransfiguredSecondaryKeepsVaalTags(t *testing.T) {
	assertBuildStats(t, "0634-reliquarian-vaal-reap-of-butchery.xml", []string{"GemLevel", "AverageHit", "TotalDot"}, nil)
}
