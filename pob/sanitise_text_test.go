package pob

import "testing"

// TestSanitiseTextDropsZeroWidthSpace: Common.lua:318; the pin's sanitiseText gives "+6 to Maximum -Rage".
func TestSanitiseTextDropsZeroWidthSpace(t *testing.T) {
	if got := SanitiseText("+6​ to Maximum –Rage"); got != "+6 to Maximum -Rage" {
		t.Errorf("%q", got)
	}
}
