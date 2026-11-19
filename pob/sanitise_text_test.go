package pob

import "testing"

// TestSanitiseTextDropsZeroWidthSpace: Common.lua:318; the pin's sanitiseText gives "+6 to Maximum -Rage".
func TestSanitiseTextDropsZeroWidthSpace(t *testing.T) {
	if got := SanitiseText("+6​ to Maximum –Rage"); got != "+6 to Maximum -Rage" {
		t.Errorf("%q", got)
	}
}

// Common.lua:311 strips tags with the balanced pattern %b<>.
func TestSanitiseTextStripsNestedTags(t *testing.T) {
	for in, want := range map[string]string{
		"<<set:MS>><<set:M>><<set:S>>Beast Grinder": "Beast Grinder",
		"<a<b>>x":   "x",
		"<open x":   "<open x",
		"a<b>c<d>e": "ace",
	} {
		if got := SanitiseText(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}
