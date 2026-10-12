package calculator

import (
	"strings"
	"testing"
)

// TestStunThresholdFromEnergyShieldParses pins ModParser.lua:4973-4980.
func TestStunThresholdFromEnergyShieldParses(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 ||
		strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line does not parse")
	}

	for _, tc := range []struct {
		line      string
		wantNames []string
	}{
		{"Stun Threshold is based on Energy Shield instead of Life",
			[]string{"StunThresholdBasedOnEnergyShieldInsteadOfLife", "StunThresholdEnergyShieldPercent"}},
		{"Stun Threshold is based on 50% of your Energy Shield instead of Life",
			[]string{"StunThresholdBasedOnEnergyShieldInsteadOfLife", "StunThresholdEnergyShieldPercent"}},
	} {
		e := ParseMod(tc.line, false)
		if e == nil {
			t.Errorf("no entry: %s", tc.line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q (line is DROPPED WHOLE): %s", extra, tc.line)
			continue
		}
		var got []string
		for _, m := range e.ModList {
			got = append(got, m.Name())
		}
		for _, want := range tc.wantNames {
			if !contains(got, want) {
				t.Errorf("produced %v, missing %q: %s", got, want, tc.line)
			}
		}
	}
}
