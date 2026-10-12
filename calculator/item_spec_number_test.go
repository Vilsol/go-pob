package calculator

import (
	"testing"
)

// TestItemSpecNumberTakesLeadingPrefix pins Item.lua:330-333.
//
//	local function specToNumber(s)
//		local n = s:match("^([%+%-]?[%d%.]+)")
//	end
func TestItemSpecNumberTakesLeadingPrefix(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want float64
		ok   bool
	}{
		{"+20%", 20, true},
		{"+20% (augmented)", 20, true},
		{"12min - 24max ( IMPORTANT! )", 12, true},
		{"-5%", -5, true},
		{"20 (augmented)", 20, true},
		{"20", 20, true},
		{"83", 83, true},
		{"1.5", 1.5, true},
		{"Large", 0, false},
		{"", 0, false},
		{"(augmented) 20", 0, false},
	} {
		got, ok := specToNumber(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("specToNumber(%q) = (%v, %v), want (%v, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
