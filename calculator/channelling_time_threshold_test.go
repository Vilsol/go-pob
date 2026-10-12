package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestChannellingTimeThresholdRowsParse pins ModParser.lua:1679-1680.
func TestChannellingTimeThresholdRowsParse(t *testing.T) {
	for _, tc := range []struct {
		line      string
		name      string
		threshold float64
	}{
		{"Gain Arcane Surge after Channelling for 1 second", "Condition:ArcaneSurge", 1},
		{"Gain Arcane Surge after Channelling for 2 seconds", "Condition:ArcaneSurge", 2},
		{"+20% to Critical Strike Multiplier if you've been Channelling for at least 1 second", "CritMultiplier", 1},
		{"+20% to Critical Strike Multiplier if you've been Channelling for at least 3 seconds", "CritMultiplier", 3},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v mods, want 1", tc.line, e)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — processModLine discards the whole line", e.Extra)
			}
			if got := e.ModList[0].Name(); got != tc.name {
				t.Errorf("mod name = %q, want %q", got, tc.name)
			}
			var found *mod.MultiplierThresholdTag
			for _, tag := range e.ModList[0].Tags() {
				if mt, ok := tag.(*mod.MultiplierThresholdTag); ok {
					found = mt
				}
			}
			if found == nil {
				t.Fatalf("no MultiplierThresholdTag on %q; tags = %v", tc.line, e.ModList[0].Tags())
			}
			if found.Variable != "ChannellingTime" {
				t.Errorf("tag var = %q, want ChannellingTime", found.Variable)
			}
			if found.TagThreshold == nil || *found.TagThreshold != tc.threshold {
				t.Errorf("tag threshold = %v, want %v", found.TagThreshold, tc.threshold)
			}
		})
	}

	e := ParseMod("10% increased Damage while Channelling", false)
	if e == nil || len(e.ModList) != 1 {
		t.Fatalf("control line produced %v", e)
	}
	for _, tag := range e.ModList[0].Tags() {
		if _, ok := tag.(*mod.MultiplierThresholdTag); ok {
			t.Errorf("`while channelling` must stay a Condition tag, got a MultiplierThreshold")
		}
	}
}
