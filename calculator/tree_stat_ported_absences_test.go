package calculator

import (
	"strings"
	"testing"
)

// TestRunegraftLimitLinesStayDropped closes a question that has been reopened three times.
func TestRunegraftLimitLinesStayDropped(t *testing.T) {
	for _, line := range []string{
		"Limited to 1 Runegraft of the Fortress",
		"Limited to 1 Runegraft of Treachery",
		"Limited to 1 Runegraft of the Warp",
	} {
		if e := ParseMod(line, false); strings.TrimSpace(e.Extra) == "" {
			t.Errorf("%q now parses whole (Extra empty). PoB's ModParser has no rule for it at "+
				"this pin, so we would be emitting a modifier the oracle does not have.", line)
		}
	}

	const control = "+10 to Strength"
	if e := ParseMod(control, false); strings.TrimSpace(e.Extra) != "" {
		t.Errorf("control: %q left Extra=%q — this test cannot tell a ported absence from a "+
			"broken parser", control, e.Extra)
	}
}
