package calculator

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
)

const nanPairsPrintCap = 60

func formatNaNPairs(pairs []string, limit int) string {
	if len(pairs) == 0 {
		return ""
	}
	sorted := append([]string(nil), pairs...)
	sort.Strings(sorted)

	var b strings.Builder
	fmt.Fprintf(&b, "\nNaN/Inf PAIRS (%d)\n", len(sorted))
	shown := sorted
	if limit > 0 && len(shown) > limit {
		shown = shown[:limit]
	}
	for _, p := range shown {
		fmt.Fprintf(&b, "  %s\n", strings.ReplaceAll(p, "\t", "  "))
	}
	if len(shown) < len(sorted) {
		fmt.Fprintf(&b, "  ... %d more not listed (cap %d)\n", len(sorted)-len(shown), limit)
	}
	return b.String()
}

func TestFormatNaNPairsNamesEveryPairUnderTheCap(t *testing.T) {
	got := formatNaNPairs([]string{"b2\tStatB", "b1\tStatA"}, 60)
	testza.AssertContains(t, got, "NaN/Inf PAIRS (2)")
	testza.AssertContains(t, got, "b1  StatA")
	testza.AssertContains(t, got, "b2  StatB")
	testza.AssertTrue(t, strings.Index(got, "b1  StatA") < strings.Index(got, "b2  StatB"))
	testza.AssertNotContains(t, got, "not listed")
}

// TestFormatNaNPairsSaysWhatItDropped: The elision line is the load-bearing half: without it the listing's length reads as the whole class.
func TestFormatNaNPairsSaysWhatItDropped(t *testing.T) {
	pairs := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		pairs = append(pairs, fmt.Sprintf("build%d\tStat", i))
	}
	got := formatNaNPairs(pairs, 2)
	testza.AssertContains(t, got, "NaN/Inf PAIRS (5)")
	testza.AssertContains(t, got, "... 3 more not listed (cap 2)")
	testza.AssertNotContains(t, got, "build4")
}

// TestFormatNaNPairsIsEmptyWhenThereAreNone: An empty result must be EMPTY, not a header over nothing.
func TestFormatNaNPairsIsEmptyWhenThereAreNone(t *testing.T) {
	testza.AssertEqual(t, "", formatNaNPairs(nil, 60))
}
