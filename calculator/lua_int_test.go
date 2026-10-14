package calculator

import (
	"fmt"
	"math"
	"testing"
)

// TestLuaInt: LuaJIT x86's string.format("%d", x), which the oracle prints, for the values Go's int conversion leaves platform-defined.
func TestLuaInt(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{math.NaN(), "-9223372036854775808"},
		{math.Inf(1), "-9223372036854775808"},
		{math.Inf(-1), "-9223372036854775808"},
		{1e300, "-9223372036854775808"},
		{-1e300, "-9223372036854775808"},
		{2.9, "2"},
		{-2.9, "-2"},
		{0, "0"},
		{1234567, "1234567"},
	} {
		if got := fmt.Sprintf("%d", luaInt(tc.in)); got != tc.want {
			t.Errorf("luaInt(%v) prints %s, want %s", tc.in, got, tc.want)
		}
	}
}
