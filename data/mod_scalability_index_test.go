package data

import (
	"maps"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
)

// TestScalabilityIndexMatchesTheNumberPattern: the index skips the number pattern on keys without a digit.
func TestScalabilityIndexMatchesTheNumberPattern(t *testing.T) {
	want := map[string][]string{}
	digits := 0
	for key := range raw.ModScalability {
		b := scalabilityKey(scalabilityNumberRe.ReplaceAllString(key, "#"))
		want[b] = append(want[b], key)
		if b != scalabilityKey(key) {
			digits++
		}
	}
	if digits == 0 {
		t.Fatal("REFUSING: no key carries a number, so the pattern's arm is never compared")
	}
	got := scalabilityIndex()
	if !slices.Equal(slices.Sorted(maps.Keys(got)), slices.Sorted(maps.Keys(want))) {
		t.Fatalf("index keys differ: %d vs %d", len(got), len(want))
	}
	for b, keys := range want {
		if !slices.Equal(slices.Sorted(slices.Values(got[b])), slices.Sorted(slices.Values(keys))) {
			t.Fatalf("%q: %q, want %q", b, got[b], keys)
		}
	}
}
