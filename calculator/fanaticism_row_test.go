package calculator

import (
	"testing"
)

// TestGainFanaticismRowTakesAnyDuration pins ModParser.lua:2904, whose key captures the duration.
func TestGainFanaticismRowTakesAnyDuration(t *testing.T) {
	for _, line := range []string{
		"Gain Fanaticism for 5 seconds on reaching Maximum Fanatic Charges",
		"Gain Fanaticism for 4 seconds on reaching Maximum Fanatic Charges",
	} {
		e := ParseMod(line, false)
		if e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "Condition:CanGainFanaticism" {
			t.Errorf("%q: remainder %q, mods %v", line, e.Extra, e.ModList)
		}
	}
}
